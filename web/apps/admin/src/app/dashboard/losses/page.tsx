"use client";

/**
 * **خسائرُ المنصة** — من الدفتر لا من تقدير.
 *
 * قاعدةُ المالك: «نحسب الخسارة الفعلية فقط وليس الخسارة الافتراضية».
 *
 * **والفرقُ ليس لفظياً**: طلبٌ أُلغي قبل التحضير خسارتُه صفر — لم يُطبخ طعامٌ
 * ولم يقد سائق. **وشاشةٌ تعدّه خسارةً تجعل المنصةَ تبدو خاسرةً وهي لم تدفع
 * شيئاً**، فيُتّخذ قرارٌ على رقمٍ لا وجود له.
 *
 * فما يُعرض هنا **قيودُ مصروفٍ خرجت من الخزينة فعلاً**: بضاعةٌ لم يستردّها
 * متجر، وتعويضُ سائقٍ عن طلبٍ فشل.
 *
 * **والرصيدُ بجانبها** — خسارةٌ بلا ما يقابلها رقمٌ يُفزع بلا معنى.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  Input,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  useLiveData,
  IconWallet,
  IconDate,
  IconStatus,
  IconOrder,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const L = m.admin.losses;

interface Loss {
  order_number: number | null;
  amount: number;
  note: string;
  created_at: string;
}

/** تاريخُ اليوم بصيغة الاستعلام — بلا مناطق زمنية تُزحزح اليوم. */
function isoDay(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export default function LossesPage() {
  const today = new Date();
  const [to, setTo] = useState(isoDay(today));
  const [from, setFrom] = useState(
    isoDay(new Date(today.getFullYear(), today.getMonth(), today.getDate() - 29)),
  );
  const [view, setView] = useViewMode("losses");

  const { data } = useLiveData<{
    losses: Loss[];
    total: number;
    treasury_balance: number;
  }>(() => api(`/api/v1/admin/reports/losses?from=${from}&to=${to}`), ["wallet", "order"]);

  const columns: DataColumn<Loss>[] = [
    {
      id: "amount",
      header: L.amount,
      icon: <IconWallet />,
      cell: (x) => (
        <span dir="ltr" className="font-bold text-danger">
          {fmtNum(x.amount)}
        </span>
      ),
    },
    {
      id: "reason",
      header: L.reason,
      icon: <IconStatus />,
      cell: (x) => <span className="line-clamp-2">{x.note}</span>,
    },
    {
      id: "order",
      header: m.terms.order,
      icon: <IconOrder />,
      cell: (x) => (x.order_number === null ? "—" : <span dir="ltr">#{fmtRef(x.order_number)}</span>),
    },
    {
      id: "date",
      header: L.date,
      icon: <IconDate />,
      cell: (x) => (
        <span dir="ltr" className="text-xs text-ink-muted">
          {fmtDateTime(x.created_at)}
        </span>
      ),
    },
  ];

  return (
    <PageContainer>
      <PageHeader icon={IconWallet} title={L.title} subtitle={L.hint} />

      <div className="mb-4 flex flex-wrap items-end gap-2">
        <Input
          label={m.shared.statement.from}
          type="date"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
        />
        <Input label={m.shared.statement.to} type="date" value={to} onChange={(e) => setTo(e.target.value)} />
      </div>

      {!data ? (
        <LoadingState />
      ) : (
        <>
          <StatGrid>
            <StatCard
              label={L.total}
              value={fmtNum(data.total)}
              icon={IconWallet}
              tone={data.total > 0 ? "danger" : "default"}
            />
            {/* **رصيدُ الخزينة لا مجموعُ قيود**: هو الصافي بعد كلّ ما دخل
                وخرج، **ولا يحتاج جمعاً ثانياً يُخطئ.** */}
            <StatCard
              label={L.treasury}
              value={fmtNum(data.treasury_balance)}
              icon={IconWallet}
              tone={data.treasury_balance < 0 ? "danger" : "success"}
            />
          </StatGrid>

          {data.losses.length === 0 ? (
            <EmptyState icon={IconStatus} title={L.empty} />
          ) : (
            <>
              <div className="mb-2 flex justify-end">
                <ViewToggle
                  view={view}
                  onChange={setView}
                  tableLabel={m.common.viewTable}
                  cardsLabel={m.common.viewCards}
                />
              </div>
              <DataView
                items={data.losses}
                getKey={(x) => x.created_at + String(x.amount)}
                columns={columns}
                view={view}
                empty={L.empty}
              />
            </>
          )}
        </>
      )}
    </PageContainer>
  );
}
