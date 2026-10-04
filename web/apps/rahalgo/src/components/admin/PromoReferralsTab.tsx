"use client";

/**
 * **«ادعُ صديقاً»** — تبويبٌ في صفحة العروض (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٧).
 *
 * المبالغُ والشرطُ تُقرأ هنا وتُعدَّل من الإعدادات (رابط)، وجدولُ الدعوات:
 * مين دعا مين، إمتى، انصرفت المكافأة أو لا، وقديش.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  DataView,
  Pagination,
  StatGrid,
  StatCard,
  type DataColumn,
  IconPromos,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

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

  const columns: DataColumn<Row>[] = [
    { id: "inviter", header: O.refInviter, primary: true, cell: (r) => r.inviter_name || "—" },
    { id: "invitee", header: O.refInvitee, cell: (r) => r.invitee_name || "—" },
    { id: "when", header: O.refWhen, cell: (r) => fmtDate(r.created_at) },
    {
      id: "reward",
      header: O.refRewardState,
      cell: (r) =>
        r.rewarded_at ? (
          <Badge variant="success">
            {O.refPaidOn.replace("{date}", fmtDate(r.rewarded_at))} · {fmtMoney(r.reward_amount)}
          </Badge>
        ) : (
          <Badge variant="neutral">{O.refNotYet}</Badge>
        ),
    },
  ];

  const s = data?.settings;
  return (
    <div className="space-y-4">
      {error && <Alert>{error}</Alert>}
      {data && s && (
        <>
          <StatGrid>
            <StatCard icon={IconPromos} label={O.refTotal} value={fmtNum(data.total)} />
            <StatCard icon={IconPromos} label={O.refRewarded} value={fmtNum(data.rewarded)} />
            <StatCard icon={IconWallet} label={O.refPaid} value={fmtMoney(data.paid)} />
          </StatGrid>
          <div className="surface space-y-2 p-4 text-sm">
            <p>
              <span className="font-medium">{O.refCondition}: </span>
              {s.reward_on === "first_order" ? O.refOnFirstOrder : O.refOnSignup}
            </p>
            <p className="font-medium">{O.refRewards}</p>
            <ul className="grid grid-cols-2 gap-1 sm:grid-cols-5">
              {[s.reward_1, s.reward_2, s.reward_3, s.reward_4].map((v, i) => (
                <li key={i}>
                  {O.refRank.replace("{n}", fmtNum(i + 1))}: {fmtMoney(v)}
                </li>
              ))}
              <li>
                {O.refRest}: {fmtMoney(s.reward_rest)}
              </li>
            </ul>
            <Link href="/dashboard/settings" className="text-accent underline">
              {O.refEditInSettings}
            </Link>
          </div>
        </>
      )}
      <DataView
        items={data?.referrals ?? []}
        loading={data === null && !error}
        getKey={(r) => `${r.created_at}-${r.invitee_name}-${r.rank}`}
        columns={columns}
        view="table"
        empty={O.refEmpty}
      />
      {data && (
        <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
      )}
    </div>
  );
}
