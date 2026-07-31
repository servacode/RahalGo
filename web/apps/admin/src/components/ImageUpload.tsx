"use client";

/**
 * مكوّن رفع الصور الموحد — يُستخدم في كل نماذج المنصة
 * (شعار المتجر، صورة الصنف، صورة البانر).
 * يرفع فوراً عند الاختيار ويعيد معرف الوسائط للنموذج الأب.
 */

import { useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, IconAdd, IconDelete, IconLoading } from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

interface MediaResult {
  id: string;
  url: string;
  thumb_url: string;
  width: number;
  height: number;
}

export default function ImageUpload({
  kind,
  label,
  initialUrl,
  onChange,
}: {
  kind: "merchant_logo" | "menu_item" | "banner" | "avatar";
  label: string;
  /** الصورة الحالية للكيان (عند التعديل) */
  initialUrl?: string | null;
  /** يُستدعى بمعرف الوسائط الجديد، أو "" عند الإزالة */
  onChange: (mediaID: string) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<string | null>(mediaUrl(initialUrl));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function upload(file: File) {
    setBusy(true);
    setError("");
    try {
      const form = new FormData();
      form.append("kind", kind);
      form.append("file", file);
      const res = await api<MediaResult>("/api/v1/admin/media", { method: "POST", body: form });
      setPreview(mediaUrl(res.thumb_url));
      onChange(res.id);
    } catch (err) {
      if (err instanceof ApiError && err.body.message_key === "errors.image_too_large") {
        setError(m.errors.image_too_large);
      } else if (err instanceof ApiError && err.body.message_key === "errors.invalid_image") {
        setError(m.errors.invalid_image);
      } else {
        setError(m.errors.internal);
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={busy}
          className="relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-control border border-dashed border-line bg-page text-ink-muted transition-colors hover:border-primary hover:text-primary"
          aria-label={label}
        >
          {busy ? (
            <IconLoading size={22} className="animate-spin" />
          ) : preview ? (
            // صور الوسائط ديناميكية من خادمنا — لا تمر بمحسّن Next
            // eslint-disable-next-line @next/next/no-img-element
            <img src={preview} alt="" className="h-full w-full object-cover" />
          ) : (
            <IconAdd size={24} />
          )}
        </button>
        <div className="space-y-1.5">
          <p className="text-xs text-ink-muted">{m.common.media.hint}</p>
          {preview && !busy && (
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                setPreview(null);
                onChange("");
                if (inputRef.current) inputRef.current.value = "";
              }}
              className="flex items-center gap-1 !px-2 !py-1 text-xs text-danger"
            >
              <IconDelete size={13} />
              {m.common.media.remove}
            </Button>
          )}
        </div>
      </div>
      {error && <p className="mt-1.5 text-xs text-danger">{error}</p>}
      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) void upload(f);
        }}
      />
    </div>
  );
}

// عنصر عرض مصغّر موحد للجداول والبطاقات (شعار/صورة أو حرف بديل)
export function MediaThumb({
  url,
  alt,
  fallback,
  size = 40,
}: {
  url?: string | null;
  alt: string;
  fallback: string;
  size?: number;
}) {
  const src = mediaUrl(url);
  if (!src) {
    return (
      <span
        className="flex shrink-0 items-center justify-center rounded-control bg-primary-light font-bold text-primary-dark"
        style={{ width: size, height: size, fontSize: size * 0.42 }}
      >
        {fallback.trim().charAt(0) || m.terms.avatarFallback}
      </span>
    );
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={src}
      alt={alt}
      width={size}
      height={size}
      className="shrink-0 rounded-control object-cover"
      style={{ width: size, height: size }}
    />
  );
}
