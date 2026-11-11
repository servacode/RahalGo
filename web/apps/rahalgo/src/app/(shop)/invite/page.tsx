"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ادع صديقاً — رمزُه ورابطُه وما ناله** (`/auth/referral`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والرابطُ يُبنى في الخادم لا في المتصفّح** — عنوانُ الموقع يتبدّل من نشرٍ
 * إلى نشر. **والمكافأةُ ودرجاتُها من اللوحة** — تُعرض كما قالها المحرّك.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum, withPlatform } from "@rahalgo/i18n";
import { Button, CopyCode, LoadingState, Money, ReloadState, StatTile, usePlatform, IconCopy, IconShare } from "@rahalgo/ui";
import { OUTLINE_CARD } from "@/components/customer/kit";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { fill } from "@/lib/customer/texts";
import type { Referral } from "@/lib/customer/types";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const m = getMessages(defaultLocale);
const V = m.customer.invite;

export default function InvitePage() {
  const { user, loading } = useAuth();
  const { name } = usePlatform();
  const [r, setR] = useState<Referral | null>(null);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  const load = useCallback(() => {
    if (!user) return;
    setError("");
    api<Referral>("/api/v1/auth/referral")
      .then(setR)
      .catch((err) => setError(errorText(err)));
  }, [user]);

  useEffect(load, [load]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  if (!r) return error ? <ReloadState label={error} onRetry={load} /> : <LoadingState variant="list" rows={3} />;

  // **والمشاركةُ جملةٌ فيها الرمزُ والرابط** (مرآةُ `inv_share_text`) — رابطٌ وحدَه على واتساب لا يُعرف ما هو.
  const share = fill(withPlatform(V.shareText, name), { code: r.code, link: r.link });
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(r.link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* **ومتصفّحٌ يمنع النسخ** — الرابطُ ظاهرٌ تحت يده. */
    }
  };
  const tierLabels = [V.tier1, V.tier2, V.tier3];

  return (
    <div className="mx-auto flex w-full flex-col gap-4 max-w-none lg:grid lg:grid-cols-2 lg:items-start lg:gap-6">
      <div className="lg:col-span-2">
        <ShopPageHeader title={V.title} />
        <p className="text-sm text-ink-muted">
          {r.next_reward > 0 ? fill(V.subtitle, { n: fmtNum(r.next_reward) }) : V.subtitleNoReward}
        </p>
      </div>

      <section className={`${OUTLINE_CARD} flex flex-col gap-3 lg:row-span-2`}>
        {/* صندوقُ الرمز: الرمزُ بخطٍّ كبير (لاتينيّ) وزرُّ نسخٍ صغيرٌ جنبه. */}
        <div className="flex items-center justify-between gap-rg-md rounded-rg-md bg-field px-rg-md py-rg-sm">
          <div className="min-w-0">
            <p className="text-caption text-ink-muted">{m.shop.invite.code}</p>
            <p dir="ltr" className="truncate text-price-lg font-mono text-start">
              {r.code}
            </p>
          </div>
          <CopyCode code={r.code} title={m.shop.invite.code} />
        </div>
        <p className="text-caption text-ink-muted">{V.yourLink}</p>
        <p dir="ltr" className="break-all rounded-rg-md bg-field px-rg-md py-rg-sm text-body">
          {r.link}
        </p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <Button variant="secondary" size="sm" className="flex min-h-12 items-center justify-center gap-2" onClick={copy}>
            <IconCopy size={16} />
            {copied ? V.copied : V.copy}
          </Button>
          <a
            href={`https://wa.me/?text=${encodeURIComponent(share)}`}
            target="_blank"
            rel="noreferrer"
            className="flex min-h-12 items-center justify-center gap-2 rounded-control bg-success px-4 py-2 text-sm font-bold text-on-bright hover:opacity-90"
          >
            <IconShare size={16} />
            {V.shareWhatsApp}
          </a>
        </div>
        <p className="text-caption text-ink-muted">{r.reward_on === "first_order" ? V.onFirstOrder : V.onSignup}</p>
      </section>

      {/* مربّعاتُ الأرقام المشتركة (`StatTile`) — الاسمُ فوق والرقمُ تحت. */}
      <div className="grid grid-cols-2 gap-rg-sm sm:grid-cols-3">
        <StatTile label={V.invited} value={r.invited} />
        <StatTile label={V.ordered} value={r.rewarded} />
        <div className="col-span-2 sm:col-span-1"><StatTile label={V.earned} value={fmtMoney(r.earned)} /></div>
      </div>

      {r.tiers.length > 0 && (
        <section className={`${OUTLINE_CARD} flex flex-col gap-2`}>
          <h2 className="text-section-title">{V.tiersTitle}</h2>
          <ul className="flex flex-col divide-y divide-line-soft text-sm">
            {r.tiers.slice(0, 3).map((amount, i) => (
              <li key={i} className="flex items-center justify-between gap-2 py-2">
                <span>{tierLabels[i]}</span>
                <span className="flex items-center gap-2">
                  {i === r.invited && <span className="text-xs text-accent-text">{V.tierNow}</span>}
                  {amount > 0 ? <Money value={amount} /> : <span className="text-ink-muted">{V.tierNone}</span>}
                </span>
              </li>
            ))}
            <li className="flex items-center justify-between gap-2 py-2">
              <span>{V.tierRest}</span>
              {r.rest > 0 ? <Money value={r.rest} /> : <span className="text-ink-muted">{V.tierNone}</span>}
            </li>
          </ul>
        </section>
      )}
    </div>
  );
}
