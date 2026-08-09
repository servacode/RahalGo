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
 * # والعنوانُ يُكتب ويُؤشَّر — كما في السلّة حرفاً بحرف
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الخريطةُ والعنوانُ غيرُ موجودين مثل ما طلبت
 *  لتحديد المكان والعنوان».)
 *
 * **كانت قائمةَ عناوينَ محفوظةٍ وحدَها** — **ومن لا عنوانَ محفوظاً له يقف
 * أمام إنذارٍ يقول «أضف عنواناً في حسابك أوّلاً»** فيخرج من الصفحة ليعود
 * إليها. **وأوّلُ طلبٍ لزبونٍ جديدٍ هو بالضبط الحالُ التي لا عنوانَ فيها.**
 *
 * **والدبّوسُ ليس زينةً**: السائقُ يمشي إليه — **ونصُّ العنوان يُقرأ ولا
 * يُلاحَق**، «خلف الجامع» تكفي من يعرف الحيّ ولا تكفي من لا يعرفه.
 *
 * **والمكوّناتُ نفسُها التي في السلّة** — `AddressBook` و`PickMap`:
 * **وشاشتان تسألان العنوانَ بطريقتين تُربكان من ملأ إحداهما.**
 */

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Input,
  Select,
  Textarea,
  AddressBook,
  PageContainer,
  PageHeader,
  IconOrder,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

/** **الخريطةُ ثقيلةٌ ولا تعمل على الخادم** — تُحمَّل عند الحاجة كما في السلّة. */
const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});

const m = getMessages(defaultLocale);
const C = m.site.custom;

/** **مركزُ الرقّة** — نقطةُ البدء لمن لا عنوانَ محفوظاً له. */
const RAQQA = { lat: 35.9528, lng: 39.0079 };

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
  /** **العنوانُ المحفوظُ المختار** — ويُفرَّغ متى عُدّل النصُّ بيده. */
  const [pickedID, setPickedID] = useState("");
  const [address, setAddress] = useState("");
  const [lat, setLat] = useState(RAQQA.lat);
  const [lng, setLng] = useState(RAQQA.lng);
  const [request, setRequest] = useState("");
  const [payment, setPayment] = useState("cash");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api<SavedAddress[]>("/api/v1/my/addresses")
      .then((a) => {
        setSaved(a);
        // **ومن له عنوانٌ محفوظٌ لا يُسأل عنه** — يُملأ النصُّ والدبّوسُ معاً،
        // **ونصٌّ بلا دبّوسٍ يرسل السائقَ إلى مركز المدينة.**
        const def = a.find((x) => x.is_default) ?? a[0];
        if (def) {
          setPickedID(def.id);
          setAddress(def.address_text);
          setLat(def.lat);
          setLng(def.lng);
        }
      })
      // @empty-ok **ولا عنوانَ محفوظ** — يكتبه الآن ويؤشّر على الخريطة.
      .catch(() => undefined);
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!request.trim() || !address.trim() || busy) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/orders/custom", {
        method: "POST",
        body: JSON.stringify({
          request: request.trim(),
          address_text: address.trim(),
          lat,
          lng,
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

        {/* **ودفترُ عناوينه أوّلاً** — من حفظ عنوانَه لا يُطلب منه رسمُ
            دبّوسه ثانيةً، **والاختيارُ يملأ النصَّ والدبّوسَ معاً.** */}
        {saved.length > 0 && (
          <div>
            <p className="mb-2 text-xs font-medium">{m.site.addresses.title}</p>
            <AddressBook
              api={api}
              selectedID={pickedID}
              onPick={(a) => {
                setPickedID(a.id);
                setAddress(a.address_text);
                setLat(a.lat);
                setLng(a.lng);
              }}
            />
          </div>
        )}

        <Input
          id="custom-address"
          label={m.site.cart.address}
          required
          value={address}
          /* **وتعديلُ النصّ يُلغي علامةَ المحفوظ** — علامةٌ على عنوانٍ لم يعد
             هو المستعمَل **تقول غيرَ الواقع.** */
          onChange={(e) => {
            setAddress(e.target.value);
            setPickedID("");
          }}
          placeholder={m.site.cart.addressPlaceholder}
        />

        {/* **والدبّوسُ يُحرَّك** — السائقُ يمشي إليه، **ونصُّ العنوان يُقرأ
            ولا يُلاحَق.** */}
        <div>
          <p className="mb-1.5 text-xs text-ink-muted">{m.site.cart.pinHint}</p>
          <div className="overflow-hidden rounded-control border border-line">
            <PickMap
              lat={lat}
              lng={lng}
              onPick={(la, ln) => {
                setLat(la);
                setLng(ln);
                setPickedID("");
              }}
            />
          </div>
        </div>

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

        <Button type="submit" disabled={busy || !request.trim() || !address.trim()} className="w-full">
          {busy ? m.common.loading : C.send}
        </Button>
      </form>
    </PageContainer>
  );
}
