"use client";

/**
 * **ادعُ صديقاً** — رابطٌ يُرسَل، ومكافأةٌ عند أوّل طلبٍ له.
 *
 * # ولماذا الرقمُ قبل الفعل
 *
 * **وعدٌ مبهمٌ لا يُحرّك أحداً**: «ادعُ أصدقاءك» لا تعني شيئاً، **و«ادعُ صديقاً
 * واربح ٥٬٠٠٠» تعني.** فيُقال الرقمُ في الصدر لا في الحاشية.
 *
 * # وزرُّ نسخٍ لا رمزٌ يُملى
 *
 * **رمزٌ يُملى بالهاتف يُكتب خطأً**، ورابطٌ يُلصق في واتساب يُضغط. **والفرقُ
 * بينهما هو الفرقُ بين دعوةٍ تصل ودعوةٍ تضيع.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { Button, LoadingState, StatGrid, StatCard, IconLink, IconUser, IconWallet } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const V = m.customer.invite;

interface Referral {
  code: string;
  link: string;
  invited: number;
  rewarded: number;
  earned: number;
  next_reward: number;
}

export default function InvitePage() {
  const [data, setData] = useState<Referral | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api<Referral>("/api/v1/auth/referral")
      .then(setData)
      .catch(() => setError(m.errors.internal));
  }, []);

  useEffect(load, [load]);

  async function copy() {
    if (!data) return;
    try {
      await navigator.clipboard.writeText(data.link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // **ونسخٌ لا يعمل لا يُسقط الشاشة** — الرابطُ معروضٌ ويُحدَّد باليد.
      setCopied(false);
    }
  }

  /** **وواتسابُ أوّلاً** — هو حيث يعيش الناسُ هنا، لا البريد. */
  const waHref = data
    ? `https://wa.me/?text=${encodeURIComponent(`${V.shareText}\n${data.link}`)}`
    : "";

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!data) return <LoadingState />;

  return (
    <div className="mx-auto w-full max-w-2xl space-y-5 p-4">
      <div>
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <IconLink size={20} className="text-ink-muted" />
          {V.title}
        </h1>
        {/* **والرقمُ في الصدر** — «ادعُ أصدقاءك» لا تعني شيئاً. */}
        <p className="mt-1 text-sm text-ink-muted">
          {data.next_reward > 0
            ? V.subtitle.replace("{n}", fmtNum(data.next_reward))
            : V.subtitleNoReward}
        </p>
      </div>

      <div className="rounded-card border border-line bg-surface p-4">
        <p className="text-xs text-ink-muted">{V.yourLink}</p>
        <p className="mt-1 break-all font-mono text-sm" dir="ltr">
          {data.link}
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          <Button onClick={() => void copy()}>{copied ? V.copied : V.copy}</Button>
          <a
            href={waHref}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center rounded-control border border-line px-4 py-2 text-sm font-medium transition-colors hover:border-accent"
          >
            {V.shareWhatsApp}
          </a>
        </div>
      </div>

      <StatGrid>
        <StatCard icon={IconUser} label={V.invited} value={fmtNum(data.invited)} />
        {/* **ومن سجّل غيرُ من طلب** — والمكافأةُ على الطلب لا على الرقم. */}
        <StatCard icon={IconUser} label={V.ordered} value={fmtNum(data.rewarded)} tone="success" />
        <StatCard
          icon={IconWallet}
          label={V.earned}
          value={`${fmtNum(data.earned)} ${m.common.currency}`}
          tone="accent"
        />
      </StatGrid>

      <p className="rounded-control bg-page px-3 py-2 text-xs text-ink-muted">{V.hint}</p>
    </div>
  );
}
