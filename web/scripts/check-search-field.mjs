// ══════════════════════════════════════════════════════════════════════
// **حقلٌ يمحو ما يُكتب فيه**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «حقلُ البحث يفتح الكيبورد، أكتب ولكن لا يظهر
//  أيُّ حرفٍ بالكتابة، تحقّق منه».)
//
// # ما وقع
//
// `type(text)` كانت تضع `query = text` **ثمّ تنادي `openSection` وهي
// تمحو `query`** — فيُكتب الحرفُ الأوّلُ ثمّ يُمحى في النداء نفسِه.
//
// **وحرفٌ واحدٌ أقلُّ من الحدّ الأدنى دائماً** (حرفان)، **فلا يُبلَغ
// الحرفُ الثاني أبدا** — والحقلُ لا يمسك شيئاً مهما كُتب فيه.
//
// **ولا خطأَ ولا رسالة**: لوحةُ المفاتيح تُفتح والحرفُ يُبتلع، **فيُقرأ
// عطباً في الجهاز لا في التطبيق** — وهو أسوأُ ما يقع في حقلِ إدخال.
//
// # وما يُقاس
//
// **دالّةُ الكتابة لا تنادي ما يمحو الحقل.** والقاعدةُ صريحةٌ لا
// مُخمَّنة: `type(` لا تحوي نداءً لـ`openSection(`.
//
// **ولا يُقاس «هل يظهر الحرف»** — ذاك لا يُقاس إلّا على جهاز، **وقيادةُ
// جهاز المالك ممنوعة** (قرارُه ٢٠٢٦-٠٨-١٨). **فيُقاس السببُ لا العَرَض.**

import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const VM = join(HERE, '..', '..', 'mobile', 'app-customer', 'src', 'main',
  'kotlin', 'com', 'rahalgo', 'customer', 'shop', 'ShopViewModel.kt')

const src = readFileSync(VM, 'utf8')

// **جسدُ `type`** — من ترويستها إلى الدالّة التالية.
const at = src.indexOf('fun type(')
if (at < 0) {
  console.error('لم تُوجد `fun type(` في ShopViewModel — تغيّر الاسمُ فأصلحْ الحارس')
  process.exit(1)
}
const rest = src.slice(at)
const end = rest.indexOf('\n    private companion object')
const body = end > 0 ? rest.slice(0, end) : rest

const bad = []
if (body.includes('openSection(')) {
  bad.push('`type` تنادي `openSection` — وهي تمحو `query` فيُبتلع كلُّ حرف')
}
// **و`openSection` وحدَها تملك المحو** — ومن محا في `type` أعاد العطب.
if ((body.match(/query\s*=\s*""/g) || []).length > 0) {
  bad.push('`type` تُفرّغ `query` بيدها')
}

if (bad.length) {
  console.error('حقلُ البحث يمحو ما يُكتب فيه:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: `loadSection(id)` — تجلب الأصنافَ ولا تمسّ الحقل.')
  process.exit(1)
}
console.log('حقلُ البحث يمسك ما يُكتب فيه — والكتابةُ لا تُنادي ما يمحوه.')
