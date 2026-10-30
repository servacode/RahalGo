"use client";

/**
 * **قائمةُ مشروبات المتجر من اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «المجموعات بتطبيق المتجر نفسهن لازم
 * يكونو باللوحة»).
 *
 * نسخةُ الإدارة من `store/DrinksEditor.tsx`: تُكتب مرّةً **وينسخها المحرّكُ إلى كلّ صنف** مجموعةَ
 * «مشروبات» اختياريّة (`catalog/drinks.go`)، والدولاريُّ يُحسب بسعر صرف المتجر في الخادم.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, Input, IconAdd, IconDelete, useToast } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { parseUsd, usdInput, usdText } from "@/lib/store/usd";
import ImageUpload from "@/components/admin/ImageUpload";

const m = getMessages(defaultLocale);
const D = m.store.drinks;

interface Drink {
  name: string;
  price: number;
  price_usd?: number | null;
  image_media_id?: string | null;
  image_url?: string | null;
  image_thumb_url?: string | null;
}
interface Row {
  /** **مفتاحٌ ثابتٌ للسطر** — فحذفُ سطرٍ لا ينقل صورتَه إلى جاره. */
  key: number;
  name: string;
  syp: string;
  usd: string;
  /** **صورةُ المشروب** (٢٠٢٦-١٠-٣٠) — اختياريّة، وتُحفظ كما هي ما لم تُبدَّل. */
  imageId: string | null;
  imageUrl: string | null;
}

let rowSeq = 0;
const newRow = (): Row => ({ key: ++rowSeq, name: "", syp: "", usd: "", imageId: null, imageUrl: null });

export default function DrinksAdmin({ merchantId, canEdit }: { merchantId: string; canEdit: boolean }) {
  const { push, Toaster } = useToast();
  const [rows, setRows] = useState<Row[] | null>(null);
  const [usd, setUsd] = useState(false);
  // **للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — وغيابُه من الردّ = مسموح.
  const [allowed, setAllowed] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const path = `/api/v1/admin/merchants/${merchantId}/drinks`;

  const fill = useCallback((ds: Drink[]) => {
    setUsd(ds.some((d) => d.price_usd != null));
    setRows(
      ds.map((d) => ({
        key: ++rowSeq,
        name: d.name,
        syp: String(d.price),
        usd: usdText(d.price_usd ?? null),
        imageId: d.image_media_id ?? null,
        imageUrl: d.image_thumb_url ?? d.image_url ?? null,
      })),
    );
  }, []);

  useEffect(() => {
    api<{ drinks: Drink[]; allowed?: boolean }>(path)
      .then((r) => {
        setAllowed(r.allowed !== false);
        fill(r.drinks ?? []);
      })
      .catch((e) => setError(errorText(e)));
  }, [path, fill]);

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
          ? { name: r.name.trim(), price: 0, price_usd: parseUsd(r.usd) ?? 0, image_media_id: r.imageId }
          : { name: r.name.trim(), price: Number(r.syp) || 0, image_media_id: r.imageId },
      );
      const res = await api<{ drinks: Drink[] }>(path, { method: "PUT", body: JSON.stringify({ drinks }) });
      fill(res.drinks ?? []);
      push((res.drinks?.length ?? 0) === 0 ? D.cleared : m.admin.market.drinksSaved);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (rows === null) return error ? <Alert>{error}</Alert> : null;

  if (!allowed) {
    return (
      <section className="surface mb-3 flex flex-col gap-1 p-3" aria-label={D.title}>
        <h3 className="heading-card">{D.title}</h3>
        <p className="text-sm text-ink-muted" data-testid="drinks-not-allowed">{D.notAllowed}</p>
      </section>
    );
  }

  return (
    <section className="surface mb-3 flex flex-col gap-2 p-3" aria-label={D.title}>
      <h3 className="heading-card">{D.title}</h3>
      <p className="text-xs text-ink-muted">{D.hint}</p>
      <div role="radiogroup" aria-label={D.currency} className="inline-flex w-fit overflow-hidden rounded-control border border-line">
        {([false, true] as const).map((v) => (
          <button
            key={String(v)}
            type="button"
            role="radio"
            aria-checked={usd === v}
            disabled={!canEdit}
            onClick={() => setUsd(v)}
            className={`px-3 py-1 text-sm ${usd === v ? "bg-accent text-on-bright" : "text-ink"}`}
          >
            {v ? D.usd : D.syp}
          </button>
        ))}
      </div>
      {rows.length === 0 && <p className="text-sm text-ink-muted">{D.empty}</p>}
      {rows.map((r, i) => (
        <div key={r.key} className="flex items-end gap-2">
          <ImageUpload
            kind="menu_item"
            label={D.image}
            initialUrl={r.imageUrl}
            onChange={(id) => set(i, { imageId: id || null })}
            compact={{ pick: D.pickImage, change: D.changeImage, clear: D.clearImage }}
            disabled={!canEdit}
          />
          <Input
            id={`adm-drink-name-${i}`}
            label={D.name}
            value={r.name}
            disabled={!canEdit}
            onChange={(e) => set(i, { name: e.target.value })}
            wrapperClassName="min-w-0 flex-1"
          />
          <Input
            id={`adm-drink-price-${i}`}
            label={usd ? D.priceUsd : D.priceSyp}
            inputMode={usd ? "decimal" : "numeric"}
            dir="ltr"
            disabled={!canEdit}
            value={usd ? r.usd : r.syp}
            onChange={(e) =>
              usd ? set(i, { usd: usdInput(e.target.value, 8) }) : set(i, { syp: e.target.value.replace(/\D/g, "").slice(0, 9) })
            }
            wrapperClassName="w-28 shrink-0"
          />
          {canEdit && (
            <Button
              type="button"
              variant="ghost"
              aria-label={D.remove}
              title={D.remove}
              onClick={() => setRows(rows.filter((_, j) => j !== i))}
              className="!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full"
            >
              <IconDelete size={16} className="text-danger" />
            </Button>
          )}
        </div>
      ))}
      {canEdit && (
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant="ghost" className="flex items-center gap-1" onClick={() => setRows([...rows, newRow()])}>
            <IconAdd size={13} />
            {D.add}
          </Button>
          <Button type="button" onClick={() => void save()} disabled={busy} className="ms-auto">
            {D.save}
          </Button>
        </div>
      )}
      {error && <Alert>{error}</Alert>}
      <Toaster />
    </section>
  );
}
