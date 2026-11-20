/* Single source of truth for the site navigation, shared by the home page
   and the generated documentation pages. */

const PRIMARY = [
  ['paths', 'Access paths'],
  ['gate', 'Write gate'],
  ['variants', 'Variants'],
  ['setup', 'Setup'],
  ['commands', 'Commands'],
];

const DOCS = 'https://github.com/anusara-bemal/zlt_xloader';

export function header(prefix = '', active = '', isHome = false) {
  const home = isHome ? '#top' : `${prefix}index.html`;
  const section = (hash) => (isHome ? `#${hash}` : `${prefix}index.html#${hash}`);
  const binary = isHome ? '#setup' : `${prefix}index.html#setup`;
  const docsHref = isHome ? 'docs/index.html' : 'index.html';
  const item = ([hash, label]) =>
    `<li><a href="${section(hash)}"${active === hash ? ' aria-current="page"' : ''}>${label}</a></li>`;
  const docsItem = `<li><a href="${docsHref}"${active === 'docs' ? ' aria-current="page"' : ''}>Docs ↗</a></li>`;

  return `  <header class="site-head">
    <div class="container head-inner">
      <a class="brand" href="${home}" aria-label="zlt_xloader home">
        <img src="${prefix}assets/favicon.svg" alt="" width="26" height="26">
        <span class="brand-name">zlt_xloader</span>
        <span class="brand-ver">FIELD MANUAL / 1.0</span>
      </a>

      <nav aria-label="Primary">
        <ul class="nav">
          ${PRIMARY.map(item).join('\n          ')}
          ${docsItem}
        </ul>
      </nav>

      <div class="head-end">
        <a class="head-link" href="${DOCS}" target="_blank" rel="noopener">GitHub ↗</a>
        <a class="btn" href="${binary}">Get the binary</a>
        <button id="menu-btn" class="menu-btn" type="button" aria-label="Toggle navigation" aria-expanded="false" aria-controls="mobile-nav">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <line x1="3" y1="7" x2="21" y2="7"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="17" x2="21" y2="17"/>
          </svg>
        </button>
      </div>
    </div>
    <nav id="mobile-nav" class="mobile-nav" aria-label="Mobile navigation">
      <ul>
        ${PRIMARY.map(item).join('\n        ')}
        <li><a href="${section('safety')}">Safety &amp; legal</a></li>
        <li><a href="${docsHref}">Documentation ↗</a></li>
      </ul>
    </nav>
  </header>`;
}
