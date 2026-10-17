"use client";

/**
 * **المفضلة — أصنافٌ لا متاجر** (مرآةُ «المفضلة» في قائمة أندرويد).
 *
 * **ومصدرُها واحدٌ مع القلب** (`FavoritesProvider`) — من أزال قلباً هنا
 * اختفى الصنفُ من البطاقة في السوق أيضاً.
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ButtonLink, EmptyState, LoadingState, IconHeart } from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useFav } from "@/lib/customer/favorites";
import type { Item } from "@/lib/customer/types";
import { ItemGrid, ItemSheet } from "@/components/customer/items";
import { NeedLogin } from "@/components/customer/NeedLogin";

const m = getMessages(defaultLocale);
const F = m.shop.favorites;

export default function FavoritesPage() {
  const { user, loading } = useAuth();
  const fav = useFav();
  const [open, setOpen] = useState<Item | null>(null);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  return (
    <div className="flex flex-col gap-4">
      <h1 className="heading-page">{F.title}</h1>
      {fav.rows === null ? (
        <LoadingState variant="list" rows={3} />
      ) : fav.rows.length === 0 ? (
        <EmptyState
          icon={IconHeart}
          title={F.empty}
          action={
            <div className="flex flex-col items-center gap-3">
              <span className="text-xs text-ink-muted">{F.emptyHint}</span>
              <ButtonLink href="/shop" variant="secondary">
                {m.shop.cart.browse}
              </ButtonLink>
            </div>
          }
        />
      ) : (
        <ItemGrid items={fav.rows} onOpen={setOpen} />
      )}
      <ItemSheet item={open} onClose={() => setOpen(null)} />
    </div>
  );
}
