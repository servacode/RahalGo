"use client";

/**
 * الخطُّ الزمنيّ — عنصرٌ واحد لكل ما يقع على التوالي.
 *
 * # لماذا عنصرٌ لا نمطٌ يُعاد
 *
 * ثلاثةُ مواضع في المنصة تعرض أشياءَ وقعت بترتيبها: **رحلةُ الطلب**، و**سجلُّ
 * الإشعارات**، و**أثرُ الأحداث**. وكانت كلٌّ منها ترسم خطَّها بيدها — قائمةً
 * هنا، وأرقاماً هناك، ونقاطاً ثالثةً — **فيرى المستخدمُ ثلاثَ لغاتٍ لمعنًى
 * واحد**، ويتعلّم قراءةَ كلٍّ منها على حدة.
 *
 * # وهندستُه: قضيبٌ ونقاط
 *
 * خطٌّ رأسيٌّ رفيع تجلس عليه عُقَدٌ دائرية. **والقضيبُ خلف العُقَد لا بينها** —
 * فلو رُسم قطعاً بين كل عقدتين لظهرت فجواتٌ عند اختلاف ارتفاع الأسطر، وهي
 * أوّلُ ما تلتقطه العين وآخرُ ما يُصلَح.
 *
 * # وثلاثُ حالاتٍ لا اثنتان
 *
 *   - **`done`** ما وقع: عقدةٌ مصمتة وقضيبٌ ملوّن قبلها
 *   - **`current`** ما يقع الآن: حلقةٌ بهالةٍ نابضة — **نبضةٌ واحدة في الشاشة
 *     لا أكثر**، فالنبضُ إن تعدّد صار ضجيجاً ولم يعد يدلّ
 *   - **`todo`** ما لم يقع: عقدةٌ فارغة وقضيبٌ باهت
 *
 * والزمنُ في المنتهى بخانةٍ ثابتة: **الأرقامُ تحت بعضها تُقارَن بلمحة، وإن
 * زحفت مع طول النصّ لم تُقارَن أبداً.**
 */

import type { ComponentType, ReactNode } from "react";

export type TimelineState = "done" | "current" | "todo";

export interface TimelineNode {
  id: string;
  state: TimelineState;
  icon?: ComponentType<{ size?: number; className?: string; strokeWidth?: number }>;
  title: ReactNode;
  detail?: ReactNode;
  /** يُعرض في المنتهى بخانةٍ ثابتة — الوقتُ أو المبلغ */
  trailing?: ReactNode;
  /** لونُ العقدة حين تكون `done` — الافتراضيّ لونُ المنصة */
  tone?: "primary" | "success" | "danger" | "warning" | "info" | "violet";
  onClick?: () => void;
  href?: string;
}

const TONE_SOLID: Record<string, string> = {
  primary: "bg-primary text-on-solid",
  success: "bg-success text-on-solid",
  danger: "bg-danger text-on-solid",
  warning: "bg-warning text-on-solid",
  info: "bg-info text-on-solid",
  violet: "bg-violet text-on-solid",
};

const TONE_RAIL: Record<string, string> = {
  primary: "bg-primary-fill",
  success: "bg-success-fill",
  danger: "bg-danger-fill",
  warning: "bg-warning-fill",
  info: "bg-info-fill",
  violet: "bg-violet-fill",
};

export function Timeline({
  nodes,
  Link,
  className = "",
}: {
  nodes: TimelineNode[];
  /** يلزم إن كانت في العُقَد روابط */
  Link?: ComponentType<{ href: string; className?: string; children: ReactNode }>;
  className?: string;
}) {
  return (
    <ol className={`relative ${className}`}>
      {nodes.map((n, i) => {
        const tone = n.tone ?? "primary";
        const last = i === nodes.length - 1;
        const lit = n.state !== "todo";

        const body = (
          <div className="flex min-w-0 flex-1 items-start gap-3 py-2.5">
            <div className="min-w-0 flex-1">
              <p
                className={`truncate leading-snug ${
                  n.state === "todo" ? "text-ink-muted" : "font-bold text-ink"
                }`}
              >
                {n.title}
              </p>
              {n.detail && (
                <div className="mt-0.5 text-xs leading-relaxed text-ink-muted">{n.detail}</div>
              )}
            </div>
            {n.trailing && (
              <div
                className="shrink-0 pt-0.5 text-xs tabular-nums text-ink-muted"
                dir="ltr"
              >
                {n.trailing}
              </div>
            )}
          </div>
        );

        return (
          <li key={n.id} className="relative flex gap-3">
            {/* العمودُ الأيسر: القضيبُ والعقدة معاً في حاويةٍ واحدة */}
            <div className="relative flex w-9 shrink-0 justify-center">
              {/* **القضيبُ ممتدٌّ خلف العقدة** لا مقطوعٌ بينها */}
              {!last && (
                <span
                  aria-hidden
                  className={`absolute top-3 bottom-0 w-px ${
                    lit ? TONE_RAIL[tone] : "bg-line"
                  }`}
                />
              )}
              <span
                className={`relative z-10 mt-1.5 flex h-7 w-7 items-center justify-center rounded-badge text-2xs font-bold transition-colors ${
                  n.state === "done"
                    ? TONE_SOLID[tone]
                    : n.state === "current"
                      ? // **الجاريةُ بالنبرة — والمنتهيةُ بالتعبئة الهادئة.**
                        //
                        // قاعدةُ العلامة: الهادئُ ما ثبت، والنبرةُ لِما
                        // يتحرّك. **وهنا تُرى القاعدةُ عاملةً**: عينُ الزبون
                        // تقع على مكانِ طلبه الآن قبل أن تقرأ حرفاً.
                        "bg-accent text-on-bright ring-4 ring-accent-edge"
                      : "border-2 border-line bg-surface text-ink-muted"
                }`}
              >
                {n.icon ? <n.icon size={14} strokeWidth={2.5} /> : i + 1}
              </span>
            </div>

            {n.href && Link ? (
              <Link
                href={n.href}
                className="-mx-2 flex min-w-0 flex-1 rounded-control px-2 transition-colors hover:bg-page"
              >
                {body}
              </Link>
            ) : n.onClick ? (
              <button
                type="button"
                onClick={n.onClick}
                className="-mx-2 flex min-w-0 flex-1 rounded-control px-2 text-start transition-colors hover:bg-page"
              >
                {body}
              </button>
            ) : (
              body
            )}
          </li>
        );
      })}
    </ol>
  );
}
