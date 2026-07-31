"use client";

/**
 * شاشة تسجيل الدخول المركزية — نموذج واحد للجميع، وكل شخص يدخل حسب دوره.
 * لا تُبنى شاشة دخول خاصة بأي لوحة؛ الاختلاف الوحيد نصوص العنوان والوجهة.
 */

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, IconPhone, IconLock } from "@rahalgo/ui";
import { authApi, tokenStore, ApiError, type AuthUser } from "./client";

const m = getMessages(defaultLocale);
type Mode = "password" | "otp";

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
  const [code, setCode] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function enter(result: { user: AuthUser; tokens: never }) {
    tokenStore.set(result.tokens);
    try {
      await onSuccess(result.user);
    } catch {
      setBusy(false);
    }
  }

  async function onPassword(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await enter((await authApi.loginPassword(phone, password)) as never);
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  async function onSendOtp(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await authApi.requestOtp(phone);
      setOtpSent(true);
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  async function onVerifyOtp(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await enter((await authApi.verifyOtp(phone, code)) as never);
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  const phoneField = (
    <Input
      id="phone"
      label={m.auth.phone}
      icon={<IconPhone />}
      dir="ltr"
      inputMode="tel"
      required
      value={phone}
      onChange={(e) => setPhone(e.target.value)}
      className="text-end"
      placeholder="09xxxxxxxx"
    />
  );
  const errorBox = error ? (
    <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
  ) : null;

  return (
    <div className="flex flex-1 items-center justify-center p-4">
      <div className="w-full max-w-sm rounded-card border border-line bg-surface p-8 shadow-sm">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-card bg-primary text-2xl font-bold text-white">
            {m.terms.brandInitial}
          </div>
          <h1 className="text-xl font-bold">{title}</h1>
          {subtitle && <p className="mt-1 text-sm text-ink-muted">{subtitle}</p>}
        </div>

        {methods === "both" && (
          <div role="group" className="mb-6 flex rounded-control border border-line bg-page p-1">
            {(["password", "otp"] as const).map((mo) => (
              <button
                key={mo}
                type="button"
                onClick={() => {
                  setMode(mo);
                  setError("");
                  setOtpSent(false);
                }}
                className={`flex-1 rounded-control px-3 py-1.5 text-sm transition-colors ${
                  mode === mo ? "bg-surface font-medium text-primary-dark shadow-sm" : "text-ink-muted"
                }`}
              >
                {mo === "password" ? m.auth.loginWithPassword : m.auth.loginWithOtp}
              </button>
            ))}
          </div>
        )}

        {mode === "password" && methods !== "otp" ? (
          <form onSubmit={onPassword} className="space-y-4">
            {phoneField}
            <Input
              id="password"
              label={m.auth.password}
              icon={<IconLock />}
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            {errorBox}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.shared.loggingIn : m.auth.login}
            </Button>
          </form>
        ) : !otpSent ? (
          <form onSubmit={onSendOtp} className="space-y-4">
            {phoneField}
            {errorBox}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.common.loading : m.auth.sendOtp}
            </Button>
          </form>
        ) : (
          <form onSubmit={onVerifyOtp} className="space-y-4">
            <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">
              {m.auth.otpSentWhatsapp}
              <br />
              <span className="text-xs">{m.auth.otpSentDev}</span>
            </p>
            <Input
              id="otp-code"
              label={m.auth.otpTitle}
              dir="ltr"
              inputMode="numeric"
              required
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className="text-center font-mono text-lg tracking-[0.5em]"
              placeholder="••••••"
              maxLength={6}
            />
            {errorBox}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.shared.loggingIn : m.auth.login}
            </Button>
            <button
              type="button"
              onClick={() => {
                setOtpSent(false);
                setCode("");
              }}
              className="w-full text-center text-sm text-ink-muted hover:text-primary"
            >
              {m.auth.changePhone}
            </button>
          </form>
        )}

        {footer && <div className="mt-4">{footer}</div>}
      </div>
    </div>
  );
}
