"use client";

/**
 * **«ادعُ صديقاً»** — تبويبٌ في صفحة العروض (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٧).
 *
 * المبالغُ والشرطُ تُقرأ هنا وتُعدَّل من الإعدادات (رابط)، وقائمةُ الدعوات بكروت:
 * مين دعا مين، إمتى، انصرفت المكافأة أو لا، وقديش.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  ButtonLink,
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

const m = getMessages(defaultLocale);
const O = m.admin.promosOwner;

interface Row {
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
  page: number;
  per_page: number;
  referrals: Row[];
}

export default function PromoReferralsTab() {
  const canSettings = useCanOpen().settings;
  const [data, setData] = useState<Resp | null>(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api<Resp>(`/api/v1/admin/referrals?page=${page}`)
      .then((r) => {
        setData(r);
        setError("");
      })
      .catch((e) => setError(errorText(e)));
  }, [page]);
  useEffect(load, [load]);

  const s = data?.settings;
  const rows = data?.referrals ?? [];
  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — أرقامٌ بمربّعات، والمكافآتُ بمربّعاتٍ حسب الترتيب،
  // وكرتٌ لكلّ دعوةٍ (مين دعا ← مين انْدعى) بدل الجدول.
  return (
    <div className="space-y-4">
      {error && <Alert>{error}</Alert>}
      {data && s && (
        <>
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
            <Tile icon={IconUsers} label={O.refTotal} value={fmtNum(data.total)} tone={data.total > 0 ? "accent" : "muted"} />
            <Tile
              icon={IconCheck}
              label={O.refRewarded}
              value={fmtNum(data.rewarded)}
              tone={data.rewarded > 0 ? "success" : "muted"}
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
              {/* **ولمن يفتح الإعدادات وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ لا تراها. */}
              {canSettings && (
                <ButtonLink href="/dashboard/settings" variant="secondary">
                  <IconSettings size={14} />
                  {O.refEditInSettings}
                </ButtonLink>
              )}
            </div>
            <ol className="grid grid-cols-2 gap-2 sm:grid-cols-5">
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

      {data === null && !error ? (
        <LoadingState variant="list" />
      ) : rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-field text-ink-muted">
            <IconUsers size={28} />
          </span>
          <h2 className="heading-section">{O.refEmpty}</h2>
          <p className="max-w-md text-sm text-ink-muted">{O.refEmptyHint}</p>
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {rows.map((r) => (
            <li key={`${r.created_at}-${r.invitee_name}-${r.rank}`} className="surface flex flex-col gap-3 p-4">
              <div className="flex items-center gap-2">
                <span className="min-w-0 flex-1 rounded-control bg-field px-3 py-2">
                  <span className="block text-2xs text-ink-muted">{O.refInviter}</span>
                  <span className="block truncate font-bold">{r.inviter_name || "—"}</span>
                </span>
                <IconNext size={16} className="shrink-0 text-ink-muted" />
                <span className="min-w-0 flex-1 rounded-control bg-field px-3 py-2">
                  <span className="block text-2xs text-ink-muted">{O.refInvitee}</span>
                  <span className="block truncate font-bold">{r.invitee_name || "—"}</span>
                </span>
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
                  <Badge variant="neutral">
                    {O.refRewardState}: {O.refNotYet}
                  </Badge>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
      {data && data.total > data.per_page && (
        <div className="flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}
    </div>
  );
}
