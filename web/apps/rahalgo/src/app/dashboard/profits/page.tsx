"use client";

/**
 * **الأرباح — من أين جاء المالُ وأين ذهب، ولكلِّ إنسانٍ نصيبُه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
 *
 * # ولماذا رقمان لا رقم
 *
 * **طلب المالكُ معادلة**: «الهامش + العمولة − المصاريف − الخسائر».
 *
 * **والمعنى صحيحٌ والحسابُ المباشرَ لها يخطئ**: ما تأخذه الخزينةُ من الطلب هو
 * **ما دفعه الزبونُ ناقصَ ما رُدَّ ناقصَ ما قُيّد للمتجر والسائق والمندوب** —
 * **فهو يحوي الهامشَ والعمولةَ معاً، وقد طُرح منه الخصمُ والكوبون أصلاً**
 * لأنّ الزبونَ دفع أقلّ.
 *
 * **فلو جُمعا ثمّ طُرح الخصمُ مرّةً أخرى لَحُسب مرّتين** — **ولَخالف الناتجُ
 * رصيدَ الخزينة**، ولا يُعرف أيُّهما يُصدَّق.
 *
 * **فيُعرض الاثنان**: التفصيلُ يقول **من أين**، والصافي يُقرأ من الدفتر
 * **فيطابق رصيدَ الخزينة دائماً.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Tabs,
  type TabDef,
  Input,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ReloadState,
  Pagination,
  StatGrid,
  StatCard,
  IconWallet,
  IconUser,
  IconStore,
  IconDriver,
  IconStatus,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const P = m.admin.profits;

type Tab = "platform" | "customers" | "reps" | "drivers" | "merchants";

interface Platform {
  orders: number;
  sales: number;
  margin: number;
  commission: number;
  discount: number;
  losses: number;
  opex: number;
  referrals: number;
  penalties: number;
  net: number;
}
interface Row {
  user_id: string;
  name: string;
  phone: string;
  a: number;
  b: number;
  c: number;
  earned?: number;
}

/** **وأعمدةُ كلِّ تبويبٍ تُسمّى في الشاشة** — والمحرّكُ يرسل `a·b·c`. */
const COLS: Record<Exclude<Tab, "platform">, [string, string, string]> = {
  customers: [P.colEarnedInvites, P.colPenalty, P.colSpent],
  reps: [P.colEarnedCommission, P.colPenalty, P.colSpent],
  drivers: [P.colEarnedDelivery, P.colPenalty, P.colSpent],
  merchants: [P.colOurMargin, P.colOurCommission, P.colGross],
};

export default function ProfitsPage() {
  const [tab, setTab] = useState<Tab>("platform");
  /** **ومدًى مفتوحٌ يعني «من أوّل يوم»** — (قرارُ المالك). */
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<unknown | null | "failed">(null);
  /** **وسببُ الخادم يُعرض كما قاله** — [ReloadState]. */
  const [why, setWhy] = useState("");

  const load = useCallback(() => {
    const qs = new URLSearchParams({ tab, page: String(page) });
    if (from) qs.set("from", from);
    if (to) qs.set("to", to);
    api<unknown>(`/api/v1/admin/profits?${qs}`)
      .then((d) => {
        setWhy("");
        setData(d);
      })
      // ══════════════════════════════════════════════════════════════
      // **والفشلُ ليس فراغاً — وسببُه يُقال**
      // ══════════════════════════════════════════════════════════════
      //
      // **و«لا أرباح» على قراءةٍ فشلت تُقرأ شهراً بلا دخل.**
      //
      // **وأوّلُ كتابةٍ ابتلعت السبب**: ردَّ الخادمُ خطأً داخليّاً
      // **وقالت الشاشةُ «لا يوجد اتصال بالإنترنت»** — فبحث المالكُ في
      // شبكته والعطبُ في استعلام. (٢٠٢٦-٠٨-١٦.)
      .catch((e) => {
        setWhy(errorText(e, m));
        setData("failed");
      });
  }, [tab, page, from, to]);

  useEffect(load, [load]);

  const tabs: TabDef<Tab>[] = [
    { key: "platform", label: P.tabPlatform, icon: IconWallet },
    { key: "customers", label: P.tabCustomers, icon: IconUser },
    { key: "reps", label: P.tabReps, icon: IconStatus },
    { key: "drivers", label: P.tabDrivers, icon: IconDriver },
    { key: "merchants", label: P.tabMerchants, icon: IconStore },
  ];

  return (
    <PageContainer>
      <PageHeader icon={IconWallet} title={P.title} subtitle={P.hint} />

      <Tabs
        items={tabs}
        value={tab}
        onChange={(k) => {
          setTab(k);
          setPage(1);
          setData(null);
        }}
        className="mb-4"
      />

      {/* **ومدًى مفتوحٌ يعني «من أوّل يوم»** — فلا يُجبَر أحدٌ على ملء حقلٍ
          ليرى الكلّ. */}
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="w-40">
          <Input
            id="pf-from"
            type="date"
            label={m.shared.statement.from}
            value={from}
            onChange={(e) => {
              setFrom(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-40">
          <Input
            id="pf-to"
            type="date"
            label={m.shared.statement.to}
            value={to}
            onChange={(e) => {
              setTo(e.target.value);
              setPage(1);
            }}
          />
        </div>
      </div>

      {data === "failed" ? (
        <ReloadState onRetry={load} label={why || undefined} />
      ) : data === null ? (
        <LoadingState />
      ) : tab === "platform" ? (
        <PlatformView d={data as Platform} />
      ) : (
        <PartyView
          d={data as { rows: Row[]; total: number; per_page: number }}
          cols={COLS[tab]}
          showEarned={tab === "merchants"}
          page={page}
          onPage={setPage}
        />
      )}
    </PageContainer>
  );
}

/** **صفٌّ في كشف المنصّة** — اسمٌ ورقمٌ ولونُه يقول أداخلٌ هو أم خارج. */
function Line({ label, value, out }: { label: string; value: number; out?: boolean }) {
  return (
    <li className="flex items-center justify-between gap-3 px-3 py-2">
      <span className="text-sm">{label}</span>
      <span
        dir="ltr"
        className={`font-bold tabular-nums ${out ? "text-danger" : "text-success"}`}
      >
        {out ? "−" : "+"}
        {fmtNum(value)}
      </span>
    </li>
  );
}

function PlatformView({ d }: { d: Platform }) {
  return (
    <>
      <StatGrid>
        <StatCard
          label={`${P.net} (${m.common.currency})`}
          value={fmtNum(d.net)}
          icon={IconWallet}
          tone={d.net >= 0 ? "success" : "danger"}
        />
        <StatCard label={P.ordersCount} value={fmtNum(d.orders)} icon={IconStatus} />
        <StatCard
          label={`${P.sales} (${m.common.currency})`}
          value={fmtNum(d.sales)}
          icon={IconStore}
        />
      </StatGrid>

      {/* ══════════════════════════════════════════════════════════════
          **والتفصيلُ يقول من أين — والصافي يُقرأ من الدفتر**
          ══════════════════════════════════════════════════════════════

          **وما تأخذه الخزينةُ من الطلب يحوي الهامشَ والعمولةَ معاً، وقد
          طُرح منه الخصمُ أصلاً** لأنّ الزبونَ دفع أقلّ. **فجمعُ الهامش
          والعمولة ثمّ طرحُ الخصم يحسبه مرّتين** — **ويخالف الناتجُ رصيدَ
          الخزينة**، ولا يُعرف أيُّهما يُصدَّق.

          **فهذه القائمةُ تُقرأ تفصيلاً لا معادلة.** */}
      <p className="mb-2 mt-4 text-sm text-ink-muted">{P.breakdownHint}</p>
      <ul className="divide-y divide-line surface">
        <Line label={P.margin} value={d.margin} />
        <Line label={P.commission} value={d.commission} />
        <Line label={P.discount} value={d.discount} out />
        <Line label={P.referrals} value={d.referrals} out />
        <Line label={P.losses} value={d.losses} out />
        <Line label={P.opex} value={d.opex} out />
        <Line label={P.penalties} value={d.penalties} />
      </ul>
    </>
  );
}

function PartyView({
  d,
  cols,
  showEarned,
  page,
  onPage,
}: {
  d: { rows: Row[]; total: number; per_page: number };
  cols: [string, string, string];
  showEarned: boolean;
  page: number;
  onPage: (p: number) => void;
}) {
  const rows = d.rows ?? [];
  if (rows.length === 0) return <EmptyState icon={IconUser} title={P.empty} />;
  return (
    <>
      <div className="surface overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="text-2xs text-ink-muted">
            <tr className="border-b border-line-soft">
              <th className="p-2 text-start">{m.terms.name}</th>
              {cols.map((c) => (
                <th key={c} className="p-2 text-start">
                  {c}
                </th>
              ))}
              {showEarned && <th className="p-2 text-start">{P.colStoreEarned}</th>}
            </tr>
          </thead>
          <tbody>
            {rows.map((x) => (
              <tr key={x.user_id} className="border-b border-line-soft">
                <td className="p-2">
                  <span className="block font-medium">{x.name || x.phone}</span>
                  <span dir="ltr" className="block text-2xs text-ink-muted">
                    {x.phone}
                  </span>
                </td>
                <td dir="ltr" className="p-2 font-bold tabular-nums text-success">
                  {fmtNum(x.a)}
                </td>
                <td dir="ltr" className="p-2 tabular-nums">
                  {fmtNum(x.b)}
                </td>
                <td dir="ltr" className="p-2 tabular-nums text-ink-muted">
                  {fmtNum(x.c)}
                </td>
                {showEarned && (
                  <td dir="ltr" className="p-2 font-bold tabular-nums text-success">
                    {fmtNum(x.earned ?? 0)}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination page={page} total={d.total} perPage={d.per_page} onChange={onPage} />
    </>
  );
}
