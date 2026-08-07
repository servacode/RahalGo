"use client";

/**
 * **رافعُ ملفٍّ — لا صورة.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «حقلٌ لإدخال الرابط من غوغل بلاي أو رفع التطبيق
 *  بشكلٍ مباشر من خيار رفع أيضاً».)
 *
 * `ImageUpload` تُظهر معاينةً ومصغَّرةً وتفحص أنّها صورة — **وملفُّ تطبيقٍ
 * لا يُعاين.** فالمعروضُ هنا حالةٌ لا صورة: مرفوعٌ أو لا، وزرٌّ يرفع وزرٌّ
 * يحذف.
 *
 * **والرفعُ يقول ما جرى**: ملفٌّ بمئة ميغابايت يأخذ وقتاً، **وزرٌّ ساكنٌ في
 * أثنائه يُقرأ معطّلاً فيُضغط ثانيةً.**
 */

import { useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "./components";
import { IconApp, IconClose, IconCheck } from "./icons";

const m = getMessages(defaultLocale);

export function FileUpload({
  label,
  hint,
  accept,
  present,
  path,
  api,
  errorText,
  onChange,
}: {
  label: string;
  hint?: string;
  /** امتدادٌ مقبولٌ واحد — يُمرَّر لحقل الملفّ ويُفحص في الخادم أيضاً. */
  accept: string;
  /** أمرفوعٌ ملفٌّ الآن؟ */
  present: boolean;
  /** مسارُ الرفع والحذف في الخادم. */
  path: string;
  api: <T>(p: string, init?: RequestInit) => Promise<T>;
  errorText: (e: unknown) => string;
  onChange: (present: boolean) => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function upload(file: File) {
    setBusy(true);
    setError("");
    try {
      const body = new FormData();
      body.append("file", file);
      await api(path, { method: "POST", body });
      onChange(true);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
      if (ref.current) ref.current.value = "";
    }
  }

  async function remove() {
    setBusy(true);
    setError("");
    try {
      await api(path, { method: "DELETE" });
      onChange(false);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <p className="mb-1.5 text-sm font-medium text-ink">{label}</p>
      {hint && <p className="mb-2 text-xs leading-relaxed text-ink-muted">{hint}</p>}

      <div className="flex flex-wrap items-center gap-2">
        <input
          ref={ref}
          type="file"
          accept={accept}
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void upload(f);
          }}
        />
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => ref.current?.click()}
          className="flex items-center gap-2"
        >
          <IconApp size={16} />
          {busy ? m.common.loading : present ? m.common.media.replaceFile : m.common.media.uploadFile}
        </Button>

        {present && !busy && (
          <>
            <span className="flex items-center gap-1.5 text-sm text-success">
              <IconCheck size={15} strokeWidth={3} />
              {m.common.media.fileReady}
            </span>
            <Button
              variant="ghost"
              onClick={() => void remove()}
              className="flex items-center gap-1.5 text-danger"
            >
              <IconClose size={15} />
              {m.common.media.removeFile}
            </Button>
          </>
        )}
      </div>

      {error && <p className="mt-2 text-xs text-danger">{error}</p>}
    </div>
  );
}
