import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import register from './pi.mjs';

test('Pi remains passive until an explicit workflow command loads its skill', async () => {
  const commands = new Map();
  const messages = [];
  register({
    registerCommand(name, command) {
      assert.equal(commands.has(name), false, `duplicate command ${name}`);
      commands.set(name, command);
    },
    sendUserMessage: (...args) => messages.push(args),
  });
  assert.equal(messages.length, 0, 'registration must not start an agent turn');
  assert.deepEqual([...commands.keys()].sort(), [
    'rforge_brief', 'rforge_compare', 'rforge_lit', 'rforge_paper', 'rforge_replicate',
  ]);
  const request = 'read-only; $(literal) "two paths"\nno downloads';
  for (const [name, command] of commands) {
    const file = new URL(`./skills/${name.slice('rforge_'.length)}/SKILL.md`, import.meta.url);
    const body = await readFile(file, 'utf8');
    for (const idle of [true, false]) {
      const before = messages.length;
      await command.handler(request, { isIdle: () => idle });
      assert.equal(messages.length, before + 1);
      const [message, options] = messages.at(-1);
      assert.ok(message.includes(body), `${name} must load its own workflow skill`);
      assert.ok(message.endsWith(request), 'preserve the literal user request');
      assert.deepEqual(options, idle ? undefined : { deliverAs: 'followUp' });
    }
  }
});
