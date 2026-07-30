"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Input, IconPhone, IconLock } from "@rahalgo/ui";
import { ApiError } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

/** يحوّل message_key القادم من الخادم إلى نص مترجم من الحزمة المركزية */
function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

export default function LoginPage() {
  const { login, logout } = useAuth();
  const router = useRouter();
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const user = await login(phone, password);
      if (!canAccessPanel(user)) {
        logout();
        setError(m.admin.notAllowed);
        return;
      }
      router.replace("/dashboard");
    } catch (err) {
      setError(err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-sm rounded-card border border-line bg-surface p-8 shadow-sm">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-card bg-primary text-2xl font-bold text-white">
            ر
          </div>
          <h1 className="text-xl font-bold">{m.admin.loginTitle}</h1>
          <p className="mt-1 text-sm text-ink-muted">{m.admin.loginSubtitle}</p>
        </div>

        <form onSubmit={onSubmit} className="space-y-4">
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

          <button
            type="submit"
            disabled={busy}
            className="w-full rounded-control bg-primary py-2.5 font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-60"
          >
            {busy ? m.admin.loggingIn : m.admin.loginButton}
          </button>
        </form>
      </div>
    </main>
  );
}
