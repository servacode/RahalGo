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
import { getMessages, defaultLocale } from "@rahalgo/i18n";
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
    } catch {
      setError(m.errors.internal);
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
    } catch {
      setError(m.errors.validation);
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
