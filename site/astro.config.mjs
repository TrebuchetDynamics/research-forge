import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://research-forge.pages.dev',
  integrations: [
    starlight({
      title: 'ResearchForge',
      description: 'Provenance-first systematic review tooling for researchers and agents.',
      customCss: ['./src/styles/site.css'],
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/TrebuchetDynamics/research-forge' },
      ],
      sidebar: [
        { label: 'Start', items: [
          { label: 'Overview', link: '/' },
          { label: 'Getting started', slug: 'getting-started' },
          { label: 'Core workflow', slug: 'workflow' },
        ]},
        { label: 'Reference', items: [
          { label: 'Source connectors', slug: 'sources' },
          { label: 'Review packages', slug: 'packages' },
          { label: 'Agent skill', slug: 'agent-skill' },
          { label: 'CLI cheatsheet', slug: 'cli' },
        ]},
      ],
    }),
  ],
});
