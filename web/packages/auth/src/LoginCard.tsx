"use client";

/**
 * بطاقة الدخول المركزية — نموذج واحد للجميع، وكل شخص يدخل حسب دوره.
 * تحمل أربعة أوضاع في بطاقة واحدة (دخول / رمز تحقق / استعادة / حساب جديد)
 * فلا تُبنى صفحة منفصلة لكل تدفّق، ولا تفقد البطاقة سياقها بانتقال.
 *
 * التصميم: لوحة مقسومة — جانب العلامة يحمل قصتها (نهر الفرات يعبر الصحراء،
 * BRAND.md) وجانب النموذج نظيف واسع. على الجوال ينطوي جانب العلامة إلى ترويسة
 * مضغوطة. كل الألوان من التوكنز حصراً وكل النصوص من المعجم (GROUND-RULES §1).
 */

import { useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Input,
  Checkbox,
  Modal,
  OtpInput,
  PasswordMeter,
  IconPhone,
  IconLock,
  IconUser,
  IconKey,
  IconSignup,
  IconPrev,
  IconCheck,
  IconSuccess,
  BrandMark,
  usePlatform,
} from "@rahalgo/ui";
import { authApi, tokenStore, ApiError, type AuthUser } from "./client";

const m = getMessages(defaultLocale);
const A = m.auth;

/** أوضاع البطاقة — تبديل داخلي بلا انتقال بين صفحات. */
type Mode = "password" | "otp" | "reset" | "signup";

/** ترجمة مفتاح الخطأ القادم من الخادم — منطق واحد لكل اللوحات. */
export function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key;
  const last = key.split(".").pop() ?? "";
  const known = (m.errors as Record<string, string>)[last];
  if (known) return known;
  if (key === "auth.otpInvalid") return m.auth.otpInvalid;
  return m.errors.internal;
}

/* **وذهبت `HERO_POINTS` مع الجانب الترويجيّ** — ثلاثُ ميزاتٍ تُبنى ولا
   تُرسم. **وقائمةٌ تُحسب ولا تُقرأ تبقى تُصان بلا فائدة**: يُترجم نصُّها
   ويُراجع، ثمّ يُكتشف بعد شهرٍ أنّها لا تظهر. (ونصوصُها في المعجم كما هي —
   لمن أرادها في صفحةٍ تسويقيّةٍ لاحقاً.) */

export function LoginCard({
  methods = "both",
  signup = false,
  onSuccess,
  footer,
  referral = "",
  initialMode = "password",
  onModeChange,
}: {
  /**
   * **هل يُنشئ أحدٌ حسابَه بنفسه في هذه الواجهة؟**
   *
   * (أمرُ المالك ٢٠٢٦-٠٨-٠٦: «شاشةُ تسجيل دخولٍ واحدةٌ لكلّ الواجهات، تصميمٌ
   *  واحدٌ **مع اختلاف الروابط حسب كلّ واجهة**».)
   *
   * **وكان «إنشاء حساب» يظهر في الخمس** — واللوحاتُ الأربعُ حساباتُها تُنشأ
   * من المنصة أو بمندوبٍ معتمد. **فرابطٌ يقود إلى بابٍ لا يخصّ صاحبَه يُعلّمه
   * ألّا يثق بما يُعرض عليه.**
   *
   * **وهذا هو الفرقُ المسموح**: الروابطُ تختلف والتصميمُ واحد.
   */
  signup?: boolean;
  /**
   * رمزُ من دعا هذا المستخدم — **يُمرَّر من الرابط ولا يُكتب باليد.**
   *
   * **ومن سجّل بلا دعوةٍ حسابُه كامل**: الرمزُ زيادةٌ لا شرط. **ورمزٌ خاطئٌ
   * لا يُسقط تسجيلاً** — يُحرَم المكافأةَ وحدَها.
   */
  referral?: string;
  /** طرق الدخول المتاحة — الافتراضي كلاهما */
  methods?: "both" | "password" | "otp";
  /** يُستدعى بعد نجاح الدخول (بعد تخزين التوكن) — هنا يقرر التطبيق الوجهة */
  onSuccess: (user: AuthUser) => void | Promise<void>;
  footer?: ReactNode;
  /**
   * **الوضعُ الذي تُفتح عليه** — يأتي من المسار.
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يوجد رابطُ تسجيلٍ، يبقى ضمن تسجيل الدخول
   *  — هل هذا شيءٌ طبيعيّ؟» والجواب: لا.)
   *
   * **ورابطُ الدعوة كان يقع على نموذج الدخول**: من دُعي ليُنشئ حساباً يصل
   * `‎/login?ref=CODE` **فيرى شاشةً تطلب كلمةَ مرورٍ لا يملكها**، وعليه أن
   * يجد «إنشاء حساب» بنفسه. **وصار `‎/signup?ref=CODE`.**
   */
  initialMode?: Mode;
  /**
   * **يُخبر التطبيقَ أنّ الوضعَ تبدّل** ليُبدّل المسارَ معه.
   *
   * **والتبديلُ داخلَ البطاقة يبقى كما هو** — لا انتقالَ صفحةٍ ولا فقدَ لما
   * كُتب، **إنّما يصير لكلّ شاشةٍ عنوانٌ يُشارَك ويُحدَّث ويُقاس.**
   */
  onModeChange?: (m: Mode) => void;
}) {
  /* ══════════════════════════════════════════════════════════════════
     **وبابُ رمز التحقّق يُطفأ من الإعدادات — لا من شيفرة كلّ تطبيق**
     ══════════════════════════════════════════════════════════════════

     (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «جهّز بالإعدادات بلوحة الادمن خيار لإطفاء أو
      تشغيل تسجيل الدخول برمز التحقّق. لا تنسَ أنّ كلّ شيءٍ يجب أن يكون بشكلٍ
      مركزيّ — لا نريد أن نعدّل كلّ شيءٍ بكلّ مكان».)

     **و`methods` تبقى**: هي ما يقوله التطبيقُ عن نفسِه (بوّابةٌ تقبل الرمزَ
     أو لا)، **والإعدادُ ما تقوله المنصةُ عن الرمز كلِّه.** والأضيقُ يفوز.

     **ومن مرّر `otp` وحدَها ثمّ أُطفئ الرمزُ يسقط إلى كلمة المرور** — وإلّا
     بقيت بوّابتُه بلا بابٍ يعمل. */
  const { otpLogin, authBg, authBgDim } = usePlatform();
  const allow: "both" | "password" | "otp" = otpLogin ? methods : "password";
  const [mode, setMode] = useState<Mode>(initialMode);

  /* **والوضعُ يتبع ما هو مسموح**: من فُتح على `otp` ثمّ أُطفئ البابُ بعد أن
     وصل الردُّ **يبقى في شاشةٍ لا تعمل.** */
  useEffect(() => {
    setMode((cur) => {
      if (allow === "password" && cur === "otp") return "password";
      if (allow === "otp" && cur === "password") return "otp";
      return cur;
    });
  }, [allow]);
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [code, setCode] = useState("");
  const [remember, setRemember] = useState(true);
  /** هل أُرسل الرمز في الوضع الحالي؟ (يخصّ otp/reset/signup) */
  const [sent, setSent] = useState(false);
  /**
   * **مرحلةُ التسجيل الثالثة** — تُفتح بعد أن يصدّق الخادمُ الرمز.
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «يدخل الرقم، يضغط إرسال رمز، **لا تظهر
   *  المعلومات إلّا بعد التحقّق من الرمز**، ثمّ تظهر معلومات إنشاء الحساب».)
   *
   * **وكانت البياناتُ والرمزُ في نموذجٍ واحد** — فيملأ الاسمَ وكلمتين
   * ويوافق، **ثمّ يُقال له إنّ رقمه الأوّل خطأ.** والخطأُ يُقال عند وقوعه.
   */
  const [codeOK, setCodeOK] = useState(false);
  const [password2, setPassword2] = useState("");
  const [agreed, setAgreed] = useState(false);
  /** **تمّ الإنشاء** — نافذةٌ تُقرّ ثمّ يُساق إلى الدخول اليدويّ. */
  const [created, setCreated] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  /** الوصول للحساب: تخزين التوكن ثم تسليم القرار للتطبيق. */
  async function enter(result: { user: AuthUser; tokens: never }) {
    tokenStore.set(result.tokens, remember);
    try {
      await onSuccess(result.user);
    } catch {
      setBusy(false);
    }
  }

  /** غلاف واحد لكل عملية: يمسح الخطأ، يقفل الزر، ويفكّ القفل عند الفشل. */
  function run(fn: () => Promise<void>) {
    return async (e?: React.FormEvent) => {
      e?.preventDefault();
      setBusy(true);
      setError("");
      try {
        await fn();
      } catch (err) {
        setError(errText(err));
        setBusy(false);
      }
    };
  }

  function go(next: Mode) {
    setMode(next);
    onModeChange?.(next);
    setError("");
    setSent(false);
    setCode("");
    // **ولا يبقى أثرُ محاولةٍ سابقةٍ في وضعٍ جديد** — رمزٌ صُدِّق لهاتفٍ ثمّ
    // بُدّل الوضعُ يفتح نموذجَ البيانات بلا تحقّق.
    setCodeOK(false);
    setPassword2("");
    setAgreed(false);
    setCreated(false);
  }

  // ---------- الحقول المشتركة ----------

  const phoneField = (
    <Input
      id="phone"
      label={A.phone}
      icon={<IconPhone />}
      dir="ltr"
      inputMode="tel"
      autoComplete="tel"
      required
      value={phone}
      onChange={(e) => setPhone(e.target.value)}
      className="text-end"
      placeholder="09xxxxxxxx"
    />
  );

  /**
   * خانات الرمز — `onDone` يُستدعى فور اكتمال الرقم السادس.
   *
   * ══════════════════════════════════════════════════════════════════
   * **والقيمةُ تُمرَّر ولا تُقرأ من الحالة**
   * ══════════════════════════════════════════════════════════════════
   *
   * كان `onComplete={() => onDone?.()}` — **يرمي ما تعطيه `OtpInput`**،
   * فيقرأ المستدعي `code` من الحالة. **وحالةُ React لا تكون قد تحدّثت بعد**:
   * `onComplete` يقع في المعالج نفسِه الذي نادى `onChange`.
   *
   * **فيُرسَل خمسةُ أرقامٍ من ستّة.** وقِيس (٢٠٢٦-٠٨-٠٦): الحقولُ فيها
   * `123456` **والجسمُ المُرسَل `{"code":"12345"}`** — والخادمُ يردّ ٤٠١
   * على رمزٍ صحيح.
   *
   * **وهو يمسّ الدخولَ بالرمز والاستعادةَ أيضاً لا التسجيلَ وحدَه** — كلُّها
   * تستعمل هذا الحقل. **ومن ضغط الزرَّ بيده كان ينجح** لأنّ الحالةَ تكون قد
   * لحقت، **ومن اكتفى بالإكمال التلقائيّ يُرفض** — عطبٌ يظهر لبعض الناس
   * ولا يظهر لبعض.
   */
  const codeField = (onDone?: (v: string) => void) => (
    <div className="space-y-2">
      <label className="block text-center text-sm font-medium text-ink">{A.otpTitle}</label>
      <OtpInput
        value={code}
        onChange={setCode}
        autoFocus
        boxLabel={A.otpBoxLabel}
        onComplete={(v) => onDone?.(v)}
      />
    </div>
  );

  const passwordField = (id: string, label: string, autoComplete: string, meter = false) => (
    <div className="space-y-2">
      <Input
        id={id}
        label={label}
        icon={<IconLock />}
        type="password"
        autoComplete={autoComplete}
        required
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="••••••••"
      />
      {meter && (
        <PasswordMeter
          value={password}
          labels={[A.strength.weak, A.strength.fair, A.strength.strong]}
        />
      )}
    </div>
  );

  const errorBox = error ? (
    // **وكانت لافتةً مكتوبةً بالحرف** — بأيقونتها وحدّها وحشوتها، وهي عينُ
    // ما يفعله `Alert`. (وقد كُتبت هذه الصياغةُ في أربعة ملفّاتٍ متفرّقة.)
    <Alert>{error}</Alert>
  ) : null;

  const sentNote = (
    <Alert tone="success">
      <p>
        {A.otpSentTo}{" "}
        <span dir="ltr" className="font-bold">
          {phone}
        </span>
      </p>
      <p className="mt-0.5 text-xs opacity-80">{A.otpSentDev}</p>
    </Alert>
  );

  const submit = (label: string) => (
    <Button type="submit" disabled={busy} className="w-full py-3 text-base elev-1">
      {busy ? m.shared.loggingIn : label}
    </Button>
  );

  /** رابط نصّي ثانوي داخل البطاقة — شكل واحد لكل روابط التبديل. */
  const linkBtn = (label: string, onClick: () => void, icon?: ReactNode) => (
    <button
      type="button"
      onClick={onClick}
      /* **ونصُّ الرابط بالتوكن المُعدِّ للنصّ العاري.**

         كان `text-primary` — **وتباينُه على البطاقة ٣٫٠٩** والحدُّ ٤٫٥ لنصٍّ
         بأربعةَ عشرَ بكسلاً. **و`accent-text` صُنع لهذا بعينه** (٤٫٥٨):
         التعبئةُ تبقى بالنبرة، **والنصُّ العاري يحتاج درجةً أفتح.** */
      className="inline-flex items-center gap-1 rounded-control text-sm font-medium text-accent-text transition-colors hover:text-primary-dark hover:underline"
    >
      {icon}
      {label}
    </button>
  );

  const rememberBox = (id: string) => (
    <Checkbox
      id={id}
      label={A.rememberMe}
      checked={remember}
      onChange={(e) => setRemember(e.target.checked)}
    />
  );

  /* **وذهب مؤشّرُ الخطوات.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا داعي لها
     احذفها أيضاً».)

     كان شريطاً برقمين وعنوانين فوق كلّ تدفّقٍ من مرحلتين. **وهو يصف بنيةَ
     النموذج لا يقدّم فيه خطوة**: من يرى حقلَ هاتفٍ وزرَّ إرسالٍ يعرف أنّه في
     الأوّل، **ومن وصل شاشةَ الرمز يعرف أنّه تقدّم.**

     **والشريطُ يأخذ ثمانيةً وأربعين بكسلاً من أعلى البطاقة** — في شاشةٍ
     أطولُ ما فيها ثلاثةُ حقول. */

  // ---------- الأوضاع ----------

  function body() {
    if (mode === "password") {
      return (
        <form
          onSubmit={run(async () =>
            enter((await authApi.loginPassword(phone, password)) as never),
          )}
          className="space-y-4"
        >
          {phoneField}
          {passwordField("password", A.password, "current-password")}
          {/* يلتفّان سطرين على الشاشات الضيّقة بدل أن يتكسّر النصّان معاً */}
          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2">
            {rememberBox("remember")}
            {linkBtn(A.forgotPassword, () => go("reset"))}
          </div>
          {errorBox}
          {submit(A.login)}
        </form>
      );
    }

    if (mode === "otp") {
      // **والرمزُ يأتي من الحقل لا من الحالة** — انظر `codeField`.
      const verify = (c: string = code) =>
        run(async () => enter((await authApi.verifyOtp(phone, c)) as never))();
      return (
        <>
          {!sent ? (
            <form
              onSubmit={run(async () => {
                await authApi.requestOtp(phone);
                setSent(true);
                setBusy(false);
              })}
              className="space-y-4"
            >
              {phoneField}
              {errorBox}
              {submit(A.sendOtp)}
            </form>
          ) : (
            <form onSubmit={(e) => { e.preventDefault(); void verify(); }} className="space-y-4">
              {sentNote}
              {codeField((v) => void verify(v))}
              {rememberBox("remember-otp")}
              {errorBox}
              {submit(A.login)}
              <div className="text-center">{linkBtn(A.changePhone, () => setSent(false))}</div>
            </form>
          )}
        </>
      );
    }

    if (mode === "reset") {
      // **ثلاثُ مراحلَ كالتسجيل** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يجوز أن
      // يفتح الفورمُ بمجرّد إرسال طلب استعادة»).
      //
      // **والاستعادةُ أخطرُ من التسجيل**: من فتح نموذجَ كلمةٍ جديدةٍ بمجرّد
      // إرسال الرمز **يظنّ أنّه على وشك تغييرها**، فيكتبها مرّتين ثمّ يُرفض.
      if (!sent) {
        return (
          <form
            onSubmit={run(async () => {
              await authApi.requestReset(phone);
              setSent(true);
              setBusy(false);
            })}
            className="space-y-4"
          >
            {phoneField}
            {errorBox}
            {submit(A.resetSend)}
          </form>
        );
      }

      if (!codeOK) {
        const checkReset = (c: string = code) =>
          run(async () => {
            await authApi.verifyReset(phone, c);
            setCode(c);
            setCodeOK(true);
            setBusy(false);
          })();
        return (
          <form onSubmit={(e) => { e.preventDefault(); void checkReset(); }} className="space-y-4">
            {sentNote}
            {codeField((v) => void checkReset(v))}
            {errorBox}
            {submit(A.verifyCode)}
            <div className="text-center">{linkBtn(A.changePhone, () => setSent(false))}</div>
          </form>
        );
      }

      return (
        <form
          onSubmit={run(async () => {
            if (password !== password2) {
              setError(A.passwordMismatch);
              setBusy(false);
              return;
            }
            await authApi.confirmReset(phone, code, password);
            /* **ولا يُدخَل تلقائيّاً** — كالتسجيل: **من غيّر كلمتَه ثمّ دخل
               بها يتأكّد أنّها تعمل**، ولا يكتشف بعد أسبوعٍ أنّه لا يذكرها.
               (والخادمُ يُصدر جلسةً تُهمَل هنا.) */
            setCreated(true);
            setBusy(false);
          })}
          className="space-y-4"
        >
          {passwordField("new-password", A.newPassword, "new-password", true)}
          {/* **وتأكيدُ الكلمة** — تُكتب مخفيّةً، **وخطأُ حرفٍ يُقفل الحسابَ
              على صاحبه** ولا يُكتشف إلّا عند أوّل دخول. */}
          <Input
            id="new-password2"
            label={A.confirmPassword}
            icon={<IconLock />}
            type="password"
            autoComplete="new-password"
            required
            value={password2}
            onChange={(e) => setPassword2(e.target.value)}
            placeholder="••••••••"
          />
          <p className="text-xs text-ink-muted">{A.passwordHint}</p>
          {errorBox}
          {submit(A.resetConfirm)}
        </form>
      );
    }

    // إنشاء حساب — زبون فقط
    //
    // **ثلاثُ مراحلَ لا اثنتان** (قرارُ المالك ٢٠٢٦-٠٨-٠٦):
    //   ١ · الرقمُ ← إرسالُ الرمز
    //   ٢ · الرمزُ ← تحقّقٌ عند الخادم **بلا استهلاك**
    //   ٣ · البياناتُ ← إنشاءُ الحساب، ثمّ الدخولُ يدويّاً
    if (!sent) {
      return (
        <form
          onSubmit={run(async () => {
            await authApi.requestSignup(phone);
            setSent(true);
            setBusy(false);
          })}
          className="space-y-4"
        >
          {phoneField}
          {errorBox}
          {submit(A.signupSend)}
        </form>
      );
    }

    if (!codeOK) {
      const check = (c: string = code) =>
        run(async () => {
          await authApi.verifySignup(phone, c);
          // **والرمزُ المصدَّقُ يُثبَّت في الحالة** — تستعمله المرحلةُ الثالثة
          // عند الإنشاء، **وقد يكون ما في الحالة أنقصَ رقماً.**
          setCode(c);
          setCodeOK(true);
          setBusy(false);
        })();
      return (
        <form onSubmit={(e) => { e.preventDefault(); void check(); }} className="space-y-4">
          {sentNote}
          {/* **ويُصدَّق فور اكتمال الرقم السادس** — ولا يُنتظر ضغطُ زرّ. */}
          {codeField((v) => void check(v))}
          {errorBox}
          {submit(A.verifyCode)}
          <div className="text-center">{linkBtn(A.changePhone, () => setSent(false))}</div>
        </form>
      );
    }

    return (
      <form
        onSubmit={run(async () => {
          // **والكلمتان تُقارنان قبل النداء** — لا يُرسَل ما يُعرَف رفضُه.
          if (password !== password2) {
            setError(A.passwordMismatch);
            setBusy(false);
            return;
          }
          if (!agreed) {
            setError(A.mustAgree);
            setBusy(false);
            return;
          }
          await authApi.confirmSignup(phone, code, fullName, password, referral);
          /* ══════════════════════════════════════════════════════════════
             **ولا يُدخَل تلقائيّاً — يُساق إلى الدخول اليدويّ**
             ══════════════════════════════════════════════════════════════

             (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «تظهر نافذةٌ تفيد أنّه تمّ إنشاء
              الحساب بنجاح، تنتقل إلى شاشة تسجيل الدخول ليدخل بشكلٍ يدويّ».)

             **والخادمُ يُصدر جلسةً مع الإنشاء** — تُهمَل هنا ولا تُخزَّن.
             **فمن أنشأ حساباً ثمّ دخل بيده يتأكّد أنّ كلمتَه تعمل**، ولا
             يكتشف بعد أسبوعٍ أنّه لا يذكرها. */
          setCreated(true);
          setBusy(false);
        })}
        className="space-y-4"
      >
        <Input
          id="full-name"
          label={A.fullName}
          icon={<IconUser />}
          autoComplete="name"
          required
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
        />
        {passwordField("signup-password", A.password, "new-password", true)}
        {/* **وتأكيدُ الكلمة** — تُكتب مخفيّةً، **وخطأُ حرفٍ واحدٍ يُقفل الحسابَ
            على صاحبه** ولا يُكتشف إلّا عند أوّل دخول. */}
        <Input
          id="signup-password2"
          label={A.confirmPassword}
          icon={<IconLock />}
          type="password"
          autoComplete="new-password"
          required
          value={password2}
          onChange={(e) => setPassword2(e.target.value)}
          placeholder="••••••••"
        />
        <p className="text-xs text-ink-muted">{A.passwordHint}</p>
        {/* **والموافقةُ صريحةٌ لا مضمرة** — والرابطان يُفتحان في تبويبٍ آخر
            فلا يضيع ما مُلئ. */}
        <Checkbox
          id="agree-terms"
          checked={agreed}
          onChange={(e) => setAgreed(e.target.checked)}
          label={
            <span>
              {A.agreePrefix}{" "}
              <a href="/terms" target="_blank" rel="noreferrer" className="text-accent-text underline">
                {A.agreeTerms}
              </a>{" "}
              {A.agreeAnd}{" "}
              <a href="/privacy" target="_blank" rel="noreferrer" className="text-accent-text underline">
                {A.agreePrivacy}
              </a>
            </span>
          }
        />
        {errorBox}
        {submit(A.signupConfirm)}
      </form>
    );
  }

  /* ══════════════════════════════════════════════════════════════════
     **ولا عنوانَ في شاشة الدخول — العلامةُ وحدَها**
     ══════════════════════════════════════════════════════════════════

     (أمرُ المالك ٢٠٢٦-٠٨-٠٦: «النصوصَ احذفها، بلا داعٍ أصلاً — والأيقونةُ
      وحدَها بالشاشات الخمس».)

     # ما كان

     كلُّ واجهةٍ تمرّر عنوانَها: «بوابة المتجر» · «لوحة المندوب» · «بوابة
     السائق» · «تسجيل الدخول إلى اللوحة». **فخمسُ شاشاتٍ تبدو خمساً وهي
     واحدة** — ومن فتح اثنتين ظنّهما منصّتين.

     # ولماذا لا يُوحَّد بل يُحذف

     **العنوانُ كان يقول ما تقوله الشاشةُ تحته**: حقلُ هاتفٍ وكلمةُ مرورٍ وزرٌّ
     مكتوبٌ عليه «تسجيل الدخول». **وسطرٌ يشرح ما يُرى تأخيرٌ لمن جاء ليدخل.**

     **والعلامةُ تقول لمن هذه الشاشة** — وهي فوقه أصلاً.

     # ويبقى للاستعادة والحساب الجديد

     **لأنّهما شاشتان أخريان**: من ضغط «نسيت كلمة المرور» يحتاج أن يُقال له
     أين صار. **والحذفُ هنا لأنّ العنوانَ زائد، لا لأنّ العناوينَ ممنوعة.** */
  const heads: Record<Mode, { title?: string; subtitle?: string }> = {
    password: {},
    otp: {},
    reset: { title: A.resetTitle, subtitle: A.resetSubtitle },
    // **ولا وصفَ تحت «حساب جديد»** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «حساب زبون
    // — للتسوّق والطلب في الرقة: احذفها»). **والعنوانُ يقول ما يقوله الوصف.**
    // **والاستعادةُ وحدَها تُبقيه**: «أدخل رقمك ليصلك رمزٌ ثمّ اختر كلمةً
    // جديدة» **خطوتان غيرُ بديهيّتين** — والباقي حقلٌ وزرّ.
    signup: { title: A.signupTitle },
  };
  const head = heads[mode];
  const isAuxMode = mode === "reset" || mode === "signup";

  return (
    /* ══════════════════════════════════════════════════════════════════
       **النموذجُ وحدَه — لا خلفيّةَ ولا جانبَ ترويجيّ**
       ══════════════════════════════════════════════════════════════════

       (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «شِل الخلفية خلف الفورم وأيضاً الكرت الذي
       على اليمين مع الكتابة، واترك فقط الفورم الخاص بتسجيل الدخول».)

       # ما حُذف

       **خلفيّةٌ بثلاث طبقات**: تدرّجٌ يعمّ الشاشة وهالتان مضبّبتان بثمانين
       بكسلاً. **وجانبٌ ترويجيٌّ** فيه موجةُ الفرات والشعارُ وثلاثُ ميزاتٍ
       بأيقوناتها — نحو ثمانين سطراً.

       # ولماذا كان ضرراً

       **من يفتح `/login` جاء ليدخل لا ليُقنَع.** والإقناعُ وقع قبلَه —
       بالصفحة الأولى والتسوّق. **وثلاثُ ميزاتٍ بين عينيه وبين حقل الهاتف
       تأخيرٌ خالص.**

       **وشاشةُ دخولٍ بنصف عرضٍ ترويجيّ تُقرأ إعلاناً** لا باباً: العينُ تبدأ
       من الجانب الملوّن فتقرأ ما لا تريد، **ثمّ تعود إلى ما جاءت له.**

       **والخلفيّةُ المتدرّجةُ تُنافس النموذج**: ثلاثُ طبقاتٍ ملوّنةٍ خلف
       بطاقةٍ بيضاء **تسحب العينَ عمّا يُكتب فيها.**
       ══════════════════════════════════════════════════════════════════ */
    <div className="relative flex flex-1 items-center justify-center p-3 sm:p-6">
      {/* ══════════════════════════════════════════════════════════════
          **خلفيّةٌ تُرفع من الإعدادات — ولا تُحشر في الشيفرة**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «باك‌غراوند خلف صفحات تسجيل الدخول
           والحساب الجديد والاستعادة، بخيارٍ بالإعدادات أرفع الصورة وأغيّرها
           إيمت ما بدّي».)

          **وطبقةٌ داكنةٌ فوقها لا تحتها**: صورةٌ فاتحةٌ تبتلع النصَّ الأبيضَ
          في البطاقة وحولَها، **وأيُّ صورةٍ يرفعها المالكُ غداً لا تُكسر
          الشاشة.**

          **و`fixed` لا `absolute`**: البطاقةُ تطول في وضع التسجيل، **وخلفيّةٌ
          تتبع الطولَ تنقطع عند حافّة المحتوى.** */}
      {authBg && (
        <>
          {/* @single-child — بقيت الشظيّةُ لأنّ التعليقَ ولدٌ ثانٍ في JSX.
              **والرسمُ كلُّه في `theme.css`** (`auth-bg-image`) — **الصورةُ
              وحجابُها معاً.** كانت هنا أصنافاً مبعثرةً وطبقةً ثانيةً مخبوءةً
              في `style`، **وما كان في `style` لا يراه حارسُ المركزيّة** فيدرج
              خارجَ الثيم صامتاً. **ولم يبقَ هنا إلّا ما يأتي من الإعدادات:
              أيُّ صورةٍ وكم شدّةُ حجابها.** */}
          <div
            aria-hidden
            className="auth-bg-image"
            style={
              {
                "--auth-bg": `url(${authBg})`,
                "--auth-bg-dim": authBgDim / 100,
              } as React.CSSProperties
            }
          />
        </>
      )}
      {/* **وعرضٌ يكفي حقلاً واحداً** — كان خمسةً ونصفاً لأنّ نصفَه كان دعاية.
          **ونموذجٌ ممدودٌ إلى ألفٍ يُقرأ صفحةً لا بطاقة.** */}
      <div className="w-full max-w-md">
        {/* **وبطاقةٌ زجاجيّةٌ تأخذ لونَها ممّا تحتها.**

            (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «فورم تسجيل الدخول والحساب الجديد
             والاستعادة والفوتر والتوب بار زجاجيّ شفّاف يأخذ اللون من
             الخلفيّة».)

            **وكانت `bg-surface` وحدَها بلا تضبيب** — فتُقرأ لوحاً داكناً
            شبهَ شفّاف، **وتظهر حوافُّ التدرّج داخلَها فتُنافس الحقول.**

            **و`surface-lit` هي الزجاجُ المركزيّ**: تضبيبُ ما وراءها، وخيطُ
            ضوءٍ أعلاها، وثقلٌ في قاعها. **ولا يُكتب تضبيبٌ ثانٍ هنا** —
            وإلّا صار لكلّ سطحٍ زجاجُه. */}
        <div className="overflow-hidden surface elev-2">
          {/* ---------- جانب النموذج ---------- */}
          <div className="p-6 sm:p-8 lg:p-10">
            {/* **والبطاقةُ تُقرأ من محورها**: العلامةُ فوق العنوان فوق الوصف
                — **ثلاثةٌ على محورٍ واحدٍ تُقرأ سُلَّماً**، وواحدةٌ في الطرف
                تجرّ العينَ إلى زاويةٍ ثمّ تعيدها. (قرارُ المالك ٢٠٢٦-٠٨-٠٦:
                «اجعل اللوغو بالوسط محاذاة».) */}
            <div className="mb-6 text-center">
              {/* **علامةُ المنصة أعلى البطاقة.**

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «وأيضاً أعلى لوحة تسجيل الدخول
                   والخروج واستعادة كلمة المرور».)

                  **وشاشةُ الدخول أوّلُ ما يُرى قبل أن يكون هناك حساب** —
                  وبطاقةٌ بيضاءُ بحقلَي هاتفٍ وكلمةِ مرورٍ بلا علامةٍ **لا
                  تقول لمن هي.** ومن وصلها من رابطٍ لا يعرف أين وقع.

                  **ومن الإعدادات لا من المعجم**: شعارٌ إن رُفع وإلّا أوّلُ
                  حرفٍ من الاسم المضبوط. */}
              <BrandMark size={48} rounded="card" className="mx-auto mb-4" />
              {head.title && (
                <h1 className="text-xl font-bold tracking-tight text-ink">{head.title}</h1>
              )}
              {/* **ولا سطرَ وصفٍ في شاشة الدخول** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦:
                  «أدخل رقمك — وسننقلك إلى مكانك حسب دورك: احذف هذه العبارة»).

                  **وكان يشرح ما يفعله النموذجُ تحته**: حقلُ هاتفٍ وكلمةُ مرورٍ
                  وزرٌّ اسمُه «تسجيل الدخول». **وشرحُ ما يُرى تأخيرٌ لمن جاء
                  ليدخل.**

                  **وحُذف الوسيطُ لا أُخفي به**: خمسةُ تطبيقاتٍ كانت تمرّره
                  وخمسةُ نصوصٍ تُترجَم له — **ونصٌّ يُراجَع ولا يُرسَم دَينٌ
                  صامت.** ويبقى الوصفُ للاستعادة والحساب الجديد **لأنّهما
                  يشرحان خطوةً غيرَ بديهيّة**، ونصُّهما من داخل البطاقة. */}
              {head.subtitle && (
                <p className="mt-1.5 text-sm leading-relaxed text-ink-muted">{head.subtitle}</p>
              )}
            </div>

            {/* مبدّل طريقة الدخول — مؤشر منزلق بخصائص منطقية (يعمل RTL وLTR) */}
            {allow === "both" && !isAuxMode && (
              <div role="group" className="relative mb-6 flex rounded-control bg-field p-1">
                <span
                  aria-hidden
                  className="absolute inset-y-1 rounded-[7px] bg-surface elev-1 transition-[inset-inline-start] duration-200"
                  style={{
                    insetInlineStart: mode === "password" ? "0.25rem" : "calc(50% + 0.125rem)",
                    width: "calc(50% - 0.375rem)",
                  }}
                />
                {(["password", "otp"] as const).map((mo) => (
                  <button
                    key={mo}
                    type="button"
                    onClick={() => go(mo)}
                    className={`relative z-10 flex flex-1 items-center justify-center gap-1.5 rounded-[7px] px-3 py-2 text-sm transition-colors ${
                      mode === mo ? "font-medium text-primary-dark" : "text-ink-muted hover:text-ink"
                    }`}
                  >
                    {mo === "password" ? <IconLock size={15} /> : <IconKey size={15} />}
                    {mo === "password" ? A.loginWithPassword : A.loginWithOtp}
                  </button>
                ))}
              </div>
            )}

            {body()}

            {isAuxMode ? (
              <div className="mt-6 border-t border-line-soft pt-4 text-center">
                {linkBtn(A.backToLogin, () => go("password"), <IconPrev size={15} />)}
              </div>
            ) : signup ? (
              <div className="mt-6 border-t border-line-soft pt-4 text-center text-sm text-ink-muted">
                {A.noAccount}{" "}
                {linkBtn(A.createAccount, () => go("signup"), <IconSignup size={15} />)}
              </div>
            ) : null}

            {footer && <div className="mt-4 text-center">{footer}</div>}

            {/* ══════════════════════════════════════════════════════════
                **نافذةُ الإتمام — تُقرّ ثمّ تسوق إلى الدخول**
                ══════════════════════════════════════════════════════

                (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «تظهر نافذةٌ تفيد أنّه تمّ إنشاء
                 الحساب بنجاح، تنتقل إلى شاشة تسجيل الدخول ليدخل بشكلٍ
                 يدويّ».)

                **ولا تُغلق بالنقر خارجَها** (`onClose` تسوق إلى الدخول):
                الإتمامُ خبرٌ يجب أن يُقرأ، **ونافذةٌ تُغلق سهواً تترك صاحبَها
                لا يدري أنجح أم لا.** */}
            {created && (
              <Modal
                open
                onClose={() => go("password")}
                title={mode === "reset" ? A.resetDoneTitle : A.signupDoneTitle}
              >
                <div className="text-center">
                  <IconSuccess size={44} className="mx-auto mb-3 text-success" />
                  <p className="text-sm leading-relaxed text-ink-muted">
                    {mode === "reset" ? A.resetDoneBody : A.signupDoneBody}
                  </p>
                </div>
                <Button onClick={() => go("password")} className="mt-4 w-full py-3">
                  {mode === "reset" ? A.resetDoneGo : A.signupDoneGo}
                </Button>
              </Modal>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
