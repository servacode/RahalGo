"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/dashboard/goods-balances — أرصدة من بضاعة** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * القاعدةُ الجديدة: البضاعةُ لا تدخل حساباً. **والأرصدةُ القديمةُ تبقى كما هي** — فهذه الصفحةُ تقول
 * مَن يحمل اليومَ مالاً أصلُه بضاعة، بستّة أقسام (`backend/internal/server/goods_balances.go`).
 * **قراءةٌ فقط**: لا زرَّ يحرّك مالاً، والأرقامُ من الخادم كما هي.
 */

import { getMessages, defaultLocale, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  IconBalance,
  IconCash,
  IconMoto,
  IconStore,
  IconUser,
  IconWallet,
  LoadingState,
  PageContainer,
  ReloadState,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const G = m.admin.goodsBalances;

type SectionKey =
  | "store_wallets"
  | "store_debts"
  | "driver_debts"
  | "cash_holding"
  | "customer_refunds"
  | "driver_advances";

interface Row {
  id: string;
  name: string;
  phone: string;
  amount: number;
  count: number;
  orders: number[];
  balance: number | null;
}

interface Section {
  rows: Row[];
  total: number;
}

type Report = Record<SectionKey, Section> & { generated_at: string };

/** **ترتيبُ الأقسام وأيقوناتُها** — وهل الطرفُ متجرٌ (ملفُّه في المتاجر) أم حساب. */
const SECTIONS: { key: SectionKey; icon: typeof IconStore; store: boolean }[] = [
  { key: "store_wallets", icon: IconStore, store: true },
  { key: "store_debts", icon: IconBalance, store: true },
  { key: "cash_holding", icon: IconCash, store: true },
  { key: "driver_advances", icon: IconMoto, store: false },
  { key: "driver_debts", icon: IconMoto, store: false },
  { key: "customer_refunds", icon: IconWallet, store: false },
];

const SHOWN_ORDERS = 5;

export default function GoodsBalancesPage() {
  const open = useCanOpen();
  const rep = useLiveData<Report>(() => api("/api/v1/admin/finance/goods-balances"), ["wallet", "order"]);
  const r = rep.data;

  return (
    <PageContainer>
      <PageHero icon={IconBalance} title={G.title} />
      <Alert tone="info" className="mb-3">
        {G.readOnly}
      </Alert>

      {rep.error && !r && <ReloadState onRetry={rep.reload} />}
      {!r && !rep.error && <LoadingState variant="list" rows={3} />}
      {r && (
        <>
          <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-3">
            {SECTIONS.map((s) => (
              <Tile
                key={s.key}
                icon={s.icon}
                label={G.sections[s.key]}
                value={fmtMoney(r[s.key].total)}
                sub={G.count.replace("{n}", fmtNum(r[s.key].rows.reduce((a, x) => a + x.count, 0)))}
                tone={r[s.key].total > 0 ? "warning" : "muted"}
              />
            ))}
          </div>

          {SECTIONS.map((s) => {
            const sec = r[s.key];
            return (
              <Panel key={s.key} title={G.sections[s.key]} note={G.hints[s.key]} className="mb-3">
                {sec.rows.length === 0 ? (
                  <p className="text-sm text-ink-muted">{G.none}</p>
                ) : (
                  <>
                    <ul className="divide-y divide-line-soft">
                      {sec.rows.map((row) => (
                        <li key={row.id} className="flex flex-wrap items-start justify-between gap-2 py-2">
                          <span className="min-w-0">
                            <span className="flex items-center gap-1.5 font-bold">
                              {s.store ? <IconStore size={14} /> : <IconUser size={14} />}
                              <OpenLink
                                allowed={s.store ? open.merchant : open.user}
                                href={s.store ? `/dashboard/merchants/${row.id}` : `/dashboard/users/${row.id}`}
                                className="text-primary hover:underline"
                              >
                                {row.name || row.phone}
                              </OpenLink>
                            </span>
                            <span className="block text-xs text-ink-muted" dir="ltr">
                              {row.phone}
                            </span>
                            <span className="block text-xs text-ink-muted">
                              {G.count.replace("{n}", fmtNum(row.count))}
                              {row.orders.length > 0 && (
                                <>
                                  {m.common.listSeparator}
                                  {G.orders.replace("{list}", "")}
                                  {row.orders.slice(0, SHOWN_ORDERS).map((n, i) => (
                                    <span key={n}>
                                      {i > 0 ? m.common.listSeparator : ""}
                                      <OpenLink
                                        allowed={open.order}
                                        href={`/dashboard/orders?q=${n}`}
                                        className="text-primary hover:underline"
                                      >
                                        #{n}
                                      </OpenLink>
                                    </span>
                                  ))}
                                  {row.orders.length > SHOWN_ORDERS &&
                                    m.common.listSeparator +
                                      G.more.replace("{n}", fmtNum(row.orders.length - SHOWN_ORDERS))}
                                </>
                              )}
                            </span>
                            {row.balance !== null && (
                              <span className="block text-xs text-ink-muted">
                                {G.balance.replace("{b}", fmtMoney(row.balance))}
                              </span>
                            )}
                          </span>
                          <span className="font-bold text-warning" dir="ltr">
                            {fmtMoney(row.amount)}
                          </span>
                        </li>
                      ))}
                    </ul>
                    <div className="mt-2 flex justify-between border-t border-line-soft pt-2 text-sm font-bold">
                      <span>{G.total}</span>
                      <span dir="ltr">{fmtMoney(sec.total)}</span>
                    </div>
                  </>
                )}
              </Panel>
            );
          })}
        </>
      )}
    </PageContainer>
  );
}
