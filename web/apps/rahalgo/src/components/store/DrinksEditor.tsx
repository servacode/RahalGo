"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«قائمة المشروبات» — تُكتب مرّةً وتنضاف لكلّ الأصناف** (قرارُ المالك ٢٠٢٦-١٠-١٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `DrinksEditor` في `ui/menu/Drinks.kt`، تصميمُ ٢٠٢٦-١٠-٢٩.) رأسٌ بسهم
 * رجوع، ثمّ «ليرة / دولار» زرّان متساويان، ثمّ سطرٌ لكلّ مشروب (صورةٌ صغيرةٌ اختياريّة
 * واسمٌ وسعرٌ وحذفٌ أحمر)، ثمّ «أضف مشروباً» و«حفظ / إلغاء». **ولا سطرَ شرح.**
 *
 * **والمحرّكُ ينسخها** إلى كلّ صنفٍ مجموعةَ «مشروبات» اختياريّة — **إلّا صنفاً
 * أطفأ «أضف مشروباتي»** — **ويتبع الدولاريُّ سعرَ الصرف** (`catalog/drinks.go`).
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Button, FormActions, IconAdd, IconCheck, IconClose, IconDelete, ImageUpload, LoadingState, ReloadState } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MenuSection } from "@/lib/store/types";
import { parseUsd, usdInput, usdText } from "@/lib/store/usd";
import { useShopToast } from "@/components/customer/toast";
import { fill as fillText } from "@/lib/customer/texts";
import { ColumnLabel, RowField, Segmented, ToolHeader } from "./ToolParts";

const m = getMessages(defaultLocale);
const D = m.store.drinks;
const BACK = "/store/menu";
/** **تصغيرُ الصورة قبل رفعها** — كصورة الصنف (`ItemEditor`). */
const IMAGE_EDGE = 1600;

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

export default function DrinksEditor() {
  const { store } = useStore();
  const router = useRouter();
  const toast = useShopToast();
  const storeId = store?.id ?? "";
  const [rows, setRows] = useState<Row[] | null>(null);
  const [itemCount, setItemCount] = useState(0);
  const [usd, setUsd] = useState(false);
  const [failed, setFailed] = useState(false);
  // **للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — وغيابُه من الردّ = مسموح.
  const [allowed, setAllowed] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

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

  const load = useCallback(async () => {
    if (!storeId) return;
    setFailed(false);
    try {
      const r = await api<{ drinks: Drink[]; allowed?: boolean }>(`/api/v1/merchant/stores/${storeId}/drinks`);
      setAllowed(r.allowed !== false);
      fill(r.drinks ?? []);
    } catch {
      setFailed(true);
    }
    // **كم صنفاً عند المتجر** — ليُقال بعد الحفظ كم صنفاً نال القائمة.
    api<MenuSection[]>(`/api/v1/merchant/stores/${storeId}/menu`)
      .then((menu) => setItemCount(menu.reduce((n, s) => n + s.items.length, 0)))
      .catch(() => undefined);
  }, [storeId, fill]);
  useEffect(() => {
    void load();
  }, [load]);

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
      const res = await api<{ drinks: Drink[] }>(`/api/v1/merchant/stores/${storeId}/drinks`, {
        method: "PUT",
        body: JSON.stringify({ drinks }),
      });
      const n = res.drinks?.length ?? 0;
      toast(n === 0 ? D.cleared : itemCount === 0 ? D.savedNone : fillText(D.saved, { n: fmtNum(itemCount) }), "success");
      router.push(BACK);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!store) return null;
  const suspended = store.status === "suspended";

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-4" data-testid="store-drinks">
      <ToolHeader title={D.title} href={BACK} />
      {failed ? (
        <ReloadState onRetry={() => void load()} />
      ) : rows === null ? (
        <LoadingState variant="list" rows={3} />
      ) : !allowed ? (
        <p className="text-sm text-ink-muted" data-testid="drinks-not-allowed">{D.notAllowed}</p>
      ) : (
        <>
          <Segmented
            label={D.currency}
            options={[
              ["syp", D.syp],
              ["usd", D.usd],
            ]}
            value={usd ? "usd" : "syp"}
            disabled={busy || suspended}
            onChange={(v) => setUsd(v === "usd")}
          />
          {rows.length === 0 ? (
            <p className="text-sm text-ink-muted">{D.empty}</p>
          ) : (
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-1.5">
                <span className="w-11 shrink-0" />
                <ColumnLabel text={D.name} className="min-w-0 flex-1" />
                <ColumnLabel text={usd ? D.priceUsd : D.priceSyp} className="w-24 shrink-0" />
                <span className="w-11 shrink-0" />
              </div>
              {rows.map((r, i) => (
                <div key={r.key} className="flex items-center gap-1.5" data-testid="drink-row">
                  <ImageUpload
                    kind="menu_item"
                    label={D.image}
                    initialUrl={r.imageUrl}
                    onChange={(id) => set(i, { imageId: id || null })}
                    api={api}
                    mediaUrl={mediaUrl}
                    errorText={errorText}
                    path="/api/v1/merchant/media"
                    maxEdge={IMAGE_EDGE}
                    compact={{ pick: D.pickImage, change: D.changeImage, clear: D.clearImage }}
                    disabled={busy || suspended}
                  />
                  <RowField
                    id={`drink-name-${i}`}
                    value={r.name}
                    onChange={(v) => set(i, { name: v })}
                    placeholder={D.name}
                    label={D.name}
                    disabled={busy || suspended}
                    className="min-w-0 flex-1"
                  />
                  <RowField
                    id={`drink-price-${i}`}
                    value={usd ? r.usd : r.syp}
                    onChange={(v) => (usd ? set(i, { usd: usdInput(v, 8) }) : set(i, { syp: v.replace(/\D/g, "").slice(0, 9) }))}
                    placeholder="0"
                    label={usd ? D.priceUsd : D.priceSyp}
                    numeric={!usd}
                    decimal={usd}
                    disabled={busy || suspended}
                    className="w-24 shrink-0"
                  />
                  <button
                    type="button"
                    aria-label={D.delete}
                    title={D.delete}
                    disabled={busy || suspended}
                    onClick={() => setRows(rows.filter((_, j) => j !== i))}
                    className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control text-danger hover:bg-danger-tint disabled:opacity-50"
                  >
                    <IconDelete size={18} />
                  </button>
                </div>
              ))}
            </div>
          )}
          <Button
            type="button"
            variant="secondary"
            className="inline-flex w-fit items-center gap-1.5 font-bold"
            disabled={busy || suspended}
            onClick={() => setRows([...rows, newRow()])}
          >
            <IconAdd size={18} />
            {D.add}
          </Button>
          {error && <Alert>{error}</Alert>}
          {!suspended && (
            <FormActions
              onSave={() => void save()}
              onCancel={() => router.push(BACK)}
              busy={busy}
              saveIcon={<IconCheck size={18} />}
              cancelIcon={<IconClose size={18} />}
            />
          )}
        </>
      )}
    </div>
  );
}
