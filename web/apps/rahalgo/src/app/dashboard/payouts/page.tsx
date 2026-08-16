"use client";

/** طلبات سحب الرصيد — المالية تصرف أو ترفض، والقيد يُسجَّل في دفتر المحفظة. */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Input,
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
  IconUser,
  IconStatus,
  IconDate,
  IconWarning,
  IconAdd,
  Money,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";
import { useAuth, hasRole } from "@/lib/auth";
import WalletModal from "@/components/admin/WalletModal";
import CreditPicker from "@/components/admin/CreditPicker";

const m = getMessages(defaultLocale);
const P = m.shared.payout;

interface Payout {
  id: string;
  user_id: string;
  user_name: string;
  user_phone: string;
  amount: number;
  status: "pending" | "paid" | "rejected";
  note: string;
  decision: string;
  balance: number;
  created_at: string;
}

const VARIANT: Record<Payout["status"], "warning" | "success" | "danger"> = {
  pending: "warning",
  paid: "success",
  rejected: "danger",
};

function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function PayoutsPage() {
  const { user } = useAuth();
  const canDecide = hasRole(user, "admin", "finance");
  /** **والشحنُ اليدويُّ للأدمن والمالية** — كحارس الخادم نفسِه. */
  const canCredit = canDecide;
  const [status, setStatus] = useState("");
  const [deciding, setDeciding] = useState<{ p: Payout; approve: boolean } | null>(null);
  /** **إضافةُ الرصيد** — اختيارٌ ثمّ شحن. */
  const [creditOpen, setCreditOpen] = useState(false);
  const [creditUser, setCreditUser] = useState<AuthUser | null>(null);
  const [view, setView] = useViewMode("payouts");
  /** **وصفحةٌ محدودةٌ بعدّ** — (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦).

      **كانت مئتين صامتةً**: من رشّح «مدفوع» ليراجع ما صُرف **يرى آخرَ
      مئتين ويظنّها كلَّ ما دُفع** — **ومالٌ خرج ومراجعتُه ناقصةً أسوأُ
      من عدمها.** */
  const [page, setPage] = useState(1);

  const { data, loading, reload } = useLiveData<{
    payouts: Payout[];
    total: number;
    per_page: number;
    /** **كم يُنتظر صرفُه الآن** — للمعلَّق وحدَه ولا يتبع الترشيح. */
    pending_total: number;
  }>(
    () => api(`/api/v1/admin/payouts?page=${page}${status ? `&status=${status}` : ""}`),
    ["wallet"],
    // **والحالةُ تُعيد الجلب** — وكانت تُضيء ولا تُنادي الشبكة، **ولم يشتكِ
    // منه أحدٌ بعد**: من رأى القائمةَ لا تتغيّر ظنّ أن لا طلباتٍ في تلك الحالة.
    [status, page],
  );

  if (loading) return <LoadingState />;
  const rows = data?.payouts ?? [];

  const columns: DataColumn<Payout>[] = [
    {
      id: "who",
      header: m.terms.name,
      icon: <IconUser />,
      primary: true,
      cell: (p) => (
        <span>
          {p.user_name}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {p.user_phone}
          </span>
        </span>
      ),
    },
    {
      id: "amount",
      header: P.amount,
      icon: <IconWallet />,
      primary: true,
      cell: (p) => (
        <span className="font-bold text-primary-dark" dir="ltr">
          {fmtNum(p.amount)}
        </span>
      ),
    },
    {
      id: "balance",
      header: m.terms.walletBalance,
      icon: <IconWallet />,
      cell: (p) => <span dir="ltr">{fmtNum(p.balance)}</span>,
    },
    {
      id: "status",
      header: m.terms.status,
      icon: <IconStatus />,
      cell: (p) => <Badge variant={VARIANT[p.status]}>{P.status[p.status]}</Badge>,
    },
    {
      id: "note",
      header: P.note,
      cell: (p) => <span className="text-sm text-ink-muted">{p.decision || p.note || "—"}</span>,
    },
    {
      id: "date",
      header: m.terms.date,
      icon: <IconDate />,
      cell: (p) => <span dir="ltr">{fmtDateTime(p.created_at)}</span>,
    },
  ];

  const filters = [
    { id: "", label: m.terms.allStatuses },
    { id: "pending", label: P.status.pending },
    { id: "paid", label: P.status.paid },
    { id: "rejected", label: P.status.rejected },
  ];

  return (
    <PageContainer width="full">
      {/* **والقسمُ يخرج مالاً ويُدخله** — لا يخرجه وحدَه.

          كان اسمُه «السحوبات» وفعلُه واحد: **البتُّ في طلبٍ يتقدّم به صاحبُه.**
          **وإضافةُ الرصيد اليدويةُ مبنيّةٌ منذ البداية** (`POST /users/{id}/wallet`)
          **ومخبوءةٌ في صفحة حسابٍ لا يفتحها من يفكّر في المال.**

          ومن أراد أن يشحن محفظةَ زبونٍ نقداً، أو يعوّض سائقاً خارج طلب، أو
          يسوّي حساباً مع متجر — **فتح ملفَّ كلٍّ منهم على حدة.** والفعلُ واحدٌ
          والموضعُ يجب أن يكون واحداً.

          (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «سحبٌ وإضافةُ رصيد — هذا القسم وليس فقط
          سحب، فيمكن إضافةُ رصيدٍ للمناديب أو الزبائن أو المتاجر بشكلٍ يدويّ
          وللسائقين».) */}
      <PageHeader
        icon={IconWallet}
        title={P.titleWithCredit}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            {canCredit && (
              <Button onClick={() => setCreditOpen(true)} className="flex items-center gap-1.5">
                <IconAdd size={16} />
                {P.creditBtn}
              </Button>
            )}
            <div className="flex rounded-control border border-line bg-field p-1">
              {filters.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setStatus(f.id)}
                  className={`rounded-[7px] px-3 py-1.5 text-sm transition-colors ${
                    status === f.id
                      ? "bg-surface font-medium text-primary-dark elev-1"
                      : "text-ink-muted hover:text-ink"
                  }`}
                >
                  {f.label}
                </button>
              ))}
            </div>
            <ViewToggle
              view={view}
              onChange={setView}
              tableLabel={m.common.viewTable}
              cardsLabel={m.common.viewCards}
            />
          </div>
        }
      />

      {/* ══════════════════════════════════════════════════════════════
          **وكم يُنتظر صرفُه الآن**
          ══════════════════════════════════════════════════════════════

          (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)

          **وكلُّ شاشةِ مالٍ في المنصّة تقول مجموعَها**: الخزينةُ رصيدَها،
          والخسائرُ مجموعَها، والنزاعاتُ «كم لنا عند الناس»، وأموالٌ لم
          تُستلم ما في الشارع. **وهذه وحدَها لا تقول كم عليها أن تدفع.**

          **ومالٌ لا يُرى مجموعاً لا يُخطَّط له.**

          **ولا يتبع الترشيح** — سؤالُه «كم عليّ الآن؟»، **ومجموعٌ يتبع
          مُرشِّحاً يقول صفراً لمن يقرأ المرفوض** وهو لا يخصّه. */}
      <StatGrid>
        <StatCard
          label={`${P.pendingTotal} (${m.common.currency})`}
          value={fmtNum(data?.pending_total ?? 0)}
          icon={IconWallet}
          tone={(data?.pending_total ?? 0) > 0 ? "accent" : "default"}
        />
      </StatGrid>

      {rows.length === 0 ? (
        <EmptyState icon={IconWallet} title={P.empty} />
      ) : (
        <DataView
          items={rows}
          getKey={(p) => p.id}
          columns={columns}
          view={view}
          empty={P.empty}
          actions={
            canDecide
              ? (p) =>
                  p.status === "pending" ? (
                    <>
                      <Button onClick={() => setDeciding({ p, approve: true })}>
                        {P.status.paid}
                      </Button>
                      <Button variant="danger" onClick={() => setDeciding({ p, approve: false })}>
                        {P.status.rejected}
                      </Button>
                    </>
                  ) : null
              : undefined
          }
        />
      )}

      {creditOpen && (
        <CreditPicker
          onClose={() => setCreditOpen(false)}
          onPick={(u) => {
            setCreditOpen(false);
            setCreditUser(u);
          }}
        />
      )}
      {creditUser && (
        <WalletModal
          user={creditUser}
          isAdmin
          onClose={() => {
            setCreditUser(null);
            reload();
          }}
        />
      )}

      {/* **ولا يظهر لصفحةٍ واحدة** — عنصرٌ لا يفعل شيئاً يزاحم ما يفعل. */}
      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {deciding && (
        <DecideModal
          payout={deciding.p}
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

function DecideModal({
  payout,
  approve,
  onClose,
  onDone,
}: {
  payout: Payout;
  approve: boolean;
  onClose: () => void;
  onDone: () => void;
}) {
  const [decision, setDecision] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/payouts/${payout.id}/decide`, {
        method: "POST",
        body: JSON.stringify({ status: approve ? "paid" : "rejected", decision }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={approve ? P.status.paid : P.status.rejected}>
      <form onSubmit={submit} className="space-y-4">
        <div className="rounded-control border border-line bg-field px-3 py-2 text-sm">
          <p className="font-bold">{payout.user_name}</p>
          <p className="text-ink-muted">
            <Money value={payout.amount} />
          </p>
        </div>
        <Input
          id="decision"
          label={P.note}
          required={!approve}
          value={decision}
          onChange={(e) => setDecision(e.target.value)}
          placeholder={P.notePlaceholder}
        />
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" variant={approve ? "primary" : "danger"} disabled={busy}>
            {busy ? m.common.loading : m.common.confirm}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
