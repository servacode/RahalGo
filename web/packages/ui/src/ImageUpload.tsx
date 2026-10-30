"use client";

/**
 * **رفعُ الصور — مكوّنٌ واحدٌ للمنصة كلِّها.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ منتجٍ… يكون له صورة… يجب أن تكون الطريقةُ
 *  مركزيّة».)
 *
 * # لماذا انتقل من لوحة الإدارة
 *
 * **كان في `apps/admin/src/components`** — فبوّابةُ المتجر لا تراه أصلاً.
 * **وصورةُ الصنف مبنيّةٌ في المحرّك وفي `MenuManager` منذ زمن**، ولا يد
 * ترفعها: صاحبُ المطعم يملك صنفاً بلا وجه.
 *
 * **ونسخُه إلى المتجر نسختان تفترقان يوماً** — تُضبط رسالةُ خطأٍ هنا وتبقى
 * هناك. فانتقل إلى المركز.
 *
 * # والنقطةُ تُمرَّر ولا تُفترض
 *
 * **الإدارةُ ترفع على `‎/admin/media` والمتجرُ على `‎/merchant/media`** —
 * وحارسُ الأنواع في المحرّك يختلف بينهما عمداً: **التاجرُ لا يرفع شعارَ
 * المنصة.** فالمسارُ مُعامِلٌ لا ثابت.
 *
 * **وكذلك `api` و`mediaUrl`** — لكلّ تطبيقٍ نسختُه بعنوانه ورموزه.
 */

import { useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "./components";
import { ImageCropDialog, LOGO_CROP_FRAMES, type CropFrame } from "./ImageCrop";
import { IconAdd, IconCamera, IconClose, IconDelete, IconLoading } from "./icons";

const m = getMessages(defaultLocale);

interface MediaResult {
  id: string;
  url: string;
  thumb_url: string;
  width: number;
  height: number;
}

/** **يطابق `validKinds` في المحرّك** — ونوعٌ ليس فيه يُرفض عند الرفع. */
export type MediaKind =
  | "merchant_logo"
  | "menu_item"
  | "menu_section"
  | "banner"
  | "avatar"
  | "platform_logo"
  | "auth_background"
  | "site_background"
  | "expense_receipt";

export function ImageUpload({
  kind,
  label,
  initialUrl,
  onChange,
  api,
  mediaUrl,
  path = "/api/v1/admin/media",
  errorText,
  maxEdge,
  crop = false,
  cropFrames = LOGO_CROP_FRAMES,
  fit = false,
  tile,
  compact,
  disabled = false,
}: {
  kind: MediaKind;
  label: string;
  /** الصورة الحالية للكيان (عند التعديل) */
  initialUrl?: string | null;
  /** يُستدعى بمعرف الوسائط الجديد، أو "" عند الإزالة */
  onChange: (mediaID: string) => void;
  api: <T>(path: string, init?: RequestInit) => Promise<T>;
  /** يحوّل مسارَ الوسائط النسبيَّ إلى رابطٍ كامل — لكلّ تطبيقٍ نسختُه. */
  mediaUrl: (url?: string | null) => string | null;
  /** **نقطةُ الرفع** — تختلف بالبوّابة، وحارسُ الأنواع فيها يختلف معها. */
  path?: string;
  /**
   * **ترجمةُ خطأ الخادم** — يمرّرها التطبيق لأنّ لكلٍّ نسختَه من `ApiError`.
   *
   * **وفارغٌ يعني «خطأٌ عامّ»** — ورسالةٌ عامّةٌ أهونُ من صمت.
   */
  errorText?: (err: unknown) => string;
  /**
   * **تصغيرٌ في المتصفّح قبل الرفع** — أطولُ ضلعٍ بالبكسل، **وفارغُه يرفع كما هي.**
   *
   * (لوحةُ المتجر ٢٠٢٦-١٠-١٧: صورةُ آيفون ١٢ مليون بكسل وأربعة ميغا.)
   * **وتطبيقُ المتجر يصغّرها إلى ١٦٠٠ قبل الرفع** (`readScaledImage`)، **فمن
   * رفع من الويب صنفاً لا يُردّ بـ«أكبر من الحدّ» وصاحبُ أندرويد يُقبل.**
   * **وما فشل تصغيرُه يُرسَل كما هو** — والخادمُ يحكم.
   */
  maxEdge?: number;
  /**
   * **معاينةٌ وقصٌّ مربّعٌ قبل الرفع** — مرآةُ `ImageCropDialog` في التطبيق
   * (طلبُ المالك ٢٠٢٦-١٠-٢٧). **اختياريٌّ** — وبلاه يُرفع الملفُّ كما كان.
   * والناتجُ مربّعٌ JPEG بأطول ضلعٍ ١٦٠٠، فلا يمرّ بـ`maxEdge`.
   */
  crop?: boolean;
  /** **إطاراتُ القصّ** — `ITEM_CROP_FRAMES` لصورة الصنف (كاملة افتراضاً)، والشعارُ مربّع. */
  cropFrames?: CropFrame[];
  /** **المعاينةُ كاملةً لا مقصوصة** — لصورة الصنف. */
  fit?: boolean;
  /**
   * **مربّعٌ كبيرٌ يُضغط** (محرّرُ الصنف، تصميمُ ٢٠٢٦-١٠-٢٩) — مرآةُ `ItemPhoto`
   * في التطبيق: شريطٌ أسفلَه يقول ما يفعله الضغط، **و✕ في زاويته يُزيلها.**
   */
  tile?: { pick: string; change: string; clear: string };
  /**
   * **مربّعٌ صغيرٌ في سطر** (صورةُ المشروب، ٢٠٢٦-١٠-٣٠) — ٤٤ بكسل، يُضغط ليختار صورة، **و✕ صغيرةٌ
   * في زاويته تُزيلها.** والخطأُ يُقال تحته.
   */
  compact?: { pick: string; change: string; clear: string };
  /** **معطّلٌ** — متجرٌ موقوفٌ أو قارئٌ بلا إذن تعديل (للمربّع الصغير). */
  disabled?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<string | null>(mediaUrl(initialUrl));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [cropping, setCropping] = useState<File | null>(null);

  function resetInput() {
    if (inputRef.current) inputRef.current.value = "";
  }

  async function shrink(file: File): Promise<Blob> {
    if (!maxEdge) return file;
    try {
      const bmp = await createImageBitmap(file);
      const scale = Math.min(1, maxEdge / Math.max(bmp.width, bmp.height));
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

  async function upload(file: File | Blob) {
    setBusy(true);
    setError("");
    try {
      const form = new FormData();
      form.append("kind", kind);
      const blob = file instanceof File ? await shrink(file) : file;
      form.append("file", blob, blob instanceof File ? blob.name : "image.jpg");
      const res = await api<MediaResult>(path, { method: "POST", body: form });
      setPreview(mediaUrl(res.thumb_url));
      onChange(res.id);
    } catch (err) {
      setError(errorText?.(err) ?? m.errors.internal);
    } finally {
      setBusy(false);
    }
  }

  const picker = (
    <>
      {error && <p className="mt-1.5 text-xs text-danger">{error}</p>}
      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (!f) return;
          if (crop) setCropping(f);
          else void upload(f);
        }}
      />
      {cropping && (
        <ImageCropDialog
          file={cropping}
          frames={cropFrames}
          onDone={(blob) => {
            setCropping(null);
            resetInput();
            void upload(blob);
          }}
          onAgain={() => {
            setCropping(null);
            resetInput();
            inputRef.current?.click();
          }}
          onCancel={() => {
            setCropping(null);
            resetInput();
          }}
        />
      )}
    </>
  );

  if (compact) {
    return (
      <div className="relative h-11 w-11 shrink-0">
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={busy || disabled}
          aria-label={preview ? compact.change : compact.pick}
          title={preview ? compact.change : compact.pick}
          data-testid="image-compact"
          className="flex h-full w-full items-center justify-center overflow-hidden rounded-control border border-dashed border-line bg-field text-ink-muted transition-colors hover:border-primary hover:text-primary disabled:opacity-50"
        >
          {busy ? (
            <IconLoading size={18} className="animate-spin" />
          ) : preview ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={preview} alt="" loading="lazy" className="h-full w-full object-cover" />
          ) : (
            <IconCamera size={18} />
          )}
        </button>
        {preview && !busy && !disabled && (
          <button
            type="button"
            aria-label={compact.clear}
            title={compact.clear}
            onClick={() => {
              setPreview(null);
              onChange("");
              resetInput();
            }}
            className="absolute -end-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-scrim text-on-solid"
          >
            <IconClose size={12} />
          </button>
        )}
        {picker}
      </div>
    );
  }

  if (tile) {
    return (
      <div className="w-32 shrink-0 sm:w-36">
        <div className="relative aspect-square w-full overflow-hidden rounded-card border border-line bg-field">
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            disabled={busy}
            aria-label={preview ? tile.change : tile.pick}
            data-testid="image-tile"
            className="absolute inset-0 flex items-center justify-center text-ink-muted"
          >
            {busy ? (
              <IconLoading size={24} className="animate-spin" />
            ) : preview ? (
              <img src={preview} alt="" loading="lazy" className={`h-full w-full ${fit ? "object-contain" : "object-cover"}`} />
            ) : (
              <IconCamera size={28} />
            )}
            <span className="absolute inset-x-0 bottom-0 flex items-center justify-center gap-1 bg-scrim py-1 text-xs font-medium text-on-solid">
              <IconCamera size={14} />
              {preview ? tile.change : tile.pick}
            </span>
          </button>
          {preview && !busy && (
            <button
              type="button"
              aria-label={tile.clear}
              title={tile.clear}
              onClick={() => {
                setPreview(null);
                onChange("");
                resetInput();
              }}
              className="absolute end-1 top-1 flex h-8 w-8 items-center justify-center rounded-full bg-scrim text-on-solid"
            >
              <IconClose size={16} />
            </button>
          )}
        </div>
        {picker}
      </div>
    );
  }

  return (
    <div>
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          disabled={busy}
          className="relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-control border border-dashed border-line bg-field text-ink-muted transition-colors hover:border-primary hover:text-primary"
          aria-label={label}
        >
          {busy ? (
            <IconLoading size={22} className="animate-spin" />
          ) : preview ? (
            // صور الوسائط ديناميكية من خادمنا — لا تمر بمحسّن Next
            // eslint-disable-next-line @next/next/no-img-element
            <img src={preview} alt="" loading="lazy" className={`h-full w-full ${fit ? "object-contain" : "object-cover"}`} />
          ) : (
            <IconAdd size={24} />
          )}
        </button>
        {/* **ولا سطرَ يعدّد الصيغَ والحجم.**

            (شكوى المالك ٢٠٢٦-٠٨-٠٩: «هي النصوص كلها ما تلزم».)

            **منتقي الملفّات يفلتر بنفسه** (`accept`)، **والخطأُ يُقال حين
            يقع** لا قبله: «الملف ليس صورة» · «أكبر من الحدّ». **وسطرٌ يشرح
            ما لن يحدث يزاحم ما يحدث.** */}
        <div className="space-y-1.5">
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
          if (!f) return;
          if (crop) setCropping(f);
          else void upload(f);
        }}
      />
      {cropping && (
        <ImageCropDialog
          file={cropping}
          frames={cropFrames}
          onDone={(blob) => {
            setCropping(null);
            resetInput();
            void upload(blob);
          }}
          onAgain={() => {
            setCropping(null);
            resetInput();
            inputRef.current?.click();
          }}
          onCancel={() => {
            setCropping(null);
            resetInput();
          }}
        />
      )}
    </div>
  );
}

/**
 * **مصغَّرٌ موحَّدٌ للجداول والبطاقات** — صورةٌ أو حرفٌ بديل.
 *
 * **والحرفُ ليس زينة**: صنفٌ بلا صورةٍ يترك فراغاً في الشبكة، **وفراغٌ في
 * صفٍّ من الصور يُقرأ عطباً في التحميل** لا «لم تُرفع بعد».
 */
export function MediaThumb({
  url,
  alt,
  fallback,
  size = 40,
  mediaUrl,
  round = false,
}: {
  url?: string | null;
  alt: string;
  fallback: string;
  size?: number;
  /** **دائرةٌ لا مربّع** — صورةُ حسابٍ في بطاقته (قرارُ المالك ٢٠٢٦-١٠-٢٥). */
  round?: boolean;
  mediaUrl: (url?: string | null) => string | null;
}) {
  const src = mediaUrl(url);
  if (!src) {
    return (
      <span
        className={`flex shrink-0 items-center justify-center bg-primary-tint font-bold text-primary ${round ? "rounded-full" : "rounded-control"}`}
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
      className={`shrink-0 object-cover ${round ? "rounded-full" : "rounded-control"}`}
      style={{ width: size, height: size }}
    />
  );
}
