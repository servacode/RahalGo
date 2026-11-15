"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«هل هذا عنوانك الصحيح؟» — مرآةُ نافذة تطبيق الزبون** (قرارُ المالك ٢٠٢٦-١١-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * سُلّم الطلبُ بعيداً عن دبّوس عنوانه المحفوظ، **فيُسأل مرّة**: «اعتمد هذا الموقع» ⇒ نقطةُ
 * عنوانه تصير مكانَ التسليم (`POST …/accept-dropoff-fix`)؛ «لا، عنواني صحيح» ⇒ يُمحى
 * الاقتراحُ ولا يُسأل ثانيةً (`DELETE`). **ولا يتغيّر شيءٌ بلا جوابه.**
 *
 * **والخريطةُ `PickMap` للعرض** (كخريطة المكتب في `ContactMap`) — النقرةُ تُبتلع.
 */

import dynamic from "next/dynamic";
import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { MyOrder } from "@/lib/customer/types";
import { OUTLINE_CARD } from "@/components/customer/kit";

const O = getMessages(defaultLocale).shop.orders;

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((x) => x.PickMap), { ssr: false });

/** **أوّلُ طلبٍ عليه اقتراح** — كما يختار التطبيقُ نافذتَه (`pendingDropoffFix`). */
export function pendingDropoffFix(orders: MyOrder[] | null | undefined): MyOrder | null {
  return (orders ?? []).find((o) => o.dropoff_suggest_lat != null && o.dropoff_suggest_lng != null) ?? null;
}

export function DropoffFix({
  order,
  onDone,
  onMessage,
}: {
  order: MyOrder;
  onDone: () => void;
  onMessage?: (text: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [answered, setAnswered] = useState(false);
  const lat = order.dropoff_suggest_lat;
  const lng = order.dropoff_suggest_lng;
  if (answered || lat == null || lng == null) return null;

  const answer = async (accept: boolean) => {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${order.id}/accept-dropoff-fix`, accept ? { method: "POST", body: "{}" } : { method: "DELETE" });
      setAnswered(true);
      if (accept) onMessage?.(O.fixDone);
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className={`${OUTLINE_CARD} flex flex-col gap-3`} aria-labelledby={`fix-${order.id}`}>
      <h2 id={`fix-${order.id}`} className="text-card-title">
        {O.fixTitle}
      </h2>
      <p className="text-sm text-ink-muted">{O.fixBody}</p>
      <div className="overflow-hidden rounded-control border border-line">
        <PickMap lat={lat} lng={lng} height="h-44" hideLocate onPick={() => undefined} />
      </div>
      <p className="text-caption text-ink-muted">{O.fixMapHint}</p>
      {error && <Alert>{error}</Alert>}
      <div className="flex flex-col gap-rg-sm sm:flex-row [&>*]:min-w-0 sm:[&>*]:flex-1">
        <Button className="min-h-12" disabled={busy} onClick={() => answer(true)}>
          {O.fixYes}
        </Button>
        <Button variant="secondary" className="min-h-12" disabled={busy} onClick={() => answer(false)}>
          {O.fixNo}
        </Button>
      </div>
    </section>
  );
}
