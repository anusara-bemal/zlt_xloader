/* Dev-time helper: writes the shared navigation into index.html so the home
   page and generated docs pages always use exactly the same header. */
import { readFileSync, writeFileSync } from 'node:fs';
import { header } from './nav.mjs';

const file = new URL('../index.html', import.meta.url);
let html = readFileSync(file, 'utf8');

const start = html.indexOf('  <header class="site-head">');
const endMarker = '\n  </header>';
const end = html.indexOf(endMarker, start);

if (start === -1 || end === -1) {
  console.error('Could not locate the header block in index.html');
  process.exit(1);
}

html = html.slice(0, start) + header('', '', true) + html.slice(end + endMarker.length);
writeFileSync(file, html, 'utf8');
console.log('Synchronised navigation in index.html');
