"use client";

/**
 * **أقسامُ السوق التي فيها متجري** — مرآةُ `SectionsEditor` في `StoreScreen.kt`.
 *
 * **والأقسامُ من المنصّة لا من المتجر** (قرارُ المالك ٢٠٢٦-٠٨-٠٧) — يختار
 * منها ما يخصّه، **ويُحفظ الاختيارُ عند كلّ تبديل** كما في التطبيق، **وبلا
 * قسمٍ لا يُضاف صنف** (`NeedSections`).
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { IconClose, Select } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { SectionRef } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const X = m.store.sections;

export default function SectionsEditor({
  all,
  mine,
  onChange,
}: {
  all: SectionRef[];
  mine: SectionRef[];
  onChange: (next: SectionRef[]) => void;
}) {
  const { store } = useStore();
  const toast = useShopToast();
  const [busy, setBusy] = useState(false);
  if (!store || all.length === 0) return null;
  const suspended = store.status === "suspended";
  const picked = new Set(mine.map((s) => s.id));
  const rest = all.filter((s) => !picked.has(s.id));

  async function save(ids: string[]) {
    if (!store) return;
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${store.id}/sections`, { method: "PUT", body: JSON.stringify({ sections: ids }) });
      onChange(all.filter((s) => ids.includes(s.id)));
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="flex flex-col gap-2" aria-label={X.title} data-testid="store-sections">
      <h3 className="heading-card">{X.title}</h3>
      <p className="text-xs text-ink-muted">{X.hint}</p>
      <div className="surface flex flex-wrap gap-2 rounded-card p-3">
        {mine.length === 0 ? (
          <p className="text-sm text-ink-muted">{X.empty}</p>
        ) : (
          mine.map((s) => (
            <span key={s.id} className="flex items-center gap-1 rounded-control bg-field py-1 pe-1 ps-3 text-sm">
              {s.name}
              <button
                type="button"
                disabled={busy || suspended}
                aria-label={fill(X.remove, { name: s.name })}
                onClick={() => void save(mine.filter((x) => x.id !== s.id).map((x) => x.id))}
                className="rounded-control p-1 text-ink-muted hover:text-ink disabled:opacity-50"
              >
                <IconClose size={14} />
              </button>
            </span>
          ))
        )}
      </div>
      <Select
        aria-label={X.add}
        value=""
        disabled={busy || suspended || rest.length === 0}
        onChange={(e) => e.target.value && void save([...mine.map((x) => x.id), e.target.value])}
      >
        <option value="">{rest.length === 0 ? X.allIn : X.add}</option>
        {rest.map((s) => (
          <option key={s.id} value={s.id}>
            {s.name}
          </option>
        ))}
      </Select>
    </section>
  );
}
