#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّكُ المسارات مثبَّتٌ — ولا يعود إلى `latest`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٢٩.)
 *
 * **أمرُ المالك نصّاً**: «لا يمكن بناء Thresholds واختبارات على نسخة ثم
 * السماح للبناء بأن يتغير تلقائيًا… أضف Guard/Test يمنع الرجوع إلى
 * `:latest`».
 *
 * # وما يُفقد لو عادت
 *
 * **كلُّ عتبات المرحلة ٧ قِيست على `v26.8.0`**:
 *
 *	`weight_name = routability` لا `duration`
 *	`alternatives=5` تردّ `TooBig` — فالسقفُ ثلاثة
 *	٨٧ بديلاً في ١٣٤ طلباً · اثنان فقط تشابهُهما ≥٨٠٪
 *	`overview=false` يعطي `NavRoute` مطابقة
 *
 * **ومحرّكٌ آخرُ قد يرتّب البدائلَ بطريقةٍ أخرى** — فتصير اختباراتُنا
 * تصف ماضياً، **وتُشحَن عتباتٌ لا تصف ما يعمل.**
 *
 * # وما يفحصه
 *
 *	١ · `routing/Dockerfile` لا يستعمل `:latest`
 *	٢ · فيه بصمةٌ صريحة (`@sha256:`)
 *	٣ · والنسخةُ هي التي قِيست
 */

import { readFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const ROOT = join(here, '..', '..');

/** **النسخةُ التي جرت عليها كلُّ قياسات المرحلة ٧.** */
export const MEASURED_VERSION = 'v26.8.0';

const problems = [];
const notes = [];

const path = join(ROOT, 'routing', 'Dockerfile');
if (!existsSync(path)) {
  problems.push('routing/Dockerfile غائب');
} else {
  const src = readFileSync(path, 'utf8');
  const from = /^FROM\s+(\S+)/m.exec(src);

  if (!from) {
    problems.push('routing/Dockerfile بلا سطر FROM');
  } else {
    const image = from[1];

    if (/:latest(@|$|\s)/.test(image) || !image.includes(':')) {
      problems.push(
        `محرّكُ المسارات على وسمٍ متحرّك: ${image} — ` +
        'وعتباتُ المرحلة ٧ قِيست على نسخةٍ بعينها',
      );
    }

    if (!image.includes('@sha256:')) {
      problems.push(
        'محرّكُ المسارات بلا بصمة — والوسمُ يُعاد تعيينُه في المنبع والبصمةُ لا',
      );
    }

    if (!image.includes(MEASURED_VERSION)) {
      problems.push(
        `محرّكُ المسارات ليس ${MEASURED_VERSION} — ` +
        'فإن رُفعت النسخةُ عمداً فأعِد القياسَ وحدّث العتبات',
      );
    }

    if (problems.length === 0) {
      const digest = /@sha256:([0-9a-f]{12})/.exec(image);
      notes.push(
        `محرّكُ المسارات: ${MEASURED_VERSION} · بصمة ${digest ? digest[1] : '?'}…`,
      );
    }
  }
}

// ── والعتباتُ تشير إلى مصدرها ───────────────────────────────────────
//
// **فمن غيّر رقماً وجد نصّاً يقول من أين جاء** — ولا يبقى الرقمُ يتيماً.
const routeset = join(ROOT, 'backend', 'internal', 'routing', 'routeset.go');
if (!existsSync(routeset)) {
  problems.push('backend/internal/routing/routeset.go غائب');
} else {
  const src = readFileSync(routeset, 'utf8');
  for (const needle of ['NearDuplicateShared', 'GrossPenaltyRatio', 'MaxAlternatives']) {
    if (!src.includes(needle)) problems.push(`العتبةُ ${needle} غائبة`);
  }
  const golden = join(ROOT, 'backend', 'internal', 'routing', 'testdata',
    'syria-alternatives.golden.json');
  if (!existsSync(golden)) {
    problems.push('الرفيدةُ الذهبيّةُ غائبة — والعتباتُ تصير بلا مصدر');
  } else {
    const data = JSON.parse(readFileSync(golden, 'utf8'));
    notes.push(`الرفيدةُ الذهبيّة: ${data.alts.length} بديلاً مقيساً`);
  }
}

for (const n of notes) console.log(`  · ${n}`);
if (problems.length) {
  console.error('محرّكُ المسارات — خللٌ:');
  for (const p of problems) console.error(`  ✗ ${p}`);
  process.exit(1);
}
console.log('محرّكُ المسارات مثبَّتٌ ببصمته — والعتباتُ لها مصدرٌ مقيس.');
