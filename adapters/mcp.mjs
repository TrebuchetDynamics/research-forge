// Shared host launcher. Negotiate the V-E0 contract before starting the Go
// engine; thereafter stdio, schemas and research results belong to rforge.
import { execFile, spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { promisify } from 'node:util';

const contracts = ['search', 'paper', 'brief', 'compare', 'provenance', 'mcp'];
let child;
let checking = true;
let stopping = false;
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    stopping = true;
    if (child) child.kill(checking ? 'SIGKILL' : signal);
    else process.exit(0);
  });
}

function recovery(code, message, details = {}) {
  return { ok: false, error: {
    ...details, code, message: `${message}; install or upgrade rforge with install.sh, then reconnect.`,
    install: 'install.sh',
  } };
}

async function compatibility() {
  try {
    const probe = promisify(execFile)('rforge', [
      'capabilities', '--json', '--min-rforge-version', '0.1.22',
      ...contracts.flatMap(name => ['--require', `${name}=1`]),
    ], { timeout: 10000, killSignal: 'SIGKILL', maxBuffer: 1024 * 1024 });
    child = probe.child;
    const { stdout } = await probe;
    const result = JSON.parse(stdout);
    if (result.ok === true && result.data?.name === 'rforge' && contracts.every(name =>
      Number.isInteger(result.data.capabilities?.[name]) && result.data.capabilities[name] >= 1)) return;
  } catch (error) {
    // Preserve engine diagnostics (version/capability gaps) when available.
    try {
      const result = JSON.parse(error.stdout);
      if (result.ok === false && typeof result.error?.code === 'string' && typeof result.error.message === 'string') {
        return recovery(result.error.code, result.error.message, result.error);
      }
    } catch { /* Missing/old engines may have no JSON capability response. */ }
    if (error.code === 'ENOENT') return recovery('rforge_missing', 'rforge was not found on PATH');
  } finally {
    child = undefined;
  }
  return recovery('rforge_incompatible', 'rforge compatibility check failed');
}

function serveFailure(result) {
  // Stay available until disconnect so the host receives an actionable,
  // structured JSON-RPC error instead of a generic failed-handshake message.
  process.stderr.write(JSON.stringify(result) + '\n');
  createInterface({ input: process.stdin }).on('line', line => {
    let request;
    try { request = JSON.parse(line); } catch {
      process.stdout.write(JSON.stringify({ jsonrpc: '2.0', id: null, error: { code: -32700, message: 'Parse error' } }) + '\n');
      return;
    }
    if (request?.id === undefined) return;
    process.stdout.write(JSON.stringify({ jsonrpc: '2.0', id: request.id,
      error: { code: -32000, message: result.error.message, data: result },
    }) + '\n');
  });
}

const failure = await compatibility();
if (!stopping) {
  if (failure) serveFailure(failure);
  else {
    checking = false;
    child = spawn('rforge', ['mcp', 'serve'], { stdio: 'inherit' });
    child.on('error', () => {
      child = undefined;
      serveFailure(recovery('rforge_missing', 'rforge could not start'));
    });
    child.on('exit', (code, signal) => {
      child = undefined;
      process.exitCode = code ?? (signal ? 1 : 0);
    });
  }
}
