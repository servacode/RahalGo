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
import { getMessages, defaultLocale, errorText, fmtNum, withPlatform } from "@rahalgo/i18n";
import { Button, CopyCode, LoadingState, Money, ReloadState, usePlatform, IconCopy, IconShare } from "@rahalgo/ui";
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
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <div>
        <ShopPageHeader title={V.title} />
        <p className="text-sm text-ink-muted">
          {r.next_reward > 0 ? fill(V.subtitle, { n: fmtNum(r.next_reward) }) : V.subtitleNoReward}
        </p>
      </div>

      <section className="surface flex flex-col gap-3 p-4">
        <CopyCode code={r.code} title={m.shop.invite.code} />
        <p className="text-xs text-ink-muted">{V.yourLink}</p>
        <p dir="ltr" className="break-all rounded-control bg-field px-3 py-2 text-sm">
          {r.link}
        </p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <Button variant="secondary" className="flex items-center justify-center gap-2" onClick={copy}>
            <IconCopy size={16} />
            {copied ? V.copied : V.copy}
          </Button>
          <a
            href={`https://wa.me/?text=${encodeURIComponent(share)}`}
            target="_blank"
            rel="noreferrer"
            className="flex items-center justify-center gap-2 rounded-control bg-success px-4 py-2 text-sm font-bold text-on-bright hover:opacity-90"
          >
            <IconShare size={16} />
            {V.shareWhatsApp}
          </a>
        </div>
        <p className="text-xs text-ink-muted">{r.reward_on === "first_order" ? V.onFirstOrder : V.onSignup}</p>
      </section>

      <dl className="surface grid grid-cols-1 gap-2 p-4 text-center sm:grid-cols-3">
        <div>
          <dt className="text-xs text-ink-muted">{V.invited}</dt>
          <dd className="figure">{fmtNum(r.invited)}</dd>
        </div>
        <div>
          <dt className="text-xs text-ink-muted">{V.ordered}</dt>
          <dd className="figure">{fmtNum(r.rewarded)}</dd>
        </div>
        <div>
          <dt className="text-xs text-ink-muted">{V.earned}</dt>
          <dd className="figure text-accent-text">
            <Money value={r.earned} />
          </dd>
        </div>
      </dl>

      {r.tiers.length > 0 && (
        <section className="surface flex flex-col gap-2 p-4">
          <h2 className="heading-card">{V.tiersTitle}</h2>
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
