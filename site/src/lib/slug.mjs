// slugFor maps a docs/-relative path to its site path under /docs/:
//   01-vision.md                 -> docs/vision/
//   adr/0003-mdns-discovery.md   -> docs/adr/mdns-discovery/
//   README.md                    -> docs/       adr/README.md -> docs/adr/
// The numeric prefixes order the files in the repo; URLs do not need them.
export function slugFor(rel) {
  const parts = rel.replace(/\.md$/, '').split('/');
  const out = [];
  for (const p of parts) {
    if (p === 'README') continue;
    out.push(p.replace(/^\d+-/, ''));
  }
  return 'docs/' + out.map((p) => p + '/').join('');
}

// titleOf pulls the leading H1 out of a Markdown body.
export function titleOf(body, fallback) {
  const m = body.match(/^#\s+(.+)$/m);
  return m ? m[1].trim() : fallback;
}
