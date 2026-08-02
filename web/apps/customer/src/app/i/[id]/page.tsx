/** صفحةُ الصنف — تُقدَّم من الخادم كي يكون رابطُها قابلاً للمشاركة والفهرسة. */

import { notFound } from "next/navigation";
import ItemClient, { type Group } from "./ItemClient";
import type { BrowseItem } from "@/components/ItemCard";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

export default async function ItemPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  try {
    const res = await fetch(`${API}/api/v1/public/items/${id}`, { cache: "no-store" });
    const json = (await res.json()) as {
      data?: { item?: BrowseItem; modifiers?: Group[] };
    };
    if (!json.data?.item) notFound();
    return <ItemClient item={json.data.item} modifiers={json.data.modifiers ?? []} />;
  } catch {
    notFound();
  }
}
