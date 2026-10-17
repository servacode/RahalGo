"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تنبيهُ الطلب الجديد — صوتٌ يتكرّر وشاشةٌ صاحية**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **آيفون يمنع الصوتَ قبل أوّل لمسة** — فلا رنّةَ تُسمع مهما وصل من طلبات
 * حتّى يلمس الصفحة. **فيظهر زرٌّ كبيرٌ «فعّل صوت الطلبات» عند الفتح**،
 * ويختفي حين يسمح المتصفّح. **وصمتٌ لا يُرى أسوأُ من زرٍّ زائد.**
 *
 * **وطلبٌ ينتظر وصاحبُه في شاشةٍ أخرى** يُقال له بلافتةٍ حمراء تقوده إليه.
 *
 * **والشاشةُ تُبقى صاحيةً** (`useWakeLock`) — صفحةٌ نامت شاشتُها لا ترنّ.
 */

import Link from "next/link";
import { usePathname } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconBell, IconBellOff, IconSmartphone } from "@rahalgo/ui";
import { useStoreOrders } from "@/lib/store/orders";
import { useWakeLock } from "@/lib/store/wakelock";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const A = m.store.alert;

export default function OrderAlert() {
  const path = usePathname() ?? "";
  const { pending, soundUnlocked, unlockSound, soundEnabled } = useStoreOrders();
  useWakeLock(true);
  const onOrders = path.startsWith("/store/orders");

  return (
    <>
      {/* **ومن كتم الرنّةَ بيده لا يُلحّ عليه زرُّ التفعيل** — يُقال له
          أنّها مكتومة وأين تُشغَّل (`/store/account`). */}
      {!soundEnabled && (
        <Link
          href="/store/account"
          className="flex items-center gap-2 rounded-card border border-warning-edge bg-warning-tint p-3 text-sm text-warning"
        >
          <IconBellOff size={18} className="shrink-0" />
          {A.muted}
        </Link>
      )}
      {soundEnabled && !soundUnlocked && (
        <button
          type="button"
          onClick={unlockSound}
          data-testid="enable-order-sound"
          className="flex w-full items-center justify-center gap-3 rounded-card bg-accent px-4 py-5 text-on-bright shadow-e3 hover:opacity-90"
        >
          <IconBell size={28} className="shrink-0" />
          <span className="flex flex-col items-start text-start">
            <span className="heading-card">{A.enable}</span>
            <span className="text-xs">{A.enableHint}</span>
          </span>
        </button>
      )}
      {pending.length > 0 && !onOrders && (
        <Link
          href="/store/orders"
          className="flex items-center gap-3 rounded-card border border-danger-edge bg-danger-tint p-3 text-danger motion-safe:animate-pulse"
        >
          <IconBell size={22} className="shrink-0" />
          <span className="min-w-0 flex-1">
            <span className="block font-bold">{A.newOrder}</span>
            <span className="block text-xs">{fill(A.newOrderBody, { n: pending.length })}</span>
          </span>
          <span className="shrink-0 rounded-control bg-danger-solid px-3 py-2 text-xs font-bold text-on-solid">
            {A.open}
          </span>
        </Link>
      )}
      <p className="flex items-center gap-2 text-xs text-ink-muted">
        <IconSmartphone size={14} className="shrink-0" />
        {A.keepOpen}
      </p>
    </>
  );
}
