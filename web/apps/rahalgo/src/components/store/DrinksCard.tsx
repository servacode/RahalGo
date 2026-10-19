"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ المشروبات — تُكتب مرّةً وتنضاف لكلّ الأصناف** (قرارُ المالك ٢٠٢٦-١٠-١٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والمحرّكُ ينسخها** إلى كلّ صنفٍ مجموعةَ «مشروبات» اختياريّة، **ويتبع الدولاريُّ
 * سعرَ الصرف** كالأصناف (`catalog/drinks.go`). **ومحرّرُ الصنف لا يراها.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Alert, Button, Input, IconAdd, IconDelete } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { parseUsd, usdInput, usdText, usdToSyp } from "@/lib/store/usd";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const D = m.store.drinks;

interface Drink {
  name: string;
  price: number;
  price_usd?: number | null;
}
interface Row {
  name: string;
  syp: string;
  usd: string;
}

export default function DrinksCard({ storeId }: { storeId: string }) {
  const { store } = useStore();
  const toast = useShopToast();
  const rate = store?.exchange_rate ?? null;
  const [rows, setRows] = useState<Row[] | null>(null);
  const [usd, setUsd] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const fill = useCallback((ds: Drink[]) => {
    setUsd(ds.some((d) => d.price_usd != null));
    setRows(
      ds.map((d) => ({
        name: d.name,
        syp: String(d.price),
        usd: usdText(d.price_usd ?? null),
      })),
    );
  }, []);

  useEffect(() => {
    if (!storeId) return;
    api<{ drinks: Drink[] }>(`/api/v1/merchant/stores/${storeId}/drinks`)
      .then((r) => fill(r.drinks ?? []))
      .catch((e) => setError(errorText(e)));
  }, [storeId, fill]);

  const set = (i: number, p: Partial<Row>) => {
    setError("");
    setRows((rs) => (rs ?? []).map((r, j) => (j === i ? { ...r, ...p } : r)));
  };

  async function save() {
    const list = rows ?? [];
    if (list.some((r) => !r.name.trim())) return setError(D.vName);
    setBusy(true);
    setError("");
    try {
      const drinks = list.map((r) =>
        usd
          ? { name: r.name.trim(), price: 0, price_usd: parseUsd(r.usd) ?? 0 }
          : { name: r.name.trim(), price: Number(r.syp) || 0 },
      );
      const res = await api<{ drinks: Drink[] }>(
        `/api/v1/merchant/stores/${storeId}/drinks`,
        {
          method: "PUT",
          body: JSON.stringify({ drinks }),
        },
      );
      fill(res.drinks ?? []);
      toast(D.saved, "success");
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (rows === null) return error ? <Alert>{error}</Alert> : null;

  return (
    <section
      className="surface flex flex-col gap-2 rounded-card p-3"
      aria-label={D.title}
      data-testid="store-drinks"
    >
      <h3 className="heading-card">{D.title}</h3>
      <p className="text-xs text-ink-muted">{D.hint}</p>
      <div
        role="radiogroup"
        aria-label={D.currency}
        className="inline-flex w-fit overflow-hidden rounded-control border border-line"
      >
        {([false, true] as const).map((v) => (
          <button
            key={String(v)}
            type="button"
            role="radio"
            aria-checked={usd === v}
            onClick={() => setUsd(v)}
            className={`px-3 py-1 text-sm ${usd === v ? "bg-accent text-on-bright" : "text-ink"}`}
          >
            {v ? D.usd : D.syp}
          </button>
        ))}
      </div>
      {rows.length === 0 && <p className="text-sm text-ink-muted">{D.empty}</p>}
      {rows.map((r, i) => {
        const u = parseUsd(r.usd);
        return (
          <div key={i} className="flex items-end gap-2">
            <Input
              id={`drink-name-${i}`}
              label={D.name}
              value={r.name}
              onChange={(e) => set(i, { name: e.target.value })}
              wrapperClassName="min-w-0 flex-1"
            />
            <Input
              id={`drink-price-${i}`}
              label={usd ? D.priceUsd : D.priceSyp}
              inputMode={usd ? "decimal" : "numeric"}
              dir="ltr"
              value={usd ? r.usd : r.syp}
              onChange={(e) =>
                usd
                  ? set(i, { usd: usdInput(e.target.value, 8) })
                  : set(i, {
                      syp: e.target.value.replace(/\D/g, "").slice(0, 9),
                    })
              }
              title={
                usd && rate != null && u != null && u > 0
                  ? fmtMoney(usdToSyp(u, rate))
                  : undefined
              }
              wrapperClassName="w-28 shrink-0"
            />
            <Button
              type="button"
              variant="ghost"
              aria-label={D.remove}
              onClick={() => setRows(rows.filter((_, j) => j !== i))}
            >
              <IconDelete size={14} className="text-danger" />
            </Button>
          </div>
        );
      })}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="ghost"
          className="flex items-center gap-1"
          onClick={() => setRows([...rows, { name: "", syp: "", usd: "" }])}
        >
          <IconAdd size={13} />
          {D.add}
        </Button>
        <Button
          type="button"
          onClick={() => void save()}
          disabled={busy}
          className="ms-auto"
        >
          {D.save}
        </Button>
      </div>
      {error && <Alert>{error}</Alert>}
    </section>
  );
}
