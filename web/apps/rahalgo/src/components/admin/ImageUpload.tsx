"use client";

/**
 * **غلافٌ رقيقٌ حول المكوّن المركزيّ** — يحقنه بنداءِ لوحة الإدارة ونقطتها.
 *
 * **والمكوّنُ نفسُه في `@rahalgo/ui`** (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «يجب أن تكون
 * الطريقةُ مركزيّة») — **وبقي هذا الملفّ لئلّا تُبدَّل عشرون مناداةً في
 * اللوحة دفعةً واحدة.**
 */

import { useRef, useState } from "react";
import { ImageUpload as Central, MediaThumb as CentralThumb, type MediaKind } from "@rahalgo/ui";
import { getMessages, defaultLocale, errorText} from "@rahalgo/i18n";
import { api, mediaUrl, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);


export default function ImageUpload(props: {
  kind: MediaKind;
  label: string;
  initialUrl?: string | null;
  onChange: (mediaID: string) => void;
  /** **نقطةُ الرفع** — صفحةُ السوق ترفع من بابها (`/admin/market/media`، قرارُ
   *  المالك ٢٠٢٦-١٠-٠٥) **فلا يلزم موظّفَها `content.manage`.** */
  path?: string;
}) {
  return <Central {...props} api={api} mediaUrl={mediaUrl} errorText={errorText} />;
}

export function MediaThumb(props: { url?: string | null; alt: string; fallback: string; size?: number; round?: boolean }) {
  return <CentralThumb {...props} mediaUrl={mediaUrl} />;
}

/**
 * **الضغطُ على الصورة يفتح ملفّات الجهاز فوراً** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «وقت تضغط ع صورة فوراً يفتح
 * اللابتوب لنضيف صورة، ما في داعي للإزالة ونرجع نرفع») — بلا نافذةٍ وسيطة. تُصغَّر الصورةُ إلى ١٦٠٠
 * بكسل كتطبيق المتجر، تُرفع، ثمّ يُنادى `onUploaded` بمعرّفها.
 */
export function useImagePicker({
  kind,
  path = "/api/v1/admin/media",
  onUploaded,
  onError,
}: {
  kind: MediaKind;
  path?: string;
  onUploaded: (mediaID: string) => void | Promise<void>;
  onError?: (msg: string) => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);

  async function shrink(file: File): Promise<Blob> {
    try {
      const bmp = await createImageBitmap(file);
      const scale = Math.min(1, 1600 / Math.max(bmp.width, bmp.height));
      if (scale === 1) return file;
      const canvas = document.createElement("canvas");
      canvas.width = Math.round(bmp.width * scale);
      canvas.height = Math.round(bmp.height * scale);
      const ctx = canvas.getContext("2d");
      if (!ctx) return file;
      ctx.drawImage(bmp, 0, 0, canvas.width, canvas.height);
      const blob = await new Promise<Blob | null>((res) => canvas.toBlob(res, "image/jpeg", 0.85));
      return blob ?? file;
    } catch {
      return file;
    }
  }

  async function pick(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setBusy(true);
    try {
      const form = new FormData();
      form.append("kind", kind);
      const blob = await shrink(file);
      form.append("file", blob, blob === file ? file.name : "image.jpg");
      const res = await api<{ id: string }>(path, { method: "POST", body: form });
      await onUploaded(res.id);
    } catch (err) {
      onError?.(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  const input = <input ref={ref} type="file" accept="image/*" hidden onChange={(e) => void pick(e)} />;
  return { open: () => ref.current?.click(), busy, input };
}
