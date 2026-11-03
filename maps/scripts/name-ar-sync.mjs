#!/usr/bin/env node
// يولّد web/packages/ui/src/mapNameAr.ts من maps/config/name-ar-overrides.csv
// (المصدر الوحيد). `--check` يتحقق دون أن يكتب.
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const CSV = join(root, 'maps', 'config', 'name-ar-overrides.csv');
const TS = join(root, 'web', 'packages', 'ui', 'src', 'mapNameAr.ts');

/** يقرأ CSV بسيطاً (حقل أول قد يُقتبس، والعربيّ بلا فواصل). */
export function parseOverrides(text) {
  const out = {};
  for (const line of text.split(/\r?\n/).slice(1)) {
    if (!line.trim()) continue;
    const i = line.lastIndexOf(',');
    let name = line.slice(0, i);
    if (name.startsWith('"') && name.endsWith('"')) name = name.slice(1, -1).replace(/""/g, '"');
    out[name] = line.slice(i + 1).trim();
  }
  return out;
}

export function readTs(text) {
  const a = text.indexOf('= {');
  return JSON.parse(text.slice(a + 2, text.lastIndexOf(';')));
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const want = parseOverrides(readFileSync(CSV, 'utf8'));
  const sorted = Object.fromEntries(Object.entries(want).sort(([a], [b]) => (a < b ? -1 : 1)));
  const old = readFileSync(TS, 'utf8');
  const head = old.slice(0, old.indexOf('= {') + 2);
  const next = `${head}${JSON.stringify(sorted, null, 2)};\n`;
  if (process.argv.includes('--check')) {
    process.exit(next === old ? 0 : 1);
  }
  writeFileSync(TS, next);
  console.log(Object.keys(sorted).length, 'entries');
}
