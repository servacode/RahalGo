"use client";

/**
 * **ادعُ صديقاً** — رابطٌ يُرسَل، ومكافأةٌ بشرطٍ يقوله الإعداد.
 *
 * # ولماذا الرقمُ قبل الفعل
 *
 * **وعدٌ مبهمٌ لا يُحرّك أحداً**: «ادعُ أصدقاءك» لا تعني شيئاً، **و«ادعُ صديقاً
 * واربح ٥٬٠٠٠» تعني.** فيُقال الرقمُ في الصدر لا في الحاشية.
 *
 * # ولا جملةٌ ثابتةٌ عن الشرط
 *
 * كانت الصفحةُ تقول بلفظها «**تُصرف المكافأةُ عند أوّل طلبٍ يُسلَّم لمن دعوتَه —
 * لا عند تسجيله**»، **والإعدادُ يقول عند التسجيل.** فيقرأ الزبونُ شرطاً ويقع
 * غيرُه — **ووعدٌ يخالف ما يقع أسوأُ من ألّا يُوعَد.**
 *
 * فصار الشرطُ يُقرأ من `reward_on`، **والدرجاتُ من الإعدادات نفسِها.**
 *
 * # والجدولُ كلُّه لا الدرجةُ القادمة
 *
 * **الزبونُ يقرّر أن يدعو قبل أن يدعو**: من يرى «الأولى ٥٬٠٠٠» وحدَها لا يعرف
 * **أيستمرّ العطاءُ أم ينقطع** — فيدعو واحداً ويقف.
 *
 * (شهد المالك ٢٠٢٦-٠٨-٠٥: «**ما تكون ثابتة، بحيث يفهم الزبونُ الآلية**: أوّلُ
 * دعوةٍ شقد يربح والثانية والثالثة، **وهل الشرطُ عند إكمال التسجيل أو عند طلب
 * الطرف الآخر**».)
 *
 * # وزرُّ نسخٍ لا رمزٌ يُملى
 *
 * **رمزٌ يُملى بالهاتف يُكتب خطأً**، ورابطٌ يُلصق في واتساب يُضغط. **والفرقُ
 * بينهما هو الفرقُ بين دعوةٍ تصل ودعوةٍ تضيع.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, withPlatform } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  LoadingState,
  PageContainer,
  PageHeader,
  StatGrid,
  StatCard,
  IconLink,
  IconUser,
  IconWallet,
  usePlatform,
} from "@rahalgo/ui";
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
  /** مكافآتُ الأولى والثانية والثالثة، و `rest` ما بعدهنّ. */
  tiers: number[];
  rest: number;
  /** `signup` أو `first_order` — **الشرطُ من الإعداد لا من الشيفرة.** */
  reward_on: string;
}

export default function InvitePage() {

  /** **واسمُ المنصة من الإعدادات** — لا يُكتب في نصّ. (٢٠٢٦-٠٨-٠٧.) */
  const { name: platformName } = usePlatform();
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
    ? `https://wa.me/?text=${encodeURIComponent(`${withPlatform(V.shareText, platformName)}\n${data.link}`)}`
    : "";

  // **والخطأُ لافتةٌ لا سطرٌ أحمرُ في وسط الفراغ.**
  if (error) return <Alert>{error}</Alert>;
  if (!data) return <LoadingState />;

  /**
   * **درجاتُ المكافأة صفّاً صفّاً — ومعها أين يقف صاحبُ الحساب.**
   *
   * **ولا جدولَ حين لا مكافأةَ أصلاً**: منصةٌ أطفأت الأرقامَ كلَّها تعرض
   * أربعةَ أسطرٍ تقول «بلا مكافأة» — **وهو إعلانٌ عن لا شيء**، والصفحةُ تبقى
   * للرابط وحدَه.
   */
  const labels = [V.tier1, V.tier2, V.tier3];
  // **وردٌّ ناقصُ `tiers` يُبيّض الصفحة** — `.map` على غيرِ مصفوفةٍ ترمي،
  // **والرميةُ هنا تُذهب الرابطَ والمكافأةَ معاً.** (كشفه جردُ ٢٠٢٦-٠٨-٠٦.)
  const rows: number[] = Array.isArray(data.tiers) ? data.tiers : [];
  const tiers = [...rows.map((amount, i) => ({ label: labels[i] ?? "", amount })),
    { label: V.tierRest, amount: data.rest }]
    .map((t, i) => ({ ...t, now: i === Math.min(data.invited, 3) }));
  const anyReward = tiers.some((t) => t.amount > 0);
  if (!anyReward) tiers.length = 0;

  return (
    <PageContainer>
      {/* **والرقمُ في الصدر** — «ادعُ أصدقاءك» لا تعني شيئاً. */}
      <PageHeader
        icon={IconLink}
        title={V.title}
        subtitle={
          data.next_reward > 0
            ? V.subtitle.replace("{n}", fmtNum(data.next_reward))
            : V.subtitleNoReward
        }
      />

      <div className="surface p-4">
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

      {/* **جدولُ الدرجات — كم تربح، ومتى.**

          **ومن رأى «الأولى ٥٬٠٠٠» وحدَها لا يعرف أيستمرّ العطاءُ أم ينقطع**،
          فيدعو واحداً ويقف. والدرجاتُ من الإعدادات، **ودرجةٌ صفرٌ تُقال «بلا
          مكافأة» ولا تُخفى**: من عدّ ثلاثاً في الجدول ورأى اثنتين يظنّ في
          الحساب خللاً. */}
      {tiers.length > 0 && (
        <div className="surface p-4">
          <p className="mb-2 font-bold">{V.tiersTitle}</p>
          <ul className="divide-y divide-line">
            {tiers.map((t) => (
              <li key={t.label} className="flex items-center gap-2 py-2 text-sm">
                <span className={t.now ? "font-bold" : "text-ink-muted"}>{t.label}</span>
                {/* **ودَورُك الآن** — الجدولُ يقول أين أنت منه، لا أرقاماً مجرّدة. */}
                {t.now && (
                  <span className="rounded-badge bg-accent-tint px-2 py-0.5 text-2xs font-bold text-accent-text">
                    {V.tierNow}
                  </span>
                )}
                <span
                  dir="ltr"
                  className={`ms-auto tabular-nums ${t.amount > 0 ? "font-bold text-success" : "text-ink-muted"}`}
                >
                  {t.amount > 0 ? `${fmtNum(t.amount)} ${m.common.currency}` : V.tierNone}
                </span>
              </li>
            ))}
          </ul>
          {/* **والشرطُ من الإعداد لا من الشيفرة** — قرأه الزبونُ فوقع غيرُه. */}
          <p className="mt-3 rounded-control bg-field px-3 py-2 text-xs text-ink-muted">
            {data.reward_on === "first_order" ? V.onFirstOrder : V.onSignup}
          </p>
        </div>
      )}

      <StatGrid>
        <StatCard icon={IconUser} label={V.invited} value={fmtNum(data.invited)} />
        {/* **ومن سجّل غيرُ من استحقّ** — والشرطُ قد يكون طلباً لم يقع بعد. */}
        <StatCard icon={IconUser} label={V.ordered} value={fmtNum(data.rewarded)} tone="success" />
        <StatCard
          icon={IconWallet}
          label={V.earned}
          value={`${fmtNum(data.earned)} ${m.common.currency}`}
          tone="accent"
        />
      </StatGrid>
    </PageContainer>
  );
}
