// ══════════════════════════════════════════════════════════════════════
// **حقلٌ لا وجودَ له في الردّ — يسقط إلى فراغه بلا خطأ**
// ══════════════════════════════════════════════════════════════════════
//
// # عائلةُ الخلل
//
// **`ignoreUnknownKeys` تجعل الاسمَ الخاطئَ صامتاً**: يُقرأ الحقلُ
// فيوجد غائباً **فيأخذ قيمتَه الافتراضيّة** — لا خطأ، لا سجلّ، لا شيء.
//
// **وعضّت مرّتين:**
//
// - **٢٠٢٦-٠٨-١٨**: `OrderRef.code` والمحرّكُ يرسل `number` — فكان رقمُ
//   الطلب صفراً في رسالة النجاح.
// - **٢٠٢٦-٠٨-١٨**: `Banner.link_url` والمحرّكُ يرسل `target` — **فكلُّ
//   لافتةٍ بلا وجهة**، تُنشَر من اللوحة وتُضغط فلا يقع شيء.
//
// # وما يُقاس
//
// **كلُّ `@SerialName` في النموذج له `json:"..."` في بنية المحرّك.**
//
// **والقائمةُ صريحةٌ لا مُخمَّنة**: مطابقةُ كلّ نموذجٍ ببنيةٍ تلقائيّاً
// تحتاج تحليلَ لغتين، **وحارسٌ يُخمّن يُنذر على الصواب فيُطفَأ.**
// تُضاف الأزواجُ حين يُكتشف زوجٌ جديد.

import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const ROOT = join(HERE, '..', '..')

/** أزواجٌ يُحرَسان: نموذجُ كوتلن ← بنيةُ Go. */
const PAIRS = [
  {
    kt: 'mobile/shared/src/main/kotlin/com/rahalgo/shared/model/Shop.kt',
    ktClass: 'Banner',
    go: 'backend/internal/catalog/marketing.go',
    goStruct: 'Banner',
  },
  {
    kt: 'mobile/shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt',
    ktClass: 'PromoPreview',
    go: 'backend/internal/orders/service.go',
    goStruct: 'PromoPreview',
  },
  {
    kt: 'mobile/shared/src/main/kotlin/com/rahalgo/shared/model/Shop.kt',
    ktClass: 'Section',
    go: 'backend/internal/server/sections_handlers.go',
    goStruct: 'publicSection',
  },
]

/** **جسدُ صنفٍ في كوتلن** — من ترويسته إلى قوسه الأخير. */
const ktBody = (src, name) => {
  const at = src.indexOf(`data class ${name}(`)
  if (at < 0) return null
  const from = src.indexOf('(', at)
  let depth = 0
  for (let i = from; i < src.length; i++) {
    if (src[i] === '(') depth++
    else if (src[i] === ')') {
      depth--
      if (depth === 0) return src.slice(from, i)
    }
  }
  return null
}

/** **جسدُ بنيةٍ في Go.** */
const goBody = (src, name) => {
  const at = src.indexOf(`type ${name} struct {`)
  if (at < 0) return null
  const end = src.indexOf('\n}', at)
  return end < 0 ? null : src.slice(at, end)
}

const bad = []
let checked = 0
for (const p of PAIRS) {
  const kt = ktBody(readFileSync(join(ROOT, p.kt), 'utf8'), p.ktClass)
  const go = goBody(readFileSync(join(ROOT, p.go), 'utf8'), p.goStruct)
  if (!kt || !go) {
    bad.push(`${p.ktClass} — لم يُوجد أحدُ الطرفين، فأصلحْ مسارَ الحارس`)
    continue
  }
  // **وأسماءُ الحقول في Go** — من وسم `json` وحدَه، وما قبل الفاصلة.
  const goFields = new Set(
    [...go.matchAll(/json:"([^",]+)/g)].map((m) => m[1]).filter((f) => f !== '-'),
  )
  // **وحقولُ كوتلن الموسومة** — وغيرُ الموسومة اسمُها اسمُها.
  for (const m of kt.matchAll(/@SerialName\("([^"]+)"\)/g)) {
    checked++
    if (!goFields.has(m[1])) {
      bad.push(`${p.ktClass}.${m[1]} — لا وجودَ له في ${p.goStruct}: يُقرأ فارغاً أبدا`)
    }
  }
}

if (checked === 0) {
  console.error('حارسُ الأسماء لم يفحص شيئاً — تغيّرت البنى، فأصلحْ أزواجَه')
  process.exit(1)
}
if (bad.length) {
  console.error('أسماءُ حقولٍ مخترَعة — تسقط إلى فراغها بلا خطأ:')
  for (const b of bad) console.error('  · ' + b)
  process.exit(1)
}
console.log(`أسماءُ الحقول تُطابق المحرّك — ${checked} اسماً موسوماً في زوجين.`)
