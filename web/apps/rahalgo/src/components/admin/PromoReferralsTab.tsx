"use client";

/**
 * **«ادعُ صديقاً»** — تبويبٌ في صفحة العروض (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٧).
 *
 * المبالغُ والشرطُ تُقرأ هنا وتُعدَّل من الإعدادات (رابطٌ يفتح موضوعَ المال مباشرة)، وقائمةُ
 * الدعوات بكروت: مين دعا مين (والاسمان رابطان لحسابيهما)، إمتى، انصرفت المكافأة أو لا، وقديش.
 * **وفلترٌ بأزرار**: الكل · انصرفت · لسا ما انصرفت (تدقيقُ العروض ٢٠٢٦-١٠-٢٧).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  Chips,
  Pagination,
  LoadingState,
  IconUsers,
  IconCheck,
  IconWallet,
  IconSettings,
  IconNext,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanOpen } from "@/lib/policy";
import { Tile } from "@/components/admin/DashParts";
import { OpenLink } from "@/components/admin/OpenLink";

const m = getMessages(defaultLocale);
const O = m.admin.promosOwner;

interface Row {
  inviter_id: string | null;
  invitee_id: string | null;
  inviter_name: string;
  invitee_name: string;
  rank: number;
  created_at: string;
  rewarded_at: string | null;
  reward_amount: number;
}

interface Resp {
  settings: {
    reward_on: string;
    reward_1: number;
    reward_2: number;
    reward_3: number;
    reward_4: number;
    reward_rest: number;
  };
  total: number;
  rewarded: number;
  paid: number;
  matched: number;
  page: number;
  per_page: number;
  referrals: Row[];
}

type RefState = "all" | "rewarded" | "pending";

function Person({ id, name, label, canUser }: { id: string | null; name: string; label: string; canUser: boolean }) {
  return (
    <span className="min-w-0 flex-1 rounded-control bg-field px-3 py-2">
      <span className="block text-2xs text-ink-muted">{label}</span>
      <OpenLink
        allowed={canUser && !!id}
        href={`/dashboard/users/${id ?? ""}`}
        className={`block truncate font-bold ${canUser && id ? "text-primary hover:underline" : ""}`}
      >
        {name || "—"}
      </OpenLink>
    </span>
  );
}

export default function PromoReferralsTab() {
  const open = useCanOpen();
  const [data, setData] = useState<Resp | null>(null);
  const [page, setPage] = useState(1);
  const [filter, setFilter] = useState<RefState>("all");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    const st = filter === "all" ? "" : `&state=${filter}`;
    api<Resp>(`/api/v1/admin/referrals?page=${page}${st}`)
      .then((r) => {
        setData(r);
        setError("");
      })
      .catch((e) => setError(errorText(e)))
      .finally(() => setLoading(false));
  }, [page, filter]);
  useEffect(load, [load]);

  function pick(f: RefState) {
    setFilter(f);
    setPage(1);
  }

  const s = data?.settings;
  const rows = data?.referrals ?? [];
  const matched = data?.matched ?? data?.total ?? 0;
  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — أرقامٌ بمربّعات، والمكافآتُ بمربّعاتٍ حسب الترتيب،
  // وكرتٌ لكلّ دعوةٍ (مين دعا ← مين انْدعى) بدل الجدول.
  return (
    <div className="space-y-4">
      {data && s && (
        <>
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
            <Tile
              icon={IconUsers}
              label={O.refTotal}
              value={fmtNum(data.total)}
              tone={data.total > 0 ? "accent" : "muted"}
              onClick={() => pick("all")}
            />
            <Tile
              icon={IconCheck}
              label={O.refRewarded}
              value={fmtNum(data.rewarded)}
              tone={data.rewarded > 0 ? "success" : "muted"}
              onClick={() => pick("rewarded")}
            />
            <Tile icon={IconWallet} label={O.refPaid} value={fmtMoney(data.paid)} tone={data.paid > 0 ? "success" : "muted"} />
          </div>
          <section className="surface p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div className="min-w-0">
                <h2 className="heading-card">{O.refRewards}</h2>
                <p className="text-xs text-ink-muted">
                  {O.refCondition}: {s.reward_on === "first_order" ? O.refOnFirstOrder : O.refOnSignup}
                </p>
              </div>
              {/* **ولمن يفتح الإعدادات وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ لا تراها.
                  **ويفتح موضوعَ المال** (`referral.*` فيه) لا أوّلَ الإعدادات. */}
              {open.settings && (
                <ButtonLink href="/dashboard/settings?topic=money" variant="secondary">
                  <IconSettings size={14} />
                  {O.refEditInSettings}
                </ButtonLink>
              )}
            </div>
            <ol className="grid grid-cols-3 gap-2 sm:grid-cols-5">
              {[s.reward_1, s.reward_2, s.reward_3, s.reward_4].map((v, i) => (
                <li key={i} className="rounded-control bg-field px-3 py-2">
                  <span className="block text-2xs text-ink-muted">{O.refRank.replace("{n}", fmtNum(i + 1))}</span>
                  <span className="block font-bold tabular-nums">{fmtMoney(v)}</span>
                </li>
              ))}
              <li className="rounded-control bg-field px-3 py-2">
                <span className="block text-2xs text-ink-muted">{O.refRest}</span>
                <span className="block font-bold tabular-nums">{fmtMoney(s.reward_rest)}</span>
              </li>
            </ol>
          </section>
        </>
      )}

      {/* ── الفلتر ببطاقةٍ واحدة ── */}
      <section className="surface p-3">
        <Chips
          wrap
          items={[
            { id: "all" as const, label: O.fAll, count: data?.total },
            { id: "rewarded" as const, label: O.refFilterRewarded, count: data?.rewarded },
            {
              id: "pending" as const,
              label: O.refFilterPending,
              count: data ? Math.max(0, data.total - data.rewarded) : undefined,
            },
          ]}
          value={filter}
          onChange={pick}
        />
      </section>

      {error ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-danger-solid text-on-solid">
            <IconUsers size={28} />
          </span>
          <h2 className="heading-section">{O.loadFailed}</h2>
          <Alert>{error}</Alert>
          <Button variant="secondary" onClick={load} disabled={loading}>
            {O.retry}
          </Button>
        </section>
      ) : data === null ? (
        <LoadingState variant="list" />
      ) : rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-field text-ink-muted">
            <IconUsers size={28} />
          </span>
          <h2 className="heading-section">{filter === "all" ? O.refEmpty : O.refNoMatch}</h2>
          <p className="max-w-md text-sm text-ink-muted">{filter === "all" ? O.refEmptyHint : O.refNoMatchHint}</p>
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {rows.map((r) => (
            <li key={`${r.created_at}-${r.invitee_id ?? r.invitee_name}-${r.rank}`} className="surface flex flex-col gap-3 p-4">
              <div className="flex items-center gap-2">
                <Person id={r.inviter_id} name={r.inviter_name} label={O.refInviter} canUser={open.user} />
                <IconNext size={16} className="shrink-0 text-ink-muted" />
                <Person id={r.invitee_id} name={r.invitee_name} label={O.refInvitee} canUser={open.user} />
              </div>
              <div className="flex flex-wrap items-center justify-between gap-2 text-2xs text-ink-muted">
                <span>
                  {O.refRank.replace("{n}", fmtNum(r.rank))} · {O.refWhen}: {fmtDate(r.created_at)}
                </span>
                {r.rewarded_at ? (
                  <Badge variant="success">
                    {O.refPaidOn.replace("{date}", fmtDate(r.rewarded_at))} · {fmtMoney(r.reward_amount)}
                  </Badge>
                ) : (
                  <Badge variant="neutral">{O.refFilterPending}</Badge>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
      {data && matched > data.per_page && (
        <div className="flex justify-center">
          <Pagination page={page} total={matched} perPage={data.per_page} onChange={setPage} />
        </div>
      )}
    </div>
  );
}
