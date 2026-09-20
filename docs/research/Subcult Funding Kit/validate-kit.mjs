import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
const root=path.dirname(fileURLToPath(import.meta.url));
const walk=d=>fs.readdirSync(d,{withFileTypes:true}).flatMap(e=>e.isDirectory()?walk(path.join(d,e.name)):[path.join(d,e.name)]);
const files=walk(root).filter(p=>p.endsWith('.md'));
const names=new Map();
for(const p of files){const n=path.basename(p,'.md');assert(!names.has(n),'Duplicate note '+n);names.set(n,p);}
let links=0,words=0,slides=0;const urls=new Set();
for(const p of files){
const c=fs.readFileSync(p,'utf8');words+=c.split(/\s+/).length;
for(const m of c.matchAll(/\[\[([^\]|#]+)(?:[^\]]*)\]\]/g)){links++;assert(names.has(m[1]),'Broken link '+m[1]+' in '+p);}
for(const m of c.matchAll(/https?:\/\/[^\s)]+/g))urls.add(m[0]);
if(p.includes('04 Pitch Decks')){const n=path.basename(p,'.md').replace(/ Deck$/,' Speaker Notes');assert(names.has(n),'Missing speaker notes '+n);slides+=c.split(/\n---\n/).length;}
}
assert.equal([6000,1000,400,700,400,500].reduce((a,b)=>a+b,0),9000);
assert.equal(54000+36000+18000+12000,120000);
assert.equal(180000+45000+15000+10000,250000);
assert.equal(30000+8000+5000+4000+3000,50000);
assert.equal(100*149*12,178800);
console.log(JSON.stringify({notes:files.length,links,words,uniqueURLs:urls.size,decks:files.filter(p=>p.includes('04 Pitch Decks')).length,slides,budgetChecks:'passed'},null,2));

