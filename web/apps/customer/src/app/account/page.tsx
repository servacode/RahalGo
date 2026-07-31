"use client";

/** حسابي — الصورة الشخصية وتغيير كلمة المرور (لأي مستخدم). */

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, IconUser, IconLock, IconPhone } from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);
const A = m.site.account;

interface Summary {
  full_name: string;
  avatar_thumb_url: string | null;
}

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
  }
  return m.errors.internal;
}

export default function AccountPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [avatar, setAvatar] = useState<string | null>(null);
  const [name, setName] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  const [newPhone, setNewPhone] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [phoneCode, setPhoneCode] = useState("");
  const [phoneBusy, setPhoneBusy] = useState(false);

  async function reqPhone(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setMsg("");
    setPhoneBusy(true);
    try {
      await api("/api/v1/auth/phone/request", { method: "POST", body: JSON.stringify({ phone: newPhone }) });
      setOtpSent(true);
    } catch (err) {
      setError(errText(err));
    } finally {
      setPhoneBusy(false);
    }
  }

  async function confirmPhone(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setMsg("");
    setPhoneBusy(true);
    try {
      await api("/api/v1/auth/phone/confirm", {
        method: "POST",
        body: JSON.stringify({ phone: newPhone, code: phoneCode }),
      });
      setMsg(A.phoneSaved);
      setOtpSent(false);
      setNewPhone("");
      setPhoneCode("");
    } catch (err) {
      setError(errText(err));
    } finally {
      setPhoneBusy(false);
    }
  }

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/account");
      return;
    }
    api<Summary>("/api/v1/me/summary")
      .then((s) => {
        setAvatar(s.avatar_thumb_url);
        setName(s.full_name);
      })
      .catch(() => undefined);
  }, [user, loading, router]);

  async function onUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError("");
    setMsg("");
    const fd = new FormData();
    fd.append("file", file);
    try {
      const res = await api<{ avatar_thumb_url: string }>("/api/v1/me/avatar", { method: "POST", body: fd });
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
    if (next !== confirm) return setError(A.mismatch);
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
      <h1 className="flex items-center gap-2 text-xl font-bold">
        <IconUser className="text-primary" />
        {A.title}
      </h1>

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
            <input ref={fileRef} type="file" accept="image/*" onChange={onUpload} className="hidden" />
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

      <section className="rounded-card border border-line bg-surface p-5">
        <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
          <IconLock size={16} className="text-primary" />
          {A.changePassword}
        </h2>
        <form onSubmit={onPassword} className="space-y-4">
          <Input id="cur" label={A.currentPassword} icon={<IconLock />} type="password" required value={current} onChange={(e) => setCurrent(e.target.value)} />
          <Input id="new" label={A.newPassword} icon={<IconLock />} type="password" required value={next} onChange={(e) => setNext(e.target.value)} />
          <Input id="conf" label={A.confirmPassword} icon={<IconLock />} type="password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          <Button type="submit" disabled={busy} className="w-full py-2.5">
            {busy ? m.common.loading : m.common.save}
          </Button>
        </form>
      </section>

      <section className="rounded-card border border-line bg-surface p-5">
        <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
          <IconPhone size={16} className="text-primary" />
          {A.changePhone}
        </h2>
        {!otpSent ? (
          <form onSubmit={reqPhone} className="space-y-4">
            <div>
              <Input
                id="newphone"
                label={A.newPhone}
                icon={<IconPhone />}
                dir="ltr"
                inputMode="tel"
                required
                value={newPhone}
                onChange={(e) => setNewPhone(e.target.value)}
                className="text-end"
                placeholder="09xxxxxxxx"
              />
              <p className="mt-1 text-xs text-ink-muted">{A.phoneHint}</p>
            </div>
            <Button type="submit" disabled={phoneBusy} className="w-full py-2.5">
              {phoneBusy ? m.common.loading : A.sendCode}
            </Button>
          </form>
        ) : (
          <form onSubmit={confirmPhone} className="space-y-4">
            <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">
              {A.codeSent}
            </p>
            <Input
              id="phonecode"
              label={A.code}
              dir="ltr"
              inputMode="numeric"
              required
              autoFocus
              value={phoneCode}
              onChange={(e) => setPhoneCode(e.target.value)}
              className="text-center font-mono text-lg tracking-[0.4em]"
              placeholder="••••••"
              maxLength={6}
            />
            <Button type="submit" disabled={phoneBusy} className="w-full py-2.5">
              {phoneBusy ? m.common.loading : A.confirmChange}
            </Button>
            <button
              type="button"
              onClick={() => {
                setOtpSent(false);
                setPhoneCode("");
              }}
              className="w-full text-center text-sm text-ink-muted hover:text-primary"
            >
              {m.auth.changePhone}
            </button>
          </form>
        )}
      </section>

      {msg && <p className="rounded-control bg-success/10 px-3 py-2 text-sm text-success">{msg}</p>}
      {error && <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
    </div>
  );
}
