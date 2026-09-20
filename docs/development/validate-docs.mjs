import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
const root=path.dirname(fileURLToPath(import.meta.url));
const files=fs.readdirSync(root).filter(n=>n.endsWith('.md'));
let links=0,words=0;
for(const n of files){
const c=fs.readFileSync(path.join(root,n),'utf8');
assert(c.startsWith('# '),'Missing title: '+n);
assert(!/[ \t]+$/m.test(c),'Trailing whitespace: '+n);
words+=c.split(/\s+/).length;
for(const m of c.matchAll(/\]\(([^)]+)\)/g)){
if(/^https?:/.test(m[1]))continue;
const target=decodeURIComponent(m[1].split('#')[0]);
assert(fs.existsSync(path.resolve(root,target)),'Broken link '+n+': '+target);links++;
}
}
const backlog=fs.readFileSync(path.join(root,'backlog.md'),'utf8');
const ids=[...backlog.matchAll(/^## ([A-Z]+-\d+) —/gm)].map(m=>m[1]);
assert.equal(new Set(ids).size,15);
for(const section of backlog.split(/^## /m).slice(1)){
if(!/^[A-Z]+-\d+ —/.test(section))continue;
assert(section.includes('**Acceptance:**') && section.includes('**Verification:**') && section.includes('**Rollback:**'),'Incomplete task');
}
console.log(JSON.stringify({documents:files.length,internalLinks:links,words,backlogItems:ids.length,result:'passed'},null,2));
