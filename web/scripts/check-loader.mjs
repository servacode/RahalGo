// ══════════════════════════════════════════════════════════════════════
// **الانتظارُ بموجات العلامة — لا بدوّارة النظام**
// ══════════════════════════════════════════════════════════════════════
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٨: «شاشةُ التحميل بشكلٍ احترافيٍّ يناسب التطبيقَ
//  والمشروع… بأسلوبٍ مميّزٍ وجميلٍ ومركزيّ».)
//
// # ما يُقاس
//
// **الدوّارةُ العارية** (`CircularProgressIndicator()` بلا وسائط) —
// وهي التي تُرسم في وسط الشاشة انتظاراً. **البديل `RahalLoader()`.**
//
// **ولا تُمنع الدوّارةُ الصغيرة** (`Modifier.size(18.dp)` داخل زرّ):
// موجاتٌ في ثمانيةَ عشرَ نقطةً على أرضٍ ملوّنةٍ **لا تُرى**، والدوّارةُ
// هناك تقول «هذا الزرُّ يعمل» ولا تدّعي هويّة.
//
// **وحارسٌ يمنع الصوابَ يُطفَأ** — فيُمنع ما يُرى وحدَه.

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const MOBILE = join(HERE, '..', '..', 'mobile')
const MODULES = ['app-customer', 'app-driver', 'app-rep', 'ui']

// **ملفُّ التعريف يُستثنى** — لا شيءَ فيه أصلا، ويُذكر ليُعرف السبب.
const SKIP = new Set(['Loading.kt'])

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt') && !SKIP.has(name)) out.push(p)
  }
  return out
}

const BARE = /CircularProgressIndicator\s*\(\s*\)/

const bad = []
for (const mod of MODULES) {
  for (const file of kt(join(MOBILE, mod, 'src'))) {
    const src = readFileSync(file, 'utf8')
    if (BARE.test(src)) bad.push(file.slice(MOBILE.length + 1))
  }
}

if (bad.length) {
  console.error('انتظارٌ بدوّارة النظام — وهي نفسُها في كلّ تطبيقٍ على الجهاز:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: RahalLoader() — موجاتُ العلامة، أو LoadingScreen(text).')
  process.exit(1)
}
console.log('الانتظارُ بموجات العلامة — لا دوّارةَ نظامٍ عاريةٍ في شاشة.')
