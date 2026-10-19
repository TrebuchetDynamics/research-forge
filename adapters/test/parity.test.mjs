import assert from 'node:assert/strict';
import { cp, mkdtemp, mkdir, readFile, rm } from 'node:fs/promises';
import { createServer } from 'node:http';
import { once } from 'node:events';
import { tmpdir } from 'node:os';
import { delimiter, join } from 'node:path';
import { after, before, test } from 'node:test';
import { cli, connect, exec, jobVolatile, parity, resultVolatile, root, terminal } from './support/parity.mjs';
import { artifacts, artifactParity, provenance, provenanceParity } from './support/artifacts.mjs';

let scratch, binary, env, server;
let blocked;
let hold = false;
let requests = [];
before(async () => {
  scratch = await mkdtemp(join(tmpdir(), 'rforge-parity-'));
  const bin = join(scratch, 'bin');
  await mkdir(bin);
  binary = join(bin, 'rforge');
  await exec('go', ['build', '-o', binary, './cmd/rforge'], { cwd: root, timeout: 120000 });
  const payload = await readFile(new URL('./fixtures/parity/openalex.json', import.meta.url));
  server = createServer((req, res) => {
    const query = new URL(req.url, 'http://fixture').searchParams.get('search');
    requests.push(query);
    if (query === 'failure') { res.writeHead(404); res.end('offline connector failure'); }
    else if (query === 'second' && hold) { blocked?.(); }
    else { res.setHeader('Content-Type', 'application/json'); res.end(payload); }
  });
  const port = process.env.AUTOWORK_PORT_BASE ? Number(process.env.AUTOWORK_PORT_BASE) + 90 : 0;
  server.listen(port, '127.0.0.1');
  await once(server, 'listening');
  env = { ...process.env, PATH: bin + delimiter + process.env.PATH, RFORGE_SOURCE_REQUEST_DELAY: '0s',
    RFORGE_OPENALEX_URL: `http://127.0.0.1:${server.address().port}` };
});
after(async () => {
  if (server) { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); }
  if (scratch) await rm(scratch, { recursive: true, force: true });
});

// Removing the MCP error envelope or changing only the Pi error code must fail.
test('cross-surface invalid input preserves the engine error code', { timeout: 30000 }, async t => {
  const dir = join(scratch, 'invalid');
  const expected = await cli(binary, env, ['search', 'batch', '--out', dir, '--query', 'fixture', '--sources', 'not-a-source'], 2);
  assert.equal(expected.error.code, 'unknown_source');
  for (const surface of ['mcp', 'pi']) {
    const client = await connect(t, binary, env, surface);
    parity(await client.tool('search', { dir, queries: ['fixture'], sources: ['not-a-source'] }), expected);
  }
});

// A host-only changed identifier, dedupe decision, hash, budget or stop reason
// must fail. All three producers use the real binary and connector fixture.
test('cross-surface search preserves evidence, provenance, budgets and connector errors', { timeout: 60000 }, async t => {
  const dir = join(scratch, 'search');
  const clients = {};
  for (const surface of ['mcp', 'pi']) clients[surface] = await connect(t, binary, env, surface);
  for (const scenario of [
    { name: 'identity', queries: ['first'], stop: 'completed', results: 3, deduped: 2 },
    { name: 'budget', queries: ['first', 'second'], max: 1, stop: 'budget_exhausted', results: 3, deduped: 2 },
    { name: 'connector', queries: ['failure'], stop: 'error_threshold', code: 1, error: 'search_batch_failed' },
    { name: 'output snapshot failure', queries: ['first'], snapshot: true, stop: 'error_threshold', code: 1, error: 'search_batch_output_snapshot_failed' },
  ]) {
    await t.test(scenario.name, async () => {
      let expected, expectedArtifacts, expectedStatus, expectedProvenance;
      for (const surface of ['cli', 'mcp', 'pi']) {
        await rm(dir, { recursive: true, force: true });
        if (scenario.snapshot) await mkdir(join(dir, 'results.md'), { recursive: true });
        const args = ['search', 'batch', '--out', dir, '--sources', 'openalex', '--limit', '3', '--depth', 'quick',
          ...scenario.queries.flatMap(query => ['--query', query]), ...(scenario.max ? ['--max-requests', String(scenario.max)] : [])];
        let result, status;
        if (surface === 'cli') {
          result = await cli(binary, env, args, scenario.code ?? 0);
          const list = await cli(binary, env, ['jobs', 'list', '--dir', dir]);
          assert.equal(list.data.jobs.length, 1);
          status = await cli(binary, env, ['jobs', 'status', list.data.jobs[0].id, '--dir', dir]);
        } else {
          const started = await clients[surface].tool('search', { dir, queries: scenario.queries, sources: ['openalex'], limit: 3, depth: 'quick',
            ...(scenario.max ? { max_requests: scenario.max } : {}) });
          status = await terminal(clients[surface], dir, started.data.job_id);
          result = status.data.job.result;
        }
        assert.equal(status.data.job.stop_reason, scenario.stop);
        assert.ok(result, `${surface} omitted the engine result`);
        if (scenario.error) {
          assert.equal(result.error.code, scenario.error);
          assert.equal(status.data.job.status, 'failed');
        } else {
          assert.equal(result.data.results, scenario.results);
          assert.equal(result.data.deduped, scenario.deduped);
          assert.equal(result.data.stop_reason, scenario.stop);
          const captured = await artifacts(dir);
          if (expectedArtifacts) artifactParity(captured, expectedArtifacts);
          else expectedArtifacts = captured;
          // Every surface reads the same immutable receipt/checksums for this
          // run. Do not normalize checksum fields across independent runs.
          const observed = await provenance(binary, env, dir, result);
          if (expectedProvenance) provenanceParity(observed, expectedProvenance);
          else expectedProvenance = observed;
          for (const reader of Object.values(clients)) {
            const read = await reader.tool('job_status', { dir, id: status.data.job.id });
            parity(read, status);
            parity(await provenance(binary, env, dir, read.data.job.result), observed);
          }
        }
        if (expected) {
          parity(result, expected, resultVolatile);
          parity(status, expectedStatus, jobVolatile);
        } else { expected = result; expectedStatus = status; }
      }
    });
  }
});

// A Pi-only rewrite of stable record/passage IDs, stored hashes or verification
// facets is an evidence-identity violation, even when the prose looks the same.
test('cross-surface claim checks preserve stored identities and content hashes', { timeout: 30000 }, async t => {
  const project = join(scratch, 'evidence-project');
  await cp(new URL('./fixtures/parity/project', import.meta.url), project, { recursive: true });
  const expected = await cli(binary, env, ['--project', project, 'claim', 'check', '--id', 'claim-parity']);
  assert.equal(expected.data.claim.evidence[0].identifier.value, 'paper-parity');
  assert.equal(expected.data.claim.evidence[0].contentHash, 'sha256:8cb639be582e298bc3bfed7c6cd1ba0f7e2a097c0862b20f7e94561b64ba6c79');
  for (const name of ['sourceIdentity', 'contentIntegrity', 'passageMatch']) {
    assert.equal(expected.data.claim.facets[name].state, 'passed');
  }
  const times = ['sourceIdentity', 'contentIntegrity', 'passageMatch'].map(name => `data.claim.facets.${name}.timestamp`);
  for (const surface of ['mcp', 'pi']) {
    const client = await connect(t, binary, env, surface);
    parity(await client.tool('claim_check', { project, id: 'claim-parity' }), expected, times);
  }
});

// Cancellation belongs to the host session; persisted work must remain usable
// by another host, with exactly the same final checkpoint and engine response.
test('MCP and Pi interruptions resume through CLI to the same final state', { timeout: 60000 }, async t => {
  const dir = join(scratch, 'resume');
  let expected, expectedArtifacts, expectedJob, expectedProvenance;
  for (const surface of ['mcp', 'pi']) {
    await rm(dir, { recursive: true, force: true });
    requests = [];
    hold = true;
    const reached = new Promise(resolve => { blocked = resolve; });
    const client = await connect(t, binary, env, surface);
    const started = await client.tool('search', { dir, queries: ['first', 'second'], sources: ['openalex'], limit: 3, depth: 'quick' });
    // Use a provider barrier, never a timing guess, to interrupt after one
    // completed pair and one reserved-but-unfinished connector request.
    await Promise.race([reached, new Promise((_, reject) => {
      const timer = setTimeout(() => reject(new Error('second query did not start')), 10000);
      reached.finally(() => clearTimeout(timer));
    })]);
    await client.stop();
    hold = false;
    const interrupted = await cli(binary, env, ['jobs', 'status', started.data.job_id, '--dir', dir]);
    assert.equal(interrupted.data.job.status, 'interrupted');
    assert.equal(interrupted.data.job.stop_reason, 'interrupted');
    assert.equal(interrupted.data.job.progress.completed, 1);
    assert.deepEqual(requests, ['first', 'second']);
    const result = await cli(binary, env, ['jobs', 'resume', started.data.job_id, '--dir', dir]);
    assert.deepEqual(requests, ['first', 'second', 'second'], 'resumption repeated committed work');
    assert.equal(result.data.stop_reason, 'completed');
    assert.equal(result.data.requests_used, 3, 'aborted connector attempt must consume budget');
    assert.equal(result.data.resumedPairs, 1);
    const captured = await artifacts(dir);
    const finished = await cli(binary, env, ['jobs', 'status', started.data.job_id, '--dir', dir]);
    assert.equal(finished.data.job.status, 'completed');
    assert.deepEqual(finished.data.job.progress, { completed: 2, total: 2 });
    const observed = await provenance(binary, env, dir, result);
    const reader = await connect(t, binary, env, surface === 'mcp' ? 'pi' : 'mcp');
    parity(await reader.tool('job_status', { dir, id: started.data.job_id }), finished);
    if (expected) {
      parity(result, expected, resultVolatile);
      parity(finished, expectedJob, jobVolatile);
      artifactParity(captured, expectedArtifacts);
      provenanceParity(observed, expectedProvenance);
    } else { expected = result; expectedArtifacts = captured; expectedJob = finished; expectedProvenance = observed; }
  }
});
