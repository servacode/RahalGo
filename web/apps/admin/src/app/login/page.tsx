"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Input, Button, IconPhone, IconLock } from "@rahalgo/ui";
import { ApiError, authApi, tokenStore } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

type Mode = "password" | "otp";

export default function LoginPage() {
  const { login, logout } = useAuth();
  const router = useRouter();
  const [mode, setMode] = useState<Mode>("password");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  function fail(err: unknown) {
    setError(err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal);
  }

  function enterPanel(user: Parameters<typeof canAccessPanel>[0]) {
    if (!canAccessPanel(user)) {
      logout();
      setError(m.admin.notAllowed);
      return;
    }
    router.replace("/dashboard");
  }

  async function onPasswordSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      enterPanel(await login(phone, password));
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  async function onSendOtp(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setNotice("");
    setBusy(true);
    try {
      await authApi.requestOtp(phone);
      setOtpSent(true);
      setNotice(m.auth.otpSentWhatsapp);
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  async function onVerifyOtp(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const result = await authApi.verifyOtp(phone, code);
      tokenStore.set(result.tokens);
      // إعادة تحميل الحالة عبر الدخول المباشر للوحة (AuthProvider سيقرأ /me)
      window.location.href = canAccessPanel(result.user) ? "/dashboard" : "/login";
      if (!canAccessPanel(result.user)) {
        tokenStore.clear();
        setError(m.admin.notAllowed);
      }
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-sm rounded-card border border-line bg-surface p-8 shadow-sm">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-card bg-primary text-2xl font-bold text-white">
            ر
          </div>
          <h1 className="text-xl font-bold">{m.admin.loginTitle}</h1>
          <p className="mt-1 text-sm text-ink-muted">{m.admin.loginSubtitle}</p>
        </div>

        {/* مبدّل طريقة الدخول */}
        <div role="group" className="mb-6 flex rounded-control border border-line bg-page p-1">
          {(["password", "otp"] as const).map((mo) => (
            <button
              key={mo}
              type="button"
              onClick={() => {
                setMode(mo);
                setError("");
                setNotice("");
                setOtpSent(false);
              }}
              className={`flex-1 rounded-control px-3 py-1.5 text-sm transition-colors ${
                mode === mo
                  ? "bg-surface font-medium text-primary-dark shadow-sm"
                  : "text-ink-muted"
              }`}
            >
              {mo === "password" ? m.auth.loginWithPassword : m.auth.loginWithOtp}
            </button>
          ))}
        </div>

        {mode === "password" ? (
          <form onSubmit={onPasswordSubmit} className="space-y-4">
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
            <Input
              id="password"
              label={m.auth.password}
              icon={<IconLock />}
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            {error && (
              <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
            )}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.admin.loggingIn : m.admin.loginButton}
            </Button>
          </form>
        ) : !otpSent ? (
          <form onSubmit={onSendOtp} className="space-y-4">
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
            {error && (
              <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
            )}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.common.loading : m.auth.sendOtp}
            </Button>
          </form>
        ) : (
          <form onSubmit={onVerifyOtp} className="space-y-4">
            {notice && (
              <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">
                {notice}
                <br />
                <span className="text-xs">{m.auth.otpSentDev}</span>
              </p>
            )}
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
            {error && (
              <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
            )}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.admin.loggingIn : m.admin.loginButton}
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
      </div>
    </main>
  );
}
