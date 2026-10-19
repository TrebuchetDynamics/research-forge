import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const workflows = {
  lit: 'Synthesize a research corpus',
  paper: 'Resolve a paper identifier or title',
  brief: 'Summarize a completed search directory',
  compare: 'Compare completed search directories',
  replicate: 'Generate a paper replication plan',
};

// Register explicit commands only: no tools, hooks, subprocesses or startup work.
export default function researchForge(pi) {
  for (const [name, description] of Object.entries(workflows)) {
    pi.registerCommand(`rforge_${name}`, {
      description,
      handler: async (args, ctx) => {
        const file = new URL(`./skills/${name}/SKILL.md`, import.meta.url);
        const body = await readFile(file, 'utf8');
        const message = `Use this explicit ResearchForge workflow. Source: ${fileURLToPath(file)}\n`
          + `Load the research-forge methodology skill for all research decisions.\n\n${body}\n\n`
          + `User request (preserve scope and restrictions):\n${args}`;
        pi.sendUserMessage(message, ctx.isIdle() ? undefined : { deliverAs: 'followUp' });
      },
    });
  }
}
