import assert from 'node:assert/strict';
import { execFile, spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { delimiter, join } from 'node:path';
import { createInterface } from 'node:readline';
import { promisify } from 'node:util';
import { after, before, test } from 'node:test';
import register from '../pi.mjs';

const exec = promisify(execFile);
let scratch;
let bin;
before(async () => {
  scratch = await mkdtemp(join(tmpdir(), 'rforge-host-'));
  bin = join(scratch, 'bin');
  await mkdir(bin);
  await exec('go', ['build', '-o', join(bin, 'rforge'), './cmd/rforge'], {
    cwd: new URL('../../', import.meta.url), timeout: 120000,
  });
});
after(async () => { if (scratch) await rm(scratch, { recursive: true, force: true }); });

function connect(t, path) {
  const servers = [];
  register({ registerCommand() {}, registerMcpServer: (name, config) => servers.push({ name, ...config }) });
  assert.equal(servers.length, 1);
  assert.equal(servers[0].name, 'rforge');
  const { command, args } = servers[0];
  const child = spawn(command, args, { env: { ...process.env, PATH: path }, stdio: ['pipe', 'pipe', 'pipe'] });
  const pending = new Map();
  let diagnostics = '';
  child.stderr.on('data', chunk => { diagnostics += chunk; });
  const lines = createInterface({ input: child.stdout });
  lines.on('line', line => {
    const message = JSON.parse(line);
    const waiter = pending.get(message.id);
    pending.delete(message.id);
    waiter?.resolve(message);
  });
  child.on('exit', code => {
    for (const waiter of pending.values()) waiter.reject(new Error(`MCP exited ${code}: ${diagnostics}`));
    pending.clear();
  });
  t.after(async () => {
    child.stdin.end();
    if (child.exitCode === null && child.signalCode === null) {
      const exited = once(child, 'exit');
      const timer = setTimeout(() => child.kill('SIGKILL'), 2000);
      await exited;
      clearTimeout(timer);
    }
    lines.close();
  });
  let id = 0;
  return {
    child,
    request(method, params = {}) {
      return new Promise((resolve, reject) => {
        pending.set(++id, { resolve, reject });
        child.stdin.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n');
      });
    },
    notify(method) { child.stdin.write(JSON.stringify({ jsonrpc: '2.0', method }) + '\n'); },
  };
}
const initialize = { protocolVersion: '2024-11-05', capabilities: {}, clientInfo: { name: 'host-test', version: '1' } };

// This catches stale host tool lists (notably omission of claim_check), leaked
// approval/export tools, and adapter-side rewrites of authoritative engine data.
test('Pi native MCP exposes the current read/run tools and preserves engine JSON', { timeout: 15000 }, async t => {
  const client = connect(t, bin + delimiter + process.env.PATH);
  assert.equal((await client.request('initialize', initialize)).error, undefined);
  client.notify('notifications/initialized');
  const listed = await client.request('tools/list');
  assert.deepEqual(listed.result.tools.map(tool => tool.name).sort(), [
    'claim_check', 'compare', 'evidence_retrieve', 'job_status', 'research_status', 'search', 'source_get',
  ]);
  assert.deepEqual(listed.result.tools.find(tool => tool.name === 'research_status').inputSchema.required, ['dir']);
  const dir = join(scratch, 'literal $(no-shell) research');
  await mkdir(dir);
  await writeFile(join(dir, 'manifest.json'), JSON.stringify({ queries: ['fixture'], sources: ['openalex'], results: 1, deduped: 1 }));
  const result = await client.request('tools/call', { name: 'research_status', arguments: { dir } });
  const cli = await exec(join(bin, 'rforge'), ['--json', 'brief', dir]);
  assert.equal(result.result.isError ?? false, false);
  assert.deepEqual(result.result.structuredContent, JSON.parse(cli.stdout));
});

test('missing and incompatible engines fail closed with structured install.sh recovery', { timeout: 120000 }, async t => {
  const old = join(scratch, 'old');
  await mkdir(old);
  await exec('go', ['build', '-ldflags', '-X main.version=0.1.0', '-o', join(old, 'rforge'), './cmd/rforge'], {
    cwd: new URL('../../', import.meta.url), timeout: 120000,
  });
  for (const [path, code] of [[join(scratch, 'missing'), 'rforge_missing'], [old, 'version_too_old']]) {
    await t.test(code, async t => {
      const client = connect(t, path);
      const result = await client.request('initialize', initialize);
      assert.equal(result.error.code, -32000);
      assert.equal(result.error.data.ok, false);
      assert.equal(result.error.data.error.code, code);
      assert.equal(result.error.data.error.install, 'install.sh');
      assert.match(result.error.message, /install.sh/);
    });
  }
});

async function unresponsiveEngine(t, name) {
  const path = join(scratch, name);
  const pidFile = join(path, 'pid');
  await mkdir(path);
  // Only a process-boundary double can simulate a hung/termination-ignoring
  // capability probe; successful paths above always use the real Go engine.
  await writeFile(join(path, 'rforge'), `#!${process.execPath}\n`
    + `import { writeFileSync } from 'node:fs';\n`
    + `process.on('SIGTERM', () => {});\n`
    + `writeFileSync(${JSON.stringify(pidFile)}, String(process.pid));\n`
    + `setInterval(() => {}, 1000);\n`, { mode: 0o755 });
  let pid;
  t.after(() => { if (pid) { try { process.kill(pid, 'SIGKILL'); } catch { /* Already reaped. */ } } });
  return { path, async started() {
    const end = Date.now() + 3000;
    while (Date.now() < end) {
      try { pid = Number(await readFile(pidFile, 'utf8')); return pid; } catch (error) {
        if (error.code !== 'ENOENT') throw error;
        await new Promise(resolve => setTimeout(resolve, 10));
      }
    }
    throw new Error('probe did not start');
  } };
}

test('compatibility checking times out and reaps an unresponsive engine', { timeout: 15000 }, async t => {
  const engine = await unresponsiveEngine(t, 'timeout');
  const client = connect(t, engine.path);
  const pid = await engine.started();
  const response = await client.request('initialize', initialize);
  assert.equal(response.error.data.error.code, 'rforge_incompatible');
  assert.match(response.error.message, /install.sh/);
  assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
});

test('host termination during preflight reaps the engine before exiting', { timeout: 5000 }, async t => {
  const engine = await unresponsiveEngine(t, 'shutdown');
  const client = connect(t, engine.path);
  const pid = await engine.started();
  const exited = once(client.child, 'exit');
  client.child.kill('SIGTERM');
  await exited;
  assert.throws(() => process.kill(pid, 0), { code: 'ESRCH' });
});
