"use client";

/**
 * **بابُ محرّر الصنف** — يقرأ أقسامَ المتجر (وصنفَه إن كان تعديلاً) ثمّ
 * يفتح المحرّر. (مرآةُ فرع `Tab.AddItem` في `MainActivity.kt`.)
 *
 * **ومتجرٌ موقوفٌ لا يضيف**، **ومتجرٌ بلا أقسامٍ يُدلّ على أقسامه أوّلاً**
 * (`NeedSections`) — صنفٌ بلا قسمٍ لا يراه زبون.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ButtonLink, EmptyState, IconList, IconStore, LoadingState, ReloadState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MenuItem, MenuSection, SectionRef } from "@/lib/store/types";
import ItemEditor from "./ItemEditor";

const m = getMessages(defaultLocale);
const S = m.store;
const I = S.item;

export default function ItemEditorPage({ itemId }: { itemId?: string }) {
  const { store } = useStore();
  const id = store?.id ?? "";
  const [sections, setSections] = useState<SectionRef[] | null>(null);
  const [item, setItem] = useState<MenuItem | null | undefined>(itemId ? undefined : null);
  const [failed, setFailed] = useState(false);
  const [preset, setPreset] = useState("");

  const load = useCallback(async () => {
    if (!id) return;
    setFailed(false);
    try {
      const secs = await api<{ sections: SectionRef[] }>(`/api/v1/merchant/stores/${id}/sections`);
      setSections(secs.sections ?? []);
      if (itemId) {
        const menu = await api<MenuSection[]>(`/api/v1/merchant/stores/${id}/menu`);
        setItem(menu.flatMap((s) => s.items).find((i) => i.id === itemId) ?? null);
      }
    } catch {
      setFailed(true);
    }
  }, [id, itemId]);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    // **والقسمُ المفتوحُ يصير اختياراً مسبقاً** — `?section=` من زرّ «أضف صنفاً» داخل القسم.
    try {
      setPreset(new URLSearchParams(window.location.search).get("section") ?? "");
    } catch {
      setPreset("");
    }
  }, []);

  if (!store) return null;
  if (failed) return <ReloadState onRetry={() => void load()} />;
  if (sections == null || item === undefined) return <LoadingState variant="list" rows={4} />;
  if (store.status === "suspended" && !itemId) {
    return (
      <EmptyState
        icon={IconStore}
        title={S.state.suspended}
        action={<p className="text-sm text-ink-muted">{S.suspendedBanner}</p>}
      />
    );
  }
  if (itemId && item == null) {
    return (
      <EmptyState
        icon={IconList}
        title={I.notFound}
        action={<ButtonLink href="/store/menu">{I.back}</ButtonLink>}
      />
    );
  }
  if (sections.length === 0) {
    return (
      <EmptyState
        icon={IconList}
        title={I.needSectionsTitle}
        action={
          <div className="flex flex-col items-center gap-3">
            <p className="whitespace-pre-line text-sm text-ink-muted">{I.needSectionsBody}</p>
            <ButtonLink href="/store#spot-sections">{I.needSectionsGo}</ButtonLink>
          </div>
        }
      />
    );
  }
  return (
    <ItemEditor
      key={item?.id ?? `new-${preset}`}
      item={item}
      sections={sections}
      initialSection={sections.some((s) => s.id === preset) ? preset : ""}
    />
  );
}
