import type { ReactNode } from "react";

/**
 * **رأسُ التبويب داخل «المشاكل والتعويضات»** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «التصميم الاحترافي») — سطرُ شرحٍ
 * صغيرٌ وأزرارُه، **بدل عنوانٍ كبيرٍ ثانٍ تحت عنوان الصفحة** يكرّر اسمَ التبويب المختار.
 */
export function SectionBar({ hint, actions }: { hint?: ReactNode; actions?: ReactNode }) {
  if (!hint && !actions) return null;
  return (
    <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
      {hint ? <p className="min-w-0 flex-1 text-sm text-ink-muted">{hint}</p> : <span />}
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </div>
  );
}
