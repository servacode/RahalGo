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
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  PageContainer,
  Pagination,
  PageHeader,
  LoadingState,
  ReloadState,
  StatGrid,
  StatCard,
  Input,
  useLiveData,
  IconWallet,
  IconBalance,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { CaseTable, CaseFilters, type CaseRow, type CaseStatus } from "./caseTable";

const m = getMessages(defaultLocale);
const L = m.admin.losses;

interface Loss {
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
  /** **ديونٌ شُطبت في المدّة** — من طلبات الشطب الموافَق عليها، خارجَ المجموع (لا قيدَ لها). */
  written_off?: number;
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
    reason: x.note,
    orderId: x.order_id,
    orderNumber: x.order_number,
    date: x.created_at,
    status: x.dispute_status,
  }));

  return (
    <PageContainer>
      <PageHeader icon={IconWallet} title={L.title} subtitle={L.hint} />

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
          <StatGrid>
            <StatCard label={L.lost} value={fmtNum(data.total)} icon={IconWallet} tone={data.total > 0 ? "danger" : "muted"} />
            <StatCard
              label={L.recovered}
              value={fmtNum(data.recovered)}
              icon={IconBalance}
              tone={data.recovered > 0 ? "success" : "muted"}
            />
            <StatCard
              label={L.net}
              value={fmtNum(data.net)}
              icon={IconWallet}
              tone={data.net > 0 ? "danger" : "muted"}
              emphasis
            />
            <StatCard label={L.owed} value={fmtNum(data.owed)} icon={IconBalance} tone={data.owed > 0 ? "warning" : "muted"} />
            <StatCard
              label={L.writtenOff}
              value={fmtNum(data.written_off ?? 0)}
              icon={IconBalance}
              tone={(data.written_off ?? 0) > 0 ? "warning" : "muted"}
            />
          </StatGrid>

          <div className="mt-4">
            <CaseTable screen="losses" rows={rows} empty={L.empty} noDisputeLabel={L.noDispute} />
          </div>

          {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
          {data.count > data.per_page && (
            <div className="mt-4 flex justify-center">
              <Pagination page={page} total={data.count} perPage={data.per_page} onChange={setPage} />
            </div>
          )}
        </>
      )}
    </PageContainer>
  );
}
