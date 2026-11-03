"use client";

import type { ComponentType } from "react";
import { Tile, type TileTone } from "@/components/admin/DashParts";

/**
 * **مربّعُ رقمٍ — يُضغط فقط لمن له فعل** (فحصُ ٢٠٢٦-١٠-٢٧).
 *
 * **هو `Tile` نفسُه** (والآخرُ `Stat` من العُدّة) — اسمٌ باقٍ لأن خسائرَ وتعويضَ البضاعة تستعمله.
 * و`highlight` إطارٌ متقطّعٌ لمربّعٍ مختلفِ المدّة («الآن» بين أرقام الفترة).
 */
export function StatTile({
  highlight = false,
  ...p
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: string;
  sub?: string;
  tone?: TileTone;
  onClick?: () => void;
  highlight?: boolean;
}) {
  return highlight ? (
    <div className="rounded-card border border-dashed border-line">
      <Tile {...p} />
    </div>
  ) : (
    <Tile {...p} />
  );
}
