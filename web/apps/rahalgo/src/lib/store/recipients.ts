"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مستلموه السابقون يُقترحون وقتَ الكتابة** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `DeliveryViewModel.onName/onPhone/pickRecipient/hideRecipient` في تطبيق المتجر.)
 *
 * **حرفان في الاسم أو الرقم ⇒ قائمةٌ تحت الحقل**، ثمانيةٌ على الأكثر، الأحدثُ أوّلاً.
 * **وتعثّرُ الاقتراح صامت** — النموذجُ يعمل كما كان بلاه، **والردُّ المتأخّرُ لما كُتب قبله
 * لا يُعرض** (`seq`).
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { RECIPIENT_DEBOUNCE_MS, shouldSearchRecipients } from "./recipientSuggest";
import type { DeliveryRecipient } from "./types";

export type RecipientField = "" | "name" | "phone";

export function useRecipientSuggest(storeId: string) {
  /** **الحقلُ الذي تظهر تحته القائمة** — وفارغٌ لا قائمة. */
  const [field, setField] = useState<RecipientField>("");
  const [items, setItems] = useState<DeliveryRecipient[]>([]);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const seq = useRef(0);

  const close = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    seq.current++;
    setField("");
    setItems([]);
  }, []);

  /** **يكتب في الاسم أو الرقم** — والقائمةُ تتبع ما كُتب بعد ٣٠٠ مللي ثانية. */
  const type = useCallback(
    (f: Exclude<RecipientField, "">, text: string) => {
      if (timer.current) clearTimeout(timer.current);
      const s = ++seq.current;
      if (!storeId || !shouldSearchRecipients(text)) {
        setField("");
        setItems([]);
        return;
      }
      timer.current = setTimeout(async () => {
        try {
          const r = await api<{ recipients: DeliveryRecipient[] }>(
            `/api/v1/merchant/stores/${storeId}/delivery-recipients?q=${encodeURIComponent(text.trim())}`,
          );
          if (s !== seq.current) return;
          const list = r.recipients ?? [];
          setItems(list);
          setField(list.length > 0 ? f : "");
        } catch {
          /* **وتعثّرُه لا يُقال** — قائمةٌ لا تظهر خيرٌ من خطأٍ فوق نموذجٍ يعمل. */
          if (s !== seq.current) return;
          setItems([]);
          setField("");
        }
      }, RECIPIENT_DEBOUNCE_MS);
    },
    [storeId],
  );

  /** **«شيلو من القائمة»** — يختفي الآن، ويرجع وحدَه إن بعث له من جديد. */
  const hide = useCallback(
    (r: DeliveryRecipient) => {
      const next = items.filter((x) => x.phone !== r.phone);
      setItems(next);
      if (next.length === 0) setField("");
      // **والرقمُ أرقاماً بلا `+`** — والمحرّكُ يوحّده (`RecipientPhoneKey`).
      api(`/api/v1/merchant/stores/${storeId}/delivery-recipients/${r.phone.replace(/\D/g, "")}`, {
        method: "DELETE",
      }).catch((err: unknown) => {
        /* **وتعثّرُ الإخفاء صامت** — يرجع في البحث التالي فيُعاد. */
        void err;
      });
    },
    [storeId, items],
  );

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  return { field, items, type, close, hide };
}
