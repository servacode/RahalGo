"use client";

/**
 * **توثيقُ واتساب — صندوقٌ واحدٌ يُنادى حيث يُحتاج.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «أو زرّ وثّق حسابك بشكلٍ مباشر».)
 *
 * # ولماذا مكوّنٌ لا صفحة
 *
 * **كان التوثيقُ داخلَ «حسابي» وحدَها** — ومن وقف أمام شاشةٍ مقفلةٍ يُقال له
 * «اذهب إلى حسابك». **وخطوةٌ زائدةٌ بين من يريد أن يوثّق وبين التوثيق خطوةٌ
 * يسقط فيها بعضُهم**: يذهب، فيجد صفحةً فيها عشرةُ حقول، فيبحث.
 *
 * **فيُفتح حيث وقف.**
 *
 * # ولا يُكرَّر منطقُه
 *
 * **نداءان لنقطتين** (`request` ثمّ `confirm`) — **ولو كُتبا في موضعين
 * لَافترقا**: يُضاف شرطٌ في أحدهما ويُنسى في الآخر، فيوثّق من دخل من بابٍ ولا
 * يوثّق من دخل من الآخر.
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Input } from "./components";
import { Alert } from "./feedback";

const m = getMessages(defaultLocale);
const A = m.shared.account;

type ApiFn = <T = unknown>(path: string, init?: RequestInit) => Promise<T>;

export function WhatsAppVerify({
  api,
  phone,
  onVerified,
}: {
  api: ApiFn;
  /** الرقمُ المقترَح — رقمُ الحساب، ويملك تبديلَه. */
  phone?: string;
  onVerified?: () => void;
}) {
  const [wa, setWa] = useState(phone ?? "");
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function request(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await api("/api/v1/auth/whatsapp/request", {
        method: "POST",
        body: JSON.stringify({ phone: wa }),
      });
      setSent(true);
    } catch (err) {
      /* ══════════════════════════════════════════════════════════════
         **وسببُ الخادم يُعرض — لا يُرمى ويُستبدَل برسالةٍ عامّة**
         ══════════════════════════════════════════════════════════════

         (شهده المالك ٢٠٢٦-٠٨-١١: رقمٌ بلا واتساب يقول «ضغطٌ على خادم
          الرسائل — حاول بعد قليل».)

         **كان `catch { setError(m.errors.internal) }`** — يمسك الخطأَ
         ويرميه ويكتب مكانَه رسالةً واحدةً لكلّ العلل. **فيقرأ صاحبُه
         «الخطأُ عندنا» ويعيد المحاولةَ عشراً** والعلّةُ في رقمه.

         **والخادمُ يعرف ويقول** — والشاشةُ كانت تُسكته. */
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function confirm(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await api("/api/v1/auth/whatsapp/confirm", {
        method: "POST",
        body: JSON.stringify({ phone: wa, code }),
      });
      onVerified?.();
    } catch (err) {
      /* **ورمزٌ خاطئٌ غيرُ رمزٍ منتهٍ غيرِ محاولاتٍ نفدت** — والخادمُ
         يفرّقها، **فتُعرض كما قالها.** */
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={sent ? confirm : request} className="space-y-3">
      {sent ? (
        <Input
          id="wa-code"
          label={A.whatsapp}
          dir="ltr"
          inputMode="numeric"
          required
          value={code}
          onChange={(e) => setCode(e.target.value)}
        />
      ) : phone ? (
        /* ══════════════════════════════════════════════════════════════
           **ورقمُ حسابه يُعرض ولا يُسأل عنه**
           ══════════════════════════════════════════════════════════════

           (قرارُ المالك ٢٠٢٦-٠٨-١١: «كيف المندوبُ يرجع يكتب رقمَه وهو
            موجود؟ ما يصير — خلص، بس وثّق حسابَك بدون ما يرجع يكتب رقم».)

           **وهو داخلٌ برقمه**: دخل به وأُرسل إليه رمزُ الدخول عليه،
           **فسؤالُه عنه ثانيةً يقول له إنّ المنصّةَ نسيت من هو.**

           **وكتابتُه بابُ خطأ**: رقمٌ فيه خانةٌ زائدةٌ يذهب الرمزُ إلى
           غيره — **ولا يعرف أنّه أخطأ، إنّما يرى «لم يصل الرمز».**

           **ويبقى الحقلُ لمن لا رقمَ له في حسابه** — حالةٌ لا تقع في
           المنصّة اليوم، **لكنّ صندوقاً يُنادى من أربعة مواضعَ لا يُبنى
           على أنّ الرقمَ موجودٌ دائماً.** */
        <div className="surface-inset px-3 py-2 text-center">
          <p className="text-xs text-ink-muted">{A.whatsapp}</p>
          <p className="font-bold tabular-nums" dir="ltr">
            {phone}
          </p>
        </div>
      ) : (
        <Input
          id="wa-phone"
          label={A.whatsapp}
          dir="ltr"
          inputMode="tel"
          required
          value={wa}
          onChange={(e) => setWa(e.target.value)}
        />
      )}
      {sent && <p className="text-xs text-ink-muted">{A.whatsappCodeSent}</p>}
      {error && <Alert>{error}</Alert>}
      <Button type="submit" disabled={busy} className="w-full">
        {busy ? m.common.loading : A.whatsappVerify}
      </Button>
    </form>
  );
}
