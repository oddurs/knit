// slugFor maps a content/-relative path to its site path under /docs/:
//   guides/files.md   -> docs/guides/files/
//   index.md          -> docs/
export function slugFor(rel) {
  const parts = rel.replace(/\.md$/, '').split('/').filter((p) => p !== 'index');
  return 'docs/' + parts.map((p) => p + '/').join('');
}

// titleOf pulls the leading H1 out of a Markdown body.
export function titleOf(body, fallback) {
  const m = body.match(/^#\s+(.+)$/m);
  return m ? m[1].trim() : fallback;
}
