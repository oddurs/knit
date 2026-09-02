// The knit docs site: Astro with no integrations, no client JavaScript, and
// the repository's docs/ directory as its only content. `npm run dev` serves
// it at http://localhost:1212.
import { defineConfig } from 'astro/config';

export default defineConfig({
  site: 'https://oddurs.github.io',
  base: '/knit',
  trailingSlash: 'always',
  server: { port: 1212 },
  preview: { port: 1212 },
  build: { format: 'directory' },
  markdown: {
    // GitHub's own themes, so highlighting reads exactly like GFM. Colors
    // follow the OS via light-dark(); the block's background stays the design
    // system's, so the highlighter's is dropped.
    shikiConfig: {
      themes: { light: 'github-light', dark: 'github-dark' },
      defaultColor: 'light-dark()',
      transformers: [{ pre(node) { delete node.properties.style; } }],
    },
  },
});
