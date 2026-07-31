"use client";

/**
 * بطاقة الدخول المركزية — نموذج واحد للجميع، وكل شخص يدخل حسب دوره.
 * تحمل أربعة أوضاع في بطاقة واحدة (دخول / رمز تحقق / استعادة / حساب جديد)
 * فلا تُبنى صفحة منفصلة لكل تدفّق، ولا تفقد البطاقة سياقها بانتقال.
 */

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Checkbox,
  IconPhone,
  IconLock,
  IconUser,
  IconKey,
  IconSignup,
  IconPrev,
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

export function LoginCard({
  title,
  subtitle,
  methods = "both",
  onSuccess,
  footer,
}: {
  title: string;
  subtitle?: string;
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
    return async (e: React.FormEvent) => {
      e.preventDefault();
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

  const codeField = (
    <Input
      id="otp-code"
      label={A.otpTitle}
      icon={<IconKey />}
      dir="ltr"
      inputMode="numeric"
      autoComplete="one-time-code"
      required
      autoFocus
      value={code}
      onChange={(e) => setCode(e.target.value)}
      className="text-center font-mono text-lg tracking-[0.4em]"
      placeholder="••••••"
      maxLength={6}
    />
  );

  const passwordField = (id: string, label: string, autoComplete: string) => (
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
  );

  const errorBox = error ? (
    <p
      role="alert"
      className="rounded-control border border-danger/25 bg-danger/10 px-3 py-2 text-sm text-danger"
    >
      {error}
    </p>
  ) : null;

  const sentNote = (
    <p className="rounded-control border border-primary/20 bg-primary-light/60 px-3 py-2 text-sm text-primary-dark">
      {A.otpSentTo}{" "}
      <span dir="ltr" className="font-bold">
        {phone}
      </span>
      <br />
      <span className="text-xs opacity-80">{A.otpSentDev}</span>
    </p>
  );

  const submit = (label: string) => (
    <Button type="submit" disabled={busy} className="w-full py-2.5">
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
          <div className="flex items-center justify-between gap-2">
            {rememberBox("remember")}
            {linkBtn(A.forgotPassword, () => go("reset"))}
          </div>
          {errorBox}
          {submit(A.login)}
        </form>
      );
    }

    if (mode === "otp") {
      return !sent ? (
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
        <form
          onSubmit={run(async () => enter((await authApi.verifyOtp(phone, code)) as never))}
          className="space-y-4"
        >
          {sentNote}
          {codeField}
          {rememberBox("remember-otp")}
          {errorBox}
          {submit(A.login)}
          <div className="text-center">{linkBtn(A.changePhone, () => setSent(false))}</div>
        </form>
      );
    }

    if (mode === "reset") {
      return !sent ? (
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
          {codeField}
          {passwordField("new-password", A.newPassword, "new-password")}
          <p className="text-xs text-ink-muted">{A.passwordHint}</p>
          {errorBox}
          {submit(A.resetConfirm)}
        </form>
      );
    }

    // إنشاء حساب — زبون فقط
    return !sent ? (
      <form
        onSubmit={run(async () => {
          await authApi.requestSignup(phone);
          setSent(true);
          setBusy(false);
        })}
        className="space-y-4"
      >
        {phoneField}
        <p className="rounded-control bg-page px-3 py-2 text-xs leading-relaxed text-ink-muted">
          {A.staffOnly}
        </p>
        {errorBox}
        {submit(A.signupSend)}
      </form>
    ) : (
      <form
        onSubmit={run(async () =>
          enter((await authApi.confirmSignup(phone, code, fullName, password)) as never),
        )}
        className="space-y-4"
      >
        {sentNote}
        {codeField}
        <Input
          id="full-name"
          label={A.fullName}
          icon={<IconUser />}
          autoComplete="name"
          required
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
        />
        {passwordField("signup-password", A.password, "new-password")}
        <p className="text-xs text-ink-muted">{A.passwordHint}</p>
        {errorBox}
        {submit(A.signupConfirm)}
      </form>
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
    // خلفية موحّدة بلمسة العلامة: تدرّج ناعم من لون العلامة الفاتح إلى لون الصفحة،
    // فلا تبقى شاشة الدخول رمادية مسطّحة.
    <div className="relative flex flex-1 items-center justify-center overflow-hidden bg-gradient-to-b from-primary-light/50 via-page to-page p-4">
      {/* هالتان لونيتان خفيفتان تعطيان الخلفية عمقاً بلا ضجيج */}
      <div className="pointer-events-none absolute -top-24 start-1/4 h-72 w-72 rounded-full bg-primary/10 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-24 end-1/4 h-72 w-72 rounded-full bg-accent/10 blur-3xl" />

      <div className="relative w-full max-w-[25rem]">
        <div className="overflow-hidden rounded-card border border-line/80 bg-surface shadow-[0_1px_3px_rgba(16,24,40,.05),0_12px_40px_-16px_rgba(16,24,40,.2)]">
          {/* رأس البطاقة — شريط بلون العلامة يحمل الشعار، فتبدو مؤطّرة لا سادة */}
          <div className="flex flex-col items-center gap-3 bg-gradient-to-b from-primary to-primary-dark px-7 pb-6 pt-7 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-card bg-white/15 text-2xl font-bold text-white ring-1 ring-white/25 backdrop-blur">
              {m.terms.brandInitial}
            </div>
            <div>
              <h1 className="text-lg font-bold tracking-tight text-white">{head.title}</h1>
              {head.subtitle && (
                <p className="mx-auto mt-1 max-w-[20rem] text-sm leading-relaxed text-white/80">
                  {head.subtitle}
                </p>
              )}
            </div>
          </div>

          <div className="p-6 sm:p-7">
            {/* مبدّل طريقة الدخول — يظهر في وضع الدخول فقط */}
            {methods === "both" && !isAuxMode && (
              <div role="group" className="mb-6 flex rounded-control bg-page p-1">
                {(["password", "otp"] as const).map((mo) => (
                  <button
                    key={mo}
                    type="button"
                    onClick={() => go(mo)}
                    className={`flex-1 rounded-[7px] px-3 py-1.5 text-sm transition-all ${
                      mode === mo
                        ? "bg-surface font-medium text-primary-dark shadow-sm"
                        : "text-ink-muted hover:text-ink"
                    }`}
                  >
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
          </div>
        </div>

        {footer && <div className="mt-4 text-center">{footer}</div>}
      </div>
    </div>
  );
}
