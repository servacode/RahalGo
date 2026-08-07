"use client";

/**
 * **كرتُ الشكوى والبلاغ — واحدٌ في الخمس.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الكرتُ هذا الخاصُّ بالشكاوي والبلاغات، طبّقوه
 *  على كلّ صفحات البلاغات بكلّ اللوحات».)
 *
 * # ما وجده الجرد
 *
 * **ثلاثُ صياغاتٍ لشيءٍ واحد**: كرتٌ غنيٌّ في شاشة الزبون · **سطرٌ عارٍ**
 * عند السائق والمتجر (`#12 · موضوع · تاريخ` في صفٍّ واحد) · وجدولٌ في
 * الإدارة.
 *
 * **ومن رفع شكوى وهو زبونٌ ثمّ صار سائقاً رأى شاشتين لا يجمعهما شيء** —
 * والبياناتُ هي هي.
 *
 * # ولماذا حقولٌ مُعنوَنةٌ لا أسطرٌ متتابعة
 *
 * **كانت أسطراً بلا حدود**: العينُ لا تعرف أين ينتهي خبرٌ ويبدأ آخر، فيُقرأ
 * التاريخُ جزءاً من السبب والتعويضُ جزءاً من الردّ. **وشكوى تُقرأ خطأً
 * تُعاد.**
 *
 * # والنصوصُ تُمرَّر ولا تُفترض
 *
 * **الحزمةُ المشتركةُ لا تعرف من يعرضها**: «شكوى مفتوحةٌ على طلبك» صحيحةٌ
 * للزبون وخاطئةٌ للسائق الذي رُفعت عليه. **فما يختلف بالدور يُمرَّر**، وما
 * لا يختلف يُقرأ من المعجم هنا مرّةً واحدة.
 */

import { type ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import { Card } from "./layout";
import { Badge } from "./components";
import { IconTile } from "./components";
import { IconSupport, IconReply, IconCheck } from "./icons";

const m = getMessages(defaultLocale);
const C = m.site.complaint;

/** نبرةُ الحالة — **ولكلّ حالٍ لونُه**، فتُقرأ الشكوى قبل أن تُقرأ. */
const TONE: Record<string, "warning" | "info" | "success" | "danger" | "neutral"> = {
  open: "warning",
  in_review: "info",
  resolved: "success",
  rejected: "danger",
  closed: "neutral",
};

export interface ComplaintTicket {
  id: string;
  number: number;
  subject: string;
  reason?: string;
  status: string;
  order_number?: number | null;
  compensation?: number;
  resolution?: string;
  created_at: string;
}

export function ComplaintCard({
  ticket: t,
  /**
   * سطرٌ تحت العنوان يقول **ما هذه الشكوى بالنسبة إليك.**
   *
   * **ويختلف بالدور**: «شكوى مفتوحةٌ على طلبك» للزبون، و«بلاغٌ رُفع عليك»
   * لمن رُفع عليه. **وفارغٌ يعني لا سطرَ أصلاً** — لا سطرٌ فارغٌ يُسأل عنه.
   */
  note = "",
  action,
  className = "",
}: {
  ticket: ComplaintTicket;
  note?: string;
  /**
   * **زرٌّ أسفلَ الكرت — لمن يملك فعلاً عليه.**
   *
   * (طلبُ المالك ٢٠٢٦-٠٨-٠٨: البلاغات في لوحة الإدارة كروتٌ وجداول.)
   *
   * **والزبونُ يقرأ بلاغَه ولا يفعل به شيئاً**، والإدارةُ تفتحه وتردّ.
   * فالكرتُ واحدٌ والفعلُ يأتي من صاحبه — **لا كرتان يفترقان بمرور الوقت.**
   */
  action?: ReactNode;
  className?: string;
}) {
  const reasons = m.site.complaint.reasons as Record<string, string>;
  const statuses = m.admin.tickets.status as Record<string, string>;
  return (
    <Card className={`flex flex-col gap-4 ${className}`}>
      {/* ── الترويسة: السببُ · الرقمُ · الحالة ─────────────────────── */}
      <div className="flex items-start gap-3">
        <IconTile>
          <IconSupport size={20} />
        </IconTile>
        <div className="min-w-0 flex-1">
          {/* **السببُ مُترجَمٌ إن عُرف، وإلّا فالموضوع** — ورمزٌ إنكليزيٌّ
              لا يُعرض على أحد. */}
          <p className="font-bold leading-tight">{(t.reason && reasons[t.reason]) || t.subject}</p>
          {note && <p className="mt-0.5 text-xs text-ink-muted">{note}</p>}
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1.5">
          {/* **الرقمُ بحقلٍ خاصّ** — هو ما يقوله حين يتّصل يسأل. */}
          <span
            dir="ltr"
            className="rounded-control bg-primary-tint px-2.5 py-1 text-sm font-bold tabular-nums text-primary"
          >
            #{fmtRef(t.number)}
          </span>
          <Badge variant={TONE[t.status] ?? "neutral"}>{statuses[t.status] ?? t.status}</Badge>
        </div>
      </div>

      {/* ── حقلان مستقلّان: متى · وعلى أيّ طلب ─────────────────────── */}
      <div className="grid grid-cols-2 gap-2">
        <div className="surface-inset px-3 py-2">
          <p className="text-2xs text-ink-muted">{C.fieldWhen}</p>
          <p className="mt-0.5 text-sm font-medium tabular-nums" dir="ltr">
            {fmtDateTime(t.created_at)}
          </p>
        </div>
        <div className="surface-inset px-3 py-2">
          <p className="text-2xs text-ink-muted">{C.fieldOrder}</p>
          <p className="mt-0.5 text-sm font-medium tabular-nums" dir="ltr">
            {t.order_number != null ? `#${fmtRef(t.order_number)}` : "—"}
          </p>
        </div>
      </div>

      {/* ── ردُّ المنصة — **حقلٌ مُعنوَنٌ لا سطرٌ عائم** ──────────────── */}
      <div className="flex-1 surface-inset p-3">
        <p className="mb-1 flex items-center gap-1.5 text-2xs font-bold text-ink-muted">
          <IconReply size={13} />
          {C.fieldReply}
        </p>
        {t.resolution ? (
          <p className="text-sm">{t.resolution}</p>
        ) : (
          /* **ومفتوحةٌ بلا ردٍّ تقول ذلك** — الصمتُ في الشاشة يُقرأ إهمالاً،
             **وجملةٌ واحدةٌ تحوّل الانتظارَ من قلقٍ إلى مهلة.** */
          <p className="text-sm text-ink-muted">{C.waiting}</p>
        )}
      </div>

      {/* ── التعويضُ حقلٌ قائمٌ بذاته — **وهو ماله** ─────────────────── */}
      {(t.compensation ?? 0) > 0 && (
        <div className="flex items-center justify-between rounded-control border border-success-edge bg-success-tint px-3 py-2">
          <span className="flex items-center gap-1.5 text-sm font-medium text-success">
            <IconCheck size={15} strokeWidth={3} />
            {C.compensated}
          </span>
          <span dir="ltr" className="text-base font-bold tabular-nums text-success">
            {fmtNum(t.compensation ?? 0)}{" "}
            <span className="text-xs font-normal">{m.common.currency}</span>
          </span>
        </div>
      )}

      {action && <div className="mt-auto">{action}</div>}
    </Card>
  );
}

/** **وشبكتُها واحدةٌ كذلك** — عمودٌ على الجوّال وعمودان على المتّسع. */
export function ComplaintGrid({ children }: { children: React.ReactNode }) {
  return <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">{children}</div>;
}
