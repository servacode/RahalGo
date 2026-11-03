#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **تعديلُ الطلب لحتّى التسليم** (قرارُ المالك ٢٠٢٦-١١-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما تحرسه
 *
 *	١ · **زرُّ «تعديل عرض السعر» للخاصّ يبقى بعد الاستلام** — الخادمُ يقبله
 *	    حتّى التسليم (`AdminOverrideCustomQuote`)، **فزرٌّ يُخفى بعد الاستلام
 *	    يترك المكتبَ بلا باب.** وشرطُ `!o.picked_up_at` في ظهوره يُسقط البناء.
 *	٢ · **وجملةُ ما بعد الاستلام تطابق الخادم** — لا «لا زيادة».
 *	٣ · **«تعديل الطلب» للعاديّ قائمٌ في الكرت** ببابه في المحرّك، **ومعاينتُه من الخادم**
 *	    (`dry_run`) لا حسبةٌ في الشاشة — رقمٌ يُحسب في موضعين يفترق.
 */
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const problems = [];

const read = (p) => {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
};
const strip = (s) => s.replace(/\{\/\*[\s\S]*?\*\/\}/g, "").replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");

const screenPath = join(web, "apps", "rahalgo", "src", "components", "admin", "orders", "OrdersScreen.tsx");
const arPath = join(web, "packages", "i18n", "src", "locales", "ar.json");
const screen = strip(read(screenPath));
const editorPath = join(web, "apps", "rahalgo", "src", "components", "admin", "orders", "OrderEditor.tsx");
const editor = strip(read(editorPath));
const ar = read(arPath);
if (screen === "") problems.push("شاشةُ الطلبات لم تُقرأ");
if (ar === "") problems.push("القاموسُ لم يُقرأ");

// ── ١ · زرُّ الخاصّ بعد الاستلام ───────────────────────────────────────
{
  const btn = screen.indexOf("m.admin.ordersPage.overrideBtn}</BtnLabel>");
  if (btn < 0) {
    problems.push("زرُّ «تعديل عرض السعر» غير موجود في شاشة الطلبات");
  } else {
    // شرطُ الظهور: من آخر `o.kind === "custom"` قبل الزرّ إلى الزرّ.
    const start = screen.lastIndexOf('o.kind === "custom"', btn);
    const cond = screen.slice(start, btn);
    if (/!\s*o\.picked_up_at/.test(cond)) {
      problems.push("زرُّ «تعديل عرض السعر» مخفيٌّ بعد الاستلام — والخادمُ يقبله حتّى التسليم");
    }
  }
}

// ── ٢ · جملةُ ما بعد الاستلام ───────────────────────────────────────────
{
  const m = ar.match(/"overridePostPickupHint":\s*"([^"]*)"/);
  if (!m) problems.push("مفتاحُ overridePostPickupHint غائب");
  else if (m[1].includes("لا زيادة")) {
    problems.push("جملةُ ما بعد الاستلام تقول «لا زيادة» — والخادمُ يقبل الزيادةَ بتأكيد الزبون");
  }
}

// ── ٣ · «تعديل الطلب» للعاديّ ────────────────────────────────────────────
{
  if (editor === "") problems.push("محرّرُ الطلب (OrderEditor.tsx) غائب");
  if (!screen.includes("m.admin.ordersPage.orderEdit.button") || !screen.includes("<OrderEditor")) {
    problems.push("زرُّ «تعديل الطلب» غائبٌ عن كرت الطلب");
  }
  if (!editor.includes("/edit`") || !editor.includes("dry_run")) {
    problems.push("محرّرُ الطلب لا ينادي بابَ المحرّك بمعاينةٍ (`dry_run`)");
  }
  if (!/"order_edit_reason_required"/.test(ar)) problems.push("رسالةُ «اكتب سبب التعديل» غائبةٌ عن القاموس");
}

if (problems.length) {
  console.error("✗ تعديلُ الطلب:");
  for (const p of problems) console.error("  - " + p);
  process.exit(1);
}
console.log("✓ تعديلُ الطلب: زرُّ الخاصّ قائمٌ حتّى التسليم، ومحرّرُ العاديّ بمعاينةٍ من الخادم");
