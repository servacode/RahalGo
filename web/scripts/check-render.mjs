/**
 * **حارسُ مخطّطة الرفع** — يمنع خاصّيّةً لا تعرفها المنصّة.
 *
 * # لماذا حارسٌ لملفٍّ يُقرأ مرّةً
 *
 * **`render.yaml` لا يُختبر إلّا عند النشر** — ولا بناءَ يكشف خطأه ولا نوعَ
 * يُدقّقه. **وخطؤه يوقف الرفعَ كلَّه** بعد أن تكون قد دفعتَ وانتظرت.
 *
 * **وقع فعلاً (٢٠٢٦-٠٨-١٠)**: كُتب `property: url` — **وليست في مواصفة
 * Render أصلاً**؛ المسموحُ `host` و`port` و`hostport` و`connectionString`
 * وأخواتُها. **فكانت المخطّطةُ تُرفض عند القراءة.**
 *
 * # وما يُفحص
 *
 * **الخاصّيّاتُ المسموحة** — وهي مغلقةٌ لا مفتوحة.
 * **و`sync: false` على كلّ سرٍّ** — فلا يُكتب سرٌّ في ملفٍّ يُرفع.
 * **و`plan` لكلّ خدمة** — وبلاه يُنشأ المدفوعُ بلا سؤال.
 */

import { readFileSync, existsSync } from "node:fs";

const FILE = new URL("../../render.yaml", import.meta.url).pathname.replace(
  /^\/([A-Za-z]:)/,
  "$1",
);

if (!existsSync(FILE)) {
  console.log("لا مخطّطةَ رفعٍ — يُتخطّى.");
  process.exit(0);
}

const src = readFileSync(FILE, "utf8");
const problems = [];

/** **الخاصّيّاتُ التي تعرفها المنصّة** — وما عداها يُرفض عند القراءة. */
const ALLOWED = new Set([
  "host",
  "port",
  "hostport",
  "connectionString",
  "connectionPoolString",
  "user",
  "password",
  "database",
]);

src.split("\n").forEach((line, i) => {
  // **والتعليقاتُ تُقفز** — الشرحُ يذكر `url` ليقول إنّها ممنوعة.
  if (/^\s*#/.test(line)) return;
  const m = line.match(/^\s*property:\s*([A-Za-z]+)\s*$/);
  if (m && !ALLOWED.has(m[1])) {
    problems.push({
      line: i + 1,
      what: `خاصّيّةٌ لا تعرفها المنصّة: ${m[1]}`,
      fix: `المسموح: ${[...ALLOWED].join(" · ")}`,
    });
  }
});

// **وكلُّ سرٍّ يُسأل عنه ولا يُكتب.**
for (const key of ["WEB_ORIGINS", "JWT_SECRET"]) {
  const at = src.indexOf(`key: ${key}`);
  if (at < 0) continue;
  const after = src.slice(at, at + 220);
  if (!/sync:\s*false|generateValue:\s*true|fromService|fromDatabase/.test(after)) {
    problems.push({
      line: src.slice(0, at).split("\n").length,
      what: `السرُّ ${key} مكتوبٌ بقيمةٍ صريحة`,
      fix: "استعمل sync: false أو generateValue: true",
    });
  }
}

// **وخدمةٌ بلا خطّةٍ تُنشأ مدفوعةً بلا سؤال.**
const services = src.split(/^\s{2}- type:/m).slice(1);
services.forEach((s, i) => {
  if (!/^\s*plan:/m.test(s)) {
    problems.push({ line: 0, what: `الخدمةُ رقم ${i + 1} بلا plan`, fix: "أضِف plan: free" });
  }
});

if (problems.length === 0) {
  console.log("مخطّطةُ الرفع سليمة — لا خاصّيّةَ مجهولةٌ ولا سرٌّ مكتوب.");
  process.exit(0);
}
console.log(`مخطّطةُ الرفع: ${problems.length} مخالفة\n`);
for (const p of problems) {
  console.log(`   render.yaml${p.line ? ":" + p.line : ""}  ${p.what}`);
  console.log(`      الإصلاح: ${p.fix}`);
}
process.exit(1);
