"use client";

/**
 * **بطاقاتٌ واحدةٌ للخسائر والنزاعات** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «الخسائر والنزاعات»).
 *
 * **الواقعةُ واحدةٌ فتُقرأ بشكلٍ واحد**: المبلغ · الطرف · السبب · الطلب (رابط) ·
 * التاريخ · الحالة · الإجراء. **وشريطُ الفلاتر واحدٌ لهما.**
 *
 * تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦): «ألغِ عرض الجداول» — كرتٌ لكلّ واقعة بشبكةٍ متجاوبة،
 * وفراغٌ بدائرةٍ وعنوانٍ وجملة.
 */

import type { ComponentType, ReactNode } from "react";
import { useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { getMessages, defaultLocale, fmtMoney, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import { Badge, IconDate, IconOrder, IconStore, IconUser, IconWallet } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const C = m.admin.claims;
const FAIL_REASONS: Record<string, string> = m.common.failReasons;

/** **سببٌ مصنَّفٌ يُترجَم، وحرٌّ يُعرض كما كُتب.** */
export const reasonText = (r: string) => FAIL_REASONS[r] ?? r;

export type CaseStatus = "open" | "settled" | "waived";

export interface CaseRow {
  key: string;
  amount: number;
  /** **ما بقي مفتوحاً وما استُرِدّ** — للنزاع المخصوم جزئيّاً. */
  recovered?: number;
  partyRole?: string | null;
  partyId?: string | null;
  partyName?: string | null;
  waivedBefore?: number;
  reason: string;
  orderId?: string | null;
  orderNumber?: number | null;
  date: string;
  /** **حالُ النزاع** — وفارغٌ في سطر خسارةٍ بلا نزاع. */
  status: CaseStatus | null;
  /** **ما يُكتب تحت الحالة** — «بانتظار الموافقة» مثلاً. */
  statusNote?: ReactNode;
}

const STATUS_TONE: Record<CaseStatus, "warning" | "success" | "neutral"> = {
  open: "warning",
  settled: "success",
  waived: "neutral",
};

/** **لونُ دائرة الطرف بحال الواقعة** — خسارةٌ بلا نزاعٍ حمراء. */
const AVATAR_TONE: Record<CaseStatus | "none", string> = {
  open: "bg-warning-solid text-on-solid",
  settled: "bg-success-solid text-on-solid",
  waived: "bg-field text-ink-muted",
  none: "bg-danger-solid text-on-solid",
};

/** **رابطُ الطرف** — المتجرُ إلى ملفّ المتجر، والشخصُ إلى حسابه. */
function partyHref(role: string, id: string): string {
  return role === "merchant" ? `/dashboard/merchants/${id}` : `/dashboard/users/${id}`;
}

/** **فراغٌ واحدٌ للتبويبات** — دائرةٌ بأيقونة وعنوانٌ وجملةٌ تساعد. */
export function CaseEmpty({
  icon: Icon,
  title,
  hint,
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  title: string;
  hint: string;
}) {
  return (
    <section className="surface flex flex-col items-center gap-3 p-8 text-center">
      <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
        <Icon size={28} />
      </span>
      <h2 className="heading-section">{title}</h2>
      <p className="max-w-md text-sm text-ink-muted">{hint}</p>
    </section>
  );
}

export function CaseList({
  rows,
  empty,
  emptyHint,
  actions,
  noDisputeLabel,
}: {
  rows: CaseRow[];
  empty: string;
  emptyHint: string;
  /** **أزرارُ البطاقة** — تُرسم بعرضها في ذيلها. */
  actions?: (row: CaseRow) => ReactNode;
  /** **نصُّ الحالة لسطرٍ بلا نزاع** — في الخسائر وحدَها. */
  noDisputeLabel?: string;
}) {
  // **وملفُّ الطرف والطلبُ لمن يُفتحان له** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
  const open = useCanOpen();
  const partyOne = C.partyOne as Record<string, string>;
  const statusOne = C.statusOne as Record<string, string>;

  if (rows.length === 0) return <CaseEmpty icon={IconWallet} title={empty} hint={emptyHint} />;

  return (
    <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
      {rows.map((x) => {
        const act = actions?.(x);
        const PartyIcon = x.partyRole === "merchant" ? IconStore : x.partyRole ? IconUser : IconWallet;
        return (
          <li key={x.key} className="surface flex flex-col gap-3 p-4">
            {/* **الطرفُ وحالُه** */}
            <div className="flex items-start gap-3">
              <span
                className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${AVATAR_TONE[x.status ?? "none"]}`}
              >
                <PartyIcon size={20} />
              </span>
              <span className="min-w-0 flex-1">
                {x.partyId && x.partyRole ? (
                  <OpenLink
                    allowed={x.partyRole === "merchant" ? open.merchant : open.user}
                    href={partyHref(x.partyRole, x.partyId)}
                    className="block truncate font-bold hover:underline"
                  >
                    {x.partyName || "—"}
                  </OpenLink>
                ) : (
                  <span className="block truncate font-bold text-ink-muted">—</span>
                )}
                {x.partyRole && (
                  <span className="block text-2xs text-ink-muted">{partyOne[x.partyRole] ?? x.partyRole}</span>
                )}
                {(x.waivedBefore ?? 0) > 0 && (
                  <span className="block text-2xs font-bold text-warning">
                    {C.waivedBefore.replace("{n}", fmtNum(x.waivedBefore ?? 0))}
                  </span>
                )}
              </span>
              <span className="shrink-0">
                {x.status ? (
                  <Badge variant={STATUS_TONE[x.status]}>{statusOne[x.status] ?? x.status}</Badge>
                ) : (
                  <Badge variant="neutral">{noDisputeLabel ?? "—"}</Badge>
                )}
              </span>
            </div>

            {/* **المبلغُ كبيراً والطلبُ بجانبه** */}
            <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
              <span>
                <span className="block text-2xs text-ink-muted">{C.amount}</span>
                <span className={`figure block ${x.status === null ? "text-danger" : "text-warning"}`}>
                  {fmtMoney(x.amount)}
                </span>
                {x.recovered ? (
                  <span className="block text-2xs text-ink-muted">
                    {C.recovered.replace("{n}", fmtMoney(x.recovered))}
                  </span>
                ) : null}
              </span>
              {x.orderNumber != null && (
                <span className="flex items-center gap-1 text-sm">
                  <IconOrder size={14} className="text-ink-muted" />
                  {x.orderId ? (
                    <OpenLink
                      allowed={open.order}
                      href={`/dashboard/orders?id=${x.orderId}`}
                      dir="ltr"
                      className="font-bold text-primary-dark hover:underline"
                    >
                      #{fmtRef(x.orderNumber)}
                    </OpenLink>
                  ) : (
                    <span dir="ltr" className="font-bold">
                      #{fmtRef(x.orderNumber)}
                    </span>
                  )}
                </span>
              )}
            </div>

            {x.reason && <p className="border-s-2 border-accent ps-3 text-sm leading-relaxed">{reasonText(x.reason)}</p>}

            {x.statusNote && (
              <p className="rounded-control bg-warning-solid px-2.5 py-1 text-xs font-bold text-on-solid">{x.statusNote}</p>
            )}

            <span className="flex items-center gap-1 text-2xs text-ink-muted">
              <IconDate size={12} />
              <span dir="ltr">{fmtDateTime(x.date)}</span>
            </span>

            {act && <div className="mt-auto grid grid-cols-2 gap-2 border-t border-line-soft pt-3">{act}</div>}
          </li>
        );
      })}
    </ul>
  );
}

/** **شريطُ الفلاتر الواحد** — كرتٌ واحدٌ بنفس الشكل في التبويبين. */
export function CaseFilters({ children }: { children: ReactNode }) {
  return <section className="surface mb-4 grid grid-cols-1 gap-2 p-4 sm:grid-cols-2 lg:grid-cols-4">{children}</section>;
}
