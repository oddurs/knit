import { getCollection } from 'astro:content';
import { slugFor, titleOf } from './slug.mjs';

// docsNav returns the sidebar: repository order for the numbered guides, then
// the ADRs, each entry with its site path and H1 title.
export async function docsNav() {
  const entries = await getCollection('docs');
  const items = entries
    .map((e) => ({
      id: e.id,
      path: slugFor(e.id + '.md'),
      title: titleOf(e.body ?? '', e.id).replace(/^ADR-\d+:\s*/, ''),
      adr: e.id.startsWith('adr/'),
      index: e.id === 'README' || e.id === 'adr/README',
    }))
    .sort((a, b) => a.id.localeCompare(b.id, 'en', { numeric: true }));
  return {
    guides: items.filter((i) => !i.adr && !i.index),
    adrs: items.filter((i) => i.adr && !i.index),
  };
}
