"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **البضاعةُ الراجعة — العمليّاتُ تقول أين، والماليّةُ تكتب كم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤، قسمُ «الطلبات»: البنود ١٢ و١٣ و١٤.)
 *
 *	الوجهة   بيد `orders.intervene` — **وبعد «سلّمت البضاعة» وحدَه**، ووجهتُها
 *	         مختارةٌ سلفاً من مشوار الإرجاع (`return_to`)
 *	التعويض  بيد `finance.manage` — **بسقف سعر شراء البضاعة** وتأكيدِ كلمة السرّ
 *	         (يلتقطه `StepUpGate` المركزيّ)، **ومرّةً واحدة**
 *
 * **كانت الصلاحيّةُ معكوسة**: الشاشةُ تعرض الزرَّ للماليّة والمحرّكُ يطلب
 * العمليّات — فموظّفُ الماليّة يرى «ممنوع» في كلّ ضغطة.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney, errorText } from "@rahalgo/i18n";
import { Alert, Button, FormActions, Input } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const OP = m.admin.ordersPage;
const G = OP.board.goods;

export function GoodsBox({
  orderId,
  settledTo,
  returnTo,
  handedAt,
  acceptsReturns,
  goodsCost,
  compensated,
  canIntervene,
  canFinance,
  onChanged,
}: {
  orderId: string;
  settledTo: "merchant" | "platform" | null;
  returnTo: "office" | "store" | null | undefined;
  handedAt: string | null | undefined;
  acceptsReturns: boolean;
  goodsCost: number;
  compensated: number;
  canIntervene: boolean;
  canFinance: boolean;
  onChanged: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [comp, setComp] = useState<string | null>(null);

  async function settle(to: "merchant" | "platform") {
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/goods`, {
        method: "POST",
        body: JSON.stringify({ to }),
      });
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function compensate() {
    const amount = Math.round(Number(comp));
    if (!Number.isFinite(amount) || amount <= 0 || amount > goodsCost) {
      setErr(G.compBad);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/goods/compensation`, {
        method: "POST",
        body: JSON.stringify({ amount }),
      });
      setComp(null);
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (comp !== null) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium">{G.compTitle}</p>
        <p className="text-xs text-ink-muted">
          {G.compHint.replace("{max}", fmtMoney(goodsCost))}
        </p>
        <Input
          type="number"
          inputMode="numeric"
          placeholder={G.compLabel}
          value={comp}
          onChange={(e) => {
            setComp(e.target.value);
            setErr("");
          }}
        />
        {err && <p className="text-xs text-danger">{err}</p>}
        <FormActions
          busy={busy}
          onSave={() => void compensate()}
          onCancel={() => {
            setComp(null);
            setErr("");
          }}
          saveLabel={G.compSend}
        />
      </div>
    );
  }

  // ── لم تُحسم بعد ────────────────────────────────────────────────
  if (settledTo === null) {
    if (!canIntervene) {
      return <span className="text-xs text-ink-muted">{G.opsOnly}</span>;
    }
    // **ومشوارٌ قائمٌ لم يُسلَّم** — لا حسمَ لبضاعةٍ في صندوق السائق.
    if (returnTo && !handedAt) {
      return <span className="text-xs text-warning">{G.waitingHand}</span>;
    }
    return (
      <>
        {/* **والوجهةُ مختارةٌ سلفاً من المشوار** — زرٌّ واحدٌ يؤكّدها. وطلبٌ فشل
            بلا مشوارٍ يبقى بزرّيه كما كان. */}
        {(returnTo === "store" || (!returnTo && acceptsReturns)) && (
          <Button variant="secondary" disabled={busy} onClick={() => void settle("merchant")}>
            {returnTo ? G.confirmStore : OP.goodsToMerchant}
          </Button>
        )}
        {(returnTo === "office" || !returnTo) && (
          <Button variant="secondary" disabled={busy} onClick={() => void settle("platform")}>
            {returnTo ? G.confirmOffice : OP.goodsToPlatform}
          </Button>
        )}
        {err && <Alert className="w-full">{err}</Alert>}
      </>
    );
  }

  // ── حُسمت ───────────────────────────────────────────────────────
  return (
    <>
      <span className="text-xs text-ink-muted">
        {settledTo === "merchant" ? OP.goodsSettledMerchant : OP.goodsSettledPlatform}
      </span>
      {settledTo === "merchant" && compensated > 0 && (
        <span className="text-xs text-success">
          {G.compensated.replace("{amount}", fmtMoney(compensated))}
        </span>
      )}
      {settledTo === "merchant" && compensated === 0 && canFinance && goodsCost > 0 && (
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => {
            setErr("");
            setComp("");
          }}
        >
          {G.compensate}
        </Button>
      )}
      {err && <Alert className="w-full">{err}</Alert>}
    </>
  );
}
