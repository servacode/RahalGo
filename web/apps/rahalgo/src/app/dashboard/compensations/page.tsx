"use client";

/**
 * **تعويضاتٌ بانتظار الموافقة** — (قرارُ المالك ٢٠٢٦-١٠-٠٢)
 *
 * كان المحرّكُ يقيّد نصفَ أجر السائق **لحظةَ ضغطة «لدي مشكلة»** — بلا يد.
 * **وصار يكتب طلباً معلَّقاً بمبلغٍ مقترَح**، وهذه الشاشةُ هي اليدُ التي
 * تقضي فيه: موافقةٌ بمبلغٍ (يُعدَّل إن لزم) أو رفضٌ بسبب.
 *
 * **والموافقةُ هي البابُ اليدويُّ القائم نفسُه** (`compensate-driver`):
 * يقيّد ويُغلق الطلبَ المعلَّق في معاملةٍ واحدة، **ويحرس التكرار**
 * (`driver_already_compensated`). **والتأكيدُ بكلمة المرور يلتقطه العميلُ
 * المركزيّ** (`StepUpGate`) — لا شيءَ منه هنا.
 *
 * **والقراءةُ بـ`finance.read` والقرارُ بـ`finance.manage`** — كجدول
 * السياسة في المحرّك نفسِه.
 */

import { useState } from "react";
import Link from "next/link";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtRef,
  fmtDateTime,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Input,
  Textarea,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  DataView,
  Pagination,
  StatGrid,
  StatCard,
  ViewToggle,
  useViewMode,
  type DataColumn,
  useLiveData,
  IconWallet,
  IconDriver,
  IconOrder,
  IconStatus,
  IconDate,
  IconWarning,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const C = m.admin.compensations;
const FAIL_REASONS = m.common.failReasons as Record<string, string>;
const FAULTS = C.faults as Record<string, string>;
const ORDER_STATUS = m.orders.status as Record<string, string>;

interface Compensation {
  id: string;
  order_id: string;
  order_number: number;
  order_status: string;
  driver_id: string;
  driver_name: string;
  fault: string;
  fail_reason: string;
  suggested_amount: number;
  status: string;
  created_at: string;
}

export default function CompensationsPage() {
  const { can } = useAuth();
  // **والقرارُ مالٌ يخرج** — `finance.manage`، كحارس المحرّك.
  const canDecide = can("finance.manage");
  const [view, setView] = useViewMode("compensations");
  const [page, setPage] = useState(1);
  const [deciding, setDeciding] = useState<{ c: Compensation; approve: boolean } | null>(null);

  const { data, loading, error, reload } = useLiveData<{
    compensations: Compensation[];
    total: number;
    page: number;
    per_page: number;
  }>(
    () => api(`/api/v1/admin/compensations/pending?page=${page}`),
    // **الطلبُ المعلَّقُ يُكتب مع فشل الطلب ويُغلق مع قيد المحفظة.**
    ["order", "wallet"],
    [page],
  );

  if (loading && !data) return <LoadingState />;

  const rows = data?.compensations ?? [];

  const columns: DataColumn<Compensation>[] = [
    {
      id: "order",
      header: C.order,
      icon: <IconOrder />,
      primary: true,
      cell: (c) => (
        <Link
          href={`/dashboard/orders?q=${c.order_number}`}
          className="font-bold text-primary-dark hover:underline"
          dir="ltr"
        >
          #{fmtRef(c.order_number)}
        </Link>
      ),
    },
    {
      id: "driver",
      header: C.driver,
      icon: <IconDriver />,
      primary: true,
      cell: (c) => (
        <Link href={`/dashboard/users/${c.driver_id}`} className="text-primary-dark hover:underline">
          {c.driver_name}
        </Link>
      ),
    },
    {
      id: "reason",
      header: C.reason,
      icon: <IconStatus />,
      cell: (c) =>
        FAIL_REASONS[c.fail_reason] ?? (
          <span dir="ltr" className="text-xs text-ink-muted">
            {c.fail_reason || "—"}
          </span>
        ),
    },
    {
      id: "fault",
      header: C.fault,
      icon: <IconWarning />,
      cell: (c) => FAULTS[c.fault] ?? c.fault,
    },
    {
      id: "orderStatus",
      header: C.orderStatus,
      icon: <IconStatus />,
      cell: (c) => (
        <span className="text-sm text-ink-muted">{ORDER_STATUS[c.order_status] ?? c.order_status}</span>
      ),
    },
    {
      id: "suggested",
      header: C.suggested,
      icon: <IconWallet />,
      primary: true,
      cell: (c) => <Money value={c.suggested_amount} className="font-bold" />,
    },
    {
      id: "created",
      header: C.created,
      icon: <IconDate />,
      cell: (c) => (
        <span dir="ltr" className="text-xs text-ink-muted">
          {fmtDateTime(c.created_at)}
        </span>
      ),
    },
  ];

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconWallet}
        title={C.title}
        subtitle={C.hint}
        actions={
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        }
      />

      {/* **وفشلُ التحميل يُقال لا يُبتلَع** — وقائمةٌ قديمةٌ تبقى ظاهرةً تحته. */}
      {error && (
        <Alert className="mb-3">
          <span>{C.loadFailed}</span>{" "}
          <button type="button" onClick={() => reload()} className="font-bold underline">
            {m.common.retry}
          </button>
        </Alert>
      )}

      {data && (
        <StatGrid>
          <StatCard
            label={C.pendingCount}
            value={fmtNum(data.total)}
            icon={IconWallet}
            tone={data.total > 0 ? "accent" : "default"}
          />
        </StatGrid>
      )}

      {data && rows.length === 0 ? (
        <EmptyState icon={IconStatus} title={C.empty} tone="success" />
      ) : (
        rows.length > 0 && (
          <DataView
            items={rows}
            getKey={(c) => c.id}
            columns={columns}
            view={view}
            empty={C.empty}
            actions={
              canDecide
                ? (c) => (
                    <>
                      <Button onClick={() => setDeciding({ c, approve: true })}>{C.approve}</Button>
                      <Button variant="danger" onClick={() => setDeciding({ c, approve: false })}>
                        {C.reject}
                      </Button>
                    </>
                  )
                : undefined
            }
          />
        )
      )}

      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {deciding && (
        <DecideModal
          item={deciding.c}
          approve={deciding.approve}
          onClose={() => setDeciding(null)}
          onDone={() => {
            setDeciding(null);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

/**
 * **نافذةُ القرار** — موافقةٌ بمبلغٍ مقترَحٍ يُعدَّل، أو رفضٌ بسبب.
 *
 * **والملاحظةُ إلزاميّةٌ في الحالين**: المحرّكُ يردّ `validation` لموافقةٍ
 * بلا كلمة — مالٌ يخرج بتقدير موظّفٍ ولا يُراجَع بلا سبب.
 */
function DecideModal({
  item,
  approve,
  onClose,
  onDone,
}: {
  item: Compensation;
  approve: boolean;
  onClose: () => void;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState(String(item.suggested_amount > 0 ? item.suggested_amount : ""));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const value = Number(amount);
  const amountOk = Number.isInteger(value) && value > 0;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const text = note.trim();
    if (text === "") return;
    if (approve && !amountOk) {
      setError(C.amountInvalid);
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (approve) {
        await api(`/api/v1/admin/orders/${item.order_id}/compensate-driver`, {
          method: "POST",
          body: JSON.stringify({ amount: value, note: text }),
        });
      } else {
        await api(`/api/v1/admin/orders/${item.order_id}/compensation/reject`, {
          method: "POST",
          body: JSON.stringify({ note: text }),
        });
      }
      onDone();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={approve ? C.approveTitle : C.rejectTitle}>
      <form onSubmit={submit} className="space-y-4">
        <div className="rounded-control border border-line bg-field px-3 py-2 text-sm">
          <p className="font-bold">
            {item.driver_name}{" "}
            <span dir="ltr" className="font-normal text-ink-muted">
              #{fmtRef(item.order_number)}
            </span>
          </p>
          <p className="text-ink-muted">
            {FAIL_REASONS[item.fail_reason] ?? item.fail_reason} · {FAULTS[item.fault] ?? item.fault}
          </p>
          <p className="text-ink-muted">
            {C.suggested}: <Money value={item.suggested_amount} />
          </p>
        </div>
        {approve && (
          <>
            <Input
              id="comp-amount"
              label={`${C.amount} (${m.common.currency})`}
              type="number"
              inputMode="numeric"
              min={1}
              step={1}
              dir="ltr"
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              error={amount !== "" && !amountOk ? C.amountInvalid : undefined}
            />
            <p className="text-xs text-ink-muted">
              {C.amountHint.replace("{n}", fmtNum(item.suggested_amount))}
            </p>
            {item.fault === "merchant" && <Alert tone="info">{C.merchantClaimHint}</Alert>}
          </>
        )}
        <Textarea
          id="comp-note"
          label={approve ? C.approveNote : C.rejectNote}
          required
          maxLength={300}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={approve ? C.approveNotePlaceholder : C.rejectNotePlaceholder}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions
          submit
          onCancel={onClose}
          busy={busy}
          saveLabel={busy ? m.common.loading : approve ? C.approve : C.reject}
        />
      </form>
    </Modal>
  );
}
