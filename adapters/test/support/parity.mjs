import assert from 'node:assert/strict';
import { execFile, spawn } from 'node:child_process';
import { once } from 'node:events';
import { createInterface } from 'node:readline';
import { promisify } from 'node:util';
import register from '../../pi.mjs';

export const exec = promisify(execFile);
export const root = new URL('../../../', import.meta.url);

// Exact JSON paths, not recursive key deletion: a record's id and a checker
// timestamp outside these paths remain authoritative. Presence is preserved.
export const resultVolatile = ['data.job_id', 'data.run_id'];
export const jobVolatile = [
  'data.job.id', 'data.job.created_at', 'data.job.updated_at',
  ...resultVolatile.map(path => `data.job.result.${path}`),
];
export function normalize(value, paths = []) {
  const copy = structuredClone(value);
  for (const path of paths) {
    const parts = path.split('.');
    const key = parts.pop();
    const parent = parts.reduce((value, part) => value?.[part], copy);
    if (parent && Object.hasOwn(parent, key)) parent[key] = '<volatile>';
  }
  return copy;
}
export function parity(actual, expected, paths = []) {
  assert.deepEqual(normalize(actual, paths), normalize(expected, paths));
}

export async function cli(binary, env, args, expectedCode = 0) {
  let output;
  try {
    output = await exec(binary, ['--json', ...args], { cwd: root, env, timeout: 15000 });
    assert.equal(expectedCode, 0, `CLI unexpectedly succeeded: ${args}`);
  } catch (error) {
    assert.equal(error.code, expectedCode, `${args}: ${error.stderr ?? error.message}`);
    output = error;
  }
  return JSON.parse(output.stdout);
}

export async function connect(t, binary, env, surface) {
  let command = binary;
  let args = ['mcp', 'serve'];
  if (surface === 'pi') {
    const servers = [];
    register({ registerCommand() {}, registerMcpServer: (name, config) => servers.push({ name, ...config }) });
    assert.equal(servers.length, 1);
    assert.equal(servers[0].name, 'rforge');
    ({ command, args } = servers[0]);
  }
  const child = spawn(command, args, { cwd: root, env, stdio: ['pipe', 'pipe', 'pipe'] });
  const pending = new Map();
  let diagnostics = '';
  child.stderr.on('data', data => { diagnostics += data; });
  const lines = createInterface({ input: child.stdout });
  lines.on('line', line => {
    const response = JSON.parse(line);
    const waiter = pending.get(response.id);
    if (waiter) { clearTimeout(waiter.timer); pending.delete(response.id); waiter.resolve(response); }
  });
  const rejectAll = error => {
    for (const waiter of pending.values()) { clearTimeout(waiter.timer); waiter.reject(error); }
    pending.clear();
  };
  child.on('error', rejectAll);
  child.on('exit', code => rejectAll(new Error(`MCP exited ${code}: ${diagnostics}`)));
  let stopped = false;
  async function stop() {
    if (stopped) return;
    stopped = true;
    lines.close();
    child.stdin.end();
    if (child.exitCode === null && child.signalCode === null) {
      const exited = once(child, 'exit');
      // The Pi launcher relays termination to its Go child, which joins jobs.
      child.kill('SIGTERM');
      const timer = setTimeout(() => child.kill('SIGKILL'), 3000);
      try { await exited; } finally { clearTimeout(timer); }
    }
  }
  t.after(stop);
  let id = 0;
  function request(method, params = {}) {
    return new Promise((resolve, reject) => {
      const requestID = ++id;
      const timer = setTimeout(() => {
        pending.delete(requestID);
        reject(new Error(`${surface} ${method} timed out: ${diagnostics}`));
      }, 10000);
      pending.set(requestID, { resolve, reject, timer });
      child.stdin.write(JSON.stringify({ jsonrpc: '2.0', id: requestID, method, params }) + '\n');
    });
  }
  const initialized = await request('initialize', {
    protocolVersion: '2024-11-05', capabilities: {}, clientInfo: { name: 'parity', version: '1' },
  });
  assert.equal(initialized.error, undefined);
  child.stdin.write(JSON.stringify({ jsonrpc: '2.0', method: 'notifications/initialized' }) + '\n');
  return { stop, async tool(name, args) {
    const response = await request('tools/call', { name, arguments: args });
    assert.equal(response.error, undefined, JSON.stringify(response));
    const result = response.result;
    const structured = result.structuredContent;
    assert.ok(structured, `missing engine JSON: ${JSON.stringify(result)}`);
    assert.deepEqual(JSON.parse(result.content[0].text), structured, 'text and structured content diverged');
    assert.equal(result.isError ?? false, structured.ok === false);
    return structured;
  } };
}

export async function terminal(client, dir, id) {
  const end = Date.now() + 10000;
  while (Date.now() < end) {
    const result = await client.tool('job_status', { dir, id });
    if (['completed', 'failed', 'interrupted'].includes(result.data.job.status)) return result;
    await new Promise(resolve => setTimeout(resolve, 10));
  }
  throw new Error(`job ${id} did not finish`);
}
