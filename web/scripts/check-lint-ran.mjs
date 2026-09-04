// ══════════════════════════════════════════════════════════════════════
// **حارسٌ يمنع أن يُقرأ الفراغُ نجاحاً**
// ══════════════════════════════════════════════════════════════════════
//
// (حادثةُ ٢٠٢٦-٠٩-٠٥: كتبتُ `LINT = PASS` و`turbo lint` قد طبع أمامي
//  **`No tasks were executed as part of this run`** — لأنّ لا حزمةَ
//  تعرّف `lint`. **فمرّ عطبُ خطّافاتٍ إلى الإنتاج.**)
//
// **وأمرٌ ينجح بلا أن يفعل شيئاً أسوأُ من أمرٍ يسقط**: السقوطُ يُقرأ،
// **والنجاحُ الفارغُ يُصدَّق.**
//
// **فيُشغَّل `turbo lint` هنا ويُقرأ سطرُ الحصيلة**: إن كان المجموعُ
// صفراً سقط البناء، **ولو لم يشتكِ ESLint من سطرٍ واحد.**

import { spawnSync } from "node:child_process";

const run = spawnSync(
  process.platform === "win32" ? "pnpm.cmd" : "pnpm",
  ["exec", "turbo", "lint", "--output-logs=new-only"],
  { encoding: "utf8", shell: process.platform === "win32" },
);

const out = `${run.stdout ?? ""}${run.stderr ?? ""}`;
process.stdout.write(out);

// **«Tasks: 2 successful, 2 total»** — والأرقامُ قد تأتي بأرقامٍ عربيّةٍ
// شرقيّةٍ حين تُترجَم الطرفيّة، **فتُطبَّع قبل أن تُقرأ.**
const ascii = out.replace(/[٠-٩]/g, (d) =>
  String(d.charCodeAt(0) - 0x0660),
);
const m = ascii.match(/Tasks:\s*(\d+)\s*successful,\s*(\d+)\s*total/);

if (!m) {
  console.error("\n✗ الفحصُ: لم أجد سطرَ حصيلةِ turbo — ولا أُصدّق نجاحاً لا أراه.");
  process.exit(1);
}

const [, ok, total] = m.map(Number);

if (total === 0) {
  console.error(
    "\n✗ الفحصُ: `turbo lint` نفّذ صفرَ مهامّ — **وهذا سقوطٌ لا نجاح.**\n" +
      "   أضِف سكربتَ `lint` إلى الحزمة، أو احذف المهمّةَ من `turbo.json`.",
  );
  process.exit(1);
}

if (run.status !== 0 || ok !== total) {
  console.error(`\n✗ الفحصُ: نجح ${ok} من ${total} مهمّةَ فحص.`);
  process.exit(run.status || 1);
}

console.log(`\n✓ الفحصُ نُفِّذ فعلاً — ${ok} من ${total} مهمّةً، ولا مهمّةَ فارغة.`);
