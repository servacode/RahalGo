// ══════════════════════════════════════════════════════════════════════
// **ما يلفّ التطبيقاتِ يُركَّب في الإطار — لا في تطبيقٍ واحد**
// ══════════════════════════════════════════════════════════════════════
//
// (ملاحظةُ المالك ٢٠٢٦-٠٨-١٩: «تطبيقُ الزبون لا يحوي واجهةَ الدخول
//  التي اتّفقنا عليها وقمنا ببنائها… واتّفقنا يجب أن تكون مركزيّة،
//  نستخدم نفس واجهة البداية بكلّ التطبيقات».)
//
// # ما وقع
//
// **`BrandIntro` بُنيت ٢٠٢٦-٠٨-١١ ورُكّبت في `DriverApp` بيدها** —
// **فوُلدت في تطبيقٍ وبقيت فيه** ثمانيةَ أيّام. والزبونُ والمندوبُ
// يُقلعان على بياض، **ولا شيءَ يقول إنّ شيئاً ينقص.**
//
// **وهي عائلةٌ تكرّرت**: الرسالةُ الطافية وشريطُ الشبكة وجالبُ الصور
// وإذنُ الإشعارات — **كلُّها بُنيت مرّةً ونُسيت في تطبيقٍ أو اثنين**،
// ولم تُكتشف إلّا حين اشتكى المالكُ أو حين شُرّحت الحزمة.
//
// # وما يُقاس
//
// **قطعُ الغلاف المشتركة تُنادى في `AppFrame` وحدَه** — لا في
// `MainActivity` تطبيقٍ بعينه. **ومن ناداها هناك بنى نسخةً تفترق.**
//
// **والقائمةُ صريحةٌ لا مُخمَّنة** — تُزاد كلَّما بُنيت قطعةُ غلافٍ
// جديدة.

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const MOBILE = join(HERE, '..', '..', 'mobile')
const APPS = ['app-customer', 'app-driver', 'app-rep']

/** **ما يلفّ التطبيقَ كلَّه** — موضعُه `AppFrame` لا غير. */
const SHELL = ['BrandIntro(', 'FlashHost(', 'NetBanner(']

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt')) out.push(p)
  }
  return out
}

const frame = readFileSync(
  join(MOBILE, 'ui', 'src', 'main', 'kotlin', 'com', 'rahalgo', 'ui', 'AppFrame.kt'),
  'utf8',
)

const bad = []

// **أوّلاً: الإطارُ يحملها كلَّها** — إمّا مباشرةً أو بحاملٍ لها.
for (const piece of SHELL) {
  const name = piece.replace('(', '')
  if (!frame.includes(piece) && !frame.includes(name + 'Host(')) {
    bad.push(`AppFrame لا يركّب ${name} — فلا يراها إلّا من بناها`)
  }
}

// **وثانياً: لا تطبيقَ يركّبها بيده** — نسخةٌ ثانيةٌ تفترق.
for (const app of APPS) {
  for (const file of kt(join(MOBILE, app, 'src'))) {
    const src = readFileSync(file, 'utf8')
    for (const piece of SHELL) {
      if (src.includes(piece)) {
        bad.push(`${app}/${file.split(/[\\/]/).pop()} يركّب ${piece.replace('(', '')} بيده`)
      }
    }
  }
}

if (bad.length) {
  console.error('قطعُ الغلاف المشتركة في غير موضعها:')
  for (const b of [...new Set(bad)]) console.error('  · ' + b)
  console.error('\nالبديل: تُنادى في AppFrame مرّةً — فتراها التطبيقاتُ الثلاثة.')
  process.exit(1)
}
console.log(`غلافُ التطبيقات موحَّد — ${SHELL.length} قطعاً تُركَّب في الإطار وحدَه.`)
