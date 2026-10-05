/**
 * ══════════════════════════════════════════════════════════════════════
 * **لونٌ ورمزٌ لكلّ نوعٍ وحال — موضعٌ واحد** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يقرؤه أربعة**: رموزُ الخريطة (`canvas.tsx`) وحبّاتُ الشريط ودليلُ الألوان
 * وبطاقةُ المُحدَّد. **ومن بدّل لوناً بدّله في `theme.css`** فتتبعه الأربعةُ
 * في التحديث نفسِه — ولا لونَ مكتوبٌ بيدٍ هنا.
 *
 * **والحالُ من الخادم لا من هنا**: السائقُ يأتي بـ`tone` (`opsmap.DriverTone`)،
 * والطلبُ بـ`stuck_reason` (شرطُ لوحة الطلبات)، والمتجرُ بـ`open_now`.
 * **فهذا الملفُّ يلوّن ولا يحكم.**
 */

import type { ComponentType } from "react";
import { IconMoto, IconOrder, IconStore, IconUsers, IconUser, themeColor } from "@rahalgo/ui";

/** **رمزُ نوعٍ وحالٍ على الخريطة** — اسمُه هو اسمُ صورته في MapLibre. */
export type MapTone =
  | "driver-available"
  | "driver-busy"
  | "driver-idle"
  | "order"
  | "order-stuck"
  | "store-open"
  | "store-closed"
  | "customer"
  | "rep";

type Glyph = ComponentType<{ size?: number; color?: string; strokeWidth?: number }>;

/** **التوكنُ والرمز لكلّ حال.** */
export const MAP_TONES: Record<MapTone, { token: string; glyph: Glyph }> = {
  "driver-available": { token: "map-driver-free", glyph: IconMoto },
  "driver-busy": { token: "map-driver-busy", glyph: IconMoto },
  "driver-idle": { token: "map-idle", glyph: IconMoto },
  order: { token: "map-order", glyph: IconOrder },
  "order-stuck": { token: "map-order-stuck", glyph: IconOrder },
  "store-open": { token: "map-store", glyph: IconStore },
  "store-closed": { token: "map-idle", glyph: IconStore },
  customer: { token: "map-customer", glyph: IconUsers },
  rep: { token: "map-rep", glyph: IconUser },
};

/** **لونُ حالٍ** — من الثيم في لحظة القراءة. */
export function toneColor(tone: MapTone): string {
  return themeColor(MAP_TONES[tone].token);
}

/** **حالُ السائق ← رمزُه** — والمجهولُ خامل. */
export function driverTone(tone: string | undefined): MapTone {
  if (tone === "available") return "driver-available";
  if (tone === "busy") return "driver-busy";
  return "driver-idle";
}

/** **حالُ الطلب ← رمزُه** — والعالقُ بسببه من الخادم. */
export function orderTone(stuckReason: string | undefined | null): MapTone {
  return stuckReason ? "order-stuck" : "order";
}

/** **حالُ المتجر ← رمزُه.** */
export function storeTone(openNow: boolean): MapTone {
  return openNow ? "store-open" : "store-closed";
}

/** **الصورُ التي يسجّلها اللوحُ** — كلُّ حالٍ بلونه ورمزه. */
export function mapIcons(): Array<{ name: MapTone; color: string; glyph: Glyph }> {
  return (Object.keys(MAP_TONES) as MapTone[]).map((name) => ({
    name,
    color: toneColor(name),
    glyph: MAP_TONES[name].glyph,
  }));
}
