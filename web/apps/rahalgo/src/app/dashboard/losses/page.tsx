/**
 * **الخسائرُ والنزاعاتُ وتعويضُ البضاعة صارت في «المشاكل والتعويضات»** (قرارُ المالك
 * ٢٠٢٦-١٠-٢٥). **والرابطُ القديمُ يفتح تبويبَ الخسائر هناك** — روابطُ الموافقات والأحداث
 * والإشعارات تبقى تعمل. **والتحويلُ من الخادم** فلا ومضةَ تحميلٍ في المتصفّح.
 */

import { redirect } from "next/navigation";

export default async function LossesRedirect({ searchParams }: { searchParams: Promise<{ tab?: string }> }) {
  const { tab } = await searchParams;
  redirect(`/dashboard/tickets?tab=${tab === "disputes" || tab === "goods" ? tab : "losses"}`);
}
