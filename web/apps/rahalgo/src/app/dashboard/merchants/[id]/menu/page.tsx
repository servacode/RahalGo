import { redirect } from "next/navigation";

/**
 * **قائمةُ المتجر صارت تبويباً في ملفّه** — هذا المسارُ يوجّه إليه فقط، فلا قائمتان تُصلَحان معاً
 * (وكان يجلب المتاجرَ بـ`per_page=100` ليقرأ الاسمَ فيضيع بعد المتجر المئة).
 */
export default async function MerchantMenuRedirect({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  redirect(`/dashboard/merchants/${encodeURIComponent(id)}?tab=menu`);
}
