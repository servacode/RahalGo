"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/dashboard/order-debts — ديون الطلبات** (قرارُ المالك ٢٠٢٦-١١-٢٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * زبونٌ دفع الطلبَ كلَّه من محفظته، **فالكابتنُ دفع للمتجر من جيبه والمنصّةُ مديونةٌ له.** المكتبُ
 * يعطيه المبلغَ كاشاً ويكبس «تم الدفع» — فيخرج من صندوق المكتب (`order_debt_paid`).
 *
 *  - جدولُ الكباتن: المفتوحُ وعددُه وأقدمُ دين وما انعطى
 *  - بالتوسيع: طلباتُ الكابتن (رقمٌ ومبلغٌ وتاريخٌ وحالٌ ومن دفع)
 *  - «تم الدفع» للكابتن كلِّه أو لطلبٍ واحد — **بتأكيدٍ داخل الصفحة**، وبقدرة `finance.manage` وحدَها
 *
 * الأرقامُ من الخادم كما هي (`backend/internal/server/order_debts.go`).
 */

import { Fragment, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDate, fmtDateTime, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  IconBalance,
  IconMoto,
  LoadingState,
  PageContainer,
  ReloadState,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const D = m.admin.orderDebts;

interface DriverRow {
  driver_id: string;
  driver_name: string;
  phone: string;
  open_total: number;
  open_count: number;
  paid_total: number;
  oldest_open_at?: string;
}

interface Report {
  drivers: DriverRow[];
  open_total: number;
}

interface DebtItem {
  id: string;
  order_id: string;
  order_number: number;
  amount: number;
  state: "open" | "paid";
  created_at: string;
  paid_at?: string;
  paid_by_name?: string;
}

/** **ما يُسأل عنه قبل الدفع** — كلُّ ديون الكابتن (`item` فارغ) أو دينُ طلبٍ واحد. */
interface Ask {
  driver: DriverRow;
  item: DebtItem | null;
}

export default function OrderDebtsPage() {
  const open = useCanOpen();
  const canCall = useCanCall();
  const canPay = canCall("POST", "/finance/order-debts/{driverID}/pay");
  const rep = useLiveData<Report>(() => api("/api/v1/admin/finance/order-debts"), ["wallet", "order"]);
  const r = rep.data;

  const [expanded, setExpanded] = useState("");
  const [items, setItems] = useState<DebtItem[]>([]);
  const [itemsBusy, setItemsBusy] = useState(false);
  const [itemsError, setItemsError] = useState("");
  const [ask, setAsk] = useState<Ask | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");

  const loadItems = async (driverID: string) => {
    setItemsBusy(true);
    setItemsError("");
    try {
      const res = await api<{ items: DebtItem[] }>(`/api/v1/admin/finance/order-debts/${driverID}`);
      setItems(res.items ?? []);
    } catch (err) {
      setItems([]);
      setItemsError(errorText(err));
    } finally {
      setItemsBusy(false);
    }
  };

  const toggle = (driverID: string) => {
    if (expanded === driverID) {
      setExpanded("");
      return;
    }
    setExpanded(driverID);
    setItems([]);
    void loadItems(driverID);
  };

  const pay = async () => {
    if (!ask) return;
    setBusy(true);
    setError("");
    setDone("");
    try {
      const res = await api<{ paid_total: number; paid_count: number }>(
        `/api/v1/admin/finance/order-debts/${ask.driver.driver_id}/pay`,
        { method: "POST", body: JSON.stringify(ask.item ? { ids: [ask.item.id] } : {}) },
      );
      setDone(D.done.replace("{amount}", fmtMoney(res.paid_total)));
      rep.reload();
      if (expanded) void loadItems(expanded);
    } catch (err) {
      setError(errorText(err));
      rep.reload();
    } finally {
      setBusy(false);
      setAsk(null);
    }
  };

  const drivers = r?.drivers ?? [];
  const askName = ask ? ask.driver.driver_name || ask.driver.phone : "";

  return (
    <PageContainer>
      <PageHero icon={IconBalance} title={D.title} />
      <Alert tone="info" className="mb-3">
        {D.hint}
      </Alert>
      {error && <Alert className="mb-3">{error}</Alert>}
      {done && (
        <Alert tone="success" className="mb-3">
          {done}
        </Alert>
      )}

      {rep.error && !r && <ReloadState onRetry={rep.reload} />}
      {!r && !rep.error && <LoadingState variant="list" rows={3} />}
      {r && (
        <>
          <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-3">
            <Tile
              icon={IconBalance}
              label={D.totalOpen}
              value={fmtMoney(r.open_total)}
              tone={r.open_total > 0 ? "warning" : "muted"}
            />
            <Tile icon={IconMoto} label={D.captains} value={fmtNum(drivers.length)} tone="muted" />
          </div>

          <Panel title={D.title}>
            {drivers.length === 0 ? (
              <p className="text-sm text-ink-muted">{D.none}</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-start text-ink-muted">
                      <th className="p-2 text-start">{D.colCaptain}</th>
                      <th className="p-2 text-start">{D.colOpen}</th>
                      <th className="p-2 text-start">{D.colCount}</th>
                      <th className="p-2 text-start">{D.colOldest}</th>
                      <th className="p-2 text-start">{D.colPaid}</th>
                      <th className="p-2 text-start" />
                    </tr>
                  </thead>
                  <tbody>
                    {drivers.map((d) => (
                      <Fragment key={d.driver_id}>
                        <tr className="border-t border-line-soft">
                          <td className="p-2">
                            <OpenLink
                              allowed={open.user}
                              href={`/dashboard/users/${d.driver_id}`}
                              className="font-bold text-primary hover:underline"
                            >
                              {d.driver_name || d.phone}
                            </OpenLink>
                            <span className="block text-xs text-ink-muted" dir="ltr">
                              {d.phone}
                            </span>
                          </td>
                          <td className="p-2 font-bold text-warning" dir="ltr">
                            {fmtMoney(d.open_total)}
                          </td>
                          <td className="p-2">{fmtNum(d.open_count)}</td>
                          <td className="p-2">{d.oldest_open_at ? fmtDate(d.oldest_open_at) : "—"}</td>
                          <td className="p-2" dir="ltr">
                            {fmtMoney(d.paid_total)}
                          </td>
                          <td className="p-2">
                            <span className="flex flex-wrap items-center justify-end gap-2">
                              <Button size="sm" variant="secondary" onClick={() => toggle(d.driver_id)}>
                                {expanded === d.driver_id ? D.hide : D.show}
                              </Button>
                              {canPay && d.open_count > 0 && (
                                <Button size="sm" disabled={busy} onClick={() => setAsk({ driver: d, item: null })}>
                                  {D.payAll}
                                </Button>
                              )}
                            </span>
                          </td>
                        </tr>
                        {expanded === d.driver_id && (
                          <tr>
                            <td colSpan={6} className="bg-field p-3">
                              <p className="mb-2 text-xs font-bold text-ink-muted">{D.orders}</p>
                              {itemsBusy && <LoadingState variant="list" rows={2} />}
                              {itemsError && <Alert>{itemsError}</Alert>}
                              {!itemsBusy && !itemsError && items.length === 0 && (
                                <p className="text-sm text-ink-muted">{D.none}</p>
                              )}
                              {!itemsBusy && items.length > 0 && (
                                <div className="overflow-x-auto">
                                  <table className="w-full text-sm">
                                    <thead>
                                      <tr className="text-start text-ink-muted">
                                        <th className="p-2 text-start">{D.colOrder}</th>
                                        <th className="p-2 text-start">{D.colAmount}</th>
                                        <th className="p-2 text-start">{D.colDate}</th>
                                        <th className="p-2 text-start">{D.colState}</th>
                                        <th className="p-2 text-start">{D.colBy}</th>
                                        <th className="p-2 text-start" />
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {items.map((it) => (
                                        <tr key={it.id} className="border-t border-line-soft">
                                          <td className="p-2">
                                            <OpenLink
                                              allowed={open.order}
                                              href={`/dashboard/orders?q=${it.order_number}`}
                                              className="font-bold text-primary hover:underline"
                                            >
                                              #{it.order_number}
                                            </OpenLink>
                                          </td>
                                          <td className="p-2 font-bold" dir="ltr">
                                            {fmtMoney(it.amount)}
                                          </td>
                                          <td className="p-2">
                                            {fmtDateTime(it.state === "paid" && it.paid_at ? it.paid_at : it.created_at)}
                                          </td>
                                          <td className="p-2">
                                            <Badge variant={it.state === "open" ? "warning" : "success"}>
                                              {it.state === "open" ? D.open : D.paid}
                                            </Badge>
                                          </td>
                                          <td className="p-2">{it.paid_by_name || "—"}</td>
                                          <td className="p-2 text-end">
                                            {canPay && it.state === "open" && (
                                              <Button
                                                size="sm"
                                                variant="secondary"
                                                disabled={busy}
                                                onClick={() => setAsk({ driver: d, item: it })}
                                              >
                                                {D.pay}
                                              </Button>
                                            )}
                                          </td>
                                        </tr>
                                      ))}
                                    </tbody>
                                  </table>
                                </div>
                              )}
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Panel>

          <Confirm
            open={ask !== null}
            tone="primary"
            title={ask?.item ? D.askOneTitle : D.askAllTitle}
            body={
              ask
                ? ask.item
                  ? D.askOne
                      .replace("{name}", askName)
                      .replace("{amount}", fmtMoney(ask.item.amount))
                      .replace("{n}", String(ask.item.order_number))
                  : D.askAll
                      .replace("{name}", askName)
                      .replace("{amount}", fmtMoney(ask.driver.open_total))
                      .replace("{n}", fmtNum(ask.driver.open_count))
                : ""
            }
            confirmLabel={D.pay}
            busy={busy}
            onConfirm={() => void pay()}
            onCancel={() => setAsk(null)}
          />
        </>
      )}
    </PageContainer>
  );
}
