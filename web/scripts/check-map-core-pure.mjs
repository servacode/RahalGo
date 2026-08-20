// ══════════════════════════════════════════════════════════════════════
// **و`map-core` ترسم ولا تقرّر — ولا تعرف الملاحة**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٠ من خطّة الملاحة، شرطُ المالك ٢٠٢٦-٠٨-٢٠: «بحيث لا تتحوّل
//  وحدةُ `:map` إلى God Module ضخمة».)
//
// # ما يُقاس
//
// **الاتّجاهُ في جهةٍ واحدة**: `driver-navigation` تستورد `map` — **ولا
// عكس.** وكذلك لا تعرف `map` تطبيقاً بعينه.
//
// # ولماذا حارسٌ وجرادلُ يمنع الدورةَ أصلاً
//
// **جرادل يمنع الدورةَ بين وحدتين** — **ولا يمنع أن تُعلن `:map`
// تبعيّةً على تطبيقٍ** يوماً، ولا أن تستورد حزمتَه بمسارٍ كامل.
//
// **والحارسُ يقرأ النيّةَ لا الحلقة.**

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const ROOT = join(HERE, '..', '..')
const CORE = join(ROOT, 'mobile', 'map')

const FORBIDDEN = [
  { needle: 'com.rahalgo.navigation', why: 'وحدةُ الملاحة — والاتّجاهُ معكوس' },
  { needle: 'com.rahalgo.driver', why: 'تطبيقُ السائق' },
  { needle: 'com.rahalgo.customer', why: 'تطبيقُ الزبون' },
  { needle: 'com.rahalgo.rep', why: 'تطبيقُ المندوب' },
  { needle: ':driver-navigation', why: 'تبعيّةُ بناءٍ على الملاحة' },
]

function walk(dir) {
  const out = []
  for (const name of readdirSync(dir)) {
    if (name === 'build') continue
    const p = join(dir, name)
    if (statSync(p).isDirectory()) out.push(...walk(p))
    else if (/\.(kt|kts)$/.test(name)) out.push(p)
  }
  return out
}

const files = walk(CORE)
if (files.length === 0) {
  console.error('لا ملفّاتِ في map-core — تغيّرت البنيةُ فأصلحْ الحارس.')
  process.exit(1)
}

const bad = []
for (const f of files) {
  const src = readFileSync(f, 'utf8')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '')
  for (const { needle, why } of FORBIDDEN) {
    if (src.includes(needle)) bad.push(`${f.slice(ROOT.length + 1)} → ${needle} (${why})`)
  }
}

if (bad.length) {
  console.error('map-core يعرف ما لا يجوز أن يعرفه:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\n**map-core ترسم ولا تقرّر** — والمنطقُ في driver-navigation.')
  process.exit(1)
}
console.log(`map-core نقيّةٌ — ${files.length} ملفّاً لا يعرف ملاحةً ولا تطبيقا.`)
