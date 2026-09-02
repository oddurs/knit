import { getCollection } from 'astro:content';
import { slugFor, titleOf } from './slug.mjs';

// Sidebar sections, in display order, keyed by the content/ folder.
const SECTIONS = [
  ['getting-started', 'Getting started'],
  ['guides', 'Guides'],
  ['reference', 'Reference'],
  ['trust', 'Trust and help'],
];

// docsNav returns the sidebar: one group per section, pages ordered by their
// frontmatter `order`, titled by frontmatter `title` or the page's H1.
export async function docsNav() {
  const entries = await getCollection('docs');
  return SECTIONS.map(([dir, label]) => ({
    label,
    pages: entries
      .filter((e) => e.id.startsWith(dir + '/'))
      .map((e) => ({
        path: slugFor(e.id + '.md'),
        title: e.data.title ?? titleOf(e.body ?? '', e.id),
        order: Number(e.data.order ?? 999),
      }))
      .sort((a, b) => a.order - b.order),
  }));
}
