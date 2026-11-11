"use client";

/**
 * **صورُ الطلب الخاصّ — رافعٌ وعارض** (قرارُ المالك ٢٠٢٦-١١-١١؛ مرآةُ `CustomPhotos.kt` وقسمِ الصور في `CustomScreen.kt`).
 *
 * الزبونُ يرفق صورةَ قائمة مشترياته أو الغرض (١–٣) بدل الكتابة وحدَها. **تُرفع حين تُختار**
 * (`POST /me/custom-photo`) فيُرسَل معرّفاتُها مع الطلب. **والروابطُ موقَّعةٌ من المحرّك** (صنفٌ محميّ)
 * فتُكمَّل بعنوان الخادم ولا تُعدَّل.
 */

import { useRef, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, IconClose } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);
const K = m.shop.custom;

/** **سقفُ الصور للطلب الواحد** — مرآةُ `MaxCustomPhotos` في المحرّك. */
export const MAX_CUSTOM_PHOTOS = 3;

export type UploadedPhoto = { id: string; url: string; thumb_url: string };

/** **يصغّر الصورةَ قبل الرفع** — كصورة الحساب: أطولُ ضلعٍ ١٦٠٠ وجودةٌ ٠٫٨٥. */
async function shrink(file: File): Promise<Blob> {
  const MAX = 1600;
  try {
    const bmp = await createImageBitmap(file);
    const scale = Math.min(1, MAX / Math.max(bmp.width, bmp.height));
    if (scale === 1 && file.size <= 2 * 1024 * 1024) return file;
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

/** **عارضُ الصور** — مصغّراتٌ تفتح الصورةَ كاملةً في تبويبٍ جديد. للزبون والمكتب. */
export function CustomPhotoGrid({ urls, label = true }: { urls?: string[] | null; label?: boolean }) {
  if (!urls || urls.length === 0) return null;
  return (
    <div className="flex flex-col gap-1">
      {label && <p className="text-2xs text-ink-muted">{K.photosLabel}</p>}
      <div className="flex flex-wrap gap-2">
        {urls.map((u) => {
          const full = mediaUrl(u) ?? u;
          return (
            <a key={u} href={full} target="_blank" rel="noreferrer" className="block h-16 w-16 overflow-hidden rounded-control border border-line">
              <img src={full} alt="" loading="lazy" className="h-full w-full object-cover" />
            </a>
          );
        })}
      </div>
    </div>
  );
}

/** **رافعُ الصور في نموذج الطلب** — حتّى ثلاث، بزرّ إزالةٍ على كلّ مصغّرة. */
export function CustomPhotoPicker({
  photos,
  onChange,
  disabled = false,
}: {
  photos: UploadedPhoto[];
  onChange: (next: UploadedPhoto[]) => void;
  disabled?: boolean;
}) {
  const ref = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (ref.current) ref.current.value = "";
    if (!file || photos.length >= MAX_CUSTOM_PHOTOS) return;
    setBusy(true);
    setError("");
    try {
      const body = new FormData();
      body.append("file", await shrink(file), "custom.jpg");
      const up = await api<UploadedPhoto>("/api/v1/me/custom-photo", { method: "POST", body });
      onChange([...photos, up]);
    } catch (err) {
      setError(errorText(err) || K.photosFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div>
        <p className="text-sm font-bold">{K.photosTitle}</p>
        <p className="text-xs text-ink-muted">{K.photosHint}</p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {photos.map((p) => (
          <div key={p.id} className="relative h-16 w-16 overflow-hidden rounded-control border border-line">
            <img src={mediaUrl(p.thumb_url) ?? p.thumb_url} alt="" className="h-full w-full object-cover" />
            <button
              type="button"
              aria-label={K.photosRemove}
              title={K.photosRemove}
              disabled={disabled}
              onClick={() => onChange(photos.filter((x) => x.id !== p.id))}
              className="absolute end-0 top-0 flex h-6 w-6 items-center justify-center rounded-full border border-line bg-surface text-ink"
            >
              <IconClose size={14} />
            </button>
          </div>
        ))}
        {photos.length < MAX_CUSTOM_PHOTOS && (
          <>
            <input ref={ref} type="file" accept="image/*" className="hidden" onChange={onPick} />
            <Button type="button" variant="secondary" disabled={disabled || busy} onClick={() => ref.current?.click()}>
              {K.photosAdd}
            </Button>
          </>
        )}
      </div>
      {error && <p className="text-xs text-danger">{error}</p>}
    </div>
  );
}
