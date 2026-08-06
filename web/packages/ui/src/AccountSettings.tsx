"use client";

/**
 * إعدادات الحساب المشتركة — نسخة واحدة مركزية لكل التطبيقات (زبون/مندوب/متجر/إدارة):
 * الصورة الشخصية، تغيير كلمة المرور، وتغيير رقم الهاتف (بتحقّق OTP).
 * يُمرَّر لها عميل الـapi و mediaUrl الخاصان بكل تطبيق (حقن التبعية).
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input } from "./components";
import { Alert } from "./feedback";
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

/**
 * **بطاقةٌ في سطرٍ أفقيّ — لا صندوقٌ طويل.**
 *
 * كان العنوانُ فوق والحقولُ تحته، **فكلُّ حقلٍ صغيرٍ يأخذ صندوقاً بارتفاع
 * ثلاثة أسطر** — وستُّ بطاقاتٍ تصير صفحةً تُمرَّر مرّتين.
 *
 * فصار العنوانُ **عموداً ضيّقاً يميناً** والحقولُ تمتدّ بجانبه: **سطرٌ واحدٌ
 * لكلّ شأن**، والصفحةُ تُمسح بلمحة.
 *
 * **وعلى الهاتف يعود العنوانُ فوق** — عمودان في شاشةٍ عرضُها راحةُ يدٍ يقصّان
 * الحقول. (قرارُ المالك ٢٠٢٦-٠٨-٠٣.)
 */
function Section({ title, icon, children }: { title: string; icon: ReactNode; children: ReactNode }) {
  return (
    /* **بطاقاتٌ بارتفاعٍ واحد.**

       كان `items-start` يترك كلَّ بطاقةٍ بطولها الطبيعيّ، **فيصير السطرُ
       درجاتٍ متفاوتة** — والعينُ تقرأ الاختلافَ عيباً في التنسيق لا فرقاً في
       المحتوى. **والشبكةُ تمدّها لأطولهنّ** والعنوانُ يبقى في الأعلى.
       (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */
    <section className="flex h-full flex-col surface p-4">
      <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
        <span className="text-primary [&>svg]:h-4 [&>svg]:w-4">{icon}</span>
        {title}
      </h2>
      <div className="flex-1">{children}</div>
    </section>
  );
}

export function AccountSettings({
  api,
  mediaUrl,
  phone,
  onDeleted,
  onVerified,
  onLogout,
}: {
  api: ApiFn;
  mediaUrl: (p: string | null | undefined) => string | null;
  phone?: string;
  /**
   * تسجيلُ الخروج — **يُعرض على الهاتف وحدَه.**
   *
   * زرُّ الخروج طُوي من الشريط في الشاشات الضيّقة (كان رقعةً حمراءَ تزاحم
   * وتُضغط بالخطأ)، **فلا بدّ من بابٍ يخرج منه صاحبُ الهاتف** — وموضعُه
   * المعتاد في كلّ تطبيق: صفحةُ الحساب.
   */
  onLogout?: () => void;
  /** يُستدعى بعد حذف الحساب — كل تطبيق يقرر وجهته (الخروج ثم صفحة الدخول) */
  onDeleted?: () => void;
  /** يُستدعى بعد توثيق واتساب — تفتح به الصفحات المقفلة بلا تحديث */
  onVerified?: () => void;
}) {
  const [avatar, setAvatar] = useState<string | null>(null);
  const [name, setName] = useState("");
  /** الاسمُ المكتوبُ في الحقل — **يُفصل عن المحفوظ** ليُعرف هل تغيّر. */
  const [nameDraft, setNameDraft] = useState("");
  const [nameBusy, setNameBusy] = useState(false);
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
        /* **واسمٌ غائبٌ يبقى فراغاً لا `undefined`.**

           كان `setName(s.full_name)` — **وردٌّ بلا اسمٍ يجعل الحالَ غيرَ
           معرَّف**، ثمّ ينفجر `name.trim()` في زرّ الحفظ **فتُفرَّغ صفحةُ
           الحساب كلُّها**: لا اسمَ ولا عنوانَ ولا خروج.

           **وشاشةٌ بيضاءُ من حقلٍ ناقصٍ في ردٍّ** أسوأُ ما يقع: **لا خطأَ
           يُقرأ ولا سببَ يُخمَّن.** والحالُ مهيّأةٌ بفراغٍ أصلاً
           (`useState("")`) — **والإسنادُ هو ما نقضه.**

           (وقع في فحصٍ بمتصفّحٍ حقيقيّ ٢٠٢٦-٠٨-٠٦.) */
        setName(s.full_name ?? "");
        setNameDraft(s.full_name ?? "");
        setWaVerified(s.whatsapp_verified);
        // رقم الدخول اقتراحٌ مبدئي: أغلب الناس واتسابهم عليه، فلا نطلب كتابته
        setWa(s.whatsapp_phone ?? phone ?? "");
      })
      .catch(() => undefined);
  }, [api, phone]);

  /**
   * **تُصغَّر في المتصفّح قبل أن تُرسَل.**
   *
   * حدُّ الخادم خمسةُ ميغا، **وصورةُ هاتفٍ حديثة تتجاوزه** — فيُردّ الرفعُ
   * بـ«الصورة أكبر من الحدّ». والرسالةُ صحيحةٌ **والتجربةُ فاشلة**: من التقط
   * صورةً بهاتفه لا يعرف كيف يُصغّرها، **فيترك الصورةَ ولا يرفع.**
   *
   * **والخادمُ يُصغّرها بعد الرفع أصلاً** — فرفعُ اثني عشر ميغا لتصير مئةَ
   * كيلو **إهدارُ باقةِ إنترنتٍ في مدينةٍ باقتُها غالية**، ثمّ يُردّ.
   *
   * فتُرسم على لوحةٍ بحدٍّ أقصى ١٢٠٠ بكسل وتُحوَّل JPEG. **وما فشل تصغيرُه
   * يُرسَل كما هو** — عطبٌ في اللوحة يجب ألّا يمنع رفعاً قد ينجح.
   */
  async function shrink(file: File): Promise<Blob> {
    const MAX = 1200;
    try {
      const bmp = await createImageBitmap(file);
      const scale = Math.min(1, MAX / Math.max(bmp.width, bmp.height));
      if (scale === 1 && file.size <= 2 * 1024 * 1024) return file;
      const canvas = document.createElement("canvas");
      canvas.width = Math.round(bmp.width * scale);
      canvas.height = Math.round(bmp.height * scale);
      const ctx = canvas.getContext("2d");
      if (!ctx) return file;
      ctx.drawImage(bmp, 0, 0, canvas.width, canvas.height);
      const blob = await new Promise<Blob | null>((res) =>
        canvas.toBlob(res, "image/jpeg", 0.85),
      );
      return blob ?? file;
    } catch {
      return file;
    }
  }

  async function onUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setError("");
    setMsg("");
    const fd = new FormData();
    fd.append("file", await shrink(file), "avatar.jpg");
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

  async function onName(e: React.FormEvent) {
    e.preventDefault();
    setNameBusy(true);
    setMsg("");
    setError("");
    try {
      await api("/api/v1/me/name", {
        method: "PATCH",
        body: JSON.stringify({ full_name: nameDraft.trim() }),
      });
      setName(nameDraft.trim());
      setMsg(A.nameSaved);
      // **والشريطُ العلويّ يقرأ الاسمَ** — فيُخبَر ليُحدّثه بلا تحديث صفحة.
      onVerified?.();
    } catch (err) {
      setError(errText(err));
    } finally {
      setNameBusy(false);
    }
  }

  return (
    /* **عمودٌ واحدٌ ضيّق — لا مربّعان يملآن الشاشة.**

       كانت مربّعين في السطر على الشاشات المتوسطة فأكبر: **صندوقان كبيران لكلّ
       حقلٍ صغير**، فتمتدّ الصفحةُ عرضاً وطولاً معاً و«حسابي» يصير لوحةَ تحكّم.
       **وهي صفحةٌ تُفتح مرّةً في الشهر لتغيير كلمةٍ أو رقم.**

       فصارت عموداً واحداً محدودَ العرض: **يُمسح بالعين من أعلى إلى أسفل**،
       والحقولُ قريبةٌ من بعضها لا متباعدةٌ في فراغ.
       (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «اجعل الكروت بصف واحد، ما يلزم الكروت بهذا
       الشكل ماخذة مساحات كبيرة».) */
    /* **ثلاثُ بطاقاتٍ في السطر — لا أربعٌ ولا عمود.**

       بالدمج صار لكلّ شأنٍ بطاقة: **من أنت** (الصورةُ والاسم) · **مفتاحُك**
       (كلمةُ المرور) · **كيف نصل إليك** (الرقمُ والواتساب). **وثلاثةٌ تملأ
       السطرَ بلا فراغٍ رابعٍ يتيم.** */
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {/* **بابُ تغيير الاسم.**

          كانت الصفحةُ تقرأ الاسمَ وتعرضه في الصورة الرمزية **ولا تكتبه** —
          ومن أخطأ فيه عند التسجيل، أو كُتب له بيد موظّفٍ في طلبٍ هاتفيّ،
          **يبقى عليه إلى الأبد** أو يتّصل بالمنصة ليُغيّره له إنسان.

          **والاسمُ يُقرأ حيث يهمّ**: يناديه السائقُ عند الباب، ويُكتب في
          الفاتورة، ويظهر لغرفة العمليات حين يتّصل. (شهده المالك ٢٠٢٦-٠٨-٠٣) */}
      <Section title={A.identity} icon={<IconUser />}>
        <div className="flex items-center gap-4">
          <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-light text-2xl font-bold text-primary-dark">
            {avatarUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={avatarUrl} alt="" loading="lazy" className="h-full w-full object-cover" />
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
      
        <div className="mt-4 border-t border-line pt-4">
        <form onSubmit={onName} className="space-y-3">
          <div>
            <Input
              id="my-name"
              label={A.fullName}
              icon={<IconUser />}
              value={nameDraft}
              onChange={(e) => setNameDraft(e.target.value)}
              placeholder={A.namePlaceholder}
            />
          </div>
          {/* **ولا يُفعَّل الزرُّ بلا تغيير** — زرٌّ يُضغط فلا يقع شيءٌ يُعلّم
              صاحبَه ألّا يثق بالأزرار. */}
          <Button
            type="submit"
            disabled={nameBusy || nameDraft.trim() === name.trim() || nameDraft.trim().length < 2}
            className="w-full py-2.5"
          >
            {A.saveName}
          </Button>
        </form>
      </div>
      </Section>

      <Section title={A.changePassword} icon={<IconLock />}>
        <form onSubmit={onPassword} className="space-y-3">
          <Input id="cur-pw" label={A.currentPassword} icon={<IconLock />} type="password" required value={current} onChange={(e) => setCurrent(e.target.value)} />
          <Input id="new-pw" label={A.newPassword} icon={<IconLock />} type="password" required value={next} onChange={(e) => setNext(e.target.value)} />
          <Input id="conf-pw" label={A.confirmPassword} icon={<IconLock />} type="password" required value={confirm} onChange={(e) => setConfirm(e.target.value)} />
          <Button type="submit" disabled={busy} className="w-full py-2.5">
            {busy ? m.common.loading : m.common.save}
          </Button>
        </form>
      </Section>

      <Section title={A.contact} icon={<IconPhone />}>
        {!otpSent ? (
          <form onSubmit={reqPhone} className="space-y-3">
            <div>
              <Input id="new-phone" label={A.newPhone} icon={<IconPhone />} dir="ltr" inputMode="tel" required value={newPhone} onChange={(e) => setNewPhone(e.target.value)} className="text-end" placeholder="09xxxxxxxx" />
              <p className="mt-1 text-xs text-ink-muted">{A.phoneHint}</p>
            </div>
            <Button type="submit" disabled={phoneBusy} className="w-full py-2.5">
              {phoneBusy ? m.common.loading : A.sendCode}
            </Button>
          </form>
        ) : (
          <form onSubmit={confirmPhone} className="space-y-3">
            <p className="rounded-control bg-primary-light px-3 py-2 text-sm text-primary-dark">{A.codeSent}</p>
            <div>
              <Input id="phone-code" label={A.code} dir="ltr" inputMode="numeric" required autoFocus value={phoneCode} onChange={(e) => setPhoneCode(e.target.value)} className="text-center font-mono text-lg tracking-[0.4em]" placeholder="••••••" maxLength={6} />
            </div>
            <Button type="submit" disabled={phoneBusy} className="w-full py-2.5">
              {phoneBusy ? m.common.loading : A.confirmChange}
            </Button>
          </form>
        )}
      
        <div className="mt-4 border-t border-line pt-4">
          <p className="mb-2 flex items-center gap-2 text-xs font-bold text-ink-muted">
            <IconWhatsApp className="h-4 w-4 text-primary" />
            {A.whatsapp}
          </p>
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
          <form onSubmit={reqWhatsApp} className="space-y-3">
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
          <form onSubmit={confirmWhatsApp} className="space-y-3">
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
      </div>
      </Section>

      <section className="surface-lit surface !border-danger/45 p-4 sm:col-span-2 lg:col-span-3">
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

      {/* **الخروجُ هنا على الهاتف** — طُوي من الشريط فلا يُزاحم، **وبابٌ
          يُغلق بلا بديلٍ حبسٌ لا تبسيط.** */}
      {onLogout && (
        <section className="surface p-4 sm:hidden">
          <Button variant="secondary" onClick={onLogout} className="w-full !text-danger">
            {m.auth.logout}
          </Button>
        </section>
      )}

      {msg && (
        <Alert tone="success">
          {msg}
        </Alert>
      )}
      {error && (
        <Alert>
          {error}
        </Alert>
      )}
    </div>
  );
}
