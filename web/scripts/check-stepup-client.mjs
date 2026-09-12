#!/usr/bin/env node
/**
 * ══════════════════════════════════════════════════════════════════════
 *  **تدفّقُ التأكيد في العميل — حارسٌ دائمٌ يمشي الطريقَ كلَّه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع في الإنتاج (٢٠٢٦-٠٩-١٢)
 *
 * **ضغط المالكُ «حفظ» لإنشاء دورٍ فقرأ «حدث خطأ غير متوقع، حاول
 * مجددا»** ثلاثَ مرّات. **والمحرّكُ كان يردّ تحدّياً صحيحَ المحتوى
 * مغلَّفاً في `{"data":…}`** — **والعميلُ يقرأ `error` و`step_up` في
 * أعلى الجسم**، فلم يجدهما، **فصار الخطأُ «داخليّاً» ولم تُفتح نافذةُ
 * كلمةِ المرور قطُّ.** **وصفرُ نداءٍ إلى `/admin/step-up`.**
 *
 * # ولماذا لم يُكشف
 *
 * **أُثبت التدفّقُ بـcurl وإثباتٍ صُنع باليد** — **فلم يمرّ أحدٌ
 * بالجواب الذي يعتمد عليه المتصفّح.**
 *
 * **فحصٌ يصنع الإثباتَ بنفسه لا يحرس بابَ الإثبات.**
 *
 * # وما يُحرَس هنا — الطريقُ كلُّه لا حلقةٌ منه
 *
 *	١ · تحدٍّ مكشوفٌ ⇒ `ApiError.body.code = step_up_required`
 *	٢ · و`stepUp` معرَّفٌ فيه الفعلُ والهدف
 *	٣ · وتُسأل الكلمةُ مرّةً (فرعُ `StepUpGate` يُختار)
 *	٤ · ويُنادى `/api/v1/admin/step-up`
 *	٥ · ويُعاد النداءُ الأصليُّ بترويسة `X-Step-Up`
 *	٦ · فينجح
 *	٧ · **وإلغاءُ التأكيد يُبقي الخطأ** ولا يُبتلَع صامتاً
 *	٨ · **وشاهدٌ سالبٌ دائم**: الجسمُ مغلَّفاً في `data` ⇒ لا تُسأل
 *	    الكلمةُ ويصير الخطأُ «داخليّاً» — **وهو عطبُ الإنتاج بعينه.**
 *
 * **ويُشغَّل العميلُ الحقيقيُّ لا نسخةٌ منه** — `packages/auth/src/client.ts`.
 */
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { register } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
register(pathToFileURL(join(here, "_tsload.mjs")).href);

// ── مِسندُ متصفّحٍ أصغرُ ما يكفي ──────────────────────────────────────
const mem = () => {
  const s = new Map();
  return {
    getItem: (k) => (s.has(k) ? s.get(k) : null),
    setItem: (k, v) => s.set(k, String(v)),
    removeItem: (k) => s.delete(k),
    clear: () => s.clear(),
  };
};
globalThis.localStorage = mem();
globalThis.sessionStorage = mem();
globalThis.window = globalThis;
globalThis.__RAHALGO_CONFIG__ = Object.freeze({ apiUrl: "https://api.test" });

const client = await import(
  pathToFileURL(join(here, "..", "packages", "auth", "src", "client.ts")).href
);

const problems = [];
const notes = [];

/** **جسمُ التحدّي كما يكتبه المحرّكُ** — `httpx.ErrorWith`. */
const CHALLENGE = {
  error: { code: "step_up_required", message_key: "errors.step_up_required" },
  step_up: { action: "admin.role_create", target_type: "role", target_id: "" },
};

const SENSITIVE_PATH = "/api/v1/admin/roles";
const SENSITIVE_BODY = JSON.stringify({ code: "observability", name: "مراقبة التشغيل" });

/**
 * drive **يمشي التدفّقَ بجسمِ تحدٍّ بعينه.**
 *
 * `wrap` يغلّف الجسمَ في `data` — **محاكاةُ عطب الإنتاج.**
 * `answer` قيمةُ ما يردّه سائلُ الكلمة (`null` = إلغاء).
 */
async function drive({ wrap = false, answer = "GRANT-1" } = {}) {
  const calls = [];
  let asked = 0;
  let askedNeed = null;
  let askedReq = null;

  globalThis.fetch = async (url, init = {}) => {
    const h = {};
    new Headers(init.headers).forEach((v, k) => (h[k] = v));
    const path = String(url).replace("https://api.test", "");
    calls.push({ method: (init.method ?? "GET").toUpperCase(), path, stepUp: h["x-step-up"] ?? null });
    if (path === "/api/v1/admin/step-up") {
      return new Response(JSON.stringify({ data: { id: "GRANT-1" } }), { status: 200 });
    }
    if (h["x-step-up"]) {
      return new Response(JSON.stringify({ data: { code: "observability" } }), { status: 200 });
    }
    const body = wrap ? { data: CHALLENGE } : CHALLENGE;
    return new Response(JSON.stringify(body), { status: 403 });
  };

  client.tokenStore.set({ access_token: "ACCESS", refresh_token: "REFRESH" });
  // **والسائلُ يفعل ما تفعله `StepUpGate` بالضبط**: ينادي
  // `requestStepUp` بكلمةٍ — **ولا يختلق إثباتاً من عنده.**
  // (**وكان يختلقه في أوّل تشغيل**، فمرّ الفحصُ ولم يبلغ بابَ الإثبات —
  // **وهي علّةُ فحوصِ ٧٠ب-و١ عينُها.**)
  client.setStepUpAsker(async (need, req) => {
    asked++;
    askedNeed = need;
    askedReq = req;
    if (answer === null) return null;
    return client.requestStepUp(req, "كلمةُ الفحص");
  });

  let ok = false;
  let err = null;
  try {
    await client.api(SENSITIVE_PATH, { method: "POST", body: SENSITIVE_BODY });
    ok = true;
  } catch (e) {
    err = e;
  }
  client.setStepUpAsker(null);
  return { ok, err, asked, askedNeed, askedReq, calls };
}

// ══════════════════════════════════════════════════════════════════════
//  ١ · الطريقُ كلُّه بجسمٍ مكشوف
// ══════════════════════════════════════════════════════════════════════
{
  const r = await drive();
  if (r.asked !== 1) {
    problems.push(`لم تُسأل الكلمةُ مرّةً واحدةً — ${r.asked} — ففرعُ نافذةِ التأكيد لا يُختار`);
  }
  if (r.askedNeed?.action !== "admin.role_create") {
    problems.push(`الفعلُ المعروضُ على من يؤكّد ${JSON.stringify(r.askedNeed)} — والواجبُ وصفُ المحرّك`);
  }
  if (r.askedReq?.body?.code !== "observability") {
    problems.push("جسمُ الطلبِ لا يُعرَض على من يؤكّد — فيؤكّد ما لا يرى");
  }
  const stepUpCall = r.calls.find((c) => c.path === "/api/v1/admin/step-up");
  if (!stepUpCall) {
    problems.push("لم يُنادَ `/api/v1/admin/step-up` — ولا إثباتَ بلا نداء");
  }
  const retry = r.calls.filter((c) => c.path === SENSITIVE_PATH && c.stepUp);
  if (retry.length !== 1) {
    problems.push(`إعادةُ النداء بترويسة X-Step-Up = ${retry.length} — والواجبُ واحدة`);
  }
  if (!r.ok) {
    problems.push(`التدفّقُ لم ينجح — ${r.err?.status} ${r.err?.body?.code}`);
  }
  if (problems.length === 0) {
    notes.push(
      "الطريقُ كلُّه: تحدٍّ ⇒ سؤالُ كلمةٍ ⇒ /admin/step-up ⇒ إعادةٌ بـX-Step-Up ⇒ نجاح",
    );
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٢ · وإلغاءُ التأكيد يُبقي الخطأ
// ══════════════════════════════════════════════════════════════════════
{
  const r = await drive({ answer: null });
  if (r.ok) {
    problems.push("إلغاءُ التأكيد مضى كأنّه نجاح — وفعلٌ لم يقع يُقال إنّه وقع");
  } else if (r.err?.body?.code !== "step_up_required") {
    problems.push(`إلغاءُ التأكيد بدّل الخطأ إلى ${r.err?.body?.code} — والواجبُ إبقاؤه`);
  } else if (r.calls.some((c) => c.stepUp)) {
    problems.push("أُعيد النداءُ بعد إلغاءِ التأكيد");
  } else {
    notes.push("وإلغاءُ التأكيد يُبقي الخطأ كما هو — ولا يُبتلَع");
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٣ · الشاهدُ السالبُ الدائم — عطبُ الإنتاج بعينه
// ══════════════════════════════════════════════════════════════════════
//
// **ولو رجع الغلافُ غداً لسقط هذا الفحصُ** — **وهو ما لم يكن موجوداً
// يومَ سقط الإنتاج.**
{
  const r = await drive({ wrap: true });
  const buried =
    r.asked === 0 &&
    !r.ok &&
    r.err?.body?.code === "internal" &&
    r.err?.stepUp === undefined &&
    !r.calls.some((c) => c.path === "/api/v1/admin/step-up");
  if (!buried) {
    problems.push(
      "الشاهدُ السالبُ لا يقيس: الجسمُ المغلَّفُ في `data` كان يجب أن يُدفن التحدّيَ " +
        `— asked=${r.asked} code=${r.err?.body?.code} ok=${r.ok}`,
    );
  } else {
    notes.push(
      'وشاهدٌ سالبٌ دائم: الجسمُ مغلَّفاً في `data` ⇒ لا تُسأل الكلمةُ ' +
        "ويصير الخطأُ «داخليّاً» — وهو عطبُ ٢٠٢٦-٠٩-١٢ بعينه",
    );
  }
}

// ══════════════════════════════════════════════════════════════════════
//  ٤ · وشكلُ التحدّي في المحرّك — لا يُقرَأ من نسخةٍ هنا
// ══════════════════════════════════════════════════════════════════════
//
// **والعقدُ طرفان** — **وحارسٌ يفحص طرفاً واحداً يمرّ وهو مكسور.**
// فيُقرأ كاتبُ التحدّي في المحرّك: **ظرفُ خطأٍ لا ظرفُ نجاح.**
{
  const { readFileSync } = await import("node:fs");
  const src = (() => {
    try {
      return readFileSync(join(here, "..", "..", "backend", "internal", "server", "stepup.go"), "utf8");
    } catch {
      return "";
    }
  })();
  if (!src) {
    problems.push("تعذّرت قراءةُ بوّابة التأكيد في المحرّك — والعقدُ طرفان");
  } else {
    const body = src.replace(/\/\*[\s\S]*?\*\//g, "").replace(/\/\/.*$/gm, "");
    const i = body.indexOf("func (s *Server) respondStepUpRequired");
    const fn = i < 0 ? "" : body.slice(i, body.indexOf("\n}", i));
    if (!fn) {
      problems.push("لم يُعثَر على كاتب التحدّي في المحرّك");
    } else if (fn.includes("httpx.JSON(")) {
      problems.push("المحرّكُ يكتب التحدّيَ بغلاف النجاح `data` — وهو عطبُ ٢٠٢٦-٠٩-١٢");
    } else if (!fn.includes("httpx.ErrorWith(")) {
      problems.push("المحرّكُ لا يكتب التحدّيَ بظرف الخطأ المركزيّ");
    } else {
      notes.push("وطرفُ المحرّك: التحدّي بظرف الخطأ لا بغلاف النجاح");
    }
  }
}

if (problems.length > 0) {
  console.error("تدفّقُ التأكيد — خلل:");
  for (const p of problems) console.error("  ✗ " + p);
  process.exit(1);
}
for (const n of notes) console.log("  · " + n);
console.log("تدفّقُ التأكيد يمشي كلَّ الطريق — ولا يُدفَن التحدّي.");
