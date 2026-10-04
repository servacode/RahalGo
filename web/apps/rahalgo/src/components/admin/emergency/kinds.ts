/** **نبرةُ نوع الطارئ** — الحادثُ الأخطر (قراراتُ المالك ٢٠٢٦-١٠-٠٤). */
export function kindVariant(kind: string): "danger" | "warning" | "violet" | "accent" {
  if (kind === "accident") return "danger";
  if (kind === "platform_halt") return "warning";
  if (kind === "store_closure") return "violet";
  return "accent";
}
