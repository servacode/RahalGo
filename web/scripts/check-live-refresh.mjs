// ══════════════════════════════════════════════════════════════════════
// **كلُّ شاشةٍ تجلب ما تكتبه اللوحةُ تسمع النبضة**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «في مشكلةُ التحديث اللحظيّ بالتطبيق… أيُّ
//  تعديلٍ من لوحة الأدمن فوراً يُطبَّق حتّى ولو الزبونُ فاتحٌ التطبيق،
//  ما يلزم يحدّث أو يعيد تشغيل التطبيق».)
//
// # ما وقع
//
// **النبضةُ موجودةٌ منذ ٢٠٢٦-٠٨-١٣** ويسمعها الطلباتُ والحسابُ
// والمحفظة — **وشاشةُ السوق لا تسمعها.** وهي بعينها ما تكتبه اللوحةُ:
// الأقسامُ والأصنافُ والأسعارُ والبانرات.
//
// **وشاشةٌ تُظهر سعراً قديماً لا تقول إنّه قديم** — يطلب بسعرٍ رآه
// ويُحاسَب بغيره.
//
// # وما يُقاس
//
// **من نادى نقطةً تجلب ما تكتبه اللوحةُ وجب أن يسمع `Refresh.tick`.**
// والقائمةُ صريحةٌ لا مُخمَّنة: **حارسٌ يُخمّن يُنذر على الصواب فيُطفَأ.**
//
// ولا يُقاس هنا طرفُ المحرّك — يحرسه `TestAnnounceWrites_IsWiredToBothGates`.

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const ROOT = new URL('../../mobile', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')

/** **نقاطٌ تجلب ما تكتبه لوحةُ الإدارة** — من ناداها سمع النبضة. */
const PANEL_FED = ['api.home()', 'api.sectionItems(', 'api.offers(']

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt')) out.push(p)
  }
  return out
}

const bad = []
let checked = 0
for (const file of kt(join(ROOT, 'app-customer'))) {
  const src = readFileSync(file, 'utf8')
  const calls = PANEL_FED.filter((c) => src.includes(c))
  if (calls.length === 0) continue
  checked++
  if (!src.includes('Refresh.tick')) {
    bad.push(`${file.slice(ROOT.length + 1)} — يجلب ${calls.join('، ')} ولا يسمع النبضة`)
  }
}

if (checked === 0) {
  console.error('حارسُ التحديث اللحظيّ لم يفحص شيئاً — تغيّرت أسماءُ النقاط، فأصلحْ قائمتَه')
  process.exit(1)
}
if (bad.length) {
  console.error('شاشاتٌ تعرض ما تكتبه اللوحةُ ولا تُنعشه:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: viewModelScope.launch { Refresh.tick.drop(1).collect { load() } }')
  process.exit(1)
}
console.log(`التحديثُ اللحظيُّ موصول — ${checked} شاشةً تجلب من اللوحة وكلُّها تسمع النبضة.`)
