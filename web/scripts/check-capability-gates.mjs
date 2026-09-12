#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **كلُّ بابٍ وكلُّ زرٍّ بقدرته — ولا اسمَ دورٍ في شرطِ ظهور**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما يحرسه
 *
 *	١ · **كلُّ بندٍ في القائمة له `caps`** — ولا بندَ بلا شرط
 *	٢ · **ولا `roles:` في جدول القائمة** أصلاً
 *	٣ · **والترشيحُ بالقدرة وحدَها** — لا فرعَ لأسماء الأدوار
 *	٤ · **وكلُّ قدرةٍ في القائمة مسجَّلةٌ في معجم المحرّك**
 *	٥ · **ولا `hasRole`/`roles.includes` يقرّر ظهورَ فعلٍ إداريّ**
 *	٦ · **والأفعالُ الماليّةُ بقدرةِ المال**، لا باسم `finance`
 *
 * # ولماذا حارسٌ لا مراجعة
 *
 * **وقِيس ٢٠٢٦-٠٩-١٣**: اثنان وعشرون بنداً كانت بالاسم، **تسعةٌ منها
 * بلا شرطٍ يراها كلُّ من دخل** — **وبندا «النقد» و«الخسائر» بدورٍ
 * صفرِ حاملين.** **وزرُّ إغلاق التذكرة يُعرَض للماليّة ويُردّ ٤٠٣،
 * ويُخفى عن الدعم وهو صاحبُه.**
 *
 * **وسطرٌ واحدٌ يُعيدها كلَّها** يومَ يستعجل أحدُهم.
 */
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const root = join(web, "..");
const src = join(web, "apps", "rahalgo", "src");
const problems = [];
const notes = [];

const read = (p) => {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
};
const strip = (s) =>
  s.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");

// ── المعجمُ الكانونيُّ من المحرّك ───────────────────────────────────
const catalog = read(join(root, "backend", "internal", "authz", "catalog.go"));
const KNOWN = new Set(
  [...catalog.matchAll(/Capability = "([\w.]+)"/g)].map((m) => m[1]),
);
if (KNOWN.size === 0) {
  problems.push("معجمُ المحرّك لم يُقرأ — والحارسُ بلا مرجع");
}

// ── ١·٢·٣·٤ · جدولُ القائمة ────────────────────────────────────────
const layoutPath = join(src, "app", "dashboard", "layout.tsx");
const layout = strip(read(layoutPath));
{
  const i = layout.indexOf("ALL_NAV");
  const j = layout.indexOf("\n];", i);
  if (i < 0 || j < 0) {
    problems.push("جدولُ القائمة لم يُقرأ");
  } else {
    const table = layout.slice(i, j);
    const items = [...table.matchAll(/href:\s*"([^"]+)"/g)].map((m) => m[1]);
    const blocks = table.split(/(?=\{\s*href:)/).filter((b) => b.includes("href:"));

    // ٢ · ولا اسمَ دورٍ في الجدول
    if (/roles:\s*\[/.test(table)) {
      const bad = [...table.matchAll(/href:\s*"([^"]+)"[^{}]*roles:\s*\[([^\]]*)\]/g)];
      problems.push(
        "**بندٌ في القائمة مبوَّبٌ باسم الدور**: " +
          (bad.map((b) => b[1]).join(" · ") || "(غيرُ مسمّى)") +
          " — **ودورٌ مخصَّصٌ بالقدرة نفسِها لا يراه.**",
      );
    }

    // ١ · وكلُّ بندٍ له شرطُ قدرة
    const naked = [];
    const used = new Set();
    for (const b of blocks) {
      const href = /href:\s*"([^"]+)"/.exec(b)?.[1] ?? "?";
      const caps = /caps:\s*\[([^\]]*)\]/.exec(b);
      if (!caps) {
        naked.push(href);
        continue;
      }
      for (const c of caps[1].match(/"([\w.]+)"/g) ?? []) used.add(c.slice(1, -1));
    }
    if (naked.length > 0) {
      problems.push(
        "**بندٌ في القائمة بلا شرطِ قدرة** — **يراه كلُّ من دخل اللوحة**:\n   " +
          naked.join(" · "),
      );
    }

    // ٤ · وكلُّ قدرةٍ مسجَّلةٌ في المحرّك
    const ghosts = [...used].filter((c) => !KNOWN.has(c)).sort();
    if (ghosts.length > 0) {
      problems.push(
        "**قدرةٌ في القائمة ليست في معجم المحرّك**: " + ghosts.join(" · "),
      );
    }
    if (naked.length === 0 && ghosts.length === 0) {
      notes.push(`${items.length} بنداً في القائمة — كلُّها بالقدرة (${used.size} قدرةً)`);
    }
  }

  // ٣ · والترشيحُ بالقدرة وحدَها
  if (!/return ALL_NAV\.filter\(\(i\) => !!i\.caps && i\.caps\.some\(can\)\);/.test(layout)) {
    problems.push(
      "**ترشيحُ القائمة لم يعد بالقدرة وحدَها** — " +
        "**وفرعٌ لاسم دورٍ يعيد العطبَ كلَّه**",
    );
  }
}

// ── ٥·٦ · أفعالُ الشاشات ───────────────────────────────────────────
//
// **ولا يُفحَص كلُّ `hasRole`**: **`canAccessPortal` و`isDriver`
// يقرّران أيَّ تطبيقٍ يفتح، لا أيَّ زرٍّ يظهر في الإدارة.** **والمقيسُ
// هنا شاشاتُ اللوحة وحدَها.**
const walk = (d, out = []) => {
  for (const e of readdirSync(d)) {
    const p = join(d, e);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (p.endsWith(".tsx") || p.endsWith(".ts")) out.push(p);
  }
  return out;
};
{
  const files = [
    ...walk(join(src, "app", "dashboard")),
    ...walk(join(src, "components", "admin")),
  ];
  // **و`isAdmin` في شاشة الطلبات تجاوزُ مالكٍ مكتوبٌ في المنتَج** —
  // **يقرّر أيَّ انتقالٍ يُعرَض، لا أيَّ بابٍ يُفتح** — **والمحرّكُ
  // يفرضه أيضاً** (`modes.go`). **فهو مستثنىً بسببه لا بصمت.**
  const EXEMPT = new Map([
    ["components/admin/orders/OrdersScreen.tsx", "تجاوزُ المالك في خارطة الانتقالات — يفرضه المحرّكُ كذلك"],
  ]);
  const hits = [];
  for (const f of files) {
    const rel = relative(src, f).replace(/\\/g, "/");
    const t = strip(read(f));
    // **ودورُ الناظر غيرُ دورِ صفٍّ معروض**: `u.roles.includes(r)`
    // في جدولِ حساباتٍ بيانٌ يُعرَض، **و`me?.roles.includes("admin")`
    // قرارُ ظهور.** **فيُقاس الناظرُ وحدَه.**
    //
    // **ولا حدَّ كلمةٍ في هذا الفحص**: **محرفُ الهروب يمرّ محرفَ
    // تراجعٍ عبر طبقات التحرير، فتصير القاعدةُ تمرّ أبداً وهي لا
    // تقيس شيئاً.** (وقع مرّتين — ٢٠٢٦-٠٩-١٢ و٢٠٢٦-٠٩-١٣.)
    // **فيُقرأ الاسمُ ويُقارَن، ولا يُترَك للتعبير النمطيّ.**
    const NAMES = ["me", "user", "viewer"];
    let found = 0;
    for (const g of t.matchAll(/hasRole\(\s*([A-Za-z0-9_$]+)/g)) {
      if (NAMES.includes(g[1])) found++;
    }
    for (const g of t.matchAll(/([A-Za-z0-9_$]+)\??\.roles\??\.(?:some|includes)\(/g)) {
      if (NAMES.includes(g[1])) found++;
    }
    if (found === 0) continue;
    if (EXEMPT.has(rel)) {
      notes.push(`مستثنىً: ${rel} — ${EXEMPT.get(rel)}`);
      continue;
    }
    hits.push(`${rel} (${found})`);
  }
  if (hits.length > 0) {
    problems.push(
      "**قرارُ ظهورٍ باسم الدور في شاشة إدارة**:\n   " +
        hits.join("\n   ") +
        "\n   **والقدرةُ هي المرجع** — `can(\"…\")` من المزوِّد المركزيّ.",
    );
  } else {
    notes.push(`${files.length} ملفَّ إدارةٍ فُحص — ولا قرارَ ظهورٍ باسم دور`);
  }
}

// ── ٦ · والأفعالُ الماليّةُ بقدرةِ المال ────────────────────────────
{
  const MUST = [
    ["app/dashboard/cash/page.tsx", 'can("finance.manage")', "تسويةُ نقد السائق"],
    ["app/dashboard/losses/page.tsx", 'can("finance.read")', "تبويبُ الخسائر"],
    ["app/dashboard/payouts/page.tsx", 'can("payouts.decide")', "قرارُ السحب"],
    ["components/admin/money/disputes.tsx", 'can("finance.manage")', "تسويةُ النزاع"],
    ["components/admin/support/tickets.tsx", 'can("support.manage")', "إغلاقُ التذكرة"],
    ["components/admin/orders/OrdersScreen.tsx", 'can("finance.manage")', "تعويضُ السائق"],
    // **والأفعالُ التشغيليّةُ في الشاشة نفسِها** — **والماليّةُ تقرأ
    // ولا تُسند** (بندُ المالك ٨).
    ["components/admin/orders/OrdersScreen.tsx", 'const canIntervene = can("orders.intervene")', "التدخّلُ التشغيليّ"],
    ["components/admin/orders/OrdersScreen.tsx", 'canIntervene && TRANSFERABLE', "تحويلُ الطلب"],
    ["components/admin/orders/OrdersScreen.tsx", 'canIntervene && o.closed_at', "إعادةُ الحساب"],
    ["components/admin/orders/OrdersScreen.tsx", 'can("drivers.read")', "عدّادُ الوردية"],
    // **ورايةُ تحرير الإعداد من المحرّك لا من اسم الناظر.**
    ["app/dashboard/settings/page.tsx", "s.editable ?? false", "تحريرُ مفتاح الإعداد"],
  ];
  const missing = [];
  for (const [rel, want, what] of MUST) {
    if (!strip(read(join(src, rel))).includes(want)) missing.push(`${what} ⇒ ${want}`);
  }
  if (missing.length > 0) {
    problems.push("**فعلٌ فقد بابَ قدرته**:\n   " + missing.join("\n   "));
  } else {
    notes.push(`${MUST.length} فعلاً حسّاساً — كلٌّ ببابه`);
  }
}

if (problems.length > 0) {
  console.error("أبوابُ القدرات — خلل:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(1);
}
for (const n of notes) console.log("  · " + n);
console.log("كلُّ بابٍ بقدرته · وكلُّ فعلٍ ببابه · ولا اسمَ دورٍ في شرط.");
