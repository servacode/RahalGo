/** الرئيسية — تُقدَّم من الخادم (SEO): المحتوى في HTML الأولي، والترشيح تفاعلي. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import HomeClient, { type HomeData } from "./HomeClient";

const m = getMessages(defaultLocale);
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  let data: HomeData = { banners: [], categories: [], sections: [] };
  try {
    const res = await fetch(`${API}/api/v1/public/home`, { cache: "no-store" });
    const json = (await res.json()) as { data?: HomeData };
    if (json.data) data = json.data;
  } catch {
    /* الخادم غير متاح — تُعرض صفحة فارغة بدل الانهيار */
  }
  // **وفراغُ الأقسام لا فراغُ المتاجر** — الزبونُ يتصفّح أقساماً.
  if (data.sections.length === 0) {
    return <p className="py-16 text-center text-ink-muted">{m.errors.offline}</p>;
  }
  return <HomeClient initial={data} />;
}
