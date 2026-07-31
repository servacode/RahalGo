"use client";

/**
 * إعدادات الحساب المشتركة — نسخة واحدة مركزية لكل التطبيقات (زبون/مندوب/متجر/إدارة):
 * الصورة الشخصية، تغيير كلمة المرور، وتغيير رقم الهاتف (بتحقّق OTP).
 * يُمرَّر لها عميل الـapi و mediaUrl الخاصان بكل تطبيق (حقن التبعية).
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input } from "./components";
import { IconUser, IconLock, IconPhone } from "./icons";

const m = getMessages(defaultLocale);
const A = m.shared.account;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

function errText(err: unknown): string {
  const key =
    typeof err === "object" && err && "body" in err
      ? ((err as { body?: { message_key?: string } }).body?.message_key ?? "").split(".").pop() ?? ""
      : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

function Section({ title, icon, children }: { title: string; icon: ReactNode; children: ReactNode }) {
  return (
    <section className="rounded-card border border-line bg-surface p-5">
      <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
        <span className="text-primary [&>svg]:h-4 [&>svg]:w-4">{icon}</span>
        {title}
      </h2>
      {children}
    </section>
  );
}

export function AccountSettings({
  api,
  mediaUrl,
  phone,
}: {
  api: ApiFn;
  mediaUrl: (p: string | null | undefined) => string | null;
  phone?: string;
}) {
  const [avatar, setAvatar] = useState<string | null>(null);
  const [name, setName] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);

  const [newPhone, setNewPhone] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [phoneCode, setPhoneCode] = useState("");
  const [phoneBusy, setPhoneBusy] = useState(false);

  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ full_name: string; avatar_thumb_url: string | null }>("/api/v1/me/summary")
      .then((s) => {
        setAvatar(s.avatar_thumb_url);
        setName(s.full_name);
      })
      .catch(() => undefined);
  }, [api]);

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
    if (next !== confirm) return setError(m.errors.password_mismatch);
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

  const avatarUrl = mediaUrl(avatar);

  return (
    <div className="space-y-6">
      <Section title={A.photo} icon={<IconUser />}>
        <div className="flex items-center gap-4">
          <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border border-line bg-primary-light text-2xl font-bold text-primary-dark">
            {avatarUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={avatarUrl} alt="" className="h-full w-full object-cover" />
            ) : (
              (name || phone || m.terms.avatarFallback).slice(0, 1)
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
      </Section>

      <Section title={A.changePassword} icon={<IconLock />}>
        <form onSubmit={onPassword} className="space-y-4">
          <Input id="cur-pw" label={A.currentPassword} icon={<IconLock />} type="password" required value={current} onChange={(e) => setCurrent(e.target.value)} />
          <Input id="new-pw" label={A.newPassword} icon={<IconLock />} type="password" required value={next} onChange={(e) => setNext(e.target.value)} />
          <Input id="conf-pw" label={A.confirmPassword} icon={<IconLock />} type="password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          <Button type="submit" disabled={busy} className="w-full py-2.5">
            {busy ? m.common.loading : m.common.save}
          </Button>
        </form>
      </Section>

      <Section title={A.changePhone} icon={<IconPhone />}>
        {!otpSent ? (
          <form onSubmit={reqPhone} className="space-y-4">
            <div>
              <Input id="new-phone" label={A.newPhone} icon={<IconPhone />} dir="ltr" inputMode="tel" required value={newPhone} onChange={(e) => setNewPhone(e.target.value)} className="text-end" placeholder="09xxxxxxxx" />
              <p className="mt-1 text-xs text-ink-muted">{A.phoneHint}</p>
            </div>
            <Button type="submit" disabled={phoneBusy} className="w-full py-2.5">
              {phoneBusy ? m.common.loading : A.sendCode}
            </Button>
          </form>
        ) : (
          <form onSubmit={confirmPhone} className="space-y-4">
            <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">{A.codeSent}</p>
            <Input id="phone-code" label={A.code} dir="ltr" inputMode="numeric" required autoFocus value={phoneCode} onChange={(e) => setPhoneCode(e.target.value)} className="text-center font-mono text-lg tracking-[0.4em]" placeholder="••••••" maxLength={6} />
            <Button type="submit" disabled={phoneBusy} className="w-full py-2.5">
              {phoneBusy ? m.common.loading : A.confirmChange}
            </Button>
          </form>
        )}
      </Section>

      {msg && <p className="rounded-control bg-success/10 px-3 py-2 text-sm text-success">{msg}</p>}
      {error && <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
    </div>
  );
}
