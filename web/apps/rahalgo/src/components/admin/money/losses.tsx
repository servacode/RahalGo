"use client";

/**
 * **خسائرُ المنصة** — من الدفتر لا من تقدير.
 *
 * قاعدةُ المالك: «نحسب الخسارة الفعلية فقط وليس الخسارة الافتراضية». فما يُعرض
 * هنا **قيودُ مصروفٍ خرجت من الخزينة فعلاً**: بضاعةٌ لم يستردّها متجر، وتعويضُ
 * سائقٍ عن طلبٍ فشل.
 *
 * # وأربعةُ أرقامٍ جنباً إلى جنب (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٤)
 *
 * خرج من الخزينة · رجع من النزاعات · **الخسارةُ الصافية** · ولنا عند الأطراف الآن.
 * **وكان المجموعُ إجماليّاً وحدَه** — فلا يُعرف كم خسرنا فعلاً بعد الاسترجاع.
 * **وسطرُ الخسارة يحمل حالَ نزاعه** (مفتوح · انخصم · انسقط · لا نزاع).
 *
 * **والأيّامُ بتوقيت دمشق** — الخادمُ يحسب حدَّي اليوم بها.
 *
 * # وكلُّ خسارةٍ بأرقام صفحة الأرباح (قرارُ المالك ٢٠٢٦-١٠-١٥)
 *
 * خسائرُ الطلبات الفاشلة والملغاة والمستردّة · التعويضات · نقصُ الصندوق · التسويةُ
 * اليدويّةُ إيداعاً · الديونُ المشطوبة — **من التصنيف الواحد في المحرّك** (`LossLines`)،
 * والمستردُّ مستردُّ صفحة الأرباح. **فالصافي هنا = سطورُ الخسارة هناك − المستردّ.**
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";
import {
  PageContainer,
  Pagination,
  LoadingState,
  ReloadState,
  Input,
  useLiveData,
  IconWallet,
  IconBalance,
  IconTrendDown,
  IconTrendUp,
  IconBlock,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { Tile } from "@/components/admin/DashParts";
import { CaseList, CaseFilters, type CaseRow, type CaseStatus } from "./caseTable";

import { SectionBar } from "@/components/admin/support/SectionBar";
const m = getMessages(defaultLocale);
const L = m.admin.losses;
const KINDS: Record<string, string> = L.kinds;
const PR = m.admin.profits;
/** **أسماءُ السطور كما في صفحة الأرباح** — المفاتيحُ نفسُها من المحرّك (`LossLines`). */
const LINE_LABEL: Record<string, string> = {
  lost_failed: PR.lineLostFailed,
  lost_cancelled: PR.lineLostCancelled,
  lost_refunded: PR.lineLostRefunded,
  compensations: PR.lineCompensations,
  shortfalls: PR.lineShortfalls,
  adjustment_credits: PR.lineAdjustmentCredits,
  written_off: PR.lineWrittenOff,
};

interface Loss {
  /** **بابُ الخسارة** — من التصنيف الواحد مع صفحة الأرباح. */
  kind: string;
  order_id: string | null;
  order_number: number | null;
  amount: number;
  note: string;
  created_at: string;
  dispute_id: string | null;
  dispute_status: CaseStatus | null;
  party_role: string | null;
  party_id: string | null;
  party_name: string | null;
}

interface LossPage {
  losses: Loss[];
  /** **مجموعُ خسائر المدّة كلِّها** — لا مجموعُ الصفحة. */
  total: number;
  recovered: number;
  net: number;
  owed: number;
  /** **ديونٌ شُطبت في المدّة** — من طلبات الشطب الموافَق عليها، **وهي داخلَ المجموع** (٢٠٢٦-١٠-١٥). */
  written_off?: number;
  /** **سطورُ الخسارة نفسُها التي في صفحة الأرباح** — رقماً موجباً. */
  lines?: { key: string; amount: number }[];
  count: number;
  per_page: number;
}

/** **يومُ دمشق** — لا يومُ جهاز المتصفّح ولا غرينتش. */
function damascusDay(offsetDays = 0): string {
  const d = new Date(Date.now() + 3 * 3600_000 + offsetDays * 86400_000);
  return d.toISOString().slice(0, 10);
}

export function LossesView() {
  const [to, setTo] = useState(damascusDay());
  const [from, setFrom] = useState(damascusDay(-29));
  /** **صفحةُ الكشف** — (قرارُ المالك ٢٠٢٦-٠٨-١٠). */
  const [page, setPage] = useState(1);

  const { data, error, reload } = useLiveData<LossPage>(
    () => api(`/api/v1/admin/reports/losses?from=${from}&to=${to}&page=${page}`),
    ["wallet", "order", "dispute"],
    [from, to, page],
  );

  const rows: CaseRow[] = (data?.losses ?? []).map((x, i) => ({
    key: `${x.created_at}-${x.amount}-${i}`,
    amount: x.amount,
    partyRole: x.party_role,
    partyId: x.party_id,
    partyName: x.party_name,
    reason: [KINDS[x.kind] ?? x.kind, x.note].filter(Boolean).join(" — "),
    orderId: x.order_id,
    orderNumber: x.order_number,
    date: x.created_at,
    status: x.dispute_status,
  }));

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — التاريخُ بكرت، خمسةُ أرقامٍ بمربّعات، توزيعُ الخسارة بكرت،
  // وكرتٌ لكلّ خسارة.
  return (
    <PageContainer>
      <SectionBar hint={L.hint} />

      <CaseFilters>
        <Input
          label={m.shared.statement.from}
          type="date"
          value={from}
          onChange={(e) => {
            setPage(1);
            setFrom(e.target.value);
          }}
        />
        <Input
          label={m.shared.statement.to}
          type="date"
          value={to}
          onChange={(e) => {
            setPage(1);
            setTo(e.target.value);
          }}
        />
      </CaseFilters>

      {/* **الخطأُ غيرُ التحميل** — كان الفشلُ يُبقي «جاري التحميل» للأبد. */}
      {error ? (
        <ReloadState label={L.loadError} onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-5">
            <Tile icon={IconTrendDown} label={L.lost} value={fmtMoney(data.total)} tone={data.total > 0 ? "danger" : "muted"} />
            <Tile
              icon={IconTrendUp}
              label={L.recovered}
              value={fmtMoney(data.recovered)}
              tone={data.recovered > 0 ? "success" : "muted"}
            />
            <Tile icon={IconWallet} label={L.net} value={fmtMoney(data.net)} tone={data.net > 0 ? "danger" : "muted"} />
            <Tile icon={IconBalance} label={L.owed} value={fmtMoney(data.owed)} tone={data.owed > 0 ? "warning" : "muted"} />
            <Tile
              icon={IconBlock}
              label={L.writtenOff}
              value={fmtMoney(data.written_off ?? 0)}
              tone={(data.written_off ?? 0) > 0 ? "warning" : "muted"}
            />
          </div>

          {/* **سطورُ الخسارة بأرقام صفحة الأرباح نفسِها** (قرارُ المالك ٢٠٢٦-١٠-١٥). */}
          {(data.lines ?? []).some((l) => l.amount !== 0) && (
            <section className="surface p-4">
              <h2 className="heading-card mb-3">{L.linesTitle}</h2>
              <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {(data.lines ?? [])
                  .filter((l) => l.amount !== 0)
                  .map((l) => (
                    <li
                      key={l.key}
                      className="flex items-center justify-between gap-3 rounded-control bg-field px-3 py-2 text-sm"
                    >
                      <span className="text-ink-muted">{LINE_LABEL[l.key] ?? l.key}</span>
                      <span className="font-bold tabular-nums text-ink">{fmtMoney(l.amount)}</span>
                    </li>
                  ))}
              </ul>
            </section>
          )}

          <CaseList rows={rows} empty={L.empty} emptyHint={L.emptyHint} noDisputeLabel={L.noDispute} />

          {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
          {data.count > data.per_page && (
            <div className="flex justify-center">
              <Pagination page={page} total={data.count} perPage={data.per_page} onChange={setPage} />
            </div>
          )}
        </>
      )}
    </PageContainer>
  );
}
