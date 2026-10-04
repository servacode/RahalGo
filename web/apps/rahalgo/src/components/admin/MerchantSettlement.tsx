"use client";

/**
 * **تسويةُ مستحقّات المتجر — الطريقةُ والمستحقّاتُ النقديّة.**
 *
 * # ما هنا
 *
 * **طريقةُ التسوية** (نقد/محفظة): تُعرَض للجميع، **وتُغيَّر بقدرةِ
 * `settings.financial.manage`** — وهي فعلٌ ماليٌّ خلفَ تأكيدٍ ثمّ خطوةِ
 * تحقّق. **والتغييرُ على الطلبات الجديدة وحدَها.**
 *
 * **وكشفُ المستحقّات النقديّة** (بقدرةِ `finance.read`): مجموعُ ما لم
 * يُسدَّد وسطرُ كلِّ مستحقّ، **وتأكيدُ الدفع نقداً بقدرةِ `finance.manage`**
 * خلفَ تأكيدٍ وخطوةِ تحقّق.
 *
 * # ولماذا تُغلق نافذةُ التأكيد قبل النداء
 *
 * **نافذةُ خطوة التحقّق (`Modal`) عند `z-50`، وهذه (`Confirm`) عند
 * `z-[70]`** — فلو بقيت مفتوحةً سترت سؤالَ الكلمة. **فتُغلق أوّلاً ثمّ
 * يُنادى الخادم**، والحدُّ في الخادم لا هنا.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDate, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, Confirm, LoadingState, Money } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const S = m.admin.merchantProfile.settlement;

type Method = "cash" | "wallet";

/** **صفُّ مستحقٍّ نقديٍّ كما يرسله المحرّك.** */
interface CashSettlement {
  id: string;
  order_id: string;
  amount: number;
  reversed_amount: number;
  outstanding: number;
  state: "cash_due" | "cash_paid" | "cash_reversed";
  paid_by?: string | null;
  paid_at?: string | null;
  created_at: string;
}

interface CashSummary {
  outstanding_total: number;
  settlements: CashSettlement[];
}

const STATE_LABEL: Record<string, string> = {
  cash_due: S.stateDue,
  cash_paid: S.statePaid,
  cash_reversed: S.stateReversed,
};

const STATE_VARIANT: Record<string, "warning" | "success" | "danger"> = {
  cash_due: "warning",
  cash_paid: "success",
  cash_reversed: "danger",
};

function methodLabel(v: Method): string {
  return v === "cash" ? S.cash : S.wallet;
}

export function MerchantSettlement({
  merchantId,
  method,
  onChanged,
}: {
  merchantId: string;
  method: Method;
  /** **يُعاد جلبُ المتجر بعد التغيير** — ليُقرأ الوضعُ الجديدُ من القاعدة. */
  onChanged: () => void;
}) {
  const { can } = useAuth();
  const canManageMethod = can("settings.financial.manage");
  const canReadCash = can("finance.read");
  const canPayCash = can("finance.manage");

  const [pendingMethod, setPendingMethod] = useState<Method | null>(null);
  const [changing, setChanging] = useState(false);
  const [error, setError] = useState("");

  const [summary, setSummary] = useState<CashSummary | null>(null);
  const [cashError, setCashError] = useState("");
  const [payId, setPayId] = useState<string | null>(null);
  const [paying, setPaying] = useState(false);

  // **والمستحقُّ القديمُ لا يختفي بتغيير الطريقة** (فحصُ قسم النقد ٢٠٢٦-١٠-٠٤، المشكلة ٣):
  // كان الكشفُ لا يُعرض إلّا إن كانت الطريقةُ اليومَ «نقد» — فمتجرٌ له مئتا ألفٍ
  // حُوّل إلى المحفظة صار مستحقُّه بلا من يراه أو يؤكّد دفعه. فيُحمَّل لمن يقرأ
  // المال دائماً، ويُعرض إن كانت الطريقةُ نقداً أو كان له تاريخٌ نقديّ.
  const loadCashAllowed = canReadCash;
  const showCash =
    canReadCash &&
    (method === "cash" || (summary?.settlements.length ?? 0) > 0 || (summary?.outstanding_total ?? 0) > 0);

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
    if (loadCashAllowed) void loadCash();
  }, [loadCashAllowed, loadCash]);

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

  async function pay(id: string) {
    setPayId(null);
    setPaying(true);
    setCashError("");
    try {
      await api(`/api/v1/admin/merchant-cash-settlements/${id}/pay`, {
        method: "POST",
        body: JSON.stringify({ note: "" }),
      });
      await loadCash();
    } catch (err) {
      setCashError(errorText(err));
    } finally {
      setPaying(false);
    }
  }

  return (
    <div className="mt-4 space-y-4">
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

      {showCash && (
        <div className="surface p-4">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
            <span className="heading-card">{S.cashHistory}</span>
            <span className="flex items-center gap-2 text-sm">
              <span className="text-ink-muted">{S.outstandingTotal}</span>
              <Money value={summary?.outstanding_total ?? 0} small />
            </span>
          </div>
          {cashError && <Alert>{cashError}</Alert>}
          {!summary && !cashError ? (
            <LoadingState variant="text" />
          ) : summary && summary.settlements.length === 0 ? (
            <p className="py-6 text-center text-sm text-ink-muted">{S.empty}</p>
          ) : (
            <ul className="divide-y divide-line">
              {(summary?.settlements ?? []).map((s) => (
                <li
                  key={s.id}
                  className="flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-sm"
                >
                  <Badge variant={STATE_VARIANT[s.state] ?? "warning"}>
                    {STATE_LABEL[s.state] ?? s.state}
                  </Badge>
                  <span className="min-w-0 flex-1">
                    <Money value={s.amount} small />
                    {s.state === "cash_due" && s.outstanding !== s.amount && (
                      <span className="text-ink-muted">
                        {" · "}
                        {S.outstanding} <Money value={s.outstanding} small />
                      </span>
                    )}
                  </span>
                  <span className="shrink-0 text-xs text-ink-muted">
                    {S.order}{" "}
                    <span dir="ltr" className="font-mono">
                      {s.order_id.slice(0, 8)}
                    </span>
                  </span>
                  {s.paid_at && (
                    <span dir="ltr" className="shrink-0 text-xs text-ink-muted">
                      {fmtDate(s.paid_at)}
                    </span>
                  )}
                  {s.state === "cash_due" && canPayCash && (
                    <Button
                      variant="primary"
                      className="!px-2.5"
                      disabled={paying}
                      onClick={() => setPayId(s.id)}
                    >
                      {S.markPaid}
                    </Button>
                  )}
                </li>
              ))}
            </ul>
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
        open={payId !== null}
        title={S.markPaidTitle}
        body={S.markPaidNote}
        confirmLabel={S.markPaid}
        tone="primary"
        busy={paying}
        onConfirm={() => payId && void pay(payId)}
        onCancel={() => setPayId(null)}
      />
    </div>
  );
}
