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
    syntaxHighlight: false, // one monochrome code block everywhere
  },
});
