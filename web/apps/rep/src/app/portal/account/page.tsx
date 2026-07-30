"use client";

/** إعدادات حساب المندوب — الصورة الشخصية وتغيير كلمة المرور. */

import { useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, IconUser, IconLock, IconView } from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const A = m.rep.account;

interface MeSummary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
}

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
  }
  return m.errors.internal;
}

export default function AccountPage() {
  const { user } = useAuth();
  const [avatar, setAvatar] = useState<string | null>(null);
  const [name, setName] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api<MeSummary>("/api/v1/me/summary")
      .then((s) => {
        setAvatar(s.avatar_thumb_url);
        setName(s.full_name);
      })
      .catch(() => undefined);
  }, []);

  async function onUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError("");
    setMsg("");
    const fd = new FormData();
    fd.append("file", file);
    try {
      const res = await api<{ avatar_thumb_url: string }>("/api/v1/me/avatar", {
        method: "POST",
        body: fd,
      });
      setAvatar(res.avatar_thumb_url);
      setMsg(A.photoSaved);
    } catch (err) {
      setError(errText(err));
    }
    if (fileRef.current) fileRef.current.value = "";
  }

  async function onRemovePhoto() {
    setError("");
    setMsg("");
    try {
      await api("/api/v1/me/avatar", { method: "DELETE" });
      setAvatar(null);
      setMsg(A.photoSaved);
    } catch (err) {
      setError(errText(err));
    }
  }

  async function onPassword(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setMsg("");
    if (next !== confirm) return setError(m.errors.password_mismatch ?? m.errors.internal);
    setBusy(true);
    try {
      await api("/api/v1/auth/password", {
        method: "POST",
        body: JSON.stringify({ password: next, current_password: current }),
      });
      setMsg(A.saved);
      setCurrent("");
      setNext("");
      setConfirm("");
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  const avatarUrl = mediaUrl(avatar);

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <IconUser size={20} className="text-primary" />
        {A.title}
      </h1>

      {/* الصورة الشخصية */}
      <section className="rounded-card border border-line bg-surface p-5">
        <h2 className="mb-3 text-sm font-bold">{A.photo}</h2>
        <div className="flex items-center gap-4">
          <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border border-line bg-primary-light text-2xl font-bold text-primary-dark">
            {avatarUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={avatarUrl} alt="" className="h-full w-full object-cover" />
            ) : (
              (name || user?.phone || "؟").slice(0, 1)
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              onChange={onUpload}
              className="hidden"
            />
            <Button variant="secondary" onClick={() => fileRef.current?.click()}>
              {A.uploadPhoto}
            </Button>
            {avatar && (
              <Button variant="secondary" onClick={onRemovePhoto} className="!text-danger">
                {A.removePhoto}
              </Button>
            )}
          </div>
        </div>
      </section>

      {/* تغيير كلمة المرور */}
      <section className="rounded-card border border-line bg-surface p-5">
        <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
          <IconLock size={16} className="text-primary" />
          {A.changePassword}
        </h2>
        <form onSubmit={onPassword} className="space-y-4">
          <Input
            id="current"
            label={A.currentPassword}
            icon={<IconLock />}
            type="password"
            required
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
          />
          <Input
            id="new"
            label={A.newPassword}
            icon={<IconLock />}
            type="password"
            required
            value={next}
            onChange={(e) => setNext(e.target.value)}
          />
          <Input
            id="confirm"
            label={A.confirmPassword}
            icon={<IconView />}
            type="password"
            required
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
          <Button type="submit" disabled={busy} className="w-full py-2.5">
            {busy ? m.common.loading : m.common.save}
          </Button>
        </form>
      </section>

      {msg && (
        <p className="rounded-control bg-success/10 px-3 py-2 text-sm text-success">{msg}</p>
      )}
      {error && (
        <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}
    </div>
  );
}
