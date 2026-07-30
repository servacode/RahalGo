"use client";

/** دخول لوحة المندوب — كلمة المرور افتراضياً، ورمز واتساب كبديل. */

import { Suspense, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Input, Button, IconPhone, IconLock } from "@rahalgo/ui";
import { authApi, tokenStore, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
type Mode = "password" | "otp";

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    const known = (m.errors as Record<string, string>)[key];
    if (known) return known;
    if (err.body.message_key === "auth.otpInvalid") return m.auth.otpInvalid;
  }
  return m.errors.internal;
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const { setUser } = useAuth();
  const [mode, setMode] = useState<Mode>("password");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  function enter(result: { user: { roles: string[] }; tokens: unknown }) {
    if (!result.user.roles.includes("sales")) {
      setError(m.rep.notAllowed);
      setBusy(false);
      return;
    }
    tokenStore.set(result.tokens as never);
    setUser(result.user as never);
    router.replace("/");
  }

  async function onPassword(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      enter(await authApi.loginPassword(phone, password));
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
      enter(await authApi.verifyOtp(phone, code));
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-page p-4">
      <div className="w-full max-w-sm rounded-card border border-line bg-surface p-8 shadow-sm">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-card bg-primary text-2xl font-bold text-white">
            ر
          </div>
          <h1 className="text-xl font-bold">{m.rep.loginTitle}</h1>
          <p className="mt-1 text-sm text-ink-muted">{m.rep.loginSubtitle}</p>
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

        {mode === "password" ? (
          <form onSubmit={onPassword} className="space-y-4">
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
              {busy ? m.common.loading : m.auth.login}
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
            {error && (
              <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
            )}
            <Button type="submit" disabled={busy} className="w-full py-2.5">
              {busy ? m.common.loading : m.auth.login}
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
