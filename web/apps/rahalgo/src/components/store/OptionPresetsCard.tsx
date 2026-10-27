"use client";

/**
 * **«خياراتي» — مجموعاتُ الخيارات التي تظهر في محرّر الصنف** (قرارُ المالك ٢٠٢٦-١٠-٢٧)
 *
 * **الافتراضُ مجموعاتُ تصنيف المتجر** (`custom=false`)، **ويختار صاحبُه غيرَها**
 * (`PUT {preset_ids}`) **أو يرجع لتصنيفه** (`PUT {reset:true}`). والردُّ في الحالين
 * هو ردُّ القراءة نفسُه — فيُعرض كما جاء.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, Checkbox, LoadingState, ReloadState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { StorePresets } from "@/lib/store/types";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const O = m.store.options;

export default function OptionPresetsCard() {
  const { store } = useStore();
  const toast = useShopToast();
  const id = store?.id ?? "";
  const [data, setData] = useState<StorePresets | null>(null);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const show = (r: StorePresets) => {
    setData(r);
    setPicked(new Set((r.presets ?? []).filter((p) => p.chosen).map((p) => p.id)));
  };

  const load = useCallback(async () => {
    if (!id) return;
    setFailed(false);
    try {
      show(await api<StorePresets>(`/api/v1/merchant/stores/${id}/option-presets`));
    } catch {
      setFailed(true);
    }
  }, [id]);
  useEffect(() => {
    void load();
  }, [load]);

  async function send(body: { preset_ids: string[] } | { reset: true }, ok: string) {
    setBusy(true);
    setError("");
    try {
      show(
        await api<StorePresets>(`/api/v1/merchant/stores/${id}/option-presets`, {
          method: "PUT",
          body: JSON.stringify(body),
        }),
      );
      toast(ok, "success");
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!store) return null;
  const suspended = store.status === "suspended";
  const presets = data?.presets ?? [];

  return (
    <section className="surface flex flex-col gap-3 rounded-card p-4" aria-label={O.title} data-testid="store-option-presets">
      <h3 className="heading-card">{O.title}</h3>
      <p className="text-xs text-ink-muted">{O.hint}</p>
      {failed ? (
        <ReloadState onRetry={() => void load()} />
      ) : data == null ? (
        <LoadingState variant="list" rows={3} />
      ) : presets.length === 0 ? (
        <p className="text-sm text-ink-muted">{O.empty}</p>
      ) : (
        <>
          <p className="text-xs font-bold text-accent-text">{data.custom ? O.custom : O.followsCategory}</p>
          <div className="flex flex-col gap-2">
            {presets.map((p) => (
              <Checkbox
                key={p.id}
                id={`opt-preset-${p.id}`}
                checked={picked.has(p.id)}
                disabled={busy || suspended}
                onChange={(e) =>
                  setPicked((s) => {
                    const next = new Set(s);
                    if (e.target.checked) next.add(p.id);
                    else next.delete(p.id);
                    return next;
                  })
                }
                label={
                  <span className="flex flex-wrap items-center gap-1">
                    <span className="font-medium">{p.name}</span>
                    {p.options.map((o, i) => (
                      <span key={`${o.name}-${i}`} className="rounded-control bg-field px-2 text-xs text-ink-muted">
                        {o.name}
                      </span>
                    ))}
                  </span>
                }
              />
            ))}
          </div>
          {error && <Alert>{error}</Alert>}
          <div className="flex flex-wrap gap-2">
            <Button
              disabled={busy || suspended}
              onClick={() => void send({ preset_ids: presets.filter((p) => picked.has(p.id)).map((p) => p.id) }, O.saved)}
            >
              {O.save}
            </Button>
            {data.custom && (
              <Button variant="secondary" disabled={busy || suspended} onClick={() => void send({ reset: true }, O.resetDone)}>
                {O.reset}
              </Button>
            )}
          </div>
        </>
      )}
    </section>
  );
}
