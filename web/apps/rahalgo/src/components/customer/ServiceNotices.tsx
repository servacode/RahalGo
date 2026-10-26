"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لافتةُ «الطلبُ متوقّف» — ورسالةُ المالك كما كتبها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مرآةٌ لـ`ServiceClosedBanner` في أندرويد** — تقرأ `ordering` من
 * `/public/platform`: **خارجَ الدوام، أو إيقافٌ مؤقّت، أو قبلَ الإطلاق.**
 * **والتصفّحُ يبقى مفتوحاً** — اللافتةُ تقول ولا تمنع، **والمحرّكُ يردّ
 * الطلبَ إن أُرسل** (`requireOrdering`).
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, usePlatform } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCustomer } from "@/lib/customer/context";
import { serviceText, demandKind, demandCta, demandDone } from "@/lib/customer/texts";
import type { Address, Availability } from "@/lib/customer/types";

const m = getMessages(defaultLocale);
const S = m.shop;

/** **أيُستقبَل الطلبُ الآن؟** — وما يُقال إن لم يُستقبَل. */
export function useOrderingState(): { open: boolean; text: string } {
  const { platform } = useCustomer();
  if (!platform) return { open: true, text: "" };
  if (!platform.launch.customer_orders) {
    return { open: false, text: platform.launch.notice.trim() || serviceText("launch_closed") };
  }
  const o = platform.ordering;
  if (o.ordering_available) return { open: true, text: "" };
  return { open: false, text: serviceText(o.reason || "platform_closed_now", o.message, o.next_available_at) };
}

export function OrderingBanner() {
  const { open, text } = useOrderingState();
  if (open) return null;
  return (
    <Alert tone="warning" title={S.service.closedTitle}>
      {text}
    </Alert>
  );
}

/**
 * **سببُ تعذّر الخدمة عند هذا العنوان** — ومعه «أضِف منطقتي» حيث يصلح.
 *
 * **والنيّةُ لا تُرسَل حكماً**: المحرّكُ يعيد قراءةَ الإتاحة ويقرّر
 * (`/api/v1/demand`)، **وما يُرسَل هنا يُقارَن بما يقضيه.**
 */
export function AvailabilityNotice({ av, address }: { av: Availability; address: Address | null }) {
  const { name } = usePlatform();
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");
  if (av.available) return null;
  const kind = demandKind(av.reason);
  return (
    <div className="space-y-2">
      <Alert tone="error">{serviceText(av.reason, av.message, av.next_available_at)}</Alert>
      {kind && address && !done && (
        <Button
          variant="secondary"
          className="w-full"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            setError("");
            try {
              await api("/api/v1/demand", {
                method: "POST",
                body: JSON.stringify({
                  lat: address.lat,
                  lng: address.lng,
                  kind,
                  address_text: address.address_text,
                }),
              });
              setDone(true);
            } catch (err) {
              setError(errorText(err));
            } finally {
              setBusy(false);
            }
          }}
        >
          {busy ? S.service.sending : demandCta(kind, name)}
        </Button>
      )}
      {done && <Alert tone="success">{demandDone(kind, name)}</Alert>}
      {error && <Alert>{error}</Alert>}
    </div>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أتُضاف السلّةُ الآن؟ — قبل الإضافة لا عند الإرسال**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٢٦: «لازم قبل ما يضيف للسلة أصلاً يطلعله… مو يضيف للسلة ليتفاجأ إنه
 * ما يقدر يطلب».) مرآةُ `addActionFor` و`rememberAddBlocked` في التطبيق:
 *
 *	متوقّفٌ الآن (دوامٌ · إيقافٌ · قبل الإطلاق)   ⇒ `closed`
 *	بلا عنوانِ توصيل                            ⇒ `need_address` (يُسأل أوّلاً)
 *	عنوانٌ لا يُخدَم (خارجَ التغطية · منطقتُه مغلقة) ⇒ `blocked`
 *
 * **والخادمُ يبقى الحارسَ الأخير** (`requireOrdering` · `ErrOutOfZone`) — هذا ما يُقال قبله.
 */
export type AddGate =
  | { kind: "ok" }
  | { kind: "closed"; text: string }
  | { kind: "need_address" }
  | { kind: "blocked"; av: Availability };

export function useAddGate(): AddGate {
  const ordering = useOrderingState();
  const { address, availability } = useCustomer();
  if (!ordering.open) return { kind: "closed", text: ordering.text };
  if (!address) return { kind: "need_address" };
  if (availability && !availability.available) return { kind: "blocked", av: availability };
  return { kind: "ok" };
}

/** **لافتةُ «عنوانُك لا نصله» فوق كلّ صفحة** — كـ`ServiceBlockNotice` في التطبيق، ومعها «اطلب تغطية منطقتي». */
export function ServiceBlockBanner() {
  const ordering = useOrderingState();
  const { address, availability } = useCustomer();
  // **ولافتتان لا تتراكبان**: المتوقّفُ يقوله `OrderingBanner` وحدَه.
  if (!ordering.open || !address || !availability || availability.available) return null;
  return <AvailabilityNotice av={availability} address={address} />;
}
