"use client";

/**
 * بوابة تبديل كلمة المرور المؤقتة — مركزية لكل اللوحات.
 *
 * من يعرف كلمة مرورك ليس أنت: المندوب يسلّم صاحب المتجر كلمة مرور، والإدارة
 * تعيد تعيين كلمة مرور حساب. في الحالتين تُعلَّم الكلمة **مؤقتة** في الخادم،
 * وهذه البوابة تحجب كل شيء حتى يضبط صاحب الحساب كلمة مروره بنفسه.
 *
 * تُلفّ حول محتوى أي بوابة: <PasswordGate>{children}</PasswordGate>
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, Button, Input, PasswordMeter, IconLock, IconWarning, IconCheck } from "@rahalgo/ui";
import { useAuth } from "./provider";
import { api, authApi } from "./client";
import { errText } from "./LoginCard";

const m = getMessages(defaultLocale);
const A = m.auth;
const G = A.mustChange;

export function PasswordGate({ children }: { children: React.ReactNode }) {
  const { user, loading, setUser } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  if (loading || !user?.must_change_password) return <>{children}</>;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (next.length < 8) return setError(m.errors.weak_password);
    if (next !== confirm) return setError(m.errors.password_mismatch);
    setBusy(true);
    try {
      await api("/api/v1/auth/password", {
        method: "POST",
        body: JSON.stringify({ password: next, current_password: current }),
      });
      // نعيد قراءة الحساب: الخادم رفع علامة الإجبار فتُفتح اللوحة تلقائياً
      setUser(await authApi.me());
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-4">
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-b from-primary-light/60 via-page to-page" />
      <div className="w-full max-w-md rounded-card border border-line bg-surface p-7 elev-1">
        <div className="mb-5 flex items-start gap-3">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-card bg-warning/15 text-warning">
            <IconWarning size={22} />
          </span>
          <div>
            <h1 className="text-lg font-bold text-ink">{G.title}</h1>
            <p className="mt-1 text-sm leading-relaxed text-ink-muted">{G.hint}</p>
          </div>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <Input
            id="cur-pw"
            label={G.temporary}
            icon={<IconLock />}
            type="password"
            autoComplete="current-password"
            required
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
          <div className="space-y-2">
            <Input
              id="new-pw"
              label={A.newPassword}
              icon={<IconLock />}
              type="password"
              autoComplete="new-password"
              required
              value={next}
              onChange={(e) => setNext(e.target.value)}
            />
            <PasswordMeter
              value={next}
              labels={[A.strength.weak, A.strength.fair, A.strength.strong]}
            />
          </div>
          <Input
            id="conf-pw"
            label={m.shared.account.confirmPassword}
            icon={<IconLock />}
            type="password"
            autoComplete="new-password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{A.passwordHint}</p>

          {error && (
            <Alert>{error}</Alert>
          )}

          <Button type="submit" disabled={busy} className="flex w-full items-center justify-center gap-2 py-3 text-base">
            <IconCheck size={17} strokeWidth={3} />
            {busy ? m.common.loading : G.submit}
          </Button>
        </form>
      </div>
    </main>
  );
}
