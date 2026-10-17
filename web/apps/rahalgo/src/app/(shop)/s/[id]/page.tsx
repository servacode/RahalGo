"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/s/[id] — قسمُ السوق وأصنافُه من كلّ المتاجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والقسمُ هو بابُ التسوّق في رحّال غو لا المتجر** — «شاورما» يجمع
 * شاورما المطاعم كلِّها، **واسمُ المتجر على البطاقة** يفرّق بينها.
 * (والمحرّكُ لا بابَ عامّاً فيه لمتجرٍ بعينه — انظر تقريرَ المرحلة.)
 *
 * **وشريطُ الأقسام أعلاها** — ينتقل بينها بلا رجوعٍ إلى السوق.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  EmptyState,
  LoadingState,
  ReloadState,
  SectionRail,
  IconPrev,
  IconStore,
  type RailItem,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useBrowseQuery } from "@/lib/customer/context";
import type { HomePage, Item } from "@/lib/customer/types";
import { ItemGrid, ItemSheet } from "@/components/customer/items";

const m = getMessages(defaultLocale);
const SC = m.shop.section;

export default function SectionPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const router = useRouter();
  const scope = useBrowseQuery("?");
  const [home, setHome] = useState<HomePage | null>(null);
  const [items, setItems] = useState<Item[] | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);

  useEffect(() => {
    api<HomePage>(`/api/v1/public/home${scope}`)
      .then(setHome)
      .catch(() => undefined); // **والشريطُ زينةُ تنقّل** — وغيابُه لا يحجب الأصناف.
  }, [scope]);

  const load = useCallback(() => {
    if (!id) return;
    setError("");
    setItems(null);
    api<{ items: Item[] }>(`/api/v1/public/sections/${id}/items${scope}`)
      .then((r) => setItems(r.items ?? []))
      .catch((err) => setError(errorText(err)));
  }, [id, scope]);

  useEffect(load, [load]);

  const section = home?.sections.find((s) => s.id === id);
  const rail: RailItem[] = (home?.sections ?? []).map((s) => ({
    id: s.id,
    name: s.name,
    imageUrl: mediaUrl(s.image_thumb_url ?? s.image_url),
    icon: s.icon,
    count: s.count,
  }));

  return (
    <div className="flex flex-col gap-4">
      <Link href="/shop" className="flex w-fit items-center gap-1 text-sm text-accent-text">
        <IconPrev size={16} />
        {SC.back}
      </Link>
      {rail.length > 0 && (
        <SectionRail items={rail} activeID={id} auto={false} onSelect={(sid) => router.push(`/s/${sid}`)} />
      )}
      {section && <h1 className="heading-page">{section.name}</h1>}
      {error ? (
        <ReloadState label={error} onRetry={load} />
      ) : items === null ? (
        <LoadingState variant="list" rows={4} />
      ) : items.length === 0 ? (
        <EmptyState icon={IconStore} title={SC.empty} action={<span className="text-xs text-ink-muted">{SC.emptyHint}</span>} />
      ) : (
        <ItemGrid items={items} onOpen={setOpen} />
      )}
      <ItemSheet item={open} onClose={() => setOpen(null)} />
    </div>
  );
}
