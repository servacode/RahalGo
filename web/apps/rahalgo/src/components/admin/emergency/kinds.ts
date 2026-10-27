/** **نبرةُ نوع الطارئ** — الحادثُ الأخطر (قراراتُ المالك ٢٠٢٦-١٠-٠٤). */
export function kindVariant(kind: string): "danger" | "warning" | "violet" | "accent" {
  if (kind === "accident") return "danger";
  if (kind === "platform_halt") return "warning";
  if (kind === "store_closure") return "violet";
  return "accent";
}

/** **دائرةُ أيقونة النوع** — تعبئةٌ مصمتةٌ بنبرة النوع نفسِها (تصميمُ اللوحة ٢٠٢٦-١٠-٢٧). */
export function kindTint(kind: string): string {
  switch (kindVariant(kind)) {
    case "danger":
      return "bg-danger-solid text-on-solid";
    case "warning":
      return "bg-warning-solid text-on-solid";
    case "violet":
      return "bg-violet-solid text-on-solid";
    default:
      return "bg-accent text-on-bright";
  }
}
