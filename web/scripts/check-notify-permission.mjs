// ══════════════════════════════════════════════════════════════════════
// **من سجّل للإشعارات وجب أن يطلب إذنَها**
// ══════════════════════════════════════════════════════════════════════
//
// (تدقيقُ الجاهزيّة ٢٠٢٦-٠٨-١٩: قِيس أنّ حزمةَ الزبون بلا
//  `POST_NOTIFICATIONS`، وتطبيقُ السائق يحمله — **فالزبونُ وحدَه
//  ناقص**، ولم يُكتشف إلّا بتشريح الحزمة.)
//
// # وبلاه تُبتلع الإشعاراتُ بصمت
//
// **من أندرويد ١٣ لا يُعرض إشعارٌ بلا إذن** — يُرسله الخادمُ ويقبله
// الجهازُ **ولا يظهر.** لا خطأَ ولا سجلّ ولا شيءَ يُبحث عنه.
//
// **فلا يعرف الزبونُ أنّ طلبَه قُبل ولا أنّ السائقَ وصل** — ويفتح
// التطبيقَ كلَّ دقيقةٍ يسأل. **وهي عائلةُ الخلل نفسُها: فشلٌ يُقرأ
// صمتا.**
//
// # وما يُقاس
//
// **كلُّ تطبيقٍ يسجّل جهازَه لدى المحرّك** (`DevicesApi` — أي ينتظر
// إشعارات) **يجب أن يحمل الإذنَ في بيانه.**
//
// **ولا يُقاس أنّه يطلبه في الشاشة** — ذاك سلوكٌ يختلف بين تطبيقٍ
// وآخر، **والإذنُ في البيان شرطٌ لا يختلف**: بلاه لا يُعرض شيءٌ
// مهما طُلب.

import { readFileSync, readdirSync, statSync, existsSync } from 'node:fs'
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

// **والعدّةُ المشتركةُ تسجّل نيابةً عنها** — فمن استعملها ينتظر إشعارا.
const uiRegisters = kt(join(MOBILE, 'ui', 'src'))
  .some((f) => readFileSync(f, 'utf8').includes('DevicesApi'))

const bad = []
let checked = 0
for (const app of APPS) {
  const manifest = join(MOBILE, app, 'src', 'main', 'AndroidManifest.xml')
  if (!existsSync(manifest)) continue
  const src = kt(join(MOBILE, app, 'src')).map((f) => readFileSync(f, 'utf8')).join('\n')
  const wantsPush = src.includes('DevicesApi') || src.includes('FirebaseMessaging') ||
    (uiRegisters && src.includes('AppCore'))
  if (!wantsPush) continue
  checked++
  if (!readFileSync(manifest, 'utf8').includes('POST_NOTIFICATIONS')) {
    bad.push(`${app} — يسجّل للإشعارات ولا يحمل الإذن: تُبتلع كلُّها بصمت`)
  }
}

if (checked === 0) {
  console.error('حارسُ الإشعارات لم يفحص شيئاً — تغيّر اسمُ العميل، فأصلحْ الحارس')
  process.exit(1)
}
if (bad.length) {
  console.error('تطبيقٌ ينتظر إشعاراتٍ ولا يطلب إذنَها:')
  for (const b of bad) console.error('  · ' + b)
  console.error('\nالبديل: POST_NOTIFICATIONS في البيان، و AskNotifyPermission بعد الدخول.')
  process.exit(1)
}
console.log(`إذنُ الإشعارات مطلوبٌ حيث يلزم — ${checked} تطبيقاً ينتظر إشعاراتٍ وكلُّها تحمله.`)
