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

export async function GET(req: Request, ctx: { params: Promise<{ app: string }> }) {
  const { app } = await ctx.params;
  if (!(APP_KEYS as readonly string[]).includes(app)) {
    return NextResponse.redirect(new URL("/download", req.url), 302);
  }
  const href = downloadHref(await readRelease(app as AppKey));
  if (!href) {
    return NextResponse.redirect(new URL(`/download/${app}`, req.url), 302);
  }
  return NextResponse.redirect(href, 302);
}
