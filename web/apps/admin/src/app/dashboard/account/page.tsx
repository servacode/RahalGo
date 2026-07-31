"use client";

/** حسابي: تغيير كلمة المرور الذاتي (بإثبات الحالية) — لكل موظفي اللوحة والأدمن. */

import { useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, FormSection, IconLock, IconUser, IconStatus, IconPhone } from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import RoleBadge from "@/components/RoleBadge";

const m = getMessages(defaultLocale);
const A = m.admin.myAccount;
const P = m.site.account; // مفاتيح تغيير رقم الهاتف المشتركة

interface Login {
  action: string;
  ip: string;
  created_at: string;
}
const LOGIN_LABELS: Record<string, string> = {
  "auth.otp_login": "دخول برمز تحقق",
  "auth.password_login": "دخول بكلمة المرور",
  "auth.password_failed": "محاولة فاشلة",
};

export default function MyAccountPage() {
  const { user } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");
  const [logins, setLogins] = useState<Login[]>([]);
  const [avatar, setAvatar] = useState<string | null>(null);

  const [newPhone, setNewPhone] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [phoneCode, setPhoneCode] = useState("");
  const [phoneBusy, setPhoneBusy] = useState(false);
  const [phoneMsg, setPhoneMsg] = useState("");

  async function reqPhone(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPhoneMsg("");
    setPhoneBusy(true);
    try {
      await api("/api/v1/auth/phone/request", { method: "POST", body: JSON.stringify({ phone: newPhone }) });
      setOtpSent(true);
    } catch {
      setError(m.errors.internal);
    } finally {
      setPhoneBusy(false);
    }
  }

  async function confirmPhone(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setPhoneMsg("");
    setPhoneBusy(true);
    try {
      await api("/api/v1/auth/phone/confirm", {
        method: "POST",
        body: JSON.stringify({ phone: newPhone, code: phoneCode }),
      });
      setPhoneMsg(P.phoneSaved);
      setOtpSent(false);
      setNewPhone("");
      setPhoneCode("");
    } catch {
      setError(m.errors.internal);
    } finally {
      setPhoneBusy(false);
    }
  }
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api<Login[]>("/api/v1/auth/my-logins")
      .then(setLogins)
      .catch(() => undefined);
    api<{ avatar_thumb_url: string | null }>("/api/v1/me/summary")
      .then((s) => setAvatar(s.avatar_thumb_url))
      .catch(() => undefined);
  }, [done]);

  async function onUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append("file", file);
    try {
      const res = await api<{ avatar_thumb_url: string }>("/api/v1/me/avatar", {
        method: "POST",
        body: fd,
      });
      setAvatar(res.avatar_thumb_url);
    } catch {
      setError(m.errors.internal);
    }
    if (fileRef.current) fileRef.current.value = "";
  }

  async function onRemovePhoto() {
    try {
      await api("/api/v1/me/avatar", { method: "DELETE" });
      setAvatar(null);
    } catch {
      setError(m.errors.internal);
    }
  }

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
        <span className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-card bg-primary-light text-lg font-bold text-primary-dark">
          {mediaUrl(avatar) ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={mediaUrl(avatar) ?? ""} alt="" className="h-full w-full object-cover" />
          ) : (
            (user?.full_name || "؟").charAt(0)
          )}
        </span>
        <div className="min-w-0 flex-1">
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
        <div className="flex shrink-0 flex-col gap-1.5">
          <input ref={fileRef} type="file" accept="image/*" onChange={onUpload} className="hidden" />
          <Button variant="secondary" onClick={() => fileRef.current?.click()} className="text-xs">
            {A.uploadPhoto}
          </Button>
          {avatar && (
            <Button variant="secondary" onClick={onRemovePhoto} className="text-xs !text-danger">
              {A.removePhoto}
            </Button>
          )}
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

      <div className="mt-5">
        <FormSection title={P.changePhone} icon={<IconPhone />}>
          {!otpSent ? (
            <form onSubmit={reqPhone} className="space-y-4">
              <Input
                id="newphone"
                label={P.newPhone}
                dir="ltr"
                inputMode="tel"
                required
                value={newPhone}
                onChange={(e) => setNewPhone(e.target.value)}
                className="text-end"
                placeholder="09xxxxxxxx"
              />
              <p className="text-xs text-ink-muted">{P.phoneHint}</p>
              <Button type="submit" disabled={phoneBusy}>
                {phoneBusy ? m.common.loading : P.sendCode}
              </Button>
            </form>
          ) : (
            <form onSubmit={confirmPhone} className="space-y-4">
              <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">
                {P.codeSent}
              </p>
              <Input
                id="phonecode"
                label={P.code}
                dir="ltr"
                inputMode="numeric"
                required
                value={phoneCode}
                onChange={(e) => setPhoneCode(e.target.value)}
                className="text-center font-mono text-lg tracking-[0.4em]"
                placeholder="••••••"
                maxLength={6}
              />
              <Button type="submit" disabled={phoneBusy}>
                {phoneBusy ? m.common.loading : P.confirmChange}
              </Button>
            </form>
          )}
          {phoneMsg && (
            <p className="mt-3 rounded-control bg-success/10 px-3 py-2 text-sm text-success">
              {phoneMsg}
            </p>
          )}
        </FormSection>
      </div>

      <div className="mt-5">
        <FormSection title={A.recentLogins} icon={<IconStatus />}>
          <p className="mb-2 text-xs text-ink-muted">{A.loginsHint}</p>
          {logins.length === 0 ? (
            <p className="py-4 text-center text-sm text-ink-muted">{A.loginsEmpty}</p>
          ) : (
            <ul className="space-y-1.5">
              {logins.map((l, i) => (
                <li
                  key={i}
                  className="flex items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                >
                  <span
                    className={`font-medium ${l.action === "auth.password_failed" ? "text-danger" : ""}`}
                  >
                    {LOGIN_LABELS[l.action] ?? l.action}
                  </span>
                  <span className="flex items-center gap-3 text-xs text-ink-muted" dir="ltr">
                    <span>{l.ip}</span>
                    <span>
                      {new Date(l.created_at).toLocaleString("ar-SY", {
                        dateStyle: "short",
                        timeStyle: "short",
                      })}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </FormSection>
      </div>
    </div>
  );
}
