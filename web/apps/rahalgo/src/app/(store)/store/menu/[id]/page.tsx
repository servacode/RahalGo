"use client";

/** **‏/store/menu/[id] — تعديلُ صنفٍ أو حذفُه** (`editItem` في `MenuViewModel.kt`). */

import { useParams } from "next/navigation";
import ItemEditorPage from "@/components/store/ItemEditorPage";

export default function EditItemPage() {
  const params = useParams<{ id: string }>();
  return <ItemEditorPage itemId={params?.id ?? ""} />;
}
