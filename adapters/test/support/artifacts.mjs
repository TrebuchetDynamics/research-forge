import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { cli, parity } from './parity.mjs';

export const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
export const json = async path => JSON.parse(await readFile(path, 'utf8'));
export const jsonl = async path => (await readFile(path, 'utf8')).split('\n').filter(Boolean).map(line => JSON.parse(line));

// Observe the receipt selected by the surface's run_id, not merely a declared
// level. Every checksum remains literal and is checked against actual bytes.
export async function provenance(binary, env, dir, result) {
  const path = join(dir, 'provenance', 'runs', `${result.data.run_id}.json`);
  const receipt = await json(path);
  const validation = await cli(binary, env, ['provenance', 'validate', path]);
  assert.equal(validation.data.provenance_level, 'engine_verified');
  assert.equal(result.data.provenance_level, validation.data.provenance_level);
  assert.equal(receipt.run_id, result.data.run_id);
  assert.ok(receipt.outputs.length >= 6, 'missing checksum-bound search artifacts');
  const volatileOutputs = {};
  for (const output of receipt.outputs) {
    const bytes = await readFile(join(dir, output.path));
    assert.equal(sha256(bytes), output.sha256, output.path);
    if (output.path === 'manifest.json' || output.path === 'batch-state.json') {
      volatileOutputs[output.path] = JSON.parse(bytes);
    }
  }
  return { validation, receipt, volatileOutputs };
}

// Compare every receipt field and output entry across independent producers.
// Only two outputs contain timestamps/job IDs. Their *raw* checksums are still
// validated above; if hashes differ, compare the source JSON using exact allowed
// paths instead of hiding or normalizing any checksum. Raw connector artifacts
// and every other output must have literally identical SHA256 values.
export function provenanceParity(actual, expected) {
  const { outputs: actualOutputs, ...actualMetadata } = actual.receipt;
  const { outputs: expectedOutputs, ...expectedMetadata } = expected.receipt;
  parity(actualMetadata, expectedMetadata, ['timestamp', 'run_id']);
  assert.deepEqual(actualOutputs.map(output => output.path), expectedOutputs.map(output => output.path));
  for (let i = 0; i < actualOutputs.length; i++) {
    const output = actualOutputs[i];
    const baseline = expectedOutputs[i];
    if (output.path === 'manifest.json' || output.path === 'batch-state.json') {
      const { sha256: actualHash, ...actualFields } = output;
      const { sha256: expectedHash, ...expectedFields } = baseline;
      parity(actualFields, expectedFields);
      // All non-checksum fields still participate, even unexpected additions.
      assert.match(actualHash, /^[a-f0-9]{64}$/);
      assert.match(expectedHash, /^[a-f0-9]{64}$/);
      parity(actual.volatileOutputs[output.path], expected.volatileOutputs[output.path],
        output.path === 'manifest.json' ? ['createdAt'] : ['createdAt', 'updatedAt', 'job_id']);
    } else parity(output, baseline);
  }
}

// Independent runs have distinct manifest/checkpoint bytes (timestamps/job IDs).
// Compare their complete JSON using only those exact volatile paths; compare
// deterministic evidence outputs byte-for-byte, including their hashes.
export async function artifacts(dir) {
  return {
    manifest: await json(join(dir, 'manifest.json')),
    state: await json(join(dir, 'batch-state.json')),
    records: await jsonl(join(dir, 'results.jsonl')),
    deduped: await jsonl(join(dir, 'results-deduped.jsonl')),
    hashes: Object.fromEntries(await Promise.all(['results.jsonl', 'results-deduped.jsonl', 'results.md', 'failures.jsonl'].map(async name =>
      [name, sha256(await readFile(join(dir, name)))]))),
  };
}
export function artifactParity(actual, expected) {
  parity(actual, expected, [
    'manifest.createdAt', 'state.createdAt', 'state.updatedAt', 'state.job_id',
  ]);
}
