"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«خيارات الأصناف» — مجموعاتُ الخيارات التي تظهر في محرّر الصنف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `OptionPresetsEditor` و`OptionPresetsState` في `ui/menu/MenuTools.kt`،
 *  تصميمُ ٢٠٢٦-١٠-٢٩.) بطاقاتٌ متساويةٌ في قائمة: الاسمُ ورقاقةُ طريقة السعر،
 * وتحتهما معاينةُ القيم، **ومربّعُ اختيارٍ واضح — والبطاقةُ كلُّها تُضغط.**
 *
 * **الافتراضُ مجموعاتُ تصنيف المتجر** (`custom=false`)، **ويختار صاحبُه غيرَها**
 * (`PUT {preset_ids}`) **أو يرجع لتصنيفه** (`PUT {reset:true}`). **وحفظٌ لم
 * يُغيّر شيئاً لا يُرسَل** — كان يقلب المتجرَ من «تصنيفه» إلى «اختياره».
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, FormActions, IconCheck, IconClose, IconHistory, LoadingState, ReloadState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { OptionPreset, StorePresets } from "@/lib/store/types";
import { priceModeOf } from "@/lib/store/optionModes";
import { useShopToast } from "@/components/customer/toast";
import { TagChip, ToolHeader } from "./ToolParts";

const m = getMessages(defaultLocale);
const O = m.store.options;
const I = m.store.item;
const BACK = "/store/menu";

function ModeChip({ p }: { p: OptionPreset }) {
  const mode = priceModeOf(p.price_mode);
  if (mode === "none") return <TagChip text={I.chipNone} tone="muted" />;
  if (mode === "weight") return <TagChip text={I.chipWeight} tone="warning" />;
  return <TagChip text={I.chipPriced} tone="accent" />;
}

export default function OptionPresetsEditor() {
  const { store } = useStore();
  const router = useRouter();
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

  async function send(body: { preset_ids: string[] } | { reset: true }, ok: string, leave: boolean) {
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
      if (leave) router.push(BACK);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!store) return null;
  const suspended = store.status === "suspended";
  const presets = data?.presets ?? [];
  const unchanged = presets.every((p) => !!p.chosen === picked.has(p.id));

  function submit() {
    if (unchanged) return router.push(BACK);
    void send({ preset_ids: presets.filter((p) => picked.has(p.id)).map((p) => p.id) }, O.saved, true);
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-rg-lg md:mx-0 md:max-w-none" data-testid="store-option-presets">
      <ToolHeader title={O.title} href={BACK} />
      {failed ? (
        <ReloadState onRetry={() => void load()} />
      ) : data == null ? (
        <LoadingState variant="list" rows={4} label={O.loading} />
      ) : presets.length === 0 ? (
        <p className="text-body text-ink-muted">{O.empty}</p>
      ) : (
        <>
          <ul className="flex flex-col gap-rg-sm">
            {presets.map((p) => {
              const on = picked.has(p.id);
              const values = p.options.map((o) => o.name).join(" · ");
              return (
                <li key={p.id}>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={on}
                    disabled={busy || suspended}
                    data-testid="option-preset"
                    onClick={() =>
                      setPicked((s) => {
                        const next = new Set(s);
                        if (on) next.delete(p.id);
                        else next.add(p.id);
                        return next;
                      })
                    }
                    className={`flex h-[4.75rem] w-full items-center gap-rg-md rounded-control border px-rg-md text-start transition-colors disabled:opacity-60 ${
                      on ? "border-2 border-accent bg-accent-tint" : "border-line bg-surface hover:bg-row-hover"
                    }`}
                  >
                    <span className="flex min-w-0 flex-1 flex-col gap-rg-xxs">
                      <span className="flex min-w-0 items-center gap-rg-sm">
                        <span className="truncate font-bold text-ink">{p.name}</span>
                        <ModeChip p={p} />
                      </span>
                      {values && <span className="break-words text-caption text-ink-muted">{values}</span>}
                    </span>
                    <span
                      className={`flex h-6 w-6 shrink-0 items-center justify-center rounded border-2 ${
                        on ? "border-accent bg-accent text-on-bright" : "border-line bg-surface"
                      }`}
                      aria-hidden
                    >
                      {on && <IconCheck size={16} />}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
          {error && <Alert>{error}</Alert>}
          {!suspended && (
            <FormActions
              onSave={submit}
              onCancel={() => router.push(BACK)}
              busy={busy}
              saveIcon={<IconCheck size={18} />}
              cancelIcon={<IconClose size={18} />}
            />
          )}
          {data.custom && !suspended && (
            <Button
              variant="ghost"
              className="inline-flex w-full items-center justify-center gap-rg-sm"
              disabled={busy}
              onClick={() => void send({ reset: true }, O.resetDone, false)}
            >
              <IconHistory size={18} />
              {O.reset}
            </Button>
          )}
        </>
      )}
    </div>
  );
}
