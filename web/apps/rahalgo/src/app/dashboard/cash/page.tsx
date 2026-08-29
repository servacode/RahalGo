"use client";

/**
 * **أموالٌ لم تُستلم** — ما في أيدي السائقين ولم يبلغ المكتبَ بعد.
 *
 * # لماذا شاشةٌ قائمةٌ بذاتها
 *
 * النقدُ الذي يقبضه السائقُ **مالُ المنصة يحمله**، لا مالُه. وكان لا يُرى
 * مجموعاً في مكان: **من أراد أن يعرف كم في الشارع فتح كشفَ كلِّ سائقٍ على
 * حدة** — فلا يفعل، فلا يعرف.
 *
 * **ومالٌ لا يُرى مجموعاً لا يُطالَب به**: سائقٌ يحمل مئتي ألفٍ منذ ثلاثة أيام
 * لا يلفت أحداً، **وسائقان يفعلان ذلك يجعلان نصفَ يومٍ من دخل المنصة خارجها.**
 *
 * # والقِدَمُ هو الإشارة لا المقدار
 *
 * خمسون ألفاً قُبضت قبل ساعة **عملٌ يجري**، وخمسون ألفاً منذ أسبوعٍ **مسألةٌ
 * أخرى**. فالعمودُ الذي يُنظر إليه أوّلاً هو «منذ متى» لا «كم».
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  StatGrid,
  StatCard,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  useLiveData,
  IconWallet,
  IconUser,
  IconDate,
  IconStatus,
  FormActions,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth, hasRole } from "@/lib/auth";

const m = getMessages(defaultLocale);
const C = m.admin.cashOutstanding;

interface Holder {
  driver_id: string;
  name: string;
  phone: string;
  held: number;
  oldest_at: string | null;
  on_shift: boolean;
}

/** أيامٌ مضت على أقدم قبضٍ لم يُسوَّ — و`0` إن لا تاريخ. */
function daysHeld(iso: string | null): number {
  if (!iso) return 0;
  return Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);
}

export default function CashOutstandingPage() {
  const { user } = useAuth();
  const canSettle = hasRole(user, "admin") || hasRole(user, "finance");
  const [view, setView] = useViewMode("cash-outstanding");
  const [target, setTarget] = useState<Holder | null>(null);

  const { data, reload } = useLiveData<{
    holders: Holder[];
    total: number;
    limit: number;
  }>(() => api("/api/v1/admin/cash/outstanding"), ["wallet", "order"]);

  if (!data) return <LoadingState />;
  const holders = data.holders ?? [];

  const columns: DataColumn<Holder>[] = [
    {
      id: "driver",
      header: m.terms.driver,
      icon: <IconUser />,
      cell: (h) => (
        <span className="flex flex-col">
          <span className="font-medium">{h.name || h.phone}</span>
          <span className="text-2xs text-ink-muted" dir="ltr">
            {h.phone}
          </span>
        </span>
      ),
    },
    {
      id: "held",
      header: C.held,
      icon: <IconWallet />,
      cell: (h) => (
        <span
          dir="ltr"
          className={`font-bold ${h.held >= data.limit ? "text-danger" : "text-warning"}`}
        >
          {fmtNum(h.held)}
        </span>
      ),
    },
    {
      id: "age",
      header: C.age,
      icon: <IconDate />,
      // **القِدَمُ يُلوَّن لا المقدار**: يومان عملٌ يجري، وأسبوعٌ مسألةٌ أخرى.
      cell: (h) => {
        const d = daysHeld(h.oldest_at);
        return (
          <Badge variant={d >= 3 ? "danger" : d >= 1 ? "warning" : "neutral"}>
            {C.days.replace("{n}", fmtNum(d))}
          </Badge>
        );
      },
    },
    {
      id: "shift",
      header: C.shift,
      cell: (h) => (
        <Badge variant={h.on_shift ? "success" : "neutral"}>
          {h.on_shift ? C.onShift : C.offShift}
        </Badge>
      ),
    },
    {
      id: "act",
      header: "",
      cell: (h) =>
        canSettle ? (
          <Button variant="secondary" onClick={() => setTarget(h)}>
            {C.receive}
          </Button>
        ) : null,
    },
  ];

  return (
    <PageContainer>
      <PageHeader icon={IconWallet} title={C.title} subtitle={C.hint} />

      <StatGrid>
        <StatCard
          label={C.total}
          value={fmtNum(data.total)}
          icon={IconWallet}
          tone={data.total > 0 ? "danger" : "default"}
        />
        <StatCard label={C.holders} value={fmtNum(holders.length)} icon={IconUser} />
      </StatGrid>

      {holders.length === 0 ? (
        <EmptyState icon={IconStatus} title={C.empty} />
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
                items={holders}
                getKey={(h) => h.driver_id}
                columns={columns}
                view={view}
                empty={C.empty}
              />
        </>
      )}

      {target && (
        <ReceiveModal
          holder={target}
          onClose={() => setTarget(null)}
          onDone={() => {
            setTarget(null);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

/** استلامُ نقدٍ من سائق — بمبلغٍ يُكتب، وافتراضُه كلُّ ما بذمّته. */
function ReceiveModal({
  holder,
  onClose,
  onDone,
}: {
  holder: Holder;
  onClose: () => void;
  onDone: () => void;
}) {
  // **الافتراضُ كلُّ ما بذمّته** — وهو الغالب. ومن سلّم جزءاً عدّل الرقم،
  // **ومن سلّم كلَّه لا يُكلَّف كتابةَ ما نعرفه.**
  const [amount, setAmount] = useState(String(holder.held));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    const n = Number(amount);
    if (!Number.isFinite(n) || n <= 0 || n > holder.held) return setError(C.badAmount);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/drivers/${holder.driver_id}/settle`, {
        method: "POST",
        body: JSON.stringify({ amount: n, note: note.trim() }),
      });
      onDone();
    } catch (err) {
      const key = err instanceof ApiError ? (err.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal open title={C.receiveFrom.replace("{n}", holder.name || holder.phone)} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">
          {C.held}: <span dir="ltr">{fmtNum(holder.held)}</span>
          {holder.oldest_at && (
            <>
              {" · "}
              <span dir="ltr">{fmtDateTime(holder.oldest_at)}</span>
            </>
          )}
        </p>
        <Input
          label={C.amount}
          type="number"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <Input label={C.note} value={note} onChange={(e) => setNote(e.target.value)} />
        {error && <p className="text-sm text-danger">{error}</p>}
        <FormActions onSave={submit} onCancel={onClose} busy={busy} saveLabel={C.confirm} />
      </div>
    </Modal>
  );
}
