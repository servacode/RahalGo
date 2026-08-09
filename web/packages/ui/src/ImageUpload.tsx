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
import { IconAdd, IconDelete, IconLoading } from "./icons";

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
  | "site_background";

export function ImageUpload({
  kind,
  label,
  initialUrl,
  onChange,
  api,
  mediaUrl,
  path = "/api/v1/admin/media",
  errorText,
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
      const res = await api<MediaResult>(path, { method: "POST", body: form });
      setPreview(mediaUrl(res.thumb_url));
      onChange(res.id);
    } catch (err) {
      setError(errorText?.(err) ?? m.errors.internal);
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
          className="relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-control border border-dashed border-line bg-field text-ink-muted transition-colors hover:border-primary hover:text-primary"
          aria-label={label}
        >
          {busy ? (
            <IconLoading size={22} className="animate-spin" />
          ) : preview ? (
            // صور الوسائط ديناميكية من خادمنا — لا تمر بمحسّن Next
            // eslint-disable-next-line @next/next/no-img-element
            <img src={preview} alt="" loading="lazy" className="h-full w-full object-cover" />
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
          if (f) void upload(f);
        }}
      />
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
}: {
  url?: string | null;
  alt: string;
  fallback: string;
  size?: number;
  mediaUrl: (url?: string | null) => string | null;
}) {
  const src = mediaUrl(url);
  if (!src) {
    return (
      <span
        className="flex shrink-0 items-center justify-center rounded-control bg-primary-tint font-bold text-primary"
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
