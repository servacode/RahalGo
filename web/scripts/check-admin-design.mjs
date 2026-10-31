// ══════════════════════════════════════════════════════════════════════
// **حارسُ تصميم اللوحة — سقفٌ لكلّ ملفٍّ ينزل ولا يصعد** (دفعةُ تصميم اللوحة ١، ٢٠٢٦-١٠-٣١)
// ══════════════════════════════════════════════════════════════════════
//
// قرّر المالك إعادةَ تصميم لوحة الإدارة على دفعات. **وهذه الدفعةُ بنت العُدّة**
// (`packages/ui/src/admin.tsx` و`status.tsx`) — **وما لم يُنقل بعد يبقى كما هو.**
//
// فالحارسُ يعدّ في `app/dashboard/**` و`components/admin/**` خمسةَ أشياء:
//
//   tinyText      نصٌّ أصغرُ من ١٢ بكسل (`text-2xs` · `text-[11px]` …)
//   subtitle      سطرُ شرحٍ تحت عنوان الصفحة (`subtitle=` على PageHero/PageHeader/AdminPage)
//   rawButton     `<button>` مبنيٌّ باليد — والبديلُ `Button` بمقاساته (`sm` · `md` · `lg`)
//   handEmpty     حالةُ فراغٍ مبنيّةٌ باليد — والبديلُ `AdminEmpty`
//   statusColor   خريطةُ ألوانٍ للحالات خارجَ القاموس الواحد — والبديلُ `StatusBadge`/`statusMeta`
//
// **ولكلّ ملفٍّ سقفُه** في `admin-design-baseline.json`: لا يصعد (مخالفةٌ جديدة)،
// **ولا ينزل بصمت** — من أصلح موضعاً أنزل السقفَ بـ`--record`، فيصير التقدّمُ مكتوباً.
// **وملفٌّ جديدٌ سقفُه صفر.**

import { readFileSync, writeFileSync, readdirSync, statSync, existsSync } from "node:fs";
import { join, relative } from "node:path";

const ROOT = new URL("..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
const BASELINE = join(ROOT, "scripts", "admin-design-baseline.json");
const SCOPE = ["apps/rahalgo/src/app/dashboard", "apps/rahalgo/src/components/admin"];

const STATUS_KEYS =
  "pending|processing|paid|rejected|failed|reversed|accepted|preparing|dispatching|assigned|at_pickup|" +
  "picked_up|on_the_way|at_dropoff|delivered|cancelled|refunded|open|in_progress|awaiting_finance|" +
  "resolved|voided|written_off|acked";

const RULES = {
  tinyText: /\btext-2xs\b|text-\[(?:\d|1[01])(?:\.\d+)?px\]|text-\[0?\.(?:[0-6]\d*|7[0-4]\d*)rem\]/g,
  rawButton: /<button\b/g,
  handEmpty: /function\s+Empty[A-Z]?\w*\s*\(|\bsurface\b[^"'`\n]*\btext-center\b|\btext-center\b[^"'`\n]*\bsurface\b/g,
  statusColor: new RegExp(
    `\\b(?:${STATUS_KEYS})\\s*:\\s*["'\`](?:warning|info|success|danger|neutral|violet|accent|primary|[^"'\`]*\\b(?:bg|text|border)-(?:warning|info|success|danger|violet|accent|primary))\\b`,
    "g",
  ),
};
/** **سطرُ الشرح** — يُقرأ في وسم الرأس كاملاً (قد يمتدّ أسطراً). */
const HEADER = /<(?:PageHero|PageHeader|AdminPage)\b[\s\S]*?(?:\/>|\n\s*>)/g;

function walk(dir, out = []) {
  if (!existsSync(dir)) return out;
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (name.endsWith(".tsx")) out.push(p);
  }
  return out;
}

/** **التعليقاتُ لا تُعدّ** — تُبيَّض مع إبقاء الأسطر. (و`://` في الروابط ليس تعليقاً.) */
function stripComments(src) {
  return src
    .replace(/\/\*[\s\S]*?\*\//g, (c) => c.replace(/[^\n]/g, " "))
    .replace(/(^|[^:"'`])\/\/[^\n]*/g, (c, pre) => pre + " ".repeat(c.length - pre.length));
}

const now = {};
for (const dir of SCOPE) {
  for (const abs of walk(join(ROOT, dir))) {
    const file = relative(ROOT, abs).replace(/\\/g, "/");
    const src = stripComments(readFileSync(abs, "utf8"));
    const counts = {};
    for (const [rule, re] of Object.entries(RULES)) {
      const n = (src.match(re) || []).length;
      if (n) counts[rule] = n;
    }
    const subs = (src.match(HEADER) || []).filter((h) => /\bsubtitle=/.test(h)).length;
    if (subs) counts.subtitle = subs;
    if (Object.keys(counts).length) now[file] = counts;
  }
}

const sorted = (o) => Object.fromEntries(Object.keys(o).sort().map((k) => [k, o[k]]));

if (process.argv.includes("--record")) {
  writeFileSync(BASELINE, JSON.stringify(sorted(now), null, 2) + "\n", "utf8");
  console.log("سُجّل سقفُ تصميم اللوحة في admin-design-baseline.json");
  process.exit(0);
}

const base = existsSync(BASELINE) ? JSON.parse(readFileSync(BASELINE, "utf8")) : {};
const up = [];
const down = [];
const rules = [...Object.keys(RULES), "subtitle"];
for (const file of new Set([...Object.keys(base), ...Object.keys(now)])) {
  for (const rule of rules) {
    const was = base[file]?.[rule] ?? 0;
    const is = now[file]?.[rule] ?? 0;
    if (is > was) up.push(`${file} · ${rule}: ${was} -> ${is}`);
    else if (is < was) down.push(`${file} · ${rule}: ${was} -> ${is}`);
  }
}

const FIX = {
  tinyText: "أصغرُ نصٍّ text-xs (12 بكسل)",
  subtitle: "لا سطرَ شرحٍ تحت العنوان — AdminPage بلا subtitle",
  rawButton: "Button بمقاسه sm|md|lg · والنادرُ في MoreMenu",
  handEmpty: "AdminEmpty · AdminReload · AdminLoading",
  statusColor: "StatusBadge أو statusMeta من @rahalgo/ui (status.tsx)",
};

if (up.length) {
  console.error("مخالفةٌ جديدةٌ في تصميم اللوحة — والسقفُ لا يصعد:");
  for (const u of up) console.error("  · " + u + "   ⇒ " + FIX[u.split(" · ")[1].split(":")[0]]);
  process.exit(1);
}
if (down.length) {
  console.error("نزلت المخالفات — فأنزل السقفَ معها (node scripts/check-admin-design.mjs --record):");
  for (const d of down) console.error("  · " + d);
  process.exit(1);
}

const total = Object.values(now).reduce((s, c) => s + Object.values(c).reduce((a, b) => a + b, 0), 0);
console.log(`سقفُ تصميم اللوحة ثابت — ${total} موضعاً قديماً باقياً، ولا موضعَ جديد.`);
