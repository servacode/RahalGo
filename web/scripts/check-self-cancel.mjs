// ══════════════════════════════════════════════════════════════════════
// **أثرٌ يُلغي نفسَه بما أحدثه**
// ══════════════════════════════════════════════════════════════════════
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «لاحظتُ أنّ التبديل التلقائيّ لا يعمل على
//  الجوّال».)
//
// # ما وقع
//
// `LaunchedEffect(pager.currentPage, pager.isScrollInProgress, …)` ثمّ
// `pager.animateScrollToPage(…)` في جسده:
//
//	١ · يمضي الانتظارُ فيبدأ الانزلاق
//	٢ · فيصير `isScrollInProgress = true`
//	٣ · **وهو مفتاحُ الأثر — فيُلغى الأثرُ ومعه الانزلاقُ نفسُه**
//	٤ · فلا يتبدّل شيءٌ أبدا
//
// **ولا خطأَ يظهر**: يبدو ساكناً كأنّ الإعدادَ مطفأ — **فيُبحث في
// لوحة الإدارة عن عطبٍ في الشيفرة.**
//
// # وما يُقاس
//
// **`isScrollInProgress` مفتاحاً لأثرٍ يُحرّك السلايدر.** والقاعدةُ
// ضيّقةٌ وصريحة: **حارسٌ يُخمّن يُنذر على الصواب فيُطفَأ.**

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'

const HERE = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')
const MOBILE = join(HERE, '..', '..', 'mobile')
const MODULES = ['app-customer', 'app-driver', 'app-rep', 'ui']

const kt = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) kt(p, out)
    else if (name.endsWith('.kt')) out.push(p)
  }
  return out
}

// **ترويسةُ الأثر ومفاتيحُه** — حتّى القوس المغلق قبل `{`.
const EFFECT = /LaunchedEffect\s*\(([^)]*)\)\s*\{/g

const bad = []
let checked = 0
for (const mod of MODULES) {
  for (const file of kt(join(MOBILE, mod, 'src'))) {
    const src = readFileSync(file, 'utf8')
    if (!src.includes('isScrollInProgress')) continue
    for (const m of src.matchAll(EFFECT)) {
      checked++
      if (!m[1].includes('isScrollInProgress')) continue
      // **ويُقرأ جسدُه** — ألفُ حرفٍ تكفي لأثرِ سلايدر.
      const body = src.slice(m.index, m.index + 1000)
      if (/animateScrollTo|scrollTo|scrollBy/.test(body)) {
        bad.push(file.slice(MOBILE.length + 1))
      }
    }
  }
}

if (checked === 0) {
  console.log('لا أثرَ يقرأ حالَ السحب — لا شيءَ يُحرَس بعد.')
  process.exit(0)
}
if (bad.length) {
  console.error('أثرٌ مفتاحُه حالُ السحب وهو الذي يُحرّكه — يُلغي نفسَه فلا يعمل:')
  for (const b of [...new Set(bad)]) console.error('  · ' + b)
  console.error(
    '\nالبديل: المفاتيحُ ما لا يتحرّك (العددُ والمهلة)، والحلقةُ داخلَ الأثر،' +
      '\nوحالُ السحب يُقرأ في الجسد لا في المفتاح.',
  )
  process.exit(1)
}
console.log(`لا أثرَ يُلغي نفسَه — ${checked} أثراً فُحص حيث يُقرأ حالُ السحب.`)
