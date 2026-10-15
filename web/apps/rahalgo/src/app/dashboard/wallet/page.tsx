import { redirect } from "next/navigation";

/**
 * **«محفظتي» أُزيلت** (قرارُ المالك ٢٠٢٦-١٠-١٥) — محفظةُ الأدمن هي الخزينة، وكشفُها في
 * قسمها برصيدٍ جارٍ وتصدير. **والعنوانُ القديمُ لا يُترك صفحةً ميّتة**: من حفظه يصل الكشف.
 */
export default function Page() {
  redirect("/dashboard/treasury?tab=statement");
}
