"use client";

/**
 * **نظرةُ الخزينة العامّة** — رصيدُها، وما في درج المكتب، والنقدُ في الشارع، وما
 * ينتظر قراراً، **وسحبُ الأدمن من رصيدها** (قرارُ المالك ٢٠٢٦-١٠-٠٤: «حتّى لو دفع
 * من المحفظة رح يكون واضح إنّ الأدمن سحب من رصيد الخزينة»).
 *
 * **وصافي الربح اليومَ والأسبوعَ والشهرَ ومن أوّل يوم** (قرارُ المالك ٢٠٢٦-١٠-١٥) — الرقمُ
 * نفسُه في صفحة الأرباح للمدى نفسِه، يحسبه المحرّك (`profit_cards`) ولا تحسبه الشاشة.
 */

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtMoney, fmtNum, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  FormActions,
  Input,
  LoadingState,
  Modal,
  ReloadState,
  IconWallet,
  IconBalance,
  IconDriver,
  IconStore,
  IconCheck,
  IconWarning,
  IconTrendUp,
  IconTrendDown,
  IconArrowIn,
  IconArrowOut,
  IconDate,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { T, EmptyCard, fmtDay, dayStartsLine } from "./shared";
import { DiffBadge, type CashClose } from "./Cashbox";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const O = T.overview;
const W = T.withdraw;

interface Overview {
  treasury_balance: number;
  has_treasury: boolean;
  cashbox: { expected: number };
  today_in: number;
  today_out: number;
  cash_in_street: number;
  merchant_cash_due: number;
  pending_payouts: number;
  withdrawals_month: number;
  open_shortfalls: number;
  pending_count: number;
  pending_total: number;
  /** **ما بلا مبلغٍ بعد** — تعويضٌ مقترحُه صفر، خارجَ المجموع. */
  pending_undecided?: number;
  profit_cards?: ProfitCard[];
  last_close: CashClose | null;
  /** **ساعةُ بدء يوم الحساب** (`platform.day_start_hour`). */
  day_start_hour?: number;
}
interface ProfitCard {
  period: "day" | "week" | "month" | "all";
  from: string;
  to: string;
  net: number;
}
interface Withdrawal {
  id: string;
  amount: number;
  note: string;
  from_cashbox: boolean;
  by_name: string;
  created_at: string;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Date.now());
}

export function OverviewTab({ onTab }: { onTab: (tab: "approvals" | "cashbox") => void }) {
  const router = useRouter();
  const canCall = useCanCall();
  const canWithdraw = canCall("POST", "/treasury/withdrawals");
  const ov = useLiveData<Overview>(() => api(`/api/v1/admin/treasury/overview`), ["wallet"]);
  const wd = useLiveData<{ withdrawals: Withdrawal[] }>(() => api(`/api/v1/admin/treasury/withdrawals`), ["wallet"]);
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [fromCashbox, setFromCashbox] = useState(true);
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");
  const [done, setDone] = useState("");
  const [idemKey, setIdemKey] = useState(newKey);

  async function withdraw() {
    setBusy(true);
    try {
      await api(`/api/v1/admin/treasury/withdrawals`, {
        method: "POST",
        headers: { "Idempotency-Key": idemKey },
        body: JSON.stringify({ amount: Math.round(Number(amount)), note: note.trim(), from_cashbox: fromCashbox }),
      });
      setOpen(false);
      setAmount("");
      setNote("");
      setIdemKey(newKey());
      setFail("");
      setDone(W.done);
      ov.reload();
      wd.reload();
    } catch (err) {
      setFail(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (ov.error && !ov.data) return <ReloadState onRetry={ov.reload} />;
  if (!ov.data) return <LoadingState variant="stats" />;
  const d = ov.data;
  const amountNum = Number(amount);
  const lc = d.last_close;

  /** **كلُّ بطاقة ربحٍ تفتح الأرباحَ على مدّتها** — كان الرابطُ بلا مدّةٍ فيفتح الشهرَ دائماً. */
  function openProfits(c: ProfitCard) {
    const q = new URLSearchParams();
    if (c.from) q.set("from", c.from);
    if (c.to) q.set("to", c.to);
    if (c.period === "all") q.set("period", "all");
    const qs = q.toString();
    router.push(qs ? `/dashboard/profits?${qs}` : "/dashboard/profits");
  }

  return (
    <div className="space-y-4">
      {done && (
        <Alert tone="success" onDismiss={() => setDone("")}>
          {done}
        </Alert>
      )}
      {!d.has_treasury && <Alert tone="warning">{O.noTreasury}</Alert>}
      {/* **الخزينةُ بالتصميم الجديد** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — الرصيدُ أوّلاً بلونه، ثمّ الربحُ بمُدَده،
          ثمّ حركةُ النقد (أين المالُ الآن)، ثمّ ما ينتظر قراراً. */}
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <section
          className={`surface flex flex-col justify-center gap-1 p-5 ${
            d.treasury_balance < 0 ? "border-danger" : ""
          }`}
        >
          <span className="text-sm text-ink-muted">{O.treasuryBalance}</span>
          <span className={`figure text-3xl ${d.treasury_balance < 0 ? "text-danger" : "text-success"}`}>
            {fmtMoney(d.treasury_balance)}
          </span>
          <span className="text-xs text-ink-muted">{O.balanceHint}</span>
        </section>
        {(d.profit_cards ?? []).length > 0 && (
          <Panel title={O.profitTitle} note={O.profitHint} className="xl:col-span-2">
            <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
              {(d.profit_cards ?? []).map((c) => (
                <Tile
                  key={c.period}
                  icon={c.net < 0 ? IconTrendDown : IconTrendUp}
                  label={O.profitCards[c.period]}
                  value={fmtMoney(c.net)}
                  sub={c.to ? (c.from && c.from !== c.to ? `${fmtDay(c.from)} ← ${fmtDay(c.to)}` : fmtDay(c.to)) : undefined}
                  tone={c.net > 0 ? "success" : c.net < 0 ? "danger" : "muted"}
                  onClick={() => openProfits(c)}
                />
              ))}
            </div>
          </Panel>
        )}
      </div>
      <p className="flex items-center gap-1.5 text-xs text-ink-muted">
        <IconDate size={14} />
        {dayStartsLine(d.day_start_hour)}
      </p>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <Panel title={O.cashTitle}>
          <div className="grid grid-cols-2 gap-2">
            <Tile
              icon={IconBalance}
              label={O.cashboxExpected}
              value={fmtMoney(d.cashbox.expected)}
              tone={d.cashbox.expected < 0 ? "danger" : "default"}
              onClick={() => onTab("cashbox")}
            />
            <Tile
              icon={IconDriver}
              label={O.cashInStreet}
              value={fmtMoney(d.cash_in_street)}
              tone={d.cash_in_street > 0 ? "warning" : "muted"}
              onClick={() => router.push("/dashboard/cash")}
            />
            <Tile
              icon={IconArrowIn}
              label={O.todayIn}
              value={fmtMoney(d.today_in)}
              tone={d.today_in > 0 ? "success" : "muted"}
              onClick={() => onTab("cashbox")}
            />
            <Tile
              icon={IconArrowOut}
              label={O.todayOut}
              value={fmtMoney(d.today_out)}
              tone={d.today_out > 0 ? "danger" : "muted"}
              onClick={() => onTab("cashbox")}
            />
          </div>
        </Panel>
        <Panel title={O.dueTitle}>
          <div className="grid grid-cols-2 gap-2">
            <Tile
              icon={IconStore}
              label={O.merchantCashDue}
              value={fmtMoney(d.merchant_cash_due)}
              tone={d.merchant_cash_due > 0 ? "warning" : "muted"}
              onClick={() => router.push("/dashboard/cash?tab=stores")}
            />
            <Tile
              icon={IconWallet}
              label={O.pendingPayouts}
              value={fmtMoney(d.pending_payouts)}
              tone={d.pending_payouts > 0 ? "warning" : "muted"}
              onClick={() => router.push("/dashboard/payouts")}
            />
            <Tile
              icon={IconCheck}
              label={O.pendingApprovals}
              value={String(d.pending_count)}
              sub={
                (d.pending_undecided ?? 0) > 0
                  ? `${fmtMoney(d.pending_total)} · ${O.undecided.replace("{n}", String(d.pending_undecided))}`
                  : fmtMoney(d.pending_total)
              }
              tone={d.pending_count > 0 ? "warning" : "muted"}
              onClick={() => onTab("approvals")}
            />
            <Tile
              icon={IconWarning}
              label={O.openShortfalls}
              value={fmtMoney(d.open_shortfalls)}
              tone={d.open_shortfalls > 0 ? "danger" : "muted"}
              onClick={() => onTab("cashbox")}
            />
            <Tile
              icon={IconArrowOut}
              label={O.withdrawalsMonth}
              value={fmtMoney(d.withdrawals_month)}
              tone={d.withdrawals_month > 0 ? "default" : "muted"}
              onClick={() => document.getElementById("treasury-withdrawals")?.scrollIntoView({ behavior: "smooth", block: "start" })}
            />
          </div>
        </Panel>
      </div>

      {/* **آخرُ إغلاقٍ بطاقة** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — يومُه بالعربيّ، والمتوقَّعُ والمعدودُ والفرق،
          وحالُه ومن سجّله. ومعاينةُ الموافقات شيلت: مربّعُ العدد فوق يفتحها. */}
      <Panel
        title={O.closeCard}
        actions={
          <Button variant="secondary" onClick={() => onTab("cashbox")}>
            {O.openCashbox}
          </Button>
        }
      >
        {lc ? (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-bold">{O.closeDay.replace("{day}", fmtDay(lc.day))}</span>
              <Badge variant={lc.status === "approved" ? "success" : lc.status === "rejected" ? "danger" : "warning"}>
                {T.cashbox.statuses[lc.status]}
              </Badge>
              <span className="text-xs text-ink-muted">
                {T.cashbox.colBy}: {lc.proposer_name}
                {lc.decider_name && ` · ${T.cashbox.colReviewer}: ${lc.decider_name}`}
              </span>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              <div className="rounded-control bg-field px-3 py-2">
                <span className="block text-2xs text-ink-muted">{T.cashbox.colExpected}</span>
                <span className="figure block leading-tight">{fmtMoney(lc.expected)}</span>
              </div>
              <div className="rounded-control bg-field px-3 py-2">
                <span className="block text-2xs text-ink-muted">{T.cashbox.colCounted}</span>
                <span className="figure block leading-tight">{fmtMoney(lc.counted)}</span>
              </div>
              <div className="col-span-2 rounded-control bg-field px-3 py-2 sm:col-span-1">
                <span className="block text-2xs text-ink-muted">{T.cashbox.difference}</span>
                <span className="mt-1 block">
                  <DiffBadge diff={lc.difference} />
                </span>
              </div>
            </div>
          </div>
        ) : (
          <EmptyCard icon={IconBalance} title={O.noClose} hint={O.noCloseHint} tone="muted" bare />
        )}
      </Panel>

      {/* **سحوباتُ الأدمن بطاقات** — المبلغُ كبيرٌ بالأحمر، ومن سحب ومتى ومن أين. */}
      <div id="treasury-withdrawals" className="scroll-mt-4">
        <Panel
          title={W.list}
          actions={canWithdraw && d.has_treasury ? <Button onClick={() => setOpen(true)}>{W.button}</Button> : undefined}
        >
          {wd.error && !wd.data ? (
            <ReloadState onRetry={wd.reload} />
          ) : !wd.data ? (
            <LoadingState variant="text" />
          ) : wd.data.withdrawals.length === 0 ? (
            <EmptyCard icon={IconWallet} title={W.empty} hint={W.emptyHint} bare />
          ) : (
            <ul className="grid grid-cols-1 gap-2 lg:grid-cols-2 2xl:grid-cols-3">
              {wd.data.withdrawals.map((x) => (
                <li key={x.id} className="flex flex-col gap-2 rounded-card border border-line-soft p-3">
                  <div className="flex items-start gap-3">
                    <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-danger-solid text-on-solid">
                      <IconArrowOut size={18} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="figure block leading-none text-danger">
                        <span dir="ltr">−{fmtNum(x.amount)}</span> {m.common.currency}
                      </span>
                      <span className="mt-1 block truncate text-xs text-ink-muted">
                        {W.by}: {x.by_name || "—"}
                      </span>
                    </span>
                    <Badge variant={x.from_cashbox ? "warning" : "neutral"}>{x.from_cashbox ? W.cashbox : W.notCashbox}</Badge>
                  </div>
                  {x.note && <p className="border-s-2 border-line ps-2 text-sm">{x.note}</p>}
                  <span className="text-2xs text-ink-muted" dir="ltr">
                    {fmtDateTime(x.created_at)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>

      <Modal open={open} onClose={() => setOpen(false)} title={W.title}>
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{W.hint}</p>
          {fail && <Alert>{fail}</Alert>}
          <p className="text-sm">
            {O.treasuryBalance}:{" "}
            <span dir="ltr" className="font-bold">
              {fmtMoney(d.treasury_balance)}
            </span>
          </p>
          <Input
            id="tw-amount"
            type="number"
            inputMode="numeric"
            dir="ltr"
            label={W.amount}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <Input id="tw-note" label={W.note} value={note} onChange={(e) => setNote(e.target.value)} />
          <Checkbox id="tw-cash" label={W.fromCashbox} checked={fromCashbox} onChange={(e) => setFromCashbox(e.target.checked)} />
          <FormActions
            busy={busy || !(amountNum > 0) || note.trim() === ""}
            saveLabel={W.confirm}
            onCancel={() => setOpen(false)}
            onSave={() => void withdraw()}
          />
        </div>
      </Modal>
    </div>
  );
}
