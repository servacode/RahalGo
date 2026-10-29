"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ المتجر وما ينقصه — للمعالج ولـ«متجري» معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `missingRequired` و`maybeStartWizard` في `StoreViewModel.kt`.)
 *
 * **ومصدرُ «الناقص» واحد** — فلا يقول المعالجُ «ناقص» والنقطةُ الحمراءُ صامتة.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import type { MerchantStore, SectionRef } from "./types";

/** **خطواتُ المعالج بترتيبها** — والشعارُ اختياريٌّ في أوّلها. */
export type SetupStep = "logo" | "sections" | "address" | "hours" | "settlement" | "returns";

/** **أقسامُ السوق كلُّها وأقسامُ المتجر** — و`known` حين تُقرأ الثانية. */
export function useStoreSections(storeId: string) {
  const [all, setAll] = useState<SectionRef[]>([]);
  const [mine, setMine] = useState<SectionRef[]>([]);
  const [known, setKnown] = useState(false);
  const load = useCallback(async () => {
    if (!storeId) return;
    try {
      const [a, b] = await Promise.all([
        api<{ sections: SectionRef[] }>("/api/v1/merchant/platform-sections"),
        api<{ sections: SectionRef[] }>(`/api/v1/merchant/stores/${storeId}/sections`),
      ]);
      setAll(a.sections ?? []);
      setMine(b.sections ?? []);
      setKnown(true);
    } catch {
      // **وتعذّرُها لا يُقال «لا أقسام»** — تبقى مجهولةً فلا يُعرض نقص.
    }
  }, [storeId]);
  useEffect(() => {
    setKnown(false);
    void load();
  }, [load]);

  /** **حفظُ الاختيار** — `PUT /sections` ثمّ تحديثٌ محلّيّ. */
  const save = useCallback(
    async (ids: string[]) => {
      await api(`/api/v1/merchant/stores/${storeId}/sections`, {
        method: "PUT",
        body: JSON.stringify({ sections: ids }),
      });
      setMine(all.filter((s) => ids.includes(s.id)));
    },
    [storeId, all],
  );
  return { all, mine, known, save, reload: load };
}

/** **أُثبتت نقطةُ المتجر وعنوانُه؟** */
export function pointSet(s: MerchantStore): boolean {
  return s.lat != null && s.lng != null && !!s.address_text.trim();
}

/**
 * **الخطواتُ الإلزاميّةُ الناقصة** — بترتيب المعالج.
 * **والدوامُ «ناقصٌ» متى لم يُحفظ له صفٌّ** (`has_hours`).
 */
export function missingRequired(
  s: MerchantStore | null,
  sec: { all: SectionRef[]; mine: SectionRef[]; known: boolean },
): SetupStep[] {
  if (!s || s.status === "suspended" || !sec.known) return [];
  const out: SetupStep[] = [];
  if (sec.all.length > 0 && sec.mine.length === 0) out.push("sections");
  if (!pointSet(s)) out.push("address");
  if (!s.has_hours) out.push("hours");
  if (!s.settlement_confirmed) out.push("settlement");
  if (!s.returns_confirmed) out.push("returns");
  return out;
}

/**
 * **المعالج: أيبدأ، وبأيّ خطوات** — مرآةُ `maybeStartWizard`.
 *
 * **والخطواتُ تُثبَّت عند البدء** فلا تختفي خطوةٌ من العدّاد وهو فيها، **والشعارُ
 * يُضاف إن غاب ولا يفتحه وحدَه.** ومتجرٌ كاملٌ لا يرى المعالجَ أبداً.
 */
export function useSetupWizard(
  store: MerchantStore | null,
  sec: { all: SectionRef[]; mine: SectionRef[]; known: boolean },
) {
  const [steps, setSteps] = useState<SetupStep[] | null>(null);
  /** **-1 الترحيب · ثمّ الخطوات · ثمّ «جاهز»** (`=== steps.length`). */
  const [index, setIndex] = useState(-1);
  const forId = useRef<string | null>(null);

  const missing = missingRequired(store, sec);
  const key = missing.join(",");
  useEffect(() => {
    if (!store) return;
    if (steps && forId.current === store.id) return;
    if (missing.length === 0) {
      setSteps(null);
      forId.current = null;
      return;
    }
    setSteps([...(store.logo_thumb_url ? [] : (["logo"] as SetupStep[])), ...missing]);
    setIndex(-1);
    forId.current = store.id;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [store?.id, key, steps]);

  /** **«جاهز»** — يُغلق المعالج؛ وإن بقي ناقصٌ (حفظٌ سقط) عاد في الفحص التالي. */
  const finish = useCallback(() => {
    forId.current = null;
    setSteps(null);
  }, []);

  return { steps, index, setIndex, finish };
}
