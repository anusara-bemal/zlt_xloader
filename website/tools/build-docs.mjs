#!/usr/bin/env node
/*
 * Zero-dependency generator for the static documentation pages.
 * Reads website/content/*.mdx (the original doc sources) and writes
 * website/docs/*.html. Run from anywhere: node website/tools/build-docs.mjs
 */

import { readdirSync, readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join, basename } from 'node:path';
import { fileURLToPath } from 'node:url';
import { header } from './nav.mjs';

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, '..');
const contentDir = join(root, 'content');
const outDir = join(root, 'docs');
const SITE = 'https://anusara-bemal.github.io';

const escapeAttr = (value) => String(value).replace(/&/g, '&amp;').replace(/"/g, '&quot;');

function slugify(text) {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-');
}

function inline(text) {
  let value = String(text)
    .replace(/&(?!(?:amp|lt|gt|quot|#\d+);)/g, '&amp;')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
    .replace(/href="\/docs\/([a-z0-9-]+)\/"/g, 'href="$1.html"');
  return value;
}

function paragraphs(inner) {
  return inner
    .split(/\n\s*\n/)
    .map((chunk) => chunk.trim())
    .filter(Boolean)
    .map((chunk) => {
      const cleaned = chunk
        .replace(/^\s*<p>/g, '')
        .replace(/<\/p>\s*$/g, '')
        .replace(/\s*<\/p>\s*<p>\s*/g, '\n\n')
        .trim();
      if (!cleaned) return '';
      if (cleaned.startsWith('<pre') || cleaned.startsWith('<table')) return cleaned;
      return `<p>${inline(cleaned.replace(/\n/g, ' '))}</p>`;
    })
    .filter(Boolean)
    .join('\n');
}

function tableHtml(lines) {
  const rows = lines
    .map((line) => line.trim())
    .filter((line) => line.startsWith('|'))
    .map((line) => line.replace(/^\||\|$/g, '').split('|').map((cell) => cell.trim()));
  if (rows.length < 2) return '';
  const head = rows[0].map((cell) => `<th>${inline(cell)}</th>`).join('');
  const body = rows
    .slice(2)
    .map((row) => `<tr>${row.map((cell) => `<td>${inline(cell)}</td>`).join('')}</tr>`)
    .join('\n');
  return `<table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table>`;
}

function renderMarkdown(mdx, headings) {
  const lines = mdx.split('\n');
  const html = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    if (!line.trim()) { i += 1; continue; }

    // Fenced code
    if (line.trim().startsWith('```')) {
      const language = line.trim().slice(3).trim();
      const code = [];
      i += 1;
      while (i < lines.length && !lines[i].trim().startsWith('```')) {
        code.push(lines[i]);
        i += 1;
      }
      i += 1;
      html.push(`<pre><code class="language-${escapeAttr(language)}">${escapeHtml(code.join('\n'))}</code></pre>`);
      continue;
    }

    // Callout component
    if (line.includes('<Callout')) {
      const block = [line];
      i += 1;
      while (i < lines.length && !lines[i].includes('</Callout>')) {
        block.push(lines[i]);
        i += 1;
      }
      if (i < lines.length) block.push(lines[i]);
      i += 1;
      const raw = block.join('\n');
      const type = (raw.match(/type="([^"]+)"/) || [])[1] || 'note';
      const title = (raw.match(/title="([^"]+)"/) || [])[1] || 'Note';
      const inner = raw.replace(/^.*?>\s*$/m, '').replace(/<\/Callout>[\s\S]*$/, '');
      html.push(`<div class="doc-callout${type !== 'note' ? ` doc-callout--${type}` : ''}"><strong>${inline(title)}</strong>${paragraphs(inner)}</div>`);
      continue;
    }

    // Command component
    if (line.includes('<Command')) {
      const block = [line];
      i += 1;
      while (i < lines.length && !lines[i].includes('</Command>')) {
        block.push(lines[i]);
        i += 1;
      }
      if (i < lines.length) block.push(lines[i]);
      i += 1;
      const raw = block.join('\n');
      const name = (raw.match(/\bname="([^"]+)"/) || [])[1] || 'command';
      const tag = (raw.match(/\btag="([^"]+)"/) || [])[1] || 'command';
      const syntax = (raw.match(/\bsyntax="([^"]+)"/) || [])[1] || '';
      const inner = raw.replace(/^[\s\S]*?^\s*>\s*$/m, '').replace(/<\/Command>[\s\S]*$/, '');
      const flags = [...raw.matchAll(/flag:\s*'([^']*)',\s*description:\s*'([^']*)'/g)];
      let flagsHtml = '';
      if (flags.length) {
        const rows = flags
          .map((match) => `<tr><td>${inline(match[1])}</td><td>${inline(match[2])}</td></tr>`)
          .join('\n');
        flagsHtml = `<table><tbody>${rows}</tbody></table>`;
      }
      html.push(
        `<div class="doc-command" id="${escapeAttr(slugify(name))}">` +
        `<div class="doc-command__head"><h3 class="doc-command__name">${inline(name)}</h3>` +
        `<span class="doc-command__tag">${inline(tag)}</span></div>` +
        `<div class="doc-command__body">${paragraphs(inner)}` +
        (syntax ? `<pre><code>${escapeHtml(syntax)}</code></pre>` : '') +
        flagsHtml + '</div></div>'
      );
      continue;
    }

    // Headings
    const heading = line.match(/^(#{2,3})\s+(.*)$/);
    if (heading) {
      const depth = heading[1].length;
      const text = heading[2].trim();
      const id = slugify(text);
      headings.push({ depth, id, text });
      html.push(`<h${depth} id="${escapeAttr(id)}">${inline(text)}</h${depth}>`);
      i += 1;
      continue;
    }

    // Tables
    if (line.trim().startsWith('|')) {
      const block = [];
      while (i < lines.length && lines[i].trim().startsWith('|')) {
        block.push(lines[i]);
        i += 1;
      }
      html.push(tableHtml(block));
      continue;
    }

    // Lists
    if (/^\s*[-*]\s+/.test(line) || /^\s*\d+\.\s+/.test(line)) {
      const ordered = /^\s*\d+\.\s+/.test(line);
      const items = [];
      while (i < lines.length && (/^\s*[-*]\s+/.test(lines[i]) || /^\s*\d+\.\s+/.test(lines[i]))) {
        items.push(lines[i].replace(/^\s*(?:[-*]|\d+\.)\s+/, ''));
        i += 1;
      }
      const tag = ordered ? 'ol' : 'ul';
      html.push(`<${tag}>${items.map((item) => `<li>${inline(item)}</li>`).join('')}</${tag}>`);
      continue;
    }

    // Paragraph
    const paragraph = [line];
    i += 1;
    while (i < lines.length && lines[i].trim() && !isBlockStart(lines[i])) {
      paragraph.push(lines[i]);
      i += 1;
    }
    html.push(`<p>${inline(paragraph.join(' ').trim())}</p>`);
  }

  return html.join('\n');
}

function isBlockStart(line) {
  return /^#{2,3}\s/.test(line) || line.includes('<Callout') || line.includes('<Command') ||
    line.trim().startsWith('```') || line.trim().startsWith('|') ||
    /^\s*[-*]\s+/.test(line) || /^\s*\d+\.\s+/.test(line);
}

function escapeHtml(text) {
  return String(text).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function parseFrontmatter(source) {
  const match = source.match(/^---\n([\s\S]*?)\n---\n/);
  const data = {};
  if (match) {
    match[1].split('\n').forEach((line) => {
      const pair = line.match(/^([a-zA-Z]+):\s*(.*)$/);
      if (pair) data[pair[1]] = pair[2].trim().replace(/^["']|["']$/g, '');
    });
  }
  return { data, body: match ? source.slice(match[0].length) : source };
}

function pageShell(title, description, canonical, body, depth) {
  const prefix = depth === 'docs' ? '../' : '';
  const ogImage = `${SITE}/assets/favicon.svg`;
  const fullTitle = `${title} — zlt_xloader docs`;
  const crumbs = [
    { '@type': 'ListItem', position: 1, name: 'zlt_xloader', item: `${SITE}/` },
    { '@type': 'ListItem', position: 2, name: 'Documentation', item: `${SITE}/docs/` },
  ];
  if (!canonical.endsWith('/docs/')) {
    crumbs.push({ '@type': 'ListItem', position: 3, name: title, item: canonical });
  }
  const schema = [
    {
      '@type': 'SoftwareApplication',
      name: 'zlt_xloader',
      description: 'Security research and device ownership toolkit for the Tozed ZLT S20 4G router.',
      applicationCategory: 'SecurityApplication',
      operatingSystem: 'Windows, Linux, macOS',
      programmingLanguage: 'Go',
      codeRepository: 'https://github.com/anusara-bemal/zlt_xloader',
      url: `${SITE}/`,
      author: { '@type': 'Person', name: 'anusara-bemal' },
      offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
    },
    { '@type': 'BreadcrumbList', itemListElement: crumbs },
  ];
  if (canonical.includes('/faq.html')) {
    schema.push({
      '@type': 'FAQPage',
      mainEntity: [
        { '@type': 'Question', name: 'Is it legal to use zlt_xloader?', acceptedAnswer: { '@type': 'Answer', text: 'Only for lawful research on devices you own or are authorised to test.' } },
        { '@type': 'Question', name: 'Does it change my router by default?', acceptedAnswer: { '@type': 'Answer', text: 'No. Mutating commands default to dry-run and destructive operations require confirmation.' } },
        { '@type': 'Question', name: 'Does telnet persist after a reboot?', acceptedAnswer: { '@type': 'Answer', text: 'No. Use the persist command to start telnet and SSH at every boot.' } },
      ],
    });
  }
  const jsonLd = JSON.stringify({ '@context': 'https://schema.org', '@graph': schema })
    .replace(/</g, '\\u003c');

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${escapeAttr(fullTitle)}</title>
  <meta name="description" content="${escapeAttr(description)}">
  <meta name="robots" content="index,follow,max-snippet:-1,max-image-preview:large">
  <meta name="theme-color" content="#f4f1ea">
  <link rel="canonical" href="${escapeAttr(canonical)}">
  <meta property="og:type" content="article">
  <meta property="og:site_name" content="zlt_xloader">
  <meta property="og:title" content="${escapeAttr(fullTitle)}">
  <meta property="og:description" content="${escapeAttr(description)}">
  <meta property="og:url" content="${escapeAttr(canonical)}">
  <meta property="og:image" content="${escapeAttr(ogImage)}">
  <meta property="og:image:alt" content="zlt_xloader logo">
  <meta property="og:image:type" content="image/svg+xml">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="${escapeAttr(fullTitle)}">
  <meta name="twitter:description" content="${escapeAttr(description)}">
  <meta name="twitter:image" content="${escapeAttr(ogImage)}">
  <link rel="icon" type="image/svg+xml" href="${prefix}assets/favicon.svg">
  <link rel="stylesheet" href="${prefix}styles.css">
  <script type="application/ld+json">${jsonLd}</script>
</head>
<body>
  <a class="skip-link" href="#main">Skip to content</a>
${header(prefix, 'docs', false)}
${body}
  <footer class="site-foot">
    <div class="container foot-grid">
      <div>
        <a class="brand" href="${prefix}index.html"><img src="${prefix}assets/favicon.svg" alt="" width="26" height="26"><span class="brand-name">zlt_xloader</span></a>
        <p>Security research and device ownership for the Tozed ZLT S20. For lawful use on hardware you own.</p>
      </div>
      <div class="foot-col"><h5>Docs</h5><ul><li><a href="index.html">Index</a></li><li><a href="disclaimer.html">Disclaimer</a></li><li><a href="quickstart.html">Quickstart</a></li></ul></div>
      <div class="foot-col"><h5>Project</h5><ul><li><a href="${prefix}index.html#paths">Features</a></li><li><a href="changelog.html">Changelog</a></li><li><a href="contributing.html">Contributing</a></li></ul></div>
      <div class="foot-col"><h5>Reference</h5><ul><li><a href="how-it-works.html">How it works</a></li><li><a href="hardware.html">Hardware notes</a></li><li><a href="faq.html">FAQ</a></li></ul></div>
    </div>
    <div class="container foot-bottom"><div>Copyright © 2026 anusara-bemal. All rights reserved.</div><div>Educational use only — do not misuse.</div></div>
  </footer>
  <script src="${prefix}script.js"></script>
</body>
</html>
`;
}

function sidebarHtml(docs, activeId) {
  const sections = [...new Set(docs.map((doc) => doc.data.section))];
  return `<aside class="doc-sidebar" aria-label="Documentation navigation">` +
    sections.map((section) => {
      const items = docs
        .filter((doc) => doc.data.section === section)
        .map((doc) => `<li><a href="${doc.id}.html"${doc.id === activeId ? ' aria-current="page"' : ''}>${inline(doc.data.label || doc.data.title)}</a></li>`)
        .join('');
      return `<div class="doc-sidebar-group"><div class="doc-sidebar-label">${inline(section)}</div><ul>${items}</ul></div>`;
    }).join('') + '</aside>';
}

function tocHtml(headings) {
  if (!headings.length) return '';
  return `<aside class="doc-toc" aria-label="On this page"><div class="doc-toc-label">On this page</div><nav><ul>` +
    headings.map((heading) => `<li class="depth-${heading.depth}"><a href="#${escapeAttr(heading.id)}">${inline(heading.text)}</a></li>`).join('') +
    '</ul></nav></aside>';
}

function pagerHtml(index, docs) {
  const prev = index > 0 ? docs[index - 1] : null;
  const next = index < docs.length - 1 ? docs[index + 1] : null;
  return `<nav class="doc-pager" aria-label="Documentation pages">` +
    (prev ? `<a href="${prev.id}.html"><span>← Previous</span><strong>${inline(prev.data.label || prev.data.title)}</strong></a>` : '<span></span>') +
    (next ? `<a href="${next.id}.html"><span>Next →</span><strong>${inline(next.data.label || next.data.title)}</strong></a>` : '<span></span>') +
    '</nav>';
}

const docs = readdirSync(contentDir)
  .filter((file) => file.endsWith('.mdx'))
  .map((file) => {
    const source = readFileSync(join(contentDir, file), 'utf8');
    const { data, body } = parseFrontmatter(source);
    const withoutImports = body.split('\n').filter((line) => !line.startsWith('import ')).join('\n');
    return {
      id: basename(file, '.mdx'),
      data: { ...data, order: Number(data.order || 99), section: data.section || 'Guides' },
      body: withoutImports,
    };
  })
  .sort((a, b) => a.data.order - b.data.order || a.data.title.localeCompare(b.data.title));

mkdirSync(outDir, { recursive: true });

docs.forEach((doc, index) => {
  const headings = [];
  const main = renderMarkdown(doc.body, headings);
  const body = `<main id="main"><div class="container doc-shell">
    ${sidebarHtml(docs, doc.id)}
    <article class="doc-main">
      <h1>${inline(doc.data.title)}</h1>
      <p class="doc-lead">${inline(doc.data.description)}</p>
      <div class="prose">${main}</div>
      ${doc.data.updated ? `<p class="hint-inline">Last reviewed: ${escapeHtml(doc.data.updated)}</p>` : ''}
      ${pagerHtml(index, docs)}
    </article>
    ${tocHtml(headings)}
  </div></main>`;
  const page = pageShell(
    doc.data.title,
    doc.data.description,
    `${SITE}/docs/${doc.id}.html`,
    body,
    'docs'
  );
  writeFileSync(join(outDir, `${doc.id}.html`), page, 'utf8');
});

// Documentation index
const sections = [...new Set(docs.map((doc) => doc.data.section))];
const indexBody = `<main id="main"><div class="container doc-shell">
  ${sidebarHtml(docs, '')}
  <article class="doc-main">
    <h1>Documentation</h1>
    <p class="doc-lead">Everything you need to evaluate, install and use zlt_xloader responsibly on a device you own.</p>
    <div class="prose">
      <p>Start with the disclaimer if you are new to the project, then follow the quickstart for a first read-only session.</p>
      ${sections.map((section) => `<h2 id="${escapeAttr(slugify(section))}">${inline(section)}</h2>
        <div class="doc-index-grid">${docs.filter((doc) => doc.data.section === section).map((doc) => `
          <a class="doc-index-card" href="${doc.id}.html"><span>${inline(doc.data.label || doc.data.title)}</span><strong>${inline(doc.data.title)}</strong><small>${inline(doc.data.description)}</small></a>`).join('')}</div>`).join('\n')}
    </div>
  </article>
  ${tocHtml(sections.map((section) => ({ depth: 2, id: slugify(section), text: section })))}
</div></main>`;

writeFileSync(
  join(outDir, 'index.html'),
  pageShell('Documentation', 'Guides and reference for zlt_xloader: installation, safety model, every command, device profiles and how the toolkit works.', `${SITE}/docs/`, indexBody, 'docs'),
  'utf8'
);

console.log(`Generated ${docs.length + 1} documentation pages in website/docs/`);
