// One collection: the user documentation under site/content/, read by a small
// loader that parses the `title`/`order` frontmatter itself and rewrites the
// relative links between pages (`../guides/files.md`) into site links, or
// GitHub links for anything outside content/. The engineering docs in the
// repository's docs/ directory are deliberately not published.
import { defineCollection, type Loader } from 'astro:content';
import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { slugFor } from './lib/slug.mjs';

const DOCS = path.resolve('content');
const REPO = 'https://github.com/oddurs/knit/blob/main/';
const BASE = '/knit/';

function docsLoader(): Loader {
  return {
    name: 'knit-docs',
    async load({ store, renderMarkdown, watcher }) {
      store.clear();
      watcher?.add(DOCS);
      for (const file of await walk(DOCS)) {
        const rel = path.relative(DOCS, file);
        const { data, content } = frontmatter(await readFile(file, 'utf8'));
        const body = rewriteLinks(content, path.dirname(rel));
        store.set({
          id: rel.replace(/\.md$/, ''),
          data,
          body,
          filePath: path.relative(process.cwd(), file),
          rendered: await renderMarkdown(body),
        });
      }
    },
  };
}

async function walk(dir: string): Promise<string[]> {
  const out: string[] = [];
  for (const d of await readdir(dir, { withFileTypes: true })) {
    const p = path.join(dir, d.name);
    if (d.isDirectory()) out.push(...(await walk(p)));
    else if (d.name.endsWith('.md')) out.push(p);
  }
  return out.sort();
}

// frontmatter splits a leading `---` block of `key: value` lines from the body.
function frontmatter(src: string): { data: Record<string, string | number>; content: string } {
  const m = src.match(/^---\n([\s\S]*?)\n---\n/);
  if (!m) return { data: {}, content: src };
  const data: Record<string, string | number> = {};
  for (const line of m[1].split('\n')) {
    const [k, ...v] = line.split(':');
    const value = v.join(':').trim();
    if (k.trim()) data[k.trim()] = /^\d+$/.test(value) ? Number(value) : value;
  }
  return { data, content: src.slice(m[0].length) };
}

// rewriteLinks resolves each relative Markdown link target against the doc's
// directory (relative to docs/). Targets inside docs/ become site paths;
// anything else points at the file on GitHub.
export function rewriteLinks(md: string, fromDir: string): string {
  return md.replace(/(\]\()([^)\s]+)(\))/g, (m, open, target, close) => {
    if (/^([a-z]+:|#|\/)/i.test(target)) return m;
    const [file, hash] = target.split('#');
    const resolved = path.posix.normalize(path.posix.join(fromDir, file));
    const anchor = hash ? '#' + hash : '';
    const href = resolved.startsWith('..')
      ? REPO + path.posix.normalize(path.posix.join('site/content', resolved)) + anchor
      : BASE + slugFor(resolved) + anchor;
    return open + href + close;
  });
}

export const collections = {
  docs: defineCollection({ loader: docsLoader() }),
};
