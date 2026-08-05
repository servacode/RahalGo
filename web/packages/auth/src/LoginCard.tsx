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

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Checkbox,
  OtpInput,
  PasswordMeter,
  IconPhone,
  IconLock,
  IconUser,
  IconKey,
  IconSignup,
  IconPrev,
  IconWhatsApp,
  IconStatus,
  IconWallet,
  IconCheck,
  IconWarning,
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

/** نقاط الثقة في جانب العلامة — تُبنى من المعجم لا من نص مكتوب. */
const HERO_POINTS = [
  { icon: IconWhatsApp, title: A.hero.secureTitle, body: A.hero.secureBody },
  { icon: IconStatus, title: A.hero.trackTitle, body: A.hero.trackBody },
  { icon: IconWallet, title: A.hero.walletTitle, body: A.hero.walletBody },
] as const;

export function LoginCard({
  title,
  subtitle,
  methods = "both",
  onSuccess,
  footer,
  referral = "",
}: {
  title: string;
  subtitle?: string;
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
}) {
  const [mode, setMode] = useState<Mode>(methods === "otp" ? "otp" : "password");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [code, setCode] = useState("");
  const [remember, setRemember] = useState(true);
  /** هل أُرسل الرمز في الوضع الحالي؟ (يخصّ otp/reset/signup) */
  const [sent, setSent] = useState(false);
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
    setError("");
    setSent(false);
    setCode("");
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

  /** خانات الرمز — onSubmit يُمرَّر ليُرسل النموذج فور اكتمال الرقم السادس. */
  const codeField = (onDone?: () => void) => (
    <div className="space-y-2">
      <label className="block text-center text-sm font-medium text-ink">{A.otpTitle}</label>
      <OtpInput
        value={code}
        onChange={setCode}
        autoFocus
        boxLabel={A.otpBoxLabel}
        onComplete={() => onDone?.()}
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
    <p
      role="alert"
      className="flex items-start gap-2 rounded-control border border-danger/25 bg-danger/10 px-3 py-2 text-sm text-danger"
    >
      <IconWarning size={16} className="mt-0.5 shrink-0" />
      <span>{error}</span>
    </p>
  ) : null;

  const sentNote = (
    <div className="flex items-start gap-2.5 rounded-control border border-success/25 bg-success/10 px-3 py-2.5 text-sm text-success">
      <IconCheck size={16} className="mt-0.5 shrink-0" strokeWidth={3} />
      <div className="min-w-0">
        <p>
          {A.otpSentTo}{" "}
          <span dir="ltr" className="font-bold">
            {phone}
          </span>
        </p>
        <p className="mt-0.5 text-xs opacity-80">{A.otpSentDev}</p>
      </div>
    </div>
  );

  const submit = (label: string) => (
    <Button type="submit" disabled={busy} className="w-full py-3 text-base shadow-sm">
      {busy ? m.shared.loggingIn : label}
    </Button>
  );

  /** رابط نصّي ثانوي داخل البطاقة — شكل واحد لكل روابط التبديل. */
  const linkBtn = (label: string, onClick: () => void, icon?: ReactNode) => (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1 rounded-control text-sm font-medium text-primary transition-colors hover:text-primary-dark hover:underline"
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

  /**
   * مؤشر الخطوات — التدفّقات ذات المرحلتين كانت تنقل المستخدم بلا إشعار بموقعه.
   * يظهر أين هو وكم بقي، فيقلّ التسرّب عند شاشة الرمز.
   */
  const steps = (labels: [string, string]) => (
    <div className="mb-5">
      <div className="mb-2 flex items-center gap-2">
        {labels.map((label, i) => {
          const done = sent && i === 0;
          const current = sent ? i === 1 : i === 0;
          return (
            <div key={label} className="flex flex-1 items-center gap-2">
              <span
                className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-badge text-xs font-bold transition-colors ${
                  done
                    ? "bg-success text-white"
                    : current
                      ? "bg-primary text-white"
                      : "border border-line text-ink-muted"
                }`}
              >
                {done ? <IconCheck size={13} strokeWidth={3} /> : i + 1}
              </span>
              <span
                className={`truncate text-xs ${current ? "font-medium text-ink" : "text-ink-muted"}`}
              >
                {label}
              </span>
              {i === 0 && <span className={`h-px flex-1 ${sent ? "bg-success" : "bg-line"}`} />}
            </div>
          );
        })}
      </div>
    </div>
  );

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
      const verify = run(async () => enter((await authApi.verifyOtp(phone, code)) as never));
      return (
        <>
          {steps([A.steps.phone, A.steps.verify])}
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
            <form onSubmit={verify} className="space-y-4">
              {sentNote}
              {codeField(() => void verify())}
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
      return (
        <>
          {steps([A.steps.phone, A.steps.newPassword])}
          {!sent ? (
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
          ) : (
            <form
              onSubmit={run(async () =>
                enter((await authApi.confirmReset(phone, code, password)) as never),
              )}
              className="space-y-4"
            >
              {sentNote}
              {codeField()}
              {passwordField("new-password", A.newPassword, "new-password", true)}
              <p className="text-xs text-ink-muted">{A.passwordHint}</p>
              {errorBox}
              {submit(A.resetConfirm)}
            </form>
          )}
        </>
      );
    }

    // إنشاء حساب — زبون فقط
    return (
      <>
        {steps([A.steps.phone, A.steps.profile])}
        {!sent ? (
          <form
            onSubmit={run(async () => {
              await authApi.requestSignup(phone);
              setSent(true);
              setBusy(false);
            })}
            className="space-y-4"
          >
            {phoneField}
            <p className="rounded-control border border-line bg-page px-3 py-2 text-xs leading-relaxed text-ink-muted">
              {A.staffOnly}
            </p>
            {errorBox}
            {submit(A.signupSend)}
          </form>
        ) : (
          <form
            onSubmit={run(async () =>
              enter((await authApi.confirmSignup(phone, code, fullName, password, referral)) as never),
            )}
            className="space-y-4"
          >
            {sentNote}
            {codeField()}
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
            <p className="text-xs text-ink-muted">{A.passwordHint}</p>
            {errorBox}
            {submit(A.signupConfirm)}
          </form>
        )}
      </>
    );
  }

  const heads: Record<Mode, { title: string; subtitle?: string }> = {
    password: { title, subtitle },
    otp: { title, subtitle },
    reset: { title: A.resetTitle, subtitle: A.resetSubtitle },
    signup: { title: A.signupTitle, subtitle: A.signupSubtitle },
  };
  const head = heads[mode];
  const isAuxMode = mode === "reset" || mode === "signup";

  return (
    <div className="relative flex flex-1 items-center justify-center overflow-hidden p-3 sm:p-6">
      {/* خلفية العلامة: تدرّج ناعم وهالتان تعطيان عمقاً بلا ضجيج */}
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-b from-primary-light/60 via-page to-page" />
      <div className="pointer-events-none absolute -top-32 start-1/4 -z-10 h-80 w-80 rounded-badge bg-primary/10 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-32 end-1/4 -z-10 h-80 w-80 rounded-badge bg-accent/10 blur-3xl" />

      <div className="w-full max-w-5xl">
        <div className="grid overflow-hidden rounded-card border border-line/70 bg-surface shadow-card lg:grid-cols-[1.05fr_1fr]">
          {/* ---------- جانب العلامة ---------- */}
          <aside className="relative overflow-hidden bg-gradient-to-br from-primary-dark via-primary to-primary-dark p-6 text-white sm:p-8 lg:p-10">
            {/* موجة الفرات — رمز العلامة (BRAND.md): النهر يعبر الصحراء */}
            <svg
              viewBox="0 0 400 300"
              preserveAspectRatio="none"
              aria-hidden
              className="pointer-events-none absolute inset-0 h-full w-full opacity-[0.18]"
            >
              <path d="M-20 210 C 80 150, 140 250, 240 190 S 380 130, 440 170" fill="none" stroke="currentColor" strokeWidth="2.5" />
              <path d="M-20 240 C 90 185, 150 280, 250 220 S 390 165, 440 200" fill="none" stroke="currentColor" strokeWidth="1.5" />
              <path d="M-20 180 C 70 120, 130 215, 230 155 S 370 100, 440 140" fill="none" stroke="currentColor" strokeWidth="1" />
            </svg>
            <div className="pointer-events-none absolute -end-16 -top-16 h-56 w-56 rounded-badge bg-accent/20 blur-3xl" />

            <div className="relative flex h-full flex-col">
              <div className="flex items-center gap-3">
                <span className="flex h-12 w-12 items-center justify-center rounded-card bg-white/15 text-xl font-bold ring-1 ring-white/25 backdrop-blur">
                  {m.terms.brandInitial}
                </span>
                <span className="text-lg font-bold tracking-tight">{m.common.appName}</span>
              </div>

              <p className="mt-5 max-w-sm text-sm leading-relaxed text-white/85 lg:mt-8 lg:text-base">
                {A.hero.tagline}
              </p>

              {/* نقاط الثقة — تظهر على الشاشات الواسعة فقط كي لا تُطيل الجوال */}
              <ul className="mt-8 hidden space-y-5 lg:block">
                {HERO_POINTS.map((p) => {
                  const Icon = p.icon;
                  return (
                    <li key={p.title} className="flex gap-3">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-white/10 ring-1 ring-white/15">
                        <Icon size={17} />
                      </span>
                      <div className="min-w-0">
                        <p className="text-sm font-bold">{p.title}</p>
                        <p className="mt-0.5 text-xs leading-relaxed text-white/70">{p.body}</p>
                      </div>
                    </li>
                  );
                })}
              </ul>

              <div className="mt-auto hidden pt-8 lg:block">
                <p className="text-xs text-white/50">{m.site.appDescription}</p>
              </div>
            </div>
          </aside>

          {/* ---------- جانب النموذج ---------- */}
          <div className="p-6 sm:p-8 lg:p-10">
            <div className="mb-6">
              <h1 className="text-xl font-bold tracking-tight text-ink">{head.title}</h1>
              {head.subtitle && (
                <p className="mt-1.5 text-sm leading-relaxed text-ink-muted">{head.subtitle}</p>
              )}
            </div>

            {/* مبدّل طريقة الدخول — مؤشر منزلق بخصائص منطقية (يعمل RTL وLTR) */}
            {methods === "both" && !isAuxMode && (
              <div role="group" className="relative mb-6 flex rounded-control bg-page p-1">
                <span
                  aria-hidden
                  className="absolute inset-y-1 rounded-[7px] bg-surface shadow-sm transition-[inset-inline-start] duration-200"
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
              <div className="mt-6 border-t border-line pt-4 text-center">
                {linkBtn(A.backToLogin, () => go("password"), <IconPrev size={15} />)}
              </div>
            ) : (
              <div className="mt-6 border-t border-line pt-4 text-center text-sm text-ink-muted">
                {A.noAccount}{" "}
                {linkBtn(A.createAccount, () => go("signup"), <IconSignup size={15} />)}
              </div>
            )}

            {footer && <div className="mt-4 text-center">{footer}</div>}
          </div>
        </div>
      </div>
    </div>
  );
}
