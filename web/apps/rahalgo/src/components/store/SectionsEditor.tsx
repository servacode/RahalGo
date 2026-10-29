"use client";

/**
 * **أقسامُ السوق — الشبكةُ نفسُها التي في المعالج** (مرآةُ `SectionsEditor` في `StoreParts.kt`).
 *
 * **وتنحفظ لحالها بلا زرّ** (طلبُ المالك ٢٠٢٦-١٠-١٢) — بعد لحظةٍ من آخر ضغطة.
 */

import { useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { useStore } from "@/lib/store/context";
import type { SectionRef } from "@/lib/store/types";
import { useShopToast } from "@/components/customer/toast";
import { SectionGrid } from "./SetupParts";

const m = getMessages(defaultLocale);
const X = m.store.sections;

export default function SectionsEditor({
  all,
  mine,
  save,
}: {
  all: SectionRef[];
  mine: SectionRef[];
  save: (ids: string[]) => Promise<void>;
}) {
  const { store } = useStore();
  const toast = useShopToast();
  const [picked, setPicked] = useState<Set<string>>(() => new Set(mine.map((s) => s.id)));
  const first = useRef(true);
  const suspended = store?.status === "suspended";

  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    const current = new Set(mine.map((s) => s.id));
    if (picked.size === current.size && [...picked].every((id) => current.has(id))) return;
    const t = setTimeout(() => {
      save([...picked]).catch((e) => toast(errorText(e), "error"));
    }, 600);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [picked]);

  if (!store || all.length === 0) return null;
  return (
    <section className="flex flex-col gap-3" aria-label={X.title} data-testid="store-sections">
      <SectionGrid
        all={all}
        picked={picked}
        disabled={suspended}
        onToggle={(id) =>
          setPicked((p) => {
            const n = new Set(p);
            if (n.has(id)) n.delete(id);
            else n.add(id);
            return n;
          })
        }
      />
      {picked.size === 0 && <p className="text-sm text-danger">{X.empty}</p>}
    </section>
  );
}
