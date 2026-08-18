// ══════════════════════════════════════════════════════════════════════
// **R8 يحذف ما لا تناديه شيفرة — وجالبُ الصور منه**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨ بصورةِ شاشة: لافتاتُ السلايدر إطاراتٌ فارغة.)
//
// # ما وقع
//
// **`coil3` تكتشف جالبَ الشبكة بـ`ServiceLoader`** — سطرٌ في
// `META-INF/services` يشير إلى صنفٍ **لا تناديه شيفرةٌ أبدا**. **وR8
// يحذفه**، فيُبنى الإصدارُ بلا جالبِ شبكة: كلُّ صورةٍ بعيدةٍ تفشل،
// **ولا انهيارَ ولا رسالة — بل إطارٌ فارغ.**
//
// **وفي التطوير تعمل** (`isMinifyEnabled = false`) — **وهو أخبثُ ما
// يكون**: يُبنى ويُجرَّب فيُرى سليماً، **ولا يظهر العطبُ إلّا في
// النسخة التي تصل الناس.**
//
// # وما يُقاس
//
// **كلُّ تطبيقٍ يُصغَّر (`isMinifyEnabled = true`) ويستعمل `coil` يجب
// أن يُسجّل `SingletonImageLoader` بيده.**
//
// **ولا يُقاس «هل تظهر الصورة»** — ذاك لا يُقاس إلّا بجهاز، **وقيادةُ
// جهاز المالك ممنوعة.** فيُقاس السببُ لا العَرَض.

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const MOBILE = join(HERE, '..', '..', 'mobile')
const APPS = ['app-customer', 'app-driver', 'app-rep']

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt')) out.push(p)
  }
  return out
}

// **والتسجيلُ قد يقع في العدّة المشتركة** — يكفي أن يُنادى من التطبيق.
const uiHasFactory = kt(join(MOBILE, 'ui', 'src'))
  .some((f) => readFileSync(f, 'utf8').includes('SingletonImageLoader'))

const bad = []
let checked = 0
for (const app of APPS) {
  const gradle = readFileSync(join(MOBILE, app, 'build.gradle.kts'), 'utf8')
  if (!/isMinifyEnabled\s*=\s*true/.test(gradle)) continue
  checked++
  const src = kt(join(MOBILE, app, 'src')).map((f) => readFileSync(f, 'utf8')).join('\n')
  const registers = src.includes('SingletonImageLoader') ||
    (uiHasFactory && /Images\.install/.test(src))
  if (!registers) {
    bad.push(`${app} — يُصغَّر ولا يُسجّل جالبَ الصور: كلُّ صورةٍ بعيدةٍ إطارٌ فارغ`)
  }
}

if (checked === 0) {
  console.error('حارسُ الصور لم يفحص شيئاً — لا تطبيقَ يُصغَّر، فأصلحْ الحارس')
  process.exit(1)
}
if (bad.length) {
  console.error('جالبُ الصور متروكٌ لـ`ServiceLoader` — وR8 يحذفه:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: Images.install(this) في onCreate — تسجيلٌ صريحٌ يُكسَر عند البناء.')
  process.exit(1)
}
console.log(`جالبُ الصور مُسجَّلٌ صراحةً — ${checked} تطبيقاً يُصغَّر وكلُّها تسجّله.`)
