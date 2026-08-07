"use client";

/**
 * **غلافٌ رقيقٌ حول المكوّن المركزيّ** — يحقنه بنداءِ لوحة الإدارة ونقطتها.
 *
 * **والمكوّنُ نفسُه في `@rahalgo/ui`** (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «يجب أن تكون
 * الطريقةُ مركزيّة») — **وبقي هذا الملفّ لئلّا تُبدَّل عشرون مناداةً في
 * اللوحة دفعةً واحدة.**
 */

import { ImageUpload as Central, MediaThumb as CentralThumb, type MediaKind } from "@rahalgo/ui";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { api, mediaUrl, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

/** **وترجمةُ خطأ الخادم تبقى هنا** — `ApiError` نسخةُ كلّ تطبيقٍ من نفسِه. */
function errText(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.body.message_key === "errors.image_too_large") return m.errors.image_too_large;
    if (err.body.message_key === "errors.invalid_image") return m.errors.invalid_image;
  }
  return m.errors.internal;
}

export default function ImageUpload(props: {
  kind: MediaKind;
  label: string;
  initialUrl?: string | null;
  onChange: (mediaID: string) => void;
}) {
  return <Central {...props} api={api} mediaUrl={mediaUrl} errorText={errText} />;
}

export function MediaThumb(props: { url?: string | null; alt: string; fallback: string; size?: number }) {
  return <CentralThumb {...props} mediaUrl={mediaUrl} />;
}
