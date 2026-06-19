import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(fileURLToPath(new URL('..', import.meta.url)));
const schemaPath = resolve(root, 'contracts/api.schema.json');
const webPath = resolve(root, 'web/src/domain.ts');
const mobilePath = resolve(root, 'mobile/src/api/types.ts');

function readText(filePath) {
  return readFileSync(filePath, 'utf8');
}

function parseSchema(filePath) {
  return JSON.parse(readText(filePath)).dtos ?? {};
}

function parseInterfaces(source) {
  const interfaces = new Map();
  const pattern = /export interface\s+([A-Za-z0-9_]+)(?:\s+extends\s+([^\{]+?))?\s*\{/g;

  for (let match; (match = pattern.exec(source)); ) {
    const name = match[1];
    const extendsList = (match[2] ?? '')
      .split(',')
      .map((part) => part.trim())
      .filter(Boolean)
      .map((part) => part.replace(/\s+$/, ''));

    const bodyStart = pattern.lastIndex;
    let depth = 1;
    let index = bodyStart;
    while (index < source.length && depth > 0) {
      const char = source[index];
      if (char === '{') depth += 1;
      if (char === '}') depth -= 1;
      index += 1;
    }

    const body = source.slice(bodyStart, index - 1);
    interfaces.set(name, { extendsList, body });
    pattern.lastIndex = index;
  }

  return interfaces;
}

function collectProps(name, interfaces, seen = new Set()) {
  if (seen.has(name)) return new Set();
  seen.add(name);

  const entry = interfaces.get(name);
  if (!entry) return new Set();

  const props = new Map();
  const lines = entry.body.split(/\r?\n/);
  for (const line of lines) {
    const match = line.match(/^\s*([A-Za-z0-9_]+)(\?)?\s*:/);
    if (match) props.set(match[1], { optional: Boolean(match[2]) });
  }

  for (const base of entry.extendsList) {
    for (const [prop, details] of collectProps(base, interfaces, seen)) {
      if (!props.has(prop)) props.set(prop, details);
    }
  }

  return props;
}

function fail(message) {
  console.error(message);
  process.exitCode = 1;
}

const schema = parseSchema(schemaPath);
const webInterfaces = parseInterfaces(readText(webPath));
const mobileInterfaces = parseInterfaces(readText(mobilePath));

for (const [dtoName, spec] of Object.entries(schema)) {
  const required = Array.isArray(spec.required) ? spec.required : [];
  for (const [label, interfaces] of [
    ['web', webInterfaces],
    ['mobile', mobileInterfaces],
  ]) {
    if (!interfaces.has(dtoName)) {
      fail(`[${label}] missing DTO ${dtoName}`);
      continue;
    }

    const props = collectProps(dtoName, interfaces);
    for (const prop of required) {
      if (!props.has(prop)) {
        fail(`[${label}] ${dtoName} missing property ${prop}`);
        continue;
      }
      if (props.get(prop).optional) {
        fail(`[${label}] ${dtoName} property ${prop} is optional but contract marks it required`);
      }
    }
  }
}

if (process.exitCode) {
  process.exit(process.exitCode);
}
