/**
 * ══════════════════════════════════════════════════════════════════════
 * **تثبيتُ الموقع على الشاشة الرئيسيّة — يبقى صالحاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون يضيف الموقعَ للشاشة فيفتح كأنّه
 *  تطبيق.)
 *
 * **وما يُكسر هنا يُكسر صامتاً**: بيانٌ بلا أيقونةٍ أو بمقاسٍ خطأ يُضاف
 * للشاشة بصورةٍ فارغة، **ولونٌ بدّله أحدٌ في `theme.css` يبقى قديماً في
 * البيان** — ولا شيءَ في الشاشة يصرخ.
 *
 * فيُفحص:
 *   ١ · ألوانُ البيان المولَّدة = `--color-shell` في `theme.css` الآن.
 *   ٢ · كلُّ أيقونةٍ موجودةٌ بمقاسها المعلَن (يُقرأ من ترويسة PNG).
 *   ٣ · البيانُ يبدأ من السوق ويفتح بلا متصفّح.
 *   ٤ · وسومُ آيفون في غلاف السوق.
 *
 * والعلاجُ واحد: `python web/scripts/make-pwa.py`
 */
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const app = join(web, "apps", "rahalgo");

let bad = 0;
const fail = (msg) => {
  console.error(`  ✗ ${msg}`);
  bad++;
};

const css = readFileSync(join(web, "packages", "ui", "src", "theme.css"), "utf8");
const shell = (/--color-shell:\s*(#[0-9a-fA-F]{6})\s*;/.exec(css)?.[1] ?? "").toLowerCase();
if (!shell) fail("لم يوجد --color-shell في theme.css");

const genPath = join(app, "src", "lib", "pwa.gen.json");
if (!existsSync(genPath)) {
  fail("pwa.gen.json غائب");
} else {
  const gen = JSON.parse(readFileSync(genPath, "utf8"));
  if (gen.theme !== shell || gen.background !== shell)
    fail(`ألوانُ البيان (${gen.theme}) غيرُ لون الأرض في التوكنز (${shell}) — شاخت`);
  for (const icon of gen.icons ?? []) {
    const file = join(app, "public", icon.src);
    if (!existsSync(file)) {
      fail(`أيقونةٌ غائبة: ${icon.src}`);
      continue;
    }
    const b = readFileSync(file);
    const isPng = b.length > 24 && b.readUInt32BE(0) === 0x89504e47;
    const w = isPng ? b.readUInt32BE(16) : 0;
    const h = isPng ? b.readUInt32BE(20) : 0;
    if (w !== icon.size || h !== icon.size)
      fail(`${icon.src}: مقاسُها ${w}×${h} والمعلَنُ ${icon.size}`);
  }
  if (!(gen.icons ?? []).some((i) => i.src.includes("maskable")))
    fail("لا أيقونةَ قابلةً للقصّ — أندرويد يقصّ الأيقونةَ بشكله");
  if (!(gen.icons ?? []).some((i) => i.src.includes("apple-touch") && i.size === 180))
    fail("لا أيقونةَ لآيفون بمقاس ١٨٠");
}

const manifest = existsSync(join(app, "src", "app", "manifest.ts"))
  ? readFileSync(join(app, "src", "app", "manifest.ts"), "utf8")
  : "";
if (!manifest) fail("لا بيانَ (app/manifest.ts)");
else {
  if (!/start_url:\s*"\/shop"/.test(manifest)) fail("البيانُ لا يبدأ من السوق");
  if (!/display:\s*"standalone"/.test(manifest)) fail("البيانُ يفتح داخلَ المتصفّح");
  if (!manifest.includes("pwa.gen.json")) fail("ألوانُ البيان مكتوبةٌ باليد لا من المولَّد");
}

const layout = readFileSync(join(app, "src", "app", "(shop)", "layout.tsx"), "utf8");
if (!/appleWebApp:\s*\{\s*capable:\s*true/.test(layout)) fail("غلافُ السوق بلا وسم آيفون للتطبيق");
if (!layout.includes("/pwa/apple-touch-icon.png")) fail("غلافُ السوق بلا أيقونة آيفون");

if (bad) {
  console.error(`\n**تثبيتُ الموقع مكسور** — ${bad} خرقاً. العلاج: python web/scripts/make-pwa.py`);
  process.exit(1);
}
console.log("تثبيتُ الموقع سليم — البيانُ من السوق · الألوانُ من التوكنز · والأيقوناتُ بمقاساتها.");
