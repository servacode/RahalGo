"use client";

/** حسابي: تغيير كلمة المرور الذاتي (بإثبات الحالية) — لكل موظفي اللوحة والأدمن. */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, FormSection, IconLock, IconUser } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import RoleBadge from "@/components/RoleBadge";

const m = getMessages(defaultLocale);
const A = m.admin.myAccount;

export default function MyAccountPage() {
  const { user } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (next !== confirm) {
      setError(m.errors.password_mismatch);
      return;
    }
    setBusy(true);
    setError("");
    setDone(false);
    try {
      await api("/api/v1/auth/password", {
        method: "POST",
        body: JSON.stringify({ password: next, current_password: current }),
      });
      setDone(true);
      setCurrent("");
      setNext("");
      setConfirm("");
    } catch (err) {
      if (err instanceof ApiError && err.body.code === "invalid_credentials") {
        setError(A.wrongCurrent);
      } else if (err instanceof ApiError && err.body.code === "weak_password") {
        setError(m.errors.weak_password);
      } else {
        setError(m.errors.internal);
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-lg">
      <h1 className="mb-1 flex items-center gap-2 text-xl font-bold">
        <IconUser className="text-primary" />
        {A.title}
      </h1>
      <p className="mb-5 text-sm text-ink-muted">{A.subtitle}</p>

      <div className="mb-5 flex items-center gap-3 rounded-card border border-line bg-surface p-4">
        <span className="flex h-12 w-12 items-center justify-center rounded-card bg-primary-light text-lg font-bold text-primary-dark">
          {(user?.full_name || "؟").charAt(0)}
        </span>
        <div>
          <p className="font-bold">{user?.full_name || "—"}</p>
          <p className="text-xs text-ink-muted" dir="ltr">
            {user?.phone}
          </p>
          <div className="mt-1 flex gap-1">
            {user?.roles.map((r) => (
              <RoleBadge key={r} role={r} />
            ))}
          </div>
        </div>
      </div>

      <FormSection title={A.changePassword} icon={<IconLock />}>
        <form onSubmit={submit} className="space-y-4">
          <Input
            id="cur-pw"
            label={A.currentPassword}
            type="password"
            dir="ltr"
            required
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
          <div className="grid grid-cols-2 gap-3">
            <Input
              id="new-pw"
              label={A.newPassword}
              type="password"
              dir="ltr"
              required
              minLength={8}
              value={next}
              onChange={(e) => setNext(e.target.value)}
            />
            <Input
              id="new-pw2"
              label={A.confirmNew}
              type="password"
              dir="ltr"
              required
              minLength={8}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />
          </div>
          {done && (
            <p className="rounded-control bg-success/10 px-3 py-2 text-sm text-success">
              {A.changed}
            </p>
          )}
          {error && (
            <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
          )}
          <Button type="submit" disabled={busy} className="w-full">
            {m.common.save}
          </Button>
        </form>
      </FormSection>
    </div>
  );
}
