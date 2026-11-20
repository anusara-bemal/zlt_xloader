# zlt_xloader website

A static product site for [zlt_xloader](https://github.com/anusara-bemal/zlt_xloader). Plain HTML, CSS and vanilla JavaScript — no framework, no package install, no build step at deploy time.

## Layout

```
website/
  index.html          full single-page product site
  styles.css          design system + all sections
  script.js           command catalogue, filters, tabs, copy buttons, nav
  assets/             favicon.svg (single logo/mark used by site and meta), real CLI capture, device cutout
  content/            documentation source (.mdx, carried over from the repo docs)
  docs/               generated static documentation pages (committed)
  tools/nav.mjs       one shared header definition for home and docs
  tools/sync-nav.mjs  writes the shared header into index.html
  tools/build-docs.mjs  zero-dependency docs generator
  sitemap.xml, robots.txt, .nojekyll
```

## Preview locally

```bash
cd website
python -m http.server 8000
# http://localhost:8000/
```

For Windows PowerShell: `py -m http.server 8000`.

## Editing docs

The docs pages in `docs/` are generated from `content/*.mdx` by:

```bash
node website/tools/build-docs.mjs
```

The generator has no dependencies. It produces one HTML page per source file plus `docs/index.html`, with sidebar, table of contents, previous/next links, canonical URLs, structured data and consistent styling. Regenerate and commit the output whenever `content/` or `tools/nav.mjs` changes.

The navigation lives in `tools/nav.mjs` only. After editing it, run both:

```bash
node website/tools/sync-nav.mjs
node website/tools/build-docs.mjs
```

Do not hand-edit the header in `index.html` or any files in `docs/` — they are overwritten by those scripts.

## Hosting

The site is hosted at the root of the domain. `.github/workflows/deploy-website.yml` uploads this folder directly to GitHub Pages using **GitHub Actions** as the Pages source.

Two optional repository variables:

| Variable | Purpose |
| --- | --- |
| `SITE_URL` | Final public origin (for example `https://example.com`). The workflow rewrites canonical, OG and sitemap URLs in the published copy. |
| `PAGES_CNAME` | Custom domain. When set, the workflow writes it as `CNAME` into the published site. |

If neither is set, the default origin `https://anusara-bemal.github.io` is used.

## Content rules

- Use real command output and real project facts only. `assets/cli.png` is a capture of the shipped binary; do not replace it with an illustration.
- Keep the educational-use and owned-hardware-only notice prominent.
- The tool's source of truth is the repository README and the Cobra command definitions; keep `content/` and the command catalogue in `script.js` in sync.
