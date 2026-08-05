"use client";

/**
 * **النزاعات — مع أربعةٍ لا مع واحد.**
 *
 * # القرار الأوّل — ولماذا لا تُخصم آلياً
 *
 * **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»** (المالك،
 * ٢٠٢٦-٠٨-٠٣)
 *
 * السائقُ يُعوَّض **لحظتَها** فلا ينتظر نزاعاً ليُقبض له، **والمطالبةُ تُحسم
 * هنا.** و**الخصمُ قرارُ إنسانٍ بعد أن يسمع الطرف**: قد يكون العذرُ حقّاً —
 * انقطعت الكهرباء، أو جاءه الطلبُ ولم يُبلَّغ. **ومالٌ يخرج من محفظةٍ قبل أن
 * يُسأل صاحبُها نزاعٌ خُسر قبل أن يُفتح**: يبقى المالُ ويذهب الشريك.
 *
 * # والقرار الثاني — **أربعةُ أطرافٍ لا واحد**
 *
 * كان القسمُ «نزاعاتِ المتاجر» وحدَها، **لأنّ المطالبةَ كانت تسكن في صفّ إنذار
 * المتجر** — والسائقُ والزبونُ والمندوب لا إنذاراتِ لهم، **فلا مكانَ لنزاعٍ
 * معهم إطلاقاً.** فيُدار بالهاتف ويُنسى، **ولا يعرف أحدٌ كم لنا عند الناس
 * مجموعاً.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «قسمُ النزاعات يحوي تبويباً للمناديب والمتاجر
 * والسائقين والزبائن لنعرف منازعةَ المنصة مع من — لأنّه قسمٌ خاصٌّ بمنازعات
 * المنصة والطرف الآخر».)
 *
 * **والعددُ على التبويب** — فمن فتح القسم عرف أين العمل قبل أن ينقر.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Tabs,
  Badge,
  Button,
  Input,
  Select,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  StatGrid,
  StatCard,
  useLiveData,
  IconStore,
  IconWallet,
  IconBalance,
  IconAdd,
  IconOrder,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";
import { useAuth, hasRole } from "@/lib/auth";
import CreditPicker from "@/components/CreditPicker";

const m = getMessages(defaultLocale);
const C = m.admin.claims;
const FAIL_REASONS: Record<string, string> = m.common.failReasons;

type Party = "" | "merchant" | "driver" | "sales" | "customer";

interface Dispute {
  id: string;
  party_role: string;
  party_id: string;
  party_name: string;
  party_phone: string;
  reason: string;
  note: string;
  amount: number;
  status: "open" | "settled" | "waived";
  settlement: string | null;
  order_number: number | null;
  created_at: string;
}

interface DisputePage {
  disputes: Dispute[];
  total: number;
  open_counts: Record<string, number>;
}

function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

/** **سببٌ مصنَّفٌ يُترجَم، وحرٌّ يُعرض كما كُتب.** */
const reasonText = (r: string) => FAIL_REASONS[r] ?? r;

const PARTY_TABS: { key: Party; label: string }[] = [
  { key: "", label: C.parties.all },
  { key: "merchant", label: C.parties.merchant },
  { key: "driver", label: C.parties.driver },
  { key: "sales", label: C.parties.sales },
  { key: "customer", label: C.parties.customer },
];

export default function DisputesPage() {
  const { user } = useAuth();
  const canSettle = hasRole(user, "admin", "finance");
  const [party, setParty] = useState<Party>("");
  const [status, setStatus] = useState("open");
  const [acting, setActing] = useState<{ d: Dispute; charge: boolean } | null>(null);
  const [openNew, setOpenNew] = useState(false);

  const { data, reload } = useLiveData<DisputePage>(
    () => api(`/api/v1/admin/disputes?party=${party}&status=${status}`),
    ["dispute", "wallet"],
    [party, status],
  );

  const rows = data?.disputes ?? [];
  const counts = data?.open_counts ?? {};
  const totalOpen = Object.values(counts).reduce((a, b) => a + b, 0);

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconBalance}
        title={C.title}
        actions={
          canSettle ? (
            <Button onClick={() => setOpenNew(true)} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {C.openBtn}
            </Button>
          ) : undefined
        }
      />
      <p className="mb-4 text-sm text-ink-muted">{C.hint}</p>

      <StatGrid>
        <StatCard label={C.count} value={fmtNum(totalOpen)} icon={IconStore} />
        <StatCard
          label={`${C.total} (${m.common.currency})`}
          value={fmtNum(data?.total ?? 0)}
          icon={IconWallet}
        />
      </StatGrid>

      {/* **تبويبُ الطرف — والعددُ عليه.** «مع من نتنازع؟» سؤالُ القسم الأوّل. */}
      {/* **والعدّادُ من المكوّن نفسِه** — كان شارةً مكتوبةً بالحرف
          بلونٍ وحشوةٍ خاصّين بهذه الشاشة وحدَها. */}
      <Tabs
        className="mb-3 mt-4"
        items={PARTY_TABS.map((t) => ({
          key: t.key,
          label: t.label,
          count: t.key ? (counts[t.key] ?? 0) : totalOpen,
        }))}
        value={party}
        onChange={setParty}
      />

      <div className="mb-4 w-44">
        <Select value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="open">{C.statuses.open}</option>
          <option value="settled">{C.statuses.settled}</option>
          <option value="waived">{C.statuses.waived}</option>
          <option value="all">{C.statuses.all}</option>
        </Select>
      </div>

      {rows.length === 0 ? (
        <EmptyState icon={IconBalance} title={C.empty} />
      ) : (
        <ul className="space-y-2">
          {rows.map((d) => (
            <li
              key={d.id}
              className="flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-3"
            >
              <span className="min-w-0 flex-1">
                <span className="block font-bold">
                  {d.party_name || "—"}{" "}
                  <Badge variant="neutral">
                    {(C.parties as Record<string, string>)[d.party_role] ?? d.party_role}
                  </Badge>
                </span>
                <span className="block text-sm text-ink-muted">
                  {reasonText(d.reason)}
                  {d.note && ` · ${d.note}`}
                </span>
                <span dir="ltr" className="block text-xs text-ink-muted">
                  {d.party_phone} · {fmtDateTime(d.created_at)}
                </span>
              </span>

              {d.order_number != null && (
                <span dir="ltr" className="flex shrink-0 items-center gap-1 text-sm text-ink-muted">
                  <IconOrder size={14} />#{fmtRef(d.order_number)}
                </span>
              )}

              <span dir="ltr" className="shrink-0 text-lg font-bold tabular-nums text-warning">
                {fmtNum(d.amount)}
              </span>

              {d.status === "open" ? (
                canSettle && (
                  <span className="flex shrink-0 gap-2">
                    <Button onClick={() => setActing({ d, charge: true })}>{C.charge}</Button>
                    <Button variant="secondary" onClick={() => setActing({ d, charge: false })}>
                      {C.waive}
                    </Button>
                  </span>
                )
              ) : (
                <Badge variant={d.status === "settled" ? "success" : "neutral"}>
                  {d.status === "settled" ? C.settledBadge : C.waivedBadge}
                </Badge>
              )}
            </li>
          ))}
        </ul>
      )}

      {acting && (
        <SettleModal
          dispute={acting.d}
          charge={acting.charge}
          onClose={() => setActing(null)}
          onDone={() => {
            setActing(null);
            reload();
          }}
        />
      )}
      {openNew && (
        <NewDisputeModal
          onClose={() => setOpenNew(false)}
          onDone={() => {
            setOpenNew(false);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

/** الحسم — **خصماً أو إسقاطاً، وكلاهما بكلمة.** */
function SettleModal({
  dispute,
  charge,
  onClose,
  onDone,
}: {
  dispute: Dispute;
  charge: boolean;
  onClose: () => void;
  onDone: () => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!note.trim()) {
      setError(C.noteRequired);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/disputes/${dispute.id}/settle`, {
        method: "POST",
        body: JSON.stringify({
          settlement: charge ? "charged" : "waived",
          note: note.trim(),
        }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={charge ? C.chargeTitle : C.waiveTitle}>
      <form onSubmit={submit} className="space-y-3">
        <p className="text-sm text-ink-muted">{charge ? C.chargeHint : C.waiveHint}</p>
        <p className="rounded-control bg-page px-3 py-2 text-sm">
          {dispute.party_name} —{" "}
          <span dir="ltr" className="font-bold tabular-nums">
            {fmtNum(dispute.amount)} {m.common.currency}
          </span>
        </p>
        <Input
          id="settle-note"
          label={C.note}
          required
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" variant={charge ? "danger" : "primary"} disabled={busy}>
            {charge ? C.chargeConfirm : C.waiveConfirm}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

/**
 * فتحُ نزاعٍ بيد إنسان — **لما لا يُولد من طلب.**
 *
 * صندوقُ سائقٍ لم يُسلَّم، وعمولةُ مندوبٍ على متجرٍ لم يعمل: **لا واقعةَ في
 * المحرّك تُنتجهما**، فيُفتحان بيدٍ ويُسجَّل من فتحهما.
 */
function NewDisputeModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [picking, setPicking] = useState(false);
  const [who, setWho] = useState<AuthUser | null>(null);
  const [role, setRole] = useState("driver");
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const value = Number(amount);
    if (!who || !Number.isFinite(value) || value <= 0 || !reason.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/disputes", {
        method: "POST",
        body: JSON.stringify({
          party_role: role,
          party_id: who.id,
          amount: Math.round(value),
          reason: reason.trim(),
        }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  if (picking) {
    return (
      <CreditPicker
        onClose={() => setPicking(false)}
        onPick={(u) => {
          setWho(u);
          // **والدورُ يُقترح من أدواره** — ومن حمل دورين يصحّحه بنفسه.
          const guess = ["driver", "sales", "merchant", "customer"].find((r) =>
            u.roles.includes(r),
          );
          if (guess) setRole(guess);
          setPicking(false);
        }}
      />
    );
  }

  return (
    <Modal open onClose={onClose} title={C.openTitle}>
      <form onSubmit={submit} className="space-y-3">
        <div>
          <p className="mb-1 text-sm font-medium">{C.party}</p>
          <Button type="button" variant="secondary" onClick={() => setPicking(true)}>
            {who ? `${who.full_name} · ${who.phone}` : C.pickParty}
          </Button>
        </div>
        <Select value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="merchant">{C.parties.merchant}</option>
          <option value="driver">{C.parties.driver}</option>
          <option value="sales">{C.parties.sales}</option>
          <option value="customer">{C.parties.customer}</option>
        </Select>
        <Input
          id="dispute-amount"
          label={`${C.amount} (${m.common.currency})`}
          type="number"
          required
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <Input
          id="dispute-reason"
          label={C.reason}
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy || !who}>
            {C.openBtn}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
