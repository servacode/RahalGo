"use client";

/**
 * **غلافٌ رقيقٌ حول المكوّن المركزيّ** — يحقنه بنداءِ لوحة الإدارة ونقطتها.
 *
 * **والمكوّنُ نفسُه في `@rahalgo/ui`** (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «يجب أن تكون
 * الطريقةُ مركزيّة») — **وبقي هذا الملفّ لئلّا تُبدَّل عشرون مناداةً في
 * اللوحة دفعةً واحدة.**
 */

import { ImageUpload as Central, MediaThumb as CentralThumb, type MediaKind } from "@rahalgo/ui";
import { getMessages, defaultLocale, errorText} from "@rahalgo/i18n";
import { api, mediaUrl, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);


export default function ImageUpload(props: {
  kind: MediaKind;
  label: string;
  initialUrl?: string | null;
  onChange: (mediaID: string) => void;
}) {
  return <Central {...props} api={api} mediaUrl={mediaUrl} errorText={errorText} />;
}

export function MediaThumb(props: { url?: string | null; alt: string; fallback: string; size?: number }) {
  return <CentralThumb {...props} mediaUrl={mediaUrl} />;
}
