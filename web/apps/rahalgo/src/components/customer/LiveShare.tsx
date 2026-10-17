"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«الكابتن بالطريق إليك» — نافذةٌ فوق أيّ شاشة، ولافتةُ المشاركة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `LiveShareDialog.kt`.) **جوابان لا ثالث**: «شارك موقعي» أخضرُ
 * و«لا، العنوان صحيح» أحمر — **وكلاهما يُذكر فلا يُسأل عن الطلب نفسِه
 * ثانيةً.** (ملاحظةُ المالك ٢٠٢٦-١٠-١٦: «لازم أزرار أخضر وأحمر ليبيّن إنه
 * يختار».)
 *
 * **ويُقال صراحةً إنّ المتصفّحَ يشارك ما دامت الصفحةُ مفتوحة** — التطبيقُ
 * يحمل خدمةً أماميّةً تعيش في الخلفيّة، **والمتصفّحُ لا.**
 */

import { createPortal } from "react-dom";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { Alert, Button, IconLocation, usePlatform } from "@rahalgo/ui";
import { useActiveOrders } from "@/lib/customer/active";
import { pendingAsk, useLiveShare } from "@/lib/customer/live";
import { quotePending } from "./orderParts";

const m = getMessages(defaultLocale);
const L = m.shop.live;

/** **السؤال** — يُركَّب في القشرة، **ولا يُسأل وتأكيدُ السعر مفتوح.** */
export function LiveAsk() {
  const { open } = useActiveOrders();
  const live = useLiveShare();
  const { name } = usePlatform();
  if (!open || typeof document === "undefined") return null;
  if (open.some(quotePending)) return null;
  const o = live.activeId ? null : pendingAsk(open, live.decided);
  if (!o) return null;
  return createPortal(
    <div className="fixed inset-0 z-[80] flex items-end justify-center scrim p-0 sm:items-center sm:p-4">
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="live-t"
        className="w-full surface-modal p-5 sm:max-w-md"
      >
        <h2 id="live-t" className="heading-card mb-2">
          {L.askTitle}
        </h2>
        <p className="text-sm">{withPlatform(L.askBody, name)}</p>
        <p className="mt-2 text-xs text-ink-muted">{L.browserNote}</p>
        <div className="mt-4 flex flex-col gap-2">
          <button
            type="button"
            onClick={() => {
              live.decide(o.id);
              live.start(o.id);
            }}
            className="w-full rounded-control bg-success px-4 py-3 text-sm font-bold text-on-bright transition-opacity hover:opacity-90"
          >
            {L.share}
          </button>
          <Button variant="danger" className="w-full py-3" onClick={() => live.decide(o.id)}>
            {L.no}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  );
}

/** **لافتةُ «عم تشارك موقعك»** — تُرى ما دامت المشاركة، ومعها زرُّ الإيقاف. */
export function LiveBanner() {
  const live = useLiveShare();
  if (live.problem && !live.activeId) {
    return <Alert tone="warning">{live.problem === "denied" ? L.denied : L.unsupported}</Alert>;
  }
  if (!live.activeId) return null;
  return (
    <div className="surface-inset flex flex-wrap items-center gap-3 p-3">
      <IconLocation size={20} className="shrink-0 text-success motion-safe:animate-pulse" />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-bold">{L.sharing}</p>
        <p className="text-xs text-ink-muted">
          {L.autoStop} · {L.browserNote}
        </p>
      </div>
      <Button variant="danger" onClick={() => live.stop("user")}>
        {L.stop}
      </Button>
    </div>
  );
}

/** **زرُّ «شارك موقعي» في تفاصيل الطلب** — لمن رفض ثمّ غيّر رأيه، أو أعاد فتحَ الصفحة. */
export function LiveStartButton({ orderId, status }: { orderId: string; status: string }) {
  const live = useLiveShare();
  if (live.activeId === orderId) return null;
  if (!["picked_up", "on_the_way", "at_dropoff"].includes(status)) return null;
  return (
    <div className="space-y-1">
      <Button
        variant="secondary"
        className="flex w-full items-center justify-center gap-2"
        onClick={() => {
          live.decide(orderId);
          live.start(orderId);
        }}
      >
        <IconLocation size={16} />
        {L.start}
      </Button>
      <p className="text-xs text-ink-muted">{L.browserNote}</p>
    </div>
  );
}
