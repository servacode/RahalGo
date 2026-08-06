"use client";

/**
 * **التغذيةُ الراجعة** — ما تقوله الشاشةُ للمستخدم عمّا وقع.
 *
 * # ما وجده الفحص
 *
 * **رسالةُ الخطأ الواحدةُ مكتوبةٌ بثمانِ صياغاتٍ في تسعةٍ وثمانين موضعاً:**
 *
 *	20 × "text-sm text-danger"
 *	20 × "rounded-control bg-danger-tint px-3 py-2 text-sm text-danger"
 *	13 × "text-danger"          13 × "mb-4 rounded-control bg-danger-tint …"
 *	 8 × "text-xs text-danger"   6 × "py-10 text-center text-danger"
 *
 * **والمستخدمُ يرى شكلاً في شاشةٍ وشكلاً في التالية** — فيقرأ الثانيةَ من
 * جديدٍ ليعرف أهي خطأٌ أم ملاحظة.
 *
 * **والنجاحُ أسوأ**: لا مكوّنَ له أصلاً — فمن حفظ إعداداً لا يعرف أحُفظ.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconWarning, IconError, IconSuccess, IconNote, IconClose } from "./icons";
import { Button } from "./components";

const m = getMessages(defaultLocale);

// ══════════════════════════════════════════════════════════════════════
//  Alert — الرسالةُ في مكانها
// ══════════════════════════════════════════════════════════════════════

/**
 * **أربعُ نبراتٍ لا لونان.**
 *
 * كان الخطرُ أحمرَ والباقي بلا شكل. **والملاحظةُ ليست خطأً والتحذيرُ ليس
 * خطراً**: «لم يُحفظ» غيرُ «انتبه أنّ الرصيد قليل» غيرُ «هذا الحقل اختياريّ».
 *
 * **والأيقونةُ تسبق اللون**: من لا يميّز الأحمرَ عن الأخضر (وهم نحو ثمانيةٍ
 * بالمئة من الرجال) **يقرأ الشكلَ لا الصبغة.**
 */
const tones = {
  error: { cls: "border-danger-edge bg-danger-tint text-danger", Icon: IconError },
  success: { cls: "border-success-edge bg-success-tint text-success", Icon: IconSuccess },
  warning: { cls: "border-warning-edge bg-warning-tint text-warning", Icon: IconWarning },
  info: { cls: "border-info-edge bg-info-tint text-info", Icon: IconNote },
} as const;

export type AlertTone = keyof typeof tones;

/**
 * Alert رسالةٌ في موضعها من الصفحة — **لا تختفي وحدَها.**
 *
 * **وما يجب أن يُقرأ قبل المتابعة لا يُعرض `Toast`**: الأخيرُ يمرّ في ثوانٍ،
 * **ومن نظر إلى لوحة المفاتيح وهو يكتب فاته.** فالخطأُ في النموذج هنا،
 * والنجاحُ العابرُ هناك.
 *
 * @param onDismiss إن مُرّر ظهر زرُّ إغلاق — **ولا يُمرَّر لخطأٍ يمنع المتابعة.**
 */
export function Alert({
  tone = "error",
  title,
  children,
  onDismiss,
  className = "",
}: {
  tone?: AlertTone;
  /** سطرٌ أوّلُ بارز — **وبلاه يُقرأ النصُّ كلُّه بوزنٍ واحد.** */
  title?: string;
  children?: ReactNode;
  onDismiss?: () => void;
  className?: string;
}) {
  const { cls, Icon } = tones[tone];
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={`flex items-start gap-2.5 rounded-control border px-3 py-2.5 text-sm ${cls} ${className}`}
    >
      <Icon size={17} className="mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        {title && <p className="font-bold">{title}</p>}
        {children && <div className={title ? "mt-0.5 opacity-90" : ""}>{children}</div>}
      </div>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          aria-label={m.common.close}
          className="-me-1 shrink-0 rounded-control p-1 opacity-70 transition-opacity hover:opacity-100"
        >
          <IconClose size={15} />
        </button>
      )}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Skeleton — **الشكلُ قبل المحتوى**
// ══════════════════════════════════════════════════════════════════════

/**
 * **حالةُ التحميل كانت سطرَ نصٍّ محلَّ صفحةٍ كاملة.**
 *
 * ```tsx
 * export function LoadingState() { return <p>جارٍ التحميل</p>; }
 * ```
 *
 * **فحين تصل البيانات يقفز كلُّ شيء** — والصفحةُ تنمو من سطرٍ إلى شاشة،
 * **ومن كان إصبعُه فوق موضعٍ ضغط غيرَه.** (وهو `Layout Shift`، وممنوعٌ صراحةً
 * في معايير القبول.)
 *
 * **والهيكلُ يحجز المساحةَ نفسَها** فلا شيءَ يقفز، **ويقول شكلَ ما هو آتٍ**:
 * من رأى ثلاثةَ صفوفٍ عرف أنّ قائمةً تُحمَّل لا رسمٌ بيانيّ.
 *
 * **والنبضُ خافتٌ عمداً**: وميضٌ قويٌّ في انتظارٍ يطول يُتعب أكثرَ ممّا يطمئن.
 */
const skelRadius = {
  control: "rounded-control",
  card: "rounded-card",
  badge: "rounded-badge",
} as const;

export function Skeleton({
  className = "",
  rounded = "control",
}: {
  className?: string;
  rounded?: keyof typeof skelRadius;
}) {
  // **والأصنافُ كاملةٌ لا مركَّبة**: تيلويند يمسح المصدرَ نصّاً، **و`rounded-${x}`
  // لا يراها فتخرج الحوافُّ حادّة.**
  return (
    <span aria-hidden className={`block animate-pulse bg-ink-faint ${skelRadius[rounded]} ${className}`} />
  );
}

/** SkeletonText أسطرٌ بأطوالٍ متفاوتة — **وأسطرٌ متساويةٌ تُقرأ جدولاً لا نصّاً.** */
export function SkeletonText({ lines = 3, className = "" }: { lines?: number; className?: string }) {
  const widths = ["w-full", "w-11/12", "w-4/5", "w-3/4", "w-5/6"];
  return (
    <span className={`block space-y-2 ${className}`}>
      {Array.from({ length: lines }, (_, i) => (
        <Skeleton key={i} className={`h-3.5 ${widths[i % widths.length]}`} />
      ))}
    </span>
  );
}

/**
 * SkeletonList صفوفٌ بصورةٍ ونصّين — **شكلُ أكثرِ قوائم المنصة.**
 *
 * **ولا هيكلَ عامٌّ لكلّ شيء**: مستطيلٌ رماديٌّ كبيرٌ لا يقول ما هو آتٍ،
 * **وهذا يقول «قائمةٌ من صفوف».**
 */
export function SkeletonList({ rows = 4 }: { rows?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="flex items-center gap-3 surface p-3">
          <Skeleton className="h-12 w-12 shrink-0" />
          <span className="min-w-0 flex-1 space-y-2">
            <Skeleton className="h-3.5 w-2/5" />
            <Skeleton className="h-3 w-1/4" />
          </span>
        </div>
      ))}
    </div>
  );
}

/** SkeletonStats بطاقاتُ مؤشّرات — **صدرُ أكثرِ اللوحات.** */
export function SkeletonStats({ count = 3 }: { count?: number }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="surface p-4">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="mt-2.5 h-6 w-28" />
        </div>
      ))}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Toast — **النجاحُ العابر**
// ══════════════════════════════════════════════════════════════════════

/**
 * **لم يكن للنجاح مكوّنٌ أصلاً.**
 *
 * فمن حفظ إعداداً أو أرسل نموذجاً **لا يعرف أوقع أم لا** — فيضغط ثانيةً.
 * (وبعضُ الشاشات تكتب سطراً أخضرَ يبقى إلى أن تُحدَّث الصفحة، **فيُقرأ حالةً
 * دائمةً لا خبرَ لحظة.**)
 *
 * # ولماذا يمرّ ولا يبقى
 *
 * **الخبرُ الذي لا يحتاج فعلاً لا يستحقّ مكاناً دائماً**: «حُفظ» تُقرأ في
 * ثانيةٍ وتُنسى، **وسطرٌ يبقى يزاحم ما يُقرأ.**
 *
 * **والخطأُ لا يمرّ**: يبقى في موضعه (`Alert`) لأنّ صاحبَه يحتاج أن يقرأه
 * وهو يُصلح.
 */
export interface ToastMsg {
  id: number;
  tone: AlertTone;
  text: string;
}

/**
 * useToast صفُّ رسائلَ عابرة — **والمنطقُ منفصلٌ عن الشكل.**
 *
 * ```tsx
 * const { toasts, push, Toaster } = useToast();
 * … push("حُفظ", "success");
 * <Toaster />
 * ```
 */
export function useToast(ms = 3200) {
  const [toasts, setToasts] = useState<ToastMsg[]>([]);
  const seq = useRef(0);
  const timers = useRef<ReturnType<typeof setTimeout>[]>([]);

  const drop = useCallback((id: number) => {
    setToasts((t) => t.filter((x) => x.id !== id));
  }, []);

  const push = useCallback(
    (text: string, tone: AlertTone = "success") => {
      const id = ++seq.current;
      setToasts((t) => [...t, { id, tone, text }]);
      timers.current.push(setTimeout(() => drop(id), ms));
    },
    [drop, ms],
  );

  // **ومؤقّتاتٌ تبقى بعد رحيل الشاشة تُحدّث ما لا وجودَ له.**
  useEffect(() => () => timers.current.forEach(clearTimeout), []);

  const Toaster = useCallback(
    () => <ToastStack toasts={toasts} onDrop={drop} />,
    [toasts, drop],
  );

  return { toasts, push, drop, Toaster };
}

/**
 * ToastStack مكانُ الرسائل — **أسفلَ الشاشة لا أعلاها.**
 *
 * **والأعلى محجوزٌ للشريط**، وعلى الجوّال يقع تحت الإبهام حيث ينظر المستخدم
 * أصلاً بعد أن يضغط.
 */
export function ToastStack({
  toasts,
  onDrop,
}: {
  toasts: ToastMsg[];
  onDrop: (id: number) => void;
}) {
  if (toasts.length === 0) return null;
  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed inset-x-3 bottom-4 z-[60] flex flex-col items-center gap-2 sm:inset-x-auto sm:end-4 sm:items-end"
    >
      {toasts.map((t) => (
        <div key={t.id} className="pointer-events-auto w-full max-w-sm elev-3">
          <Alert
            tone={t.tone}
            onDismiss={() => onDrop(t.id)}
            className="surface-raised"
          >
            {t.text}
          </Alert>
        </div>
      ))}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Confirm — **تأكيدُ ما لا يُستدرَك**
// ══════════════════════════════════════════════════════════════════════

/**
 * **ما لا يُستدرَك يُسأل عنه مرّةً واحدةً بوضوح.**
 *
 * وكان بعضُه بـ`confirm()` الأصليّ — **نافذةُ نظامٍ بخطّ النظام ولغته**،
 * تخرج من المنصة كلَّ خروج، **وأزرارُها «موافق/إلغاء» لا تقول ماذا سيقع.**
 *
 * # وزرُّ الخطر يقول فعلَه
 *
 * **«موافق» لا تعني شيئاً**: من قرأها بسرعةٍ لا يعرف على ماذا وافق. **و«احذف
 * الحساب» تقول.**
 *
 * # والافتراضيُّ هو الإلغاء
 *
 * **من ضغط `Enter` بلا قراءة لا يجب أن يحذف** — فالتركيزُ يبدأ على الإلغاء.
 */
export function Confirm({
  open,
  title,
  body,
  confirmLabel,
  onConfirm,
  onCancel,
  busy = false,
  tone = "danger",
}: {
  open: boolean;
  title: string;
  /** **ماذا يقع بالضبط** — وعمومُ الكلام يجعل الضغطَ عادةً. */
  body?: ReactNode;
  /** **نصٌّ يقول الفعل** — لا «موافق». */
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
  /** **يمنع النقرَ المتكرّر** — وضغطتان تحذفان مرّتين. */
  busy?: boolean;
  tone?: "danger" | "primary";
}) {
  const cancelRef = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    if (!open) return;
    cancelRef.current?.querySelector("button")?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onCancel();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onCancel]);

  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-[70] flex items-end justify-center scrim p-0 sm:items-center sm:p-4"
      onClick={onCancel}
    >
      {/* **وعلى الجوّال يصعد من الأسفل** — نافذةٌ في وسط شاشةٍ طويلةٍ تترك
          اليدَ بعيدةً عن أزرارها. */}
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="cf-t"
        onClick={(e) => e.stopPropagation()}
        className="surface-sheet w-full max-w-sm p-5"
      >
        <h2 id="cf-t" className="text-lg font-bold">
          {title}
        </h2>
        {body && <div className="mt-1.5 text-sm text-ink-muted">{body}</div>}
        <div className="mt-5 flex gap-2">
          <Button
            variant={tone === "danger" ? "danger" : "primary"}
            onClick={onConfirm}
            disabled={busy}
          >
            {busy ? m.common.loading : confirmLabel}
          </Button>
          {/* **والتركيزُ يبدأ هنا** — من ضغط `Enter` بلا قراءة لا يجب أن
              يحذف. و`Button` لا يمرّر المرجعَ، **فالمرجعُ على غلافه.** */}
          <span ref={cancelRef}>
            <Button variant="secondary" onClick={onCancel} disabled={busy}>
              {m.common.cancel}
            </Button>
          </span>
        </div>
      </div>
    </div>
  );
}
