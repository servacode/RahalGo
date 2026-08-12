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
import { IconUser, IconLock, IconWarning, IconCheck, IconVerified } from "./icons";
import { IconWhatsApp } from "./brand-icons";

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
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {/* **بابُ تغيير الاسم.**

          كانت الصفحةُ تقرأ الاسمَ وتعرضه في الصورة الرمزية **ولا تكتبه** —
          ومن أخطأ فيه عند التسجيل، أو كُتب له بيد موظّفٍ في طلبٍ هاتفيّ،
          **يبقى عليه إلى الأبد** أو يتّصل بالمنصة ليُغيّره له إنسان.

          **والاسمُ يُقرأ حيث يهمّ**: يناديه السائقُ عند الباب، ويُكتب في
          الفاتورة، ويظهر لغرفة العمليات حين يتّصل. (شهده المالك ٢٠٢٦-٠٨-٠٣) */}
      <Section title={A.identity} icon={<IconUser />}>
        <div className="flex items-center gap-4">
          <div className="figure flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-tint text-primary-dark">
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
      
        <div className="mt-4 border-t border-line-soft pt-4">
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

      {/* ══════════════════════════════════════════════════════════════
          **ورقمٌ واحدٌ لا رقمان — وعليه واتساب**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-١٢: «ما يصير رقم الهاتف مختلف عن واتساب،
           هيك تخرب الدنيا… رقم الهاتف حصراً عليه واتساب، هيك متّفقين».)

          **وكان رقمان**: رقمُ الدخول ورقمُ واتساب «قد يختلف عن رقم
          دخولك». **ورقمان لشخصٍ واحدٍ يفترقان**: يبدّل أحدَهما وينسى
          الآخر، **فيصله رمزُ الدخول على رقمٍ ولا يصله إشعارُ طلبه.**

          **والرمزُ نفسُه يمشي على واتساب** — فمن لا واتسابَ على رقمه لا
          يدخل أصلاً. **فالشرطُ قائمٌ في التسجيل**، وحقلٌ ثانٍ بعده
          يفتح باباً لتناقضٍ لا فائدةَ فيه.

          **والتبديلُ يبقى بابَه**: رمزٌ يصل على الرقم الجديد ويُؤكَّد. */}
      <Section title={A.whatsapp} icon={<IconWhatsApp />}>
        {/* ══════════════════════════════════════════════════════════════
            **والرقمُ يُعرض ولا يُبدَّل من هنا**
            ══════════════════════════════════════════════════════════════

            (تصحيحُ المالك ٢٠٢٦-٠٨-١٢: «رقم الواتساب حذفتَه من حسابي،
             والرقمُ المسجَّل به لازم يظلّ بالحساب مشان التوثيق، ما لازم
             ينحذف… بس زرّ تغيير الرقم نلغيه».)

            **ورقمٌ لا يُرى لا يُوثَّق**: من فتح حسابَه ليتأكّد أيَّ رقمٍ
            سجّل به — قبل أن يشتكي أو يسأل المكتب — **يجد فراغا.** وهو
            أوّلُ ما يُسأل عنه في كلّ خلاف.

            **والتبديلُ يُغلق بابُه**: الرقمُ هو الحساب — به يدخل، وعليه
            يصل رمزُه، وبه يعرفه المكتبُ والسائق. **وتبديلُه بضغطتين في
            صفحةٍ يجعل حساباً كاملاً — بمحفظته وتاريخه — ينتقل إلى رقمٍ
            آخرَ بلا أثر.**

            **ومن أراد التبديل يمرّ بالمكتب** — يُوثَّق من يطلب ولماذا. */}
        <div className="flex items-center justify-between gap-3 rounded-control border border-success-edge bg-success-tint px-3 py-2.5">
          <span dir="ltr" className="min-w-0 truncate font-medium text-ink">
            {phone}
          </span>
          <span className="flex shrink-0 items-center gap-1.5 text-xs font-bold text-success">
            <IconVerified size={16} />
            {A.whatsappVerified}
          </span>
        </div>
        <p className="mt-2 text-xs text-ink-muted">{A.whatsappHint}</p>
      </Section>

      <section className="surface-lit surface !border-danger-edge p-4 sm:col-span-2 lg:col-span-3">
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

      {/* ══════════════════════════════════════════════════════════════
          **وذهب زرُّ الخروج من هنا — لأنّ له باباً في الشريط**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-١١: «بنسخة الجوّال يوجد زرُّ تسجيل خروجٍ
           بحسابي، لا داعيَ له».)

          **وُضع يومَ طُوي الخروجُ من الشريط على الهاتف** — وحجّتُه أنّ
          باباً يُغلق بلا بديلٍ حبسٌ. **وصار له بديلٌ**: قائمةُ الحساب في
          الشريط تحمله على كلّ مقاس.

          **وزرّان لفعلٍ واحدٍ في شاشةٍ واحدةٍ يُربكان** — ومن رآه هنا ظنّ
          أنّ ما في الشريط شيءٌ آخر.

          **و`onLogout` تبقى في الواجهة** — تمرّرها اللوحاتُ ولا تُكسر،
          **وحذفُ خاصّيّةٍ لأنّ شاشةً استغنت عنها يكسر من يمرّرها.** */}

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
