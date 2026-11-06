#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  شرح إنشاء الحساب — الموقع بيقول نفس ما بيقوله التطبيق (٢٠٢٦-١١-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرار المالك: سهم بيدلّ على «إنشاء حساب جديد» وبيمشي مع الزبون خطوة خطوة،
 *  بالتطبيق وبالموقع /shop سوا.)
 *
 * # شو بيتحرس
 *
 *	١ · **نصوص الفقاعات** بـ ar.json (`shop.auth.guide*`) هي نفسها بـ strings.xml
 *	    (`guide_*`) — مين غيّر كلمة بواحد بيسقط عليه التاني.
 *	٢ · **عدد أرقام الكود** بالموقع والتطبيق = اللي بيولّده المحرّك (`randomDigits(n)`).
 *	٣ · **منطق الخطوات** (`guideSpot`) — الرقم ← الكود ← الاسم ← كلمة السر، والحقل
 *	    الملموس بيغلب، والتخزين المحجوب ما بيوقّع شي.
 *	٤ · **الشاشة فعلاً بتستعمله** — `CustomerAuth.tsx` فيها الفقاعة والإضاءة.
 */
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
const web = join(here, "..");
const repo = join(web, "..");
register(pathToFileURL(join(here, "_tsload.mjs")).href);

let fails = 0;
const fail = (msg) => {
  fails++;
  console.error(`✗ ${msg}`);
};
const eq = (name, got, want) => {
  if (got !== want) fail(`${name}\n   got:  ${JSON.stringify(got)}\n   want: ${JSON.stringify(want)}`);
};
const read = (p) => {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
};

// ── ١ · النصوص ───────────────────────────────────────────────────────
const xml = read(join(repo, "mobile", "ui", "src", "main", "res", "values", "strings.xml"));
const android = (name) => {
  const m = xml.match(new RegExp(`<string name="${name}">([\\s\\S]*?)</string>`));
  if (!m) {
    fail(`النص ${name} مو موجود بـ strings.xml`);
    return "";
  }
  return m[1].replace(/%1\$d/g, "{n}");
};
const dict = JSON.parse(read(join(web, "packages", "i18n", "src", "locales", "ar.json")));
const A = dict.shop?.auth ?? {};
const pairs = {
  guideLogin: "guide_login",
  guidePhone: "guide_phone",
  guideCode: "guide_code",
  guideName: "guide_name",
  guidePassword: "guide_password",
  guideReferral: "guide_referral",
  guideSkip: "guide_skip",
};
for (const [web, app] of Object.entries(pairs)) eq(`shop.auth.${web} ↔ ${app}`, A[web], android(app));

// ── ٢ · عدد أرقام الكود ─────────────────────────────────────────────
const go = read(join(repo, "backend", "internal", "identity", "service.go"));
const engine = [...new Set([...go.matchAll(/randomDigits\((\d+)\)/g)].map((m) => Number(m[1])))];
const helper = await import(pathToFileURL(join(web, "apps", "rahalgo", "src", "lib", "customer", "signupGuide.ts")).href);
eq("أطوال الكود بالمحرّك", engine.join(","), String(helper.SIGNUP_CODE_DIGITS));
const kt = read(join(repo, "mobile", "ui", "src", "main", "kotlin", "com", "rahalgo", "ui", "SignupGuide.kt"));
eq("عدد الكود بالتطبيق", kt.match(/const val SIGNUP_CODE_DIGITS = (\d+)/)?.[1], String(helper.SIGNUP_CODE_DIGITS));

// ── ٣ · منطق الخطوات ─────────────────────────────────────────────────
const { guideSpot, guidePasswordDone, guideSeen, markGuideSeen } = helper;
eq("خطوة الرقم", guideSpot("phone", false, false, null), "phone");
eq("خطوة الكود", guideSpot("code", false, false, null), "code");
eq("أول البيانات الاسم", guideSpot("details", false, false, null), "name");
eq("بعد الاسم كلمة السر", guideSpot("details", true, false, null), "password");
eq("بعد كلمة السر ما في فقاعة", guideSpot("details", true, true, null), null);
eq("الحقل الملموس بيغلب", guideSpot("details", false, false, "password"), "password");
eq("حقل من خطوة تانية ما بينقبل", guideSpot("phone", false, false, "name"), "phone");
eq("رقم مستعمل ما في فقاعة", guideSpot("taken", false, false, null), null);
eq("كلمة قصيرة", guidePasswordDone("1234567", "1234567", 8), false);
eq("تأكيد مختلف", guidePasswordDone("12345678", "1234567x", 8), false);
eq("كلمة تمام", guidePasswordDone("12345678", "12345678", 8), true);
const mem = new Map();
const store = { getItem: (k) => mem.get(k) ?? null, setItem: (k, v) => void mem.set(k, v) };
eq("أول مرة ما شافه", guideSeen(store), false);
markGuideSeen(store);
eq("بعدها شافه", guideSeen(store), true);
const broken = {
  getItem: () => {
    throw new Error("blocked");
  },
  setItem: () => {
    throw new Error("blocked");
  },
};
eq("تخزين محجوب بيرجّع ما شافه", guideSeen(broken), false);
try {
  markGuideSeen(broken);
} catch {
  fail("markGuideSeen رمى مع تخزين محجوب");
}

// ── ٤ · الشاشة بتستعمله ──────────────────────────────────────────────
const screen = read(join(web, "apps", "rahalgo", "src", "components", "customer", "CustomerAuth.tsx"));
for (const needle of ["<GuideSpotlight", "T.guideLogin", "guideSpot(", 'hint("phone")', 'hint("code")', 'hint("name")', 'hint("password")'])
  if (!screen.includes(needle)) fail(`CustomerAuth.tsx ما فيها ${needle}`);

if (fails) {
  console.error(`\ncheck-signup-guide: ${fails} فشل`);
  process.exit(1);
}
console.log("✓ check-signup-guide: نصوص الشرح والكود والخطوات متطابقة بين التطبيق والموقع");
