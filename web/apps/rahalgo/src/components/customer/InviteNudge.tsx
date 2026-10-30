"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«ادعُ صديقاً» ظاهرةً** (قرارُ المالك ٢٠٢٦-١٠-٣٠، مرآةُ `InviteNudge.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أوّلُ سطرٍ في «ما يخصّني» بلونٍ ومبلغ**، **وبطاقةٌ تحت البحث تُخفى بـ✕ للأبد**،
 * **وبعد أوّل طلبٍ يصله تعود مرّةً بجملة «عجبتك الخدمة؟».** والمبلغُ من المحرّك
 * (`auth/referral` ← `next_reward`) — **وبلا مبلغٍ لا يُعرض شيء.**
 */

import Link from "next/link";
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, withPlatform } from "@rahalgo/i18n";
import { Button, IconCatGifts, IconClose, IconNext, usePlatform } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { readJSON, writeJSON } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import type { Referral } from "@/lib/customer/types";
import { useShopToast } from "./toast";

const m = getMessages(defaultLocale);
const I = m.shop.invite;

const HOME_HIDDEN = "rahalgo_invite_home_hidden";
const AFTER_HIDDEN = "rahalgo_invite_after_hidden";
const DELIVERED = "rahalgo_invite_delivered";

/** **وصله أوّلُ طلب** — فتعود البطاقةُ مرّةً بجملة «عجبتك الخدمة؟». */
export function markInviteDelivered(): void {
  if (!readJSON<boolean>(DELIVERED, false)) writeJSON(DELIVERED, true);
}

/** **رمزُ الدعوة ومبلغُها** — لصاحب حسابٍ وحدَه. */
export function useReferral(): Referral | null {
  const { user } = useAuth();
  const [r, setR] = useState<Referral | null>(null);
  useEffect(() => {
    if (!user) {
      setR(null);
      return;
    }
    let alive = true;
    api<Referral>("/api/v1/auth/referral")
      .then((x) => alive && setR(x))
      .catch(() => undefined);
    return () => {
      alive = false;
    };
  }, [user]);
  return r;
}

/** **أوّلُ سطرٍ في «ما يخصّني»** — بصبغةٍ وأيقونةِ هديّةٍ ومبلغ. */
export function InviteMenuRow() {
  const r = useReferral();
  const label = r && r.next_reward > 0 ? fill(I.menu, { n: fmtNum(r.next_reward) }) : m.shop.account.invite;
  return (
    <Link
      href="/invite"
      className="mx-2 my-1 flex items-center gap-3 rounded-control bg-accent-tint px-3 py-3 text-sm font-bold text-accent-text hover:opacity-90"
    >
      <IconCatGifts size={18} className="shrink-0" />
      <span className="flex-1">{label}</span>
      <IconNext size={15} />
    </Link>
  );
}

/** **بطاقةُ الرئيسيّة تحت البحث** — «شارك رمزك» و✕ تُخفيها للأبد على هذا المتصفّح. */
export function InviteHomeCard() {
  const r = useReferral();
  const { name } = usePlatform();
  const toast = useShopToast();
  const [state, setState] = useState<{ home: boolean; after: boolean; delivered: boolean } | null>(null);

  useEffect(() => {
    setState({
      home: readJSON<boolean>(HOME_HIDDEN, false),
      after: readJSON<boolean>(AFTER_HIDDEN, false),
      delivered: readJSON<boolean>(DELIVERED, false),
    });
  }, []);

  if (!r || !state || r.next_reward <= 0 || !r.code) return null;
  const after = state.delivered && !state.after;
  if (!after && state.home) return null;

  const n = fmtNum(r.next_reward);
  const shareText = fill(withPlatform(m.customer.invite.shareText, name), { code: r.code, link: r.link });

  const share = async () => {
    try {
      if (typeof navigator !== "undefined" && navigator.share) {
        await navigator.share({ text: shareText });
        return;
      }
      await navigator.clipboard.writeText(shareText);
      toast(I.copied, "success");
    } catch {
      /* **أُغلقت نافذةُ المشاركة** — لا شيء يُقال. */
    }
  };

  const hide = () => {
    writeJSON(after ? AFTER_HIDDEN : HOME_HIDDEN, true);
    setState({ ...state, [after ? "after" : "home"]: true });
  };

  return (
    <div className="flex items-center gap-3 rounded-card bg-accent-tint p-3">
      <IconCatGifts size={22} className="shrink-0 text-accent-text" />
      <Link href="/invite" className="min-w-0 flex-1 text-sm font-bold">
        {fill(after ? I.after : I.home, { n })}
      </Link>
      <Button onClick={() => void share()} className="shrink-0">
        {I.shareCode}
      </Button>
      <button
        type="button"
        aria-label={I.hide}
        onClick={hide}
        className="taparea flex h-8 w-8 shrink-0 items-center justify-center rounded-control text-ink-muted hover:bg-row-hover"
      >
        <IconClose size={16} />
      </button>
    </div>
  );
}
