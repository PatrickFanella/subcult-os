// Optional maintenance check. Run: node validate-dossier.mjs
// Uses only Node built-ins. Does not fetch URLs, execute note content, or write files.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

const root = path.dirname(fileURLToPath(import.meta.url));
const files = [];
const errors = [];
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isSymbolicLink()) { errors.push('Symlink not permitted: ' + full); continue; }
    if (entry.isDirectory()) walk(full);
    else if (entry.name.endsWith('.md')) files.push(full);
  }
}
walk(root);
const notes = new Map();
for (const file of files) {
  const stem = path.basename(file, '.md');
  if (notes.has(stem)) errors.push('Duplicate stem: ' + stem);
  notes.set(stem, { file, text: fs.readFileSync(file, 'utf8'), incoming: 0, edges: [] });
}
let wikiLinks = 0;
let words = 0;
const urls = new Set();
for (const [stem, note] of notes) {
  const front = note.text.match(/^---\n([\s\S]*?)\n---\n/);
  if (!front) errors.push('Missing frontmatter: ' + stem);
  else {
    for (const key of ['type', 'status', 'created', 'research_as_of', 'tags']) {
      if (!new RegExp('^' + key + ':', 'm').test(front[1])) errors.push('Missing metadata ' + key + ': ' + stem);
    }
    if (!/^created: \d{4}-\d{2}-\d{2}$/m.test(front[1])) errors.push('Invalid created date: ' + stem);
  }
  const body = note.text.slice(front?.[0].length ?? 0);
  if (!/^# \S/m.test(body)) errors.push('Missing title: ' + stem);
  const count = body.trim().split(/\s+/).length;
  words += count;
  if (count < 100) errors.push('Undeveloped note under 100 tokens: ' + stem);
  if (/\b(?:TODO|TBD|lorem ipsum)\b/i.test(body)) errors.push('Placeholder marker: ' + stem);
  if (/[ \t]+\n/.test(note.text)) errors.push('Trailing whitespace: ' + stem);
  for (const match of body.matchAll(/\[\[([^\]]+)\]\]/g)) {
    wikiLinks++;
    const raw = match[1].split('|')[0];
    const [target, heading] = raw.split('#');
    const dest = notes.get(target || stem);
    if (!dest) { errors.push('Missing wikilink: ' + stem + ' -> ' + raw); continue; }
    dest.incoming++;
    note.edges.push(target || stem);
    if (heading && !dest.text.split('\n').some(line => line.replace(/^#{1,6}\s+/, '').trim() === heading)) {
      errors.push('Missing heading: ' + raw);
    }
  }
  for (const match of body.matchAll(/\[[^\]]*\]\(([^)\s]+)\)/g)) {
    const target = match[1];
    if (/^https?:\/\//.test(target)) { urls.add(target); continue; }
    const decoded = decodeURIComponent(target.split('#')[0]);
    if (decoded.startsWith('/') || decoded.startsWith('file:')) errors.push('Nonportable link: ' + stem);
    if (decoded && !fs.existsSync(path.resolve(path.dirname(note.file), decoded))) errors.push('Missing relative path: ' + target);
  }
}
for (const [stem, note] of notes) {
  if (stem !== 'README' && !note.incoming) errors.push('Orphan note: ' + stem);
}
const reached = new Set();
function visit(stem) {
  if (reached.has(stem) || !notes.has(stem)) return;
  reached.add(stem);
  for (const next of notes.get(stem).edges) visit(next);
}
visit('README');
for (const stem of notes.keys()) if (!reached.has(stem)) errors.push('Unreachable from README: ' + stem);

// Independent recomputation of every displayed quantitative scenario.
const checks = [
  [50 * 39 + 20 * 99, 3930], [(50 * 39 + 20 * 99) * 12, 47160],
  [250 * 39 + 100 * 99, 19650], [(250 * 39 + 100 * 99) * 12, 235800],
  [1000 * 0.02 * 39 * 12, 9360], [1000 * 0.05 * 59 * 12, 35400],
  [1000 * 0.1 * 99 * 12, 118800], [50 * 39 * 12, 23400],
  [50 * 99 * 12, 59400], [20 * 59 * 12, 14160], [100 * 59 * 12, 70800],
  [99 - 8 - 40 * 0.5, 71], [99 - 8 - 40, 51], [99 - 8 - 40 * 2, 11],
  [39 - 8 - 40 * 0.5, 11], [2000 * 2 * 1 * 0.02, 80], [4 * 12, 48],
  [Math.round(71 / 99 * 1000) / 10, 71.7],
  [Math.round(51 / 99 * 1000) / 10, 51.5],
  [Math.round(11 / 99 * 1000) / 10, 11.1],
];
for (const [actual, expected] of checks) assert.equal(actual, expected);
console.log(JSON.stringify({
  markdownNotes: files.length,
  whitespaceDelimitedWords: words,
  wikilinks: wikiLinks,
  uniqueExternalUrls: urls.size,
  reachableNotes: reached.size,
  arithmeticChecks: checks.length,
  errors
}, null, 2));
if (errors.length) process.exitCode = 1;

