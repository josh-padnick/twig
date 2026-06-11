// Astro + Starlight configuration for the twig documentation site,
// deployed to GitHub Pages. The account's verified custom domain means it
// serves at https://joshpadnick.com/twig/ (the github.io URL redirects).
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// Content pages cross-link with file-relative .md paths so the links also
// render on GitHub (PR previews, blob view). This plugin rewrites them at
// build time into the directory URLs Starlight serves. Non-index pages
// live at <slug>/index.html — a directory URL — so their links need one
// extra ../ in URL space.
function relativeMarkdownLinks() {
  return function transform(tree, file) {
    const isIndex = /(^|\/)index\.mdx?$/i.test(file.path ?? '');
    const walk = (node) => {
      if (node.type === 'link' && typeof node.url === 'string') {
        node.url = rewriteMdLink(node.url, isIndex);
      }
      for (const child of node.children ?? []) walk(child);
    };
    walk(tree);
  };
}

function rewriteMdLink(url, isIndex) {
  if (/^(?:[a-z][a-z0-9+.-]*:|\/|#)/i.test(url)) return url; // absolute or in-page
  const match = url.match(/^([^#?]*\.mdx?)(#.*)?$/i);
  if (!match) return url;
  let path = match[1].replace(/(^|\/)index\.mdx?$/i, '$1').replace(/\.mdx?$/i, '/');
  if (!isIndex) path = '../' + path;
  return path + (match[2] ?? '');
}

export default defineConfig({
  site: 'https://joshpadnick.com',
  base: '/twig',
  markdown: {
    remarkPlugins: [relativeMarkdownLinks],
  },
  integrations: [
    starlight({
      title: 'twig',
      description: 'Git worktrees, one short command away.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/josh-padnick/twig' },
      ],
      sidebar: [
        { label: 'Why twig', link: '/' },
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
