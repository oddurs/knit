# knit docs site

Astro, no integrations, no client JavaScript, system fonts. The pages are the
Markdown files in `content/`, written for people who use knit. The engineering
docs in the repository's `docs/` directory are internal and are not published.

```sh
cd site
npm install
npm run dev       # http://localhost:1212
npm run build     # static output in dist/
```

Pages carry `title` and `order` in frontmatter; the folder is the sidebar
section (`getting-started`, `guides`, `reference`, `trust`). Links between
pages are ordinary relative Markdown links (`../guides/files.md`).

Deployed to GitHub Pages by `.github/workflows/docs.yml` on every push to
`main` that touches `site/`.
