"use client";

/**
 * إعدادات الحساب المشتركة — نسخة واحدة مركزية لكل التطبيقات (زبون/مندوب/متجر/إدارة):
 * الصورة الشخصية، تغيير كلمة المرور، وتغيير رقم الهاتف (بتحقّق OTP).
 * يُمرَّر لها عميل الـapi و mediaUrl الخاصان بكل تطبيق (حقن التبعية).
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input } from "./components";
import { emitLocal } from "./Notifications";
import { IconUser, IconLock, IconPhone, IconWarning, IconCheck, IconWhatsApp, IconVerified } from "./icons";

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
  onDeleted,
  onVerified,
}: {
  api: ApiFn;
  mediaUrl: (p: string | null | undefined) => string | null;
  phone?: string;
  /** يُستدعى بعد حذف الحساب — كل تطبيق يقرر وجهته (الخروج ثم صفحة الدخول) */
  onDeleted?: () => void;
  /** يُستدعى بعد توثيق واتساب — تفتح به الصفحات المقفلة بلا تحديث */
  onVerified?: () => void;
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

  const [wa, setWa] = useState("");
  const [waVerified, setWaVerified] = useState(false);
  const [waSent, setWaSent] = useState(false);
  const [waCode, setWaCode] = useState("");
  const [waBusy, setWaBusy] = useState(false);
  const [waEditing, setWaEditing] = useState(false);

  const [delSent, setDelSent] = useState(false);
  const [delCode, setDelCode] = useState("");
  const [delBusy, setDelBusy] = useState(false);
  const [deleted, setDeleted] = useState(false);

  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api<{
      full_name: string;
      avatar_thumb_url: string | null;
      whatsapp_phone: string | null;
      whatsapp_verified: boolean;
    }>("/api/v1/me/summary")
      .then((s) => {
        setAvatar(s.avatar_thumb_url);
        setName(s.full_name);
        setWaVerified(s.whatsapp_verified);
        // رقم الدخول اقتراحٌ مبدئي: أغلب الناس واتسابهم عليه، فلا نطلب كتابته
        setWa(s.whatsapp_phone ?? phone ?? "");
      })
      .catch(() => undefined);
  }, [api, phone]);

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
      emitLocal("profile"); // الشريط العلوي يلتقط الصورة الجديدة فوراً
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
      emitLocal("profile");
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
      emitLocal("profile");
      setOtpSent(false);
      setNewPhone("");
      setPhoneCode("");
    } catch (err) {
      setError(errText(err));
    } finally {
      setPhoneBusy(false);
    }
  }

  async function reqWhatsApp(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setMsg("");
    setWaBusy(true);
    try {
      await api("/api/v1/auth/whatsapp/request", { method: "POST", body: JSON.stringify({ phone: wa }) });
      setWaSent(true);
    } catch (err) {
      setError(errText(err));
    } finally {
      setWaBusy(false);
    }
  }

  async function confirmWhatsApp(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setWaBusy(true);
    try {
      await api("/api/v1/auth/whatsapp/confirm", {
        method: "POST",
        body: JSON.stringify({ phone: wa, code: waCode }),
      });
      setWaVerified(true);
      setWaSent(false);
      setWaEditing(false);
      setWaCode("");
      setMsg(A.whatsappDone);
      onVerified?.();
    } catch (err) {
      setError(errText(err));
    } finally {
      setWaBusy(false);
    }
  }

  async function reqDelete() {
    setError("");
    setMsg("");
    setDelBusy(true);
    try {
      await api("/api/v1/auth/account/delete/request", { method: "POST" });
      setDelSent(true);
    } catch (err) {
      setError(errText(err));
    } finally {
      setDelBusy(false);
    }
  }

  async function confirmDelete(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setDelBusy(true);
    try {
      await api("/api/v1/auth/account/delete/confirm", {
        method: "POST",
        body: JSON.stringify({ code: delCode }),
      });
      setDeleted(true);
      // الجلسة أُبطلت في الخادم — نُخرج المستخدم بدل تركه في شاشة ميتة
      setTimeout(() => onDeleted?.(), 2500);
    } catch (err) {
      setError(errText(err));
    } finally {
      setDelBusy(false);
    }
  }

  const avatarUrl = mediaUrl(avatar);

  return (
    // مربعان في السطر على الشاشات المتوسطة فأكبر — كانت الأقسام مرصوفة طولياً
    // فيتمدّد النموذج بلا داعٍ ونصف العرض فارغ.
    <div className="grid gap-5 md:grid-cols-2">
      <Section title={A.photo} icon={<IconUser />}>
        <div className="flex items-center gap-4">
          <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-badge border border-line bg-primary-light text-2xl font-bold text-primary-dark">
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

      {/* توثيق واتساب — قناة التواصل، لا هوية الدخول (قسم مستقل عن تغيير الرقم) */}
      <Section title={A.whatsapp} icon={<IconWhatsApp />}>
        {waVerified && !waEditing ? (
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3 rounded-control border border-success/30 bg-success/5 px-3 py-2.5">
              <span dir="ltr" className="min-w-0 truncate font-medium text-ink">
                {wa}
              </span>
              <span className="flex shrink-0 items-center gap-1.5 text-xs font-bold text-success">
                <IconVerified size={16} />
                {A.whatsappVerified}
              </span>
            </div>
            <Button
              variant="secondary"
              onClick={() => {
                setWaEditing(true);
                setWaSent(false);
              }}
            >
              {A.whatsappChange}
            </Button>
          </div>
        ) : !waSent ? (
          <form onSubmit={reqWhatsApp} className="space-y-4">
            <div>
              <Input
                id="wa-phone"
                label={A.whatsapp}
                icon={<IconWhatsApp />}
                dir="ltr"
                inputMode="tel"
                required
                value={wa}
                onChange={(e) => setWa(e.target.value)}
                className="text-end"
                placeholder="09xxxxxxxx"
              />
              <p className="mt-1 text-xs text-ink-muted">{A.whatsappHint}</p>
            </div>
            {!waVerified && (
              <p className="flex items-center gap-1.5 text-xs font-medium text-warning">
                <IconWarning size={14} />
                {A.whatsappUnverified}
              </p>
            )}
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={waBusy} className="flex-1 py-2.5">
                {waBusy ? m.common.loading : A.whatsappVerify}
              </Button>
              {waEditing && (
                <Button type="button" variant="secondary" onClick={() => setWaEditing(false)}>
                  {m.common.cancel}
                </Button>
              )}
            </div>
          </form>
        ) : (
          <form onSubmit={confirmWhatsApp} className="space-y-4">
            <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">
              {A.whatsappCodeSent}
            </p>
            <Input
              id="wa-code"
              label={A.code}
              dir="ltr"
              inputMode="numeric"
              required
              autoFocus
              value={waCode}
              onChange={(e) => setWaCode(e.target.value)}
              className="text-center font-mono text-lg tracking-[0.4em]"
              placeholder="••••••"
              maxLength={6}
            />
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={waBusy} className="flex-1 py-2.5">
                {waBusy ? m.common.loading : A.whatsappVerify}
              </Button>
              <Button type="button" variant="secondary" onClick={() => setWaSent(false)}>
                {m.common.cancel}
              </Button>
            </div>
          </form>
        )}
      </Section>

      {/* منطقة الخطر — تمتدّ عبر العمودين وتُفصل بصرياً عمّا فوقها */}
      <section className="rounded-card border border-danger/30 bg-danger/5 p-5 md:col-span-2">
        <h2 className="mb-1 flex items-center gap-2 text-sm font-bold text-danger">
          <span className="[&>svg]:h-4 [&>svg]:w-4">
            <IconWarning />
          </span>
          {A.dangerTitle}
        </h2>
        <p className="mb-3 text-xs leading-relaxed text-ink-muted">{A.dangerHint}</p>

        {deleted ? (
          <p className="flex items-center gap-2 rounded-control bg-surface px-3 py-2 text-sm text-ink">
            <IconCheck size={16} strokeWidth={3} className="text-success" />
            {A.deleteDone}
          </p>
        ) : !delSent ? (
          <Button variant="danger" onClick={reqDelete} disabled={delBusy}>
            {delBusy ? m.common.loading : A.deleteSendCode}
          </Button>
        ) : (
          <form onSubmit={confirmDelete} className="max-w-sm space-y-3">
            <p className="text-sm text-ink">{A.deleteCodeSent}</p>
            <Input
              id="del-code"
              label={A.code}
              dir="ltr"
              inputMode="numeric"
              required
              autoFocus
              value={delCode}
              onChange={(e) => setDelCode(e.target.value)}
              className="text-center font-mono text-lg tracking-[0.4em]"
              placeholder="••••••"
              maxLength={6}
            />
            <p className="text-xs font-medium text-danger">{A.dangerIrreversible}</p>
            <div className="flex flex-wrap gap-2">
              <Button type="submit" variant="danger" disabled={delBusy}>
                {delBusy ? m.common.loading : A.deleteConfirm}
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  setDelSent(false);
                  setDelCode("");
                }}
              >
                {A.deleteCancel}
              </Button>
            </div>
          </form>
        )}
      </section>

      {msg && (
        <p className="rounded-control bg-success/10 px-3 py-2 text-sm text-success md:col-span-2">
          {msg}
        </p>
      )}
      {error && (
        <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger md:col-span-2">
          {error}
        </p>
      )}
    </div>
  );
}
