"use client";

/**
 * **تسويةُ مستحقّات المتجر — الطريقةُ والمستحقّاتُ النقديّة.**
 *
 * # ما هنا
 *
 * **طريقةُ التسوية** (نقد/محفظة): تُعرَض في ملفّ المتجر، **وتُغيَّر بقدرةِ
 * `settings.financial.manage`** خلفَ تأكيد. **والتغييرُ على الطلبات الجديدة وحدَها.**
 * **ولا تُعرض في نافذة الدفع** (`variant="pay"`، فحصُ اللوحة ٢٠٢٦-١٠-٢٧): تخصّ ملفَّ
 * المتجر ولا شأنَ لها بالدفع.
 *
 * **وكشفُ المستحقّات النقديّة** (بقدرةِ `finance.read`): رقمٌ كبيرٌ لما لم يُدفع، وكرتٌ
 * لكلّ طلبٍ برقمه المعروف، **وتأكيدُ الدفع بقدرةِ `finance.manage`** — طلباً طلباً أو
 * «ادفع الكل». **والمدفوعُ والملغى وراء زرّ «السجل».**
 *
 * # ولماذا تُغلق نافذةُ التأكيد قبل النداء
 *
 * **نافذةُ خطوة التحقّق (`Modal`) عند `z-50`، وهذه (`Confirm`) عند
 * `z-[70]`** — فلو بقيت مفتوحةً سترت سؤالَ الكلمة. **فتُغلق أوّلاً ثمّ
 * يُنادى الخادم**، والحدُّ في الخادم لا هنا.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDate, fmtDateTime, fmtMoney, fmtNum, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, Confirm, LoadingState, Money, IconStatus, IconWallet } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { OpenLink } from "@/components/admin/OpenLink";
import { useCanOpen } from "@/lib/policy";

const m = getMessages(defaultLocale);
const S = m.admin.merchantProfile.settlement;
const C = m.admin.cashOutstanding;

type Method = "cash" | "wallet";

/** **صفُّ مستحقٍّ نقديٍّ كما يرسله المحرّك.** */
interface CashSettlement {
  id: string;
  order_id: string;
  /** رقمُ الطلب المعروف (#N) — صفرٌ إن غاب. */
  order_number?: number;
  amount: number;
  reversed_amount: number;
  outstanding: number;
  state: "cash_due" | "cash_paid" | "cash_reversed";
  paid_by?: string | null;
  paid_via?: "office" | "driver" | null;
  paid_at?: string | null;
  created_at: string;
}

interface CashSummary {
  outstanding_total: number;
  settlements: CashSettlement[];
}

const STATE_LABEL: Record<string, string> = {
  cash_due: C.stateDue,
  cash_paid: C.statePaid,
  cash_reversed: C.stateReversed,
};

const STATE_VARIANT: Record<string, "warning" | "success" | "neutral"> = {
  cash_due: "warning",
  cash_paid: "success",
  cash_reversed: "neutral",
};

function methodLabel(v: Method): string {
  return v === "cash" ? S.cash : S.wallet;
}

function orderLabel(s: CashSettlement): string {
  return s.order_number ? C.orderNo.replace("{o}", String(s.order_number)) : C.orderUnknown;
}

export function MerchantSettlement({
  merchantId,
  method,
  onChanged,
  variant = "profile",
}: {
  merchantId: string;
  method: Method;
  /** **يُعاد جلبُ ما وراء النافذة بعد أيّ تغيير** — طريقةً أو دفعاً. */
  onChanged: () => void;
  /** `pay` نافذةُ الدفع من صفحة النقد — بلا تغيير الطريقة. */
  variant?: "profile" | "pay";
}) {
  const { can } = useAuth();
  const canOrder = useCanOpen().order;
  const canManageMethod = can("settings.financial.manage");
  const canReadCash = can("finance.read");
  const canPayCash = can("finance.manage");
  const payMode = variant === "pay";

  const [pendingMethod, setPendingMethod] = useState<Method | null>(null);
  const [changing, setChanging] = useState(false);
  const [error, setError] = useState("");

  const [summary, setSummary] = useState<CashSummary | null>(null);
  const [cashError, setCashError] = useState("");
  const [payTarget, setPayTarget] = useState<CashSettlement | "all" | null>(null);
  const [paying, setPaying] = useState(false);
  const [done, setDone] = useState("");
  const [showHistory, setShowHistory] = useState(false);

  // **والمستحقُّ القديمُ لا يختفي بتغيير الطريقة** (فحصُ قسم النقد ٢٠٢٦-١٠-٠٤، المشكلة ٣):
  // يُحمَّل لمن يقرأ المال دائماً، ويُعرض إن كانت الطريقةُ نقداً أو كان له تاريخٌ نقديّ.
  const showCash =
    canReadCash &&
    (payMode || method === "cash" || (summary?.settlements.length ?? 0) > 0 || (summary?.outstanding_total ?? 0) > 0);

  const loadCash = useCallback(async () => {
    try {
      const got = await api<CashSummary>(`/api/v1/admin/merchants/${merchantId}/cash-settlements`);
      // **والقائمةُ الفارغةُ قد تصل null** — كانت تُسقط ملفَّ المتجر كلَّه (بلاغُ المالك ٢٠٢٦-١٠-٠٤).
      setSummary({ outstanding_total: got?.outstanding_total ?? 0, settlements: got?.settlements ?? [] });
      setCashError("");
    } catch (err) {
      setCashError(errorText(err));
    }
  }, [merchantId]);

  useEffect(() => {
    if (canReadCash) void loadCash();
  }, [canReadCash, loadCash]);

  async function applyChange(target: Method) {
    setPendingMethod(null);
    setChanging(true);
    setError("");
    try {
      await api(`/api/v1/admin/merchants/${merchantId}/settlement-method`, {
        method: "PATCH",
        body: JSON.stringify({ method: target }),
      });
      onChanged();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setChanging(false);
    }
  }

  const due = (summary?.settlements ?? []).filter((s) => s.state === "cash_due");
  const history = (summary?.settlements ?? []).filter((s) => s.state !== "cash_due");

  /** **الدفعُ طلباً طلباً — و«الكل» يمرّ عليها بالترتيب ويقف عند أوّل خطأ.** */
  async function pay(rows: CashSettlement[]) {
    setPayTarget(null);
    setPaying(true);
    setCashError("");
    setDone("");
    let paid = 0;
    try {
      for (const s of rows) {
        await api(`/api/v1/admin/merchant-cash-settlements/${s.id}/pay`, {
          method: "POST",
          body: JSON.stringify({ note: "" }),
        });
        paid += s.outstanding;
      }
      setDone(`${C.payDone} · ${fmtMoney(paid)}`);
    } catch (err) {
      setCashError(errorText(err));
    } finally {
      setPaying(false);
      await loadCash();
      // **وكرتُ المتجر والمجاميعُ وراء النافذة تتحدّث** (فحصُ اللوحة ٢٠٢٦-١٠-٢٧).
      if (paid > 0) onChanged();
    }
  }

  const total = summary?.outstanding_total ?? 0;

  return (
    <div className={`space-y-4 ${payMode ? "" : "mt-4"}`}>
      {!payMode && (
        <div className="surface p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p className="text-sm text-ink-muted">{S.title}</p>
              <p className="mt-0.5 font-medium">{methodLabel(method)}</p>
            </div>
            {canManageMethod && (
              <div className="flex gap-1">
                {(["cash", "wallet"] as Method[]).map((opt) => (
                  <button
                    key={opt}
                    type="button"
                    disabled={changing}
                    onClick={() => opt !== method && setPendingMethod(opt)}
                    className={`rounded-control border px-3 py-1 text-sm transition-colors ${
                      method === opt
                        ? "border-accent bg-accent-tint font-medium"
                        : "border-line text-ink-muted hover:border-accent-edge"
                    }`}
                  >
                    {methodLabel(opt)}
                  </button>
                ))}
              </div>
            )}
          </div>
          {error && <Alert className="mt-3">{error}</Alert>}
        </div>
      )}

      {showCash && (
        <div className={payMode ? "space-y-3" : "surface space-y-3 p-4"}>
          {!payMode && <h3 className="heading-card">{S.cashHistory}</h3>}

          {cashError && <Alert>{cashError}</Alert>}
          {done && (
            <Alert tone="success" onDismiss={() => setDone("")}>
              {done}
            </Alert>
          )}

          {!summary ? (
            cashError ? null : <LoadingState variant="text" />
          ) : (
            <>
              {/* ── الرقمُ الكبير: ما لم يُدفع بعد ── */}
              <div className={`flex flex-wrap items-center gap-4 rounded-card p-4 ${total > 0 ? "bg-warning-tint" : "bg-field"}`}>
                <span
                  className={`inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full ${
                    total > 0 ? "bg-warning-solid text-on-solid" : "bg-success-solid text-on-solid"
                  }`}
                >
                  <IconWallet size={22} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block text-sm text-ink-muted">{C.payDue}</span>
                  <span className={`figure block ${total > 0 ? "text-warning" : "text-ink-muted"}`}>{fmtMoney(total)}</span>
                  {due.length > 0 && (
                    <span className="block text-xs text-ink-muted">{C.payDueCount.replace("{n}", fmtNum(due.length))}</span>
                  )}
                </span>
              </div>

              {canPayCash && due.length > 1 && (
                <Button size="lg" className="w-full" disabled={paying} onClick={() => setPayTarget("all")}>
                  {C.payAll.replace("{n}", fmtMoney(total))}
                </Button>
              )}
              {!canPayCash && due.length > 0 && <p className="text-xs text-ink-muted">{C.payReadOnly}</p>}

              {/* ── كرتٌ لكلّ طلبٍ لم يُدفع ── */}
              {due.length === 0 ? (
                <div className="flex flex-col items-center gap-2 rounded-card border border-line-soft p-6 text-center">
                  <span className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-success-solid text-on-solid">
                    <IconStatus size={22} />
                  </span>
                  <span className="font-bold">{C.payNoneDue}</span>
                  {history.length > 0 && <span className="max-w-sm text-sm text-ink-muted">{C.payNoneDueHint}</span>}
                </div>
              ) : (
                <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                  {due.map((s) => (
                    <li key={s.id} className="flex flex-col gap-2 rounded-card border border-line-soft bg-field p-3 text-sm">
                      <div className="flex items-start justify-between gap-2">
                        <span className="min-w-0">
                          <OpenLink allowed={canOrder} href={`/dashboard/orders?id=${s.order_id}`} className="block font-bold hover:underline">
                            {orderLabel(s)}
                          </OpenLink>
                          <span className="block text-2xs text-ink-muted" dir="ltr">{fmtDateTime(s.created_at)}</span>
                        </span>
                        <Badge variant="warning">{C.stateDue}</Badge>
                      </div>
                      <span className="figure block text-warning">{fmtMoney(s.outstanding)}</span>
                      {s.outstanding !== s.amount && (
                        <span className="text-xs text-ink-muted">
                          {C.partialLeft.replace("{n}", fmtMoney(s.outstanding)).replace("{t}", fmtMoney(s.amount))}
                        </span>
                      )}
                      {canPayCash && (
                        <Button variant={due.length > 1 ? "secondary" : "primary"} className="w-full" disabled={paying} onClick={() => setPayTarget(s)}>
                          {C.payOne.replace("{n}", fmtMoney(s.outstanding))}
                        </Button>
                      )}
                    </li>
                  ))}
                </ul>
              )}

              {/* ── السجلّ: المدفوعُ والملغى، وراء زرّ ── */}
              {history.length > 0 && (
                <div className="space-y-2">
                  <Button variant="ghost" className="w-full" onClick={() => setShowHistory((v) => !v)}>
                    {showHistory ? C.historyHide : C.historyShow.replace("{n}", fmtNum(history.length))}
                  </Button>
                  {showHistory && (
                    <ul className="space-y-2">
                      {history.map((s) => (
                        <li key={s.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-card border border-line-soft px-3 py-2 text-sm">
                          <Badge variant={STATE_VARIANT[s.state] ?? "neutral"}>{STATE_LABEL[s.state] ?? s.state}</Badge>
                          <OpenLink allowed={canOrder} href={`/dashboard/orders?id=${s.order_id}`} className="font-medium hover:underline">
                            {orderLabel(s)}
                          </OpenLink>
                          <span className="min-w-0 flex-1 text-xs text-ink-muted">
                            {s.paid_at && C.paidOn.replace("{d}", fmtDate(s.paid_at))}
                            {s.paid_via === "office" && <> · {C.paidViaOffice}</>}
                            {s.paid_via === "driver" && <> · {C.paidViaDriver}</>}
                          </span>
                          <Money value={s.amount} small />
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      )}

      <Confirm
        open={pendingMethod !== null}
        title={S.changeTitle}
        body={
          <>
            <p>{S.changeNote}</p>
            {pendingMethod && (
              <p className="mt-2 font-medium text-ink">
                {S.changeTo.replace("{method}", methodLabel(pendingMethod))}
              </p>
            )}
          </>
        }
        confirmLabel={S.confirmChange}
        tone="primary"
        busy={changing}
        onConfirm={() => pendingMethod && void applyChange(pendingMethod)}
        onCancel={() => setPendingMethod(null)}
      />

      <Confirm
        open={payTarget !== null}
        title={payTarget === "all" ? C.payAllTitle : C.payOneTitle}
        body={
          payTarget === "all"
            ? C.payAllBody.replace("{n}", fmtMoney(total)).replace("{c}", fmtNum(due.length))
            : payTarget
              ? C.payOneBody.replace("{n}", fmtMoney(payTarget.outstanding)).replace("{o}", orderLabel(payTarget))
              : ""
        }
        confirmLabel={C.payConfirm}
        tone="primary"
        busy={paying}
        onConfirm={() => payTarget && void pay(payTarget === "all" ? due : [payTarget])}
        onCancel={() => setPayTarget(null)}
      />
    </div>
  );
}
