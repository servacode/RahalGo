// ══════════════════════════════════════════════════════════════════════
// **الإلغاءُ ليس خطأً — ومن ابتلعه عرضه على أنّه عطب**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨ بصورةِ شاشة: «تعذّر إتمام الطلب (gs1)» تحت
//  حقل البحث.)
//
// # ما وقع
//
// `gs1` اسمُ `CancellationException` بعد تشويش R8. **وكلُّ حرفٍ يُكتب
// في البحث يُلغي بحثَ سابقِه** — وهو مقصود. فإن كان السابقُ قد تجاوز
// المهلةَ ودخل في النداء **خرج بـ`CancellationException`**،
// و`catch (e: Exception)` تبتلعها فتُعرض على أنّها عطبُ خادم.
//
// **فيُقرأ التطبيقُ مكسوراً وهو يعمل كما صُمّم** — والرمزُ المعروضُ لا
// معنى له، **ولم يُعرف سببُه إلّا بقراءة سجلّ الجهاز.**
//
// # ولماذا يُعاد رميُه لا يُهمَل
//
// **الإلغاءُ يسري في شجرة المهامّ بالاستثناء نفسِه.** ومن ابتلعه قطع
// السريان: **يُلغى النطاقُ ويبقى ما فيه يعمل** حتّى يُتلف النموذج.
//
// # وما يُقاس
//
// **كلُّ ملفٍّ يُلغي مهمّةً ويلتقط `Exception` يجب أن يذكر
// `CancellationException`.** والقاعدةُ صريحةٌ لا مُخمَّنة، **وحارسٌ
// يُخمّن يُنذر على الصواب فيُطفَأ.**

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const MOBILE = join(HERE, '..', '..', 'mobile')
const MODULES = ['app-customer', 'app-driver', 'app-rep', 'ui', 'shared', 'map']

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt')) out.push(p)
  }
  return out
}

const CANCELS = /\.cancel\s*\(\s*\)/
const CATCHES = /catch\s*\(\s*\w+\s*:\s*(Exception|Throwable)\s*\)/

const bad = []
let checked = 0
for (const mod of MODULES) {
  for (const file of kt(join(MOBILE, mod, 'src'))) {
    const src = readFileSync(file, 'utf8')
    if (!CANCELS.test(src) || !CATCHES.test(src)) continue
    checked++
    if (!src.includes('CancellationException')) {
      bad.push(file.slice(MOBILE.length + 1))
    }
  }
}

if (bad.length) {
  console.error('ملفّاتٌ تُلغي مهمّةً وتبتلع إلغاءَها فتعرضه عطبا:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: catch (e: CancellationException) { throw e } قبل التقاط Exception.')
  process.exit(1)
}
console.log(
  checked === 0
    ? 'لا ملفَّ يُلغي مهمّةً ويلتقط الاستثناءات — لا شيءَ يُحرَس بعد.'
    : `الإلغاءُ لا يُعرض عطباً — ${checked} ملفّاً يُلغي مهامَّه وكلُّها تفرّق.`,
)
