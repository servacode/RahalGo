"use client";

/**
 * **هدفي ومكافآتي** — الشاشةُ التي يراها السائقُ والمندوب.
 *
 * # لماذا شاشةٌ لا سطرٌ في المحفظة
 *
 * المحفظةُ دفترٌ: تقول «٥٠٬٠٠٠ في الثالث من الشهر». **ولا تقول لماذا، ولا كم
 * بقي على الهدف، ولا أنّ ثمّة هدفاً أصلاً.**
 *
 * **وحافزٌ لا يُرى لا يحفّز**: من لا يعرف أنّه على بُعد ثلاثةِ طلباتٍ من
 * مكافأةٍ لا يسعى إليها.
 *
 * # والعقوبةُ تُعرض كما تُعرض المكافأة
 *
 * **ومن عوقب ولا يعلم لا يُصلح شيئاً** — يُخصم منه فيُفاجأ، **ويظنّ الظلمَ
 * حيث كان خبر.** (وهي القاعدةُ نفسُها التي جعلت الشكاوى تُعرض عليه.)
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import { PageContainer, PageHeader, StatGrid, StatCard, EmptyState, LoadingState } from "./layout";
import { Badge, Button } from "./components";
import { IconStar, IconWallet, IconCheck, IconWarning } from "./icons";

const m = getMessages(defaultLocale);
const T = m.shared.incentives;

interface Standing {
  done: number;
  target: number;
  reached: boolean;
  rewarded: number;
  penalized: number;
}
interface Entry {
  id: string;
  kind: "reward" | "penalty";
  amount: number;
  reason: string;
  for_target: boolean;
  by: string;
  created_at: string;
}

/** ما يردّه الخادم — **والمكافأةُ خارج `standing`** لأنّها قاعدةٌ لا إنجاز. */
interface Payload {
  standing?: Standing;
  entries?: Entry[];
  /** **مكافأةُ بلوغ الهدف** — تُقال قبل أن يُبلَغ. صفرٌ يعني بلا مكافأةٍ آليّة. */
  target_reward?: number;
}

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

/** @param path مسارُ الدور — `/api/v1/driver/incentives` أو مسارُ المندوب. */
export function MyIncentives({ api, path }: { api: ApiFn; path: string }) {
  const [data, setData] = useState<Payload | null>(null);
  /**
   * **أيُعرض الاحتفال؟** — يُطفَأ بضغطة، ويُوسَم شهرُه في الجهاز.
   *
   * **ومن بلغ هدفَه يفرح مرّةً**: تهنئةٌ تتكرّر كلَّ يومٍ حتّى آخر الشهر تصير
   * ضجيجاً يُغلَق قبل أن يُقرأ.
   */
  const [showParty, setShowParty] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api<Payload>(path)
      .then(setData)
      .catch(() => setError(m.errors.internal));
  }, [api, path]);

  useEffect(load, [load]);

  /** وسمُ الشهر — «rahalgo.target.2026-08». */
  const partyKey = () => {
    const d = new Date();
    return `rahalgo.target.${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
  };
  const dismissParty = () => {
    try {
      localStorage.setItem(partyKey(), "1");
    } catch {
      // **وجهازٌ يمنع التخزينَ لا يُسقط الشاشة** — يُطفأ للجلسة وحدَها.
    }
    setShowParty(false);
  };
  useEffect(() => {
    if (!data?.standing?.reached) return;
    try {
      setShowParty(localStorage.getItem(partyKey()) !== "1");
    } catch {
      setShowParty(true);
    }
  }, [data?.standing?.reached]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!data) return <LoadingState />;

  /* **وردٌّ ناقصُ `standing` يُبيّض الصفحة.**

     (كشفه جردُ السائق ٢٠٢٦-٠٨-٠٦: `Cannot read properties of undefined
      (reading 'target')`.)

     **والحافزُ يُقرأ ليُحفِّز** — وشاشةٌ بيضاءُ مكانَه تقول للسائق إنّ
     المنصةَ معطوبة. **وحقلٌ ناقصٌ يُعرض صفراً أهونُ من صفحةٍ تسقط.** */
  const st = data.standing ?? { target: 0, done: 0, reached: false, rewarded: 0, penalized: 0 };
  const entries = Array.isArray(data.entries) ? data.entries : [];
  // **ونسبةٌ تُحسب هنا لا في الخادم**: هي عرضٌ لا قاعدة. **والبلوغُ يقوله
  // الخادمُ** — شرطٌ يُحسب في موضعين يفترق يوماً.
  const pct = st.target > 0 ? Math.min(100, (st.done / st.target) * 100) : 0;
  const reward = data.target_reward ?? 0;

  return (
    <PageContainer>
      <PageHeader icon={IconStar} title={T.title} subtitle={T.subtitle} />

      {/* **وشريطٌ لا يظهر بلا هدف.**

          صفرٌ يعني «لا هدفَ مضبوط»، **وشريطٌ ممتلئٌ على هدفٍ صفرٍ يُقرأ
          إنجازاً** فيُهنّئ من لم يُطلب منه شيء. */}
      {st.target > 0 && (
        <div className="surface p-4">
          <div className="mb-2 flex items-center justify-between gap-2">
            <p className="font-bold">{T.monthTarget}</p>
            {st.reached ? (
              <Badge variant="success">{T.reached}</Badge>
            ) : (
              <span className="text-sm text-ink-muted">
                {T.remaining}: {fmtNum(Math.max(0, st.target - st.done))}
              </span>
            )}
          </div>
          <div className="h-2 overflow-hidden rounded-badge bg-field">
            <div
              className={`h-full rounded-badge ${st.reached ? "bg-success" : "bg-accent"}`}
              style={{ width: `${pct}%` }}
            />
          </div>
          <p className="mt-1.5 text-sm text-ink-muted" dir="ltr">
            {fmtNum(st.done)} / {fmtNum(st.target)}
          </p>

          {/* ── **والجائزةُ تُقال قبل أن تُنال** ─────────────────────────

              (شكوى المالك ٢٠٢٦-٠٨-٠٩: «لازم يعرف شو المكافأة الي رح يحصل
               عليها وقت يحقق هدفه».)

              **وشاشةٌ تقول «٣ من ٥٠» ولا تقول ماذا بعدها تطلب جهداً بلا
              وعد.** وصفرٌ يُخفيها — **لا يُعرَض وعدٌ بلا مبلغ.** */}
          {reward > 0 && (
            <p
              className={`mt-2 rounded-control px-3 py-2 text-center text-sm font-medium ${
                st.reached ? "bg-success-tint text-success" : "bg-field text-ink"
              }`}
            >
              {st.reached
                ? T.rewardEarned.replace("{a}", fmtNum(reward)).replace("{c}", m.common.currency)
                : T.rewardPromise.replace("{a}", fmtNum(reward)).replace("{c}", m.common.currency)}
            </p>
          )}
        </div>
      )}

      {/* ── **احتفالٌ عند البلوغ — مرّةً في الشهر** ────────────────────

          (قرارُ المالك: «مع تأثير للشاشة عند وصول الهدف احتفاليّ للسائق».)

          **ولا يُعاد في كلّ فتحة**: من بلغ هدفَه يفرح مرّةً، **وتهنئةٌ تتكرّر
          كلَّ يومٍ حتّى آخر الشهر تصير ضجيجاً يُغلَق قبل أن يُقرأ.**

          **والوسمُ في الجهاز لا في الخادم**: هو شأنُ هذه الشاشة وحدَها،
          **وصفٌّ في القاعدة لأثرٍ بصريٍّ يُثقل الدفتر بما ليس منه.** */}
      {st.reached && showParty && (
        <div className="surface relative overflow-hidden p-5 text-center">
          <IconStar size={34} className="mx-auto text-warning" />
          <p className="mt-2 heading-card text-ink">{T.partyTitle}</p>
          <p className="mt-1 text-sm text-ink-muted">
            {T.partyBody.replace("{n}", fmtNum(st.target))}
          </p>
          <Button variant="secondary" className="mt-3" onClick={dismissParty}>
            {m.common.confirm}
          </Button>
        </div>
      )}

      <StatGrid>
        <StatCard
          icon={IconCheck}
          label={T.doneThisMonth}
          value={fmtNum(st.done)}
        />
        <StatCard
          icon={IconWallet}
          label={T.rewardedThisMonth}
          value={`${fmtNum(st.rewarded)} ${m.common.currency}`}
          tone="success"
        />
        <StatCard
          icon={IconWarning}
          label={T.penalizedThisMonth}
          value={`${fmtNum(st.penalized)} ${m.common.currency}`}
          tone={st.penalized > 0 ? "danger" : "default"}
        />
      </StatGrid>

      {entries.length === 0 ? (
        <EmptyState icon={IconStar} title={T.empty} />
      ) : (
        <ul className="space-y-2">
          {entries.map((e) => (
            <li key={e.id} className="surface p-4">
              <div className="flex items-center justify-between gap-2">
                <Badge variant={e.kind === "reward" ? "success" : "danger"}>
                  {e.kind === "reward" ? T.reward : T.penalty}
                </Badge>
                <span
                  className={`font-bold ${e.kind === "reward" ? "text-success" : "text-danger"}`}
                  dir="ltr"
                >
                  {e.kind === "reward" ? "+" : "−"}
                  {fmtNum(e.amount)} {m.common.currency}
                </span>
              </div>
              {/* **والسببُ إلزاميّ في الخادم** — فلا سطرَ هنا بلا كلمة. */}
              <p className="mt-1.5 text-sm">{e.reason}</p>
              <p className="mt-0.5 text-xs text-ink-muted" dir="ltr">
                {fmtDateTime(e.created_at)}
              </p>
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  );
}
