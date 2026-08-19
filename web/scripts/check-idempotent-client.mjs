// ══════════════════════════════════════════════════════════════════════
// **الحمايةُ عقدٌ بين طرفين — ونصفُه في التطبيق**
// ══════════════════════════════════════════════════════════════════════
//
// (تدقيقُ الإطلاق ٢٠٢٦-٠٨-١٩ — BUG-001.)
//
// # ما وُجد
//
// **الخادمُ يحرس `POST /orders` منذ بُني** — **والتطبيقُ لا يرسل
// `Idempotency-Key`.** والوسيطُ يتخطّى الحراسةَ حين يكون فارغاً:
// `if key == "" { next(); return }`.
//
// **فقراءةُ `server.go` وحدَها تُوهم أنّ البابَ محميّ** — وهو مكشوف.
//
// # والخطرُ ليس الضغطتين المتتاليتين
//
// **تلك يمنعها `enabled = !busy`.** الخطرُ أن يُنشأ الطلبُ في الخادم
// **ثمّ تنقطع الشبكةُ قبل أن يصل الردّ**: يرى «تعذّر» فيضغط ثانيةً —
// **فطلبان وسائقان وخصمان.**
//
// # وما يُقاس
//
// **كلُّ نداءٍ يُنشئ شيئاً يُحاسَب عليه يمرّر `idempotencyKey`.**
// والقائمةُ صريحةٌ لا مُخمَّنة — تُزاد كلَّما فُتح بابُ إنشاءٍ جديد.
//
// **ولا يُقاس أنّ المفتاحَ ثابتٌ بين المحاولات** — ذاك منطقُ نموذجٍ
// لا يُقرأ بنصّ. **وحارسٌ يُخمّن يُنذر على الصواب فيُطفَأ.**

import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const ROOT = join(HERE, '..', '..')

/** نداءاتٌ تُنشئ ما يُحاسَب عليه: الملفّ ← الدالّة. */
const CREATORS = [
  { file: 'mobile/shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt',
    fn: 'createOrder' },
  { file: 'mobile/shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt',
    fn: 'createCustom' },
  { file: 'mobile/shared/src/main/kotlin/com/rahalgo/shared/driver/MeApi.kt',
    fn: 'requestPayout' },
]

const bad = []
for (const c of CREATORS) {
  const src = readFileSync(join(ROOT, c.file), 'utf8')
  const at = src.indexOf(`suspend fun ${c.fn}(`)
  if (at < 0) {
    bad.push(`${c.fn} — لم تُوجد، تغيّر اسمُها فأصلحْ الحارس`)
    continue
  }
  // **جسدُ الدالّة** — حتّى الدالّة التالية أو ٨٠٠ حرف.
  const rest = src.slice(at)
  const next = rest.indexOf('\n    suspend fun ', 10)
  const body = rest.slice(0, next > 0 ? next : Math.min(800, rest.length))
  if (!body.includes('idempotencyKey')) {
    bad.push(`${c.fn} — تُنشئ بلا Idempotency-Key: **إعادةُ المحاولة تُنشئ ثانيا**`)
  }
}

if (bad.length) {
  console.error('نداءُ إنشاءٍ بلا مفتاحِ منعِ التكرار:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: مفتاحٌ يُولَّد للمحاولة ويبقى حتّى تنجح — لا لكلّ ضغطة.')
  process.exit(1)
}
console.log(`نداءاتُ الإنشاء تحمل مفتاحَها — ${CREATORS.length} نداءً محروسا.`)
