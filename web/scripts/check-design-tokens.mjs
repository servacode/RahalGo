// ══════════════════════════════════════════════════════════════════════
// **الشكلُ الخامُ لا يزيد — سقفٌ ينزل ولا يصعد**
// ══════════════════════════════════════════════════════════════════════
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٨: «بدنا نوحّد ستايلَ التطبيقات مشان كلّ
//  التطبيقات تطلع بنفس التصميم والستايل… يعني مو كلّ تطبيقٍ وكلّ صفحةٍ
//  وكلّ قسمٍ تصميمه على كيفه».)
//
// # لماذا سقفٌ لا منعٌ تامّ
//
// **المنعُ التامُّ يعني إصلاحَ ٥٢ شكلاً و١١٢ زرّاً في دفعةٍ واحدة** —
// وهو ما نهى عنه المالكُ صراحةً («دفعةٌ واحدةٌ في كلّ مرّة»، بعد يومِ
// السبعِ التزاماتٍ التي أُرجعت كلُّها).
//
// **والحارسُ يجب أن يعمل اليومَ لا بعد شهر**: حارسٌ يُكتب ويبقى مطفأً
// حتّى تكتمل الهجرةُ **لا يحرس الهجرةَ نفسَها** — ويُضاف شكلٌ خامٌّ
// جديدٌ في أثنائها فلا يصرخ أحد.
//
// **وقد اكتملت الهجرةُ في اليوم نفسِه** (٢٠٢٦-٠٨-١٨): نزل السقفُ من
// ١٦٥ إلى ٣ — **وهي التعريفاتُ الثلاثةُ في `ui/Buttons.kt` وحدَها**،
// وهي الموضعُ الوحيدُ الذي يُمسّ فيه Material. **فصار السقفُ عند
// الصفر منعاً تامّاً** بلا أن يُعاد كتابةُ الحارس.
//
// **وتبقى المسافاتُ خارجَه** — ٧٢٥ موضعاً، وتقريبُها إلى السلّم
// **يُغيّر ما يراه المالكُ في كلّ شاشة**، فلا يقع إلّا بأمرِه.
//
// # ولماذا يصرخ حين ينقص أيضا
//
// **سقفٌ ينزل بلا أن يُسجَّل يعود يصعد.** فمن أصلح موضعاً وجب أن يُنزل
// الرقمَ في `design-tokens-baseline.json` — **فيصير التقدّمُ مكتوباً
// في المستودع لا محفوظاً في ذاكرةِ أحد.**

import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const MOBILE = join(HERE, '..', '..', 'mobile')
const BASELINE = join(HERE, 'design-tokens-baseline.json')

// **والوحدتان مُستثناتان**: `design` تُعرّف التوكنز، و`ui` هي العدّةُ
// المشتركة — **وهي التي يُفترض أن تكتب الشكلَ نيابةً عن الشاشات.**
// (ويبقى `ui` في العدّ ليُرى، ولا يُستثنى من السقف.)
const SCOPE = ['app-customer', 'app-driver', 'app-rep', 'ui']

/** **ما يُعَدّ** — كلٌّ بسببه. */
const RULES = {
  // **نصفُ قطرٍ مكتوبٌ بيد** — البديل `Rahal.shape.md`.
  rawShape: /RoundedCornerShape\s*\(/g,
  // **زرُّ Material خام** — لا يعرف شكلَ المنصّة ولا حشوتَها.
  rawButton: /(?<![A-Za-z])(Button|OutlinedButton|TextButton|FilledTonalButton)\s*\(/g,
}

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt')) out.push(p)
  }
  return out
}

const now = {}
for (const mod of SCOPE) {
  now[mod] = { rawShape: 0, rawButton: 0 }
  for (const file of kt(join(MOBILE, mod, 'src'))) {
    const src = readFileSync(file, 'utf8')
    for (const [rule, re] of Object.entries(RULES)) {
      now[mod][rule] += (src.match(re) || []).length
    }
  }
}

// **و`--record` تكتب السقفَ الحاليّ** — تُنادى مرّةً عند إنشاء الحارس،
// **وبعدها بيدِ من أصلح** لا تلقائيّا: تحديثٌ تلقائيٌّ يعني سقفاً
// يتبع الواقعَ بدل أن يحكمه.
if (process.argv.includes('--record')) {
  writeFileSync(BASELINE, JSON.stringify(now, null, 2) + '\n', 'utf8')
  console.log('سُجّل السقفُ الحاليّ في design-tokens-baseline.json')
  process.exit(0)
}

const base = JSON.parse(readFileSync(BASELINE, 'utf8'))
const up = []
const down = []
for (const mod of SCOPE) {
  for (const rule of Object.keys(RULES)) {
    const was = base[mod]?.[rule] ?? 0
    const is = now[mod][rule]
    if (is > was) up.push(`${mod} · ${rule}: ${was} ← صارت ${is}`)
    else if (is < was) down.push(`${mod} · ${rule}: ${was} ← صارت ${is}`)
  }
}

if (up.length) {
  console.error('شكلٌ خامٌّ جديد — والسقفُ لا يصعد:')
  for (const u of up) console.error('  · ' + u)
  console.error('\nالبديل: Rahal.shape.md · Rahal.space.lg — وأزرارُ العدّة المشتركة.')
  process.exit(1)
}
if (down.length) {
  console.error('نزل الخامُّ — فأنزل السقفَ معه (node scripts/check-design-tokens.mjs --record):')
  for (const d of down) console.error('  · ' + d)
  process.exit(1)
}

const total = SCOPE.reduce((s, m) => s + now[m].rawShape + now[m].rawButton, 0)
console.log(`سقفُ الشكل الخامّ ثابت — ${total} موضعاً باقياً، ولا موضعَ جديد.`)
