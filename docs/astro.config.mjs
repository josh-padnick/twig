// Astro + Starlight configuration for the twig documentation site,
// deployed to GitHub Pages at https://josh-padnick.github.io/twig.
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://josh-padnick.github.io',
  base: '/twig',
  integrations: [
    starlight({
      title: 'twig',
      description: 'Git worktrees, one short command away.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/josh-padnick/twig' },
      ],
      sidebar: [
        {
          label: 'Getting started',
          items: [
            { slug: 'getting-started/install' },
            { slug: 'getting-started/quickstart' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { slug: 'guides/resolution' },
            { slug: 'guides/setup-scripts' },
            { slug: 'guides/trust' },
            { slug: 'guides/openers' },
            { slug: 'guides/providers' },
            { slug: 'guides/remote-pickup' },
            { slug: 'guides/shell-integration' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { slug: 'reference/cli' },
            { slug: 'reference/config' },
            { slug: 'reference/twig-toml' },
          ],
        },
        {
          label: 'Contributing',
          items: [{ slug: 'contributing/openers' }],
        },
      ],
    }),
  ],
});
