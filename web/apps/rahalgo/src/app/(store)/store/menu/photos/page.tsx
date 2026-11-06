"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/menu/photos — «التصوير السريع» للأصناف بلا صورة** (قرارُ المالك ٢٠٢٦-١١-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `QuickPhotoScreen` و`QuickPhotoState` في تطبيق المتجر.)
 *
 * اسمُ الصنف كبيراً، ثمّ صورةٌ من الجهاز أو من كاميرا الجوّال، ثمّ «احفظ وكمّل» فيظهر
 * الصنفُ التالي وحدَه. **والرجوعُ إليها يكمل من أوّل صنفٍ ما زال بلا صورة** — وما
 * تخطّاه في هذه الجلسة يأتي في الآخر. **والصنفُ يبقى مخفيّاً أو ظاهراً كما كان:
 * الصورةُ وحدَها تتغيّر.**
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Alert, Button, ImageUpload, LoadingState } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { ToolHeader } from "@/components/store/ToolParts";
import { quickPhotoOrder, type NoPhotoItem } from "@/lib/store/quickPhoto";

const m = getMessages(defaultLocale);
const Q = m.store.quickPhoto;
const IMAGE_EDGE = 1600;

interface Page {
  items: NoPhotoItem[];
  total: number;
  next_after: string;
}

export default function QuickPhotoPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const toast = useShopToast();
  const [items, setItems] = useState<NoPhotoItem[] | null>(null);
  const [error, setError] = useState("");
  const [cursor, setCursor] = useState(0);
  const [mediaId, setMediaId] = useState("");
  const [saving, setSaving] = useState(false);
  const skipped = useRef(new Set<string>());

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const all: NoPhotoItem[] = [];
      let after = "";
      do {
        const p = await api<Page>(
          `/api/v1/merchant/stores/${id}/menu/no-photo?limit=200${after ? `&after=${after}` : ""}`,
        );
        all.push(...p.items);
        after = p.next_after;
      } while (after);
      setItems(quickPhotoOrder(all, skipped.current));
      setCursor(0);
    } catch (e) {
      setError(errorText(e));
    }
  }, [id]);
  useEffect(() => {
    void load();
  }, [load]);

  const total = items?.length ?? 0;
  const current = items && cursor < total ? items[cursor] : null;
  const price = useMemo(
    () =>
      current
        ? [current.section_name, fmtMoney(current.merchant_price)].filter(Boolean).join(" · ")
        : "",
    [current],
  );

  function next() {
    setMediaId("");
    setCursor((c) => c + 1);
  }

  async function saveAndNext() {
    if (!current || !mediaId) return;
    setSaving(true);
    try {
      await api(`/api/v1/merchant/menu/items/${current.id}`, {
        method: "PATCH",
        body: JSON.stringify({ image_media_id: mediaId }),
      });
      next();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setSaving(false);
    }
  }

  function skip() {
    if (current) skipped.current.add(current.id);
    next();
  }

  return (
    <div className="flex flex-col gap-4" data-testid="quick-photo">
      <ToolHeader title={Q.title} href="/store/menu" />
      {error && <Alert tone="error">{error}</Alert>}
      {!items && !error && <LoadingState variant="list" rows={3} />}
      {items && total === 0 && <p className="heading-section py-6 text-center">{Q.none}</p>}
      {items && total > 0 && !current && (
        <div className="flex flex-col items-center gap-3 py-6">
          <p className="heading-section text-center">{Q.done}</p>
          <Button onClick={() => void load()}>{Q.again}</Button>
        </div>
      )}
      {current && (
        <section className="surface flex flex-col items-center gap-3 rounded-card p-4">
          <p className="text-sm font-bold text-accent-text" data-testid="quick-photo-counter">
            {fill(Q.counter, { n: fmtNum(cursor + 1), total: fmtNum(total) })}
          </p>
          <h2 className="heading-page text-center">{current.name}</h2>
          <p className="text-center text-xs text-ink-muted">{price}</p>
          <ImageUpload
            key={current.id}
            kind="menu_item"
            label={Q.photo}
            onChange={setMediaId}
            api={api}
            mediaUrl={mediaUrl}
            errorText={errorText}
            path="/api/v1/merchant/media"
            maxEdge={IMAGE_EDGE}
            fit
            capture
            tile={{ pick: Q.shoot, change: Q.retake, clear: Q.clear }}
          />
          <div className="flex w-full gap-2">
            <Button variant="secondary" onClick={skip} disabled={saving}>
              {Q.skip}
            </Button>
            <Button className="flex-1" onClick={() => void saveAndNext()} disabled={!mediaId || saving}>
              {Q.saveNext}
            </Button>
          </div>
        </section>
      )}
    </div>
  );
}
