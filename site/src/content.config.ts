// One collection: every Markdown file under the repository's docs/ directory,
// read in place by a small loader. The site never copies documentation; the
// loader only rewrites the repository-relative links the docs use
// (`03-protocol.md#error-codes`, `adr/0004-hmac.md`, `../roadmaps/x.toml`)
// into site links, or GitHub links for files that are not part of the site.
import { defineCollection, type Loader } from 'astro:content';
import { readdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { slugFor } from './lib/slug.mjs';

const DOCS = path.resolve('../docs');
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
        const body = rewriteLinks(await readFile(file, 'utf8'), path.dirname(rel));
        store.set({
          id: rel.replace(/\.md$/, ''),
          data: {},
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
      ? REPO + path.posix.normalize(path.posix.join('docs', resolved)) + anchor
      : BASE + slugFor(resolved) + anchor;
    return open + href + close;
  });
}

export const collections = {
  docs: defineCollection({ loader: docsLoader() }),
};
