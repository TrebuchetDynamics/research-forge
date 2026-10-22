// One sandboxed Pi observation. The custodian/gateway is a different process.
// PI_PACKAGE points to the existing pinned Pi installation, never a new install.
import fs from 'node:fs';
import http from 'node:http';
import { pathToFileURL } from 'node:url';

const root = process.env.PI_PACKAGE;
const { createAgentSession, createExtensionRuntime, ModelRuntime, SessionManager, SettingsManager } = await import(pathToFileURL(`${root}/dist/index.js`));
const { registerSearchHub } = await import('file:///native.mjs');
const input = JSON.parse(fs.readFileSync('/work/input.json', 'utf8'));
const started = performance.now();
const deadline = Number(process.env.PILOT_DEADLINE_MS);
let toolCalls = 0, inputTokens = 0, outputTokens = 0, exhausted = false;
let answer = null;
let completeAccounting = true;
let lastUsage = null;
const accounted = new Set();
const generations = new Set();
const transcript = [];

function gateway(path, data) {
  return new Promise((resolve, reject) => {
    const request = http.request({ socketPath: '/gateway/gateway.sock', method: 'POST', path, headers: { 'content-type': 'application/json' } }, response => {
      let text = '';
      response.on('data', chunk => { text += chunk; });
      response.on('end', () => {
        try {
          const value = JSON.parse(text);
          if (response.statusCode !== 200) {
            if (value.error === 'budget_exhausted') exhausted = true;
            reject(new Error(value.error));
          }
          else resolve(value);
        } catch (error) { reject(error); }
      });
    });
    request.on('error', reject);
    request.end(JSON.stringify(data));
  });
}

function check() {
  if (!completeAccounting) throw new Error('incomplete LM accounting');
  if (exhausted || toolCalls >= input.caps.tool_calls || inputTokens + outputTokens >= input.caps.tokens || Date.now() >= deadline) {
    exhausted = true;
    throw new Error('budget_exhausted');
  }
}

const native = [];
let nativeLimit = 5;
registerSearchHub({ registerCommand() {}, registerTool(tool) { native.push(tool); } }, {
  env: {},
  fetchImpl: async url => {
    const result = await gateway('/native', { url: String(url), limit: nativeLimit });
    return new Response(Buffer.from(result.body, 'base64'), { status: 200 });
  },
});
// Jina is not exposed: its server-side nested HTTP attempts are opaque. Search
// publication-date enrichment is performed by the accounted trusted gateway.
const tools = process.env.PILOT_CONDITION === 'with_rforge' ? [{
  name: 'rforge_search', label: 'ResearchForge Search',
  description: 'Run actual rforge search batch with the registered provider mix and --continue-on-error. The trusted gateway applies the date cap, leakage checks, remaining budgets and first-429 quarantine. Returns engine-normalized deduplicated paper records plus manifest/failure summaries. Choose one problem-based query per call. No failed-pair resume.',
  parameters: { type: 'object', properties: { query: { type: 'string' }, limit: { type: 'integer', minimum: 1, maximum: 50 } }, required: ['query'], additionalProperties: false },
  async execute(_id, params) {
    const result = await gateway('/rforge', params);
    return { content: [{ type: 'text', text: JSON.stringify(result) }], details: undefined };
  },
}] : native.filter(tool => tool.name === 'web_search');

tools.push({
  name: 'submit_answer', label: 'Submit Answer',
  description: 'Finish the observation with the actual ordered bibliography. Preserve ranks and duplicates. Return only identifiers of papers you retrieved, without tool/host/run attribution. Claims may be empty; do not invent supporting quotes. Partial or empty answers are allowed and stay in the analysis.',
  parameters: { type: 'object', properties: {
    bibliography: { type: 'array', items: { type: 'object', properties: { identifiers: { type: 'array', items: { type: 'object', properties: { scheme: { type: 'string' }, value: { type: 'string' } }, required: ['scheme', 'value'], additionalProperties: false } } }, required: ['identifiers'], additionalProperties: false } },
    claims: { type: 'array', items: { type: 'object', properties: { text: { type: 'string' }, quote: { type: 'string' }, locator: { type: 'string' } }, required: ['text', 'quote', 'locator'], additionalProperties: false } },
  }, required: ['bibliography', 'claims'], additionalProperties: false },
  async execute(_id, params) {
    answer = params;
    return { content: [{ type: 'text', text: 'Answer retained.' }], details: undefined, terminate: true };
  },
});
for (const tool of tools) {
  const execute = tool.execute;
  tool.execute = async (...args) => {
    check(); toolCalls++;
    if (tool.name === 'web_search') nativeLimit = Math.min(20, Math.max(1, Number(args[1].limit) || 5));
    return execute(...args);
  };
}

const modelRuntime = await ModelRuntime.create({ authPath: '/work/agent/auth.json', modelsPath: '/work/agent/models.json' });
const model = (await modelRuntime.getAvailable()).find(model => model.provider === input.model.provider && model.id === input.model.version);
if (!model) throw new Error('pinned model unavailable');
const loader = {
  getExtensions: () => ({ extensions: [], errors: [], runtime: createExtensionRuntime() }),
  getSkills: () => ({ skills: [], diagnostics: [] }), getPrompts: () => ({ prompts: [], diagnostics: [] }), getThemes: () => ({ themes: [], diagnostics: [] }),
  getAgentsFiles: () => ({ agentsFiles: [] }), getSystemPrompt: () => input.system + '\n\n' + input.tool_instructions,
  getSystemPromptSource: () => undefined, getAppendSystemPrompt: () => [], getAppendSystemPromptSources: () => [], extendResources() {}, async reload() {},
};
const { session } = await createAgentSession({ cwd: '/work', agentDir: '/work/agent', modelRuntime, model, thinkingLevel: 'off', resourceLoader: loader, tools: tools.map(t => t.name), customTools: tools, sessionManager: SessionManager.inMemory('/work'), settingsManager: SettingsManager.inMemory({ cacheWarming: 'off' }) });
const originalPayload = session.agent.onPayload;
session.agent.onPayload = async payload => {
  check();
  const updated = originalPayload ? await originalPayload(payload) : payload;
  updated.temperature = input.sampling.temperature;
  updated.top_p = input.sampling.top_p;
  updated.max_tokens = Math.min(4096, Math.max(1, input.caps.tokens - inputTokens - outputTokens));
  updated.provider = { only: ['Anthropic'], allow_fallbacks: false };
  updated.usage = { include: true };
  transcript.push({ type: 'request', payload: updated });
  return updated;
};
session.agent.onProviderStreamEvent = data => {
  if (data?.id) generations.add(data.id);
  if (data?.usage) {
    lastUsage = { id: data.id, usage: data.usage };
    transcript.push({ type: 'provider_usage', data });
  }
};
session.subscribe(event => {
  if (event.type === 'message_end') {
    transcript.push(event.message);
    if (event.message.role === 'assistant') {
      // An indivisible response can cross a cap while already containing the
      // structured final answer. Retain that observed output even when no more
      // host tools may execute; never replace it with an artificial empty row.
      for (const block of event.message.content) {
        if (block.type === 'toolCall' && block.name === 'submit_answer' && Array.isArray(block.arguments?.bibliography) && Array.isArray(block.arguments?.claims)) answer = block.arguments;
      }
      const usage = event.message.usage;
      // Missing usage must never turn into zero-cost/zero-token observations.
      // A local pre-invocation budget guard sent no additional model request.
      const localBudgetStop = exhausted && event.message.errorMessage?.includes('budget_exhausted');
      if (!localBudgetStop) {
        const raw = lastUsage?.usage;
        const measured = usage && [usage.input, usage.cacheRead, usage.cacheWrite, usage.output].every(n => Number.isSafeInteger(n) && n >= 0);
        const reconciled = raw && Number.isSafeInteger(raw.prompt_tokens) && Number.isSafeInteger(raw.completion_tokens) && raw.prompt_tokens >= 0 && raw.completion_tokens >= 0 && measured && raw.prompt_tokens === usage.input + usage.cacheRead + usage.cacheWrite && raw.completion_tokens === usage.output;
        if (!reconciled || !lastUsage.id || accounted.has(lastUsage.id)) completeAccounting = false;
        else {
          accounted.add(lastUsage.id);
          inputTokens += raw.prompt_tokens;
          outputTokens += raw.completion_tokens;
        }
        if (inputTokens + outputTokens >= input.caps.tokens) exhausted = true;
      }
    }
  }
});
const timer = setTimeout(() => { exhausted = true; void session.abort(); }, Math.max(0, deadline - Date.now()));
let error = null;
try {
  await session.prompt(`${input.task}\n\nQuestion: ${input.question}\nDomain: ${input.domain}\nInclusive publication-date cap: ${input.date_cap}\nCaps: ${JSON.stringify(input.caps)}`);
} catch (failure) { error = failure.message; }
finally { clearTimeout(timer); }
const lastText = session.getLastAssistantText();
if (!answer && lastText) {
  try {
    const parsed = JSON.parse(lastText.replace(/^```(?:json)?\s*/, '').replace(/\s*```$/, ''));
    if (Array.isArray(parsed.bibliography) && Array.isArray(parsed.claims)) answer = parsed;
  } catch { /* Non-structured partial text stays in the private transcript. */ }
}
const messages = session.messages;
const result = { answer: answer ?? { bibliography: [], claims: [] }, tool_calls: toolCalls, input_tokens: inputTokens, output_tokens: outputTokens, wall_clock_ms: Math.round(performance.now() - started), exhausted, error, complete_accounting: completeAccounting, generation_ids: [...generations], transcript, messages, effective_system: session.systemPrompt, active_tools: session.getActiveToolNames(), model: { provider: model.provider, version: model.id }, host_version: JSON.parse(fs.readFileSync(`${root}/package.json`, 'utf8')).version };
session.dispose();
console.log(JSON.stringify(result));
