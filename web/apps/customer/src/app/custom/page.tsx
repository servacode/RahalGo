"use client";

/**
 * **طلبٌ خاصّ — ما ليس في المنصّة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «الزبون يريد أكلاً من مطعمٍ محدَّد أو شيئاً من
 *  سوقٍ غير موجودٍ بالمتجر».)
 *
 * # ولا يُسأل عن سعرٍ ولا متجر
 *
 * **هو يطلب ما لا نعرف سعرَه** — والسائقُ يشتريه ويتّفق معه في المحادثة بعد
 * الإسناد. **وسؤالٌ لا جوابَ له يُوقف من يملأ نموذجاً.**
 *
 * # والعنوانُ يُملأ من دفتره
 *
 * **ومن له عنوانٌ واحد لا يُسأل عنه** — القاعدةُ نفسُها في السلّة.
 */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Select,
  Textarea,
  PageContainer,
  PageHeader,
  IconOrder,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.site.custom;

interface SavedAddress {
  id: string;
  address_text: string;
  lat: number;
  lng: number;
  is_default: boolean;
}

export default function CustomOrderPage() {
  const router = useRouter();
  const [saved, setSaved] = useState<SavedAddress[]>([]);
  const [pickedID, setPickedID] = useState("");
  const [request, setRequest] = useState("");
  const [payment, setPayment] = useState("cash");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api<SavedAddress[]>("/api/v1/my/addresses")
      .then((a) => {
        setSaved(a);
        const def = a.find((x) => x.is_default) ?? a[0];
        if (def) setPickedID(def.id);
      })
      .catch(() => undefined);
  }, []);

  /** **وعنوانٌ يُكتب الآن** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «ربّما يريد عنواناً
      مختلفاً»). **ومن يطلب لغيره أو من مكانٍ طارئٍ لا يجد عنوانَه في دفتره.** */
  const [freeText, setFreeText] = useState("");
  const useFree = pickedID === "__new";
  const picked = useFree
    ? { address_text: freeText.trim(), lat: saved[0]?.lat ?? 0, lng: saved[0]?.lng ?? 0 }
    : saved.find((a) => a.id === pickedID);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!request.trim() || !picked?.address_text || busy) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/orders/custom", {
        method: "POST",
        body: JSON.stringify({
          request: request.trim(),
          address_text: picked.address_text,
          lat: picked.lat,
          lng: picked.lng,
          payment_method: payment,
        }),
      });
      router.push("/orders");
    } catch (err) {
      const key =
        err instanceof ApiError ? (err.body.message_key ?? "").split(".").pop() ?? "" : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <PageContainer>
      <PageHeader icon={IconOrder} title={C.title} subtitle={C.subtitle} />

      <form onSubmit={submit} className="surface space-y-4 p-5">
        <Textarea
          id="custom-request"
          label={C.what}
          rows={5}
          required
          maxLength={600}
          placeholder={C.placeholder}
          value={request}
          onChange={(e) => setRequest(e.target.value)}
        />

        {/* **والعنوانُ من دفتره** — ومن لا عنوانَ له يُرسَل ليضيف واحداً،
            **لا يُترك أمام قائمةٍ فارغةٍ لا يفهم لماذا هي فارغة.** */}
        {saved.length === 0 ? (
          <Alert tone="warning">{C.noAddress}</Alert>
        ) : (
          <Select
            id="custom-address"
            label={C.address}
            value={pickedID}
            onChange={(e) => setPickedID(e.target.value)}
          >
            {saved.map((a) => (
              <option key={a.id} value={a.id}>
                {a.address_text}
              </option>
            ))}
            <option value="__new">{C.newAddress}</option>
          </Select>
        )}

        {useFree && (
          <Textarea
            id="custom-free-address"
            label={C.newAddressText}
            rows={2}
            required
            value={freeText}
            onChange={(e) => setFreeText(e.target.value)}
          />
        )}

        {/* **وطريقةُ الدفع** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩).

            **والمبلغُ يُخصم عند التسليم لا الآن**: لا يُعرف حتّى يتّفق السائقُ
            معك — **ومحفظةٌ تُخصم بتقديرٍ ثمّ يُسوّى الفرقُ دَينٌ بلا دفتر.** */}
        <Select
          id="custom-payment"
          label={m.site.cart.payment}
          value={payment}
          onChange={(e) => setPayment(e.target.value)}
        >
          <option value="cash">{m.orders.payment.cash}</option>
          <option value="wallet">{m.orders.payment.wallet}</option>
        </Select>

        {/* **وكيف يمضي الأمرُ يُقال قبل الإرسال** — من لا يعرف ماذا يقع بعد
            ضغطته يتردّد، **أو يضغط ثمّ ينتظر ما لا يعرف شكلَه.** */}
        <p className="rounded-control bg-field px-3 py-2 text-xs leading-relaxed text-ink-muted">
          {C.how}
        </p>

        {error && <Alert>{error}</Alert>}

        <Button type="submit" disabled={busy || !request.trim() || !picked?.address_text} className="w-full">
          {busy ? m.common.loading : C.send}
        </Button>
      </form>
    </PageContainer>
  );
}
