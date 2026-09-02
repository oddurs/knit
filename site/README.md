# knit docs site

Astro, no integrations, no client JavaScript, system fonts. The content is the
repository's `docs/` directory, read in place: edit a doc, the site changes.

```sh
cd site
npm install
npm run dev       # http://localhost:1212
npm run build     # static output in dist/
```

Deployed to GitHub Pages by `.github/workflows/docs.yml` on every push to
`main` that touches `docs/` or `site/`.
