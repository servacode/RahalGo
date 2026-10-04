"use client";

/**
 * **جدولٌ واحدٌ للخسائر والنزاعات** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «الخسائر والنزاعات»).
 *
 * كانت الخسائرُ جدولاً من المكتبة المشتركة والنزاعاتُ قائمةً مرسومةً باليد —
 * **الواقعةُ واحدةٌ فتُقرأ بشكلٍ واحد**: المبلغ · الطرف · السبب · الطلب (رابط) ·
 * التاريخ · الحالة · الإجراء. **وشريطُ الفلاتر واحدٌ لهما.**
 */

import type { ReactNode } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconWallet,
  IconDate,
  IconStatus,
  IconOrder,
  IconUser,
} from "@rahalgo/ui";

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

/** **رابطُ الطرف** — المتجرُ إلى ملفّ المتجر، والشخصُ إلى حسابه. */
function partyHref(role: string, id: string): string {
  return role === "merchant" ? `/dashboard/merchants/${id}` : `/dashboard/users/${id}`;
}

export function CaseTable({
  screen,
  rows,
  empty,
  actions,
  noDisputeLabel,
}: {
  /** مفتاحُ حفظ تفضيل العرض (جدول/بطاقات). */
  screen: string;
  rows: CaseRow[];
  empty: string;
  actions?: (row: CaseRow) => ReactNode;
  /** **نصُّ الحالة لسطرٍ بلا نزاع** — في الخسائر وحدَها. */
  noDisputeLabel?: string;
}) {
  const [view, setView] = useViewMode(screen);
  const partyOne = C.partyOne as Record<string, string>;
  const statusOne = C.statusOne as Record<string, string>;

  const columns: DataColumn<CaseRow>[] = [
    {
      id: "amount",
      header: C.amount,
      icon: <IconWallet />,
      primary: true,
      cell: (x) => (
        <span dir="ltr" className={`font-bold ${x.status === null ? "text-danger" : "text-warning"}`}>
          {fmtNum(x.amount)}
          {x.recovered ? (
            <span className="block text-xs font-normal text-ink-muted">
              {C.recovered.replace("{n}", fmtNum(x.recovered))}
            </span>
          ) : null}
        </span>
      ),
    },
    {
      id: "party",
      header: C.party,
      icon: <IconUser />,
      cell: (x) =>
        x.partyId && x.partyRole ? (
          <span>
            <Link href={partyHref(x.partyRole, x.partyId)} className="font-medium text-primary-dark hover:underline">
              {x.partyName || "—"}
            </Link>{" "}
            <Badge variant="neutral">{partyOne[x.partyRole] ?? x.partyRole}</Badge>
            {(x.waivedBefore ?? 0) > 0 && (
              <span className="block text-xs text-warning">
                {C.waivedBefore.replace("{n}", fmtNum(x.waivedBefore ?? 0))}
              </span>
            )}
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "reason",
      header: C.reason,
      icon: <IconStatus />,
      cell: (x) => <span className="line-clamp-2">{reasonText(x.reason)}</span>,
    },
    {
      id: "order",
      header: m.terms.order,
      icon: <IconOrder />,
      cell: (x) =>
        x.orderNumber == null ? (
          "—"
        ) : x.orderId ? (
          <Link href={`/dashboard/orders?id=${x.orderId}`} dir="ltr" className="text-primary-dark hover:underline">
            #{fmtRef(x.orderNumber)}
          </Link>
        ) : (
          <span dir="ltr">#{fmtRef(x.orderNumber)}</span>
        ),
    },
    {
      id: "date",
      header: m.admin.losses.date,
      icon: <IconDate />,
      cell: (x) => (
        <span dir="ltr" className="text-xs text-ink-muted">
          {fmtDateTime(x.date)}
        </span>
      ),
    },
    {
      id: "status",
      header: C.statusFilter,
      icon: <IconStatus />,
      cell: (x) => (
        <span>
          {x.status ? (
            <Badge variant={STATUS_TONE[x.status]}>{statusOne[x.status] ?? x.status}</Badge>
          ) : (
            <span className="text-xs text-ink-muted">{noDisputeLabel ?? "—"}</span>
          )}
          {x.statusNote && <span className="mt-1 block text-xs text-ink-muted">{x.statusNote}</span>}
        </span>
      ),
    },
  ];

  return (
    <>
      <div className="mb-2 flex justify-end">
        <ViewToggle view={view} onChange={setView} tableLabel={m.common.viewTable} cardsLabel={m.common.viewCards} />
      </div>
      <DataView items={rows} getKey={(x) => x.key} columns={columns} actions={actions} view={view} empty={empty} />
    </>
  );
}

/** **شريطُ الفلاتر الواحد** — صفٌّ واحدٌ بنفس الشكل في التبويبين. */
export function CaseFilters({ children }: { children: ReactNode }) {
  return <div className="mb-4 flex flex-wrap items-end gap-2">{children}</div>;
}
