/**
 * **التنزيلُ المباشرُ من الباركود** (قرارُ المالك ٢٠٢٦-١٠-٠٦: «لازم ينزل من الباركود بدون
 * ما يضغط تحميل»).
 *
 * `rahalgo.com/get/driver` **يحوّل فوراً إلى ملفّ التطبيق** — فيبدأ التنزيلُ لحظةَ المسح،
 * **والرابطُ على نطاقنا لا على المحرّك.** وتطبيقٌ بلا ملفٍّ يُعاد إلى صفحته.
 */
import { NextResponse } from "next/server";
import { APP_KEYS, type AppKey, readRelease, downloadHref } from "@/lib/releases";

export const dynamic = "force-dynamic";

// **وتحويلٌ نسبيٌّ داخل الموقع** — `req.url` خلف الوكيل عنوانٌ داخليّ (`localhost:3000`).
function local(path: string): Response {
  return new Response(null, { status: 302, headers: { Location: path } });
}

export async function GET(_req: Request, ctx: { params: Promise<{ app: string }> }) {
  const { app } = await ctx.params;
  if (!(APP_KEYS as readonly string[]).includes(app)) {
    return local("/download");
  }
  const href = downloadHref(await readRelease(app as AppKey));
  if (!href) {
    return local(`/download/${app}`);
  }
  return NextResponse.redirect(href, 302);
}
