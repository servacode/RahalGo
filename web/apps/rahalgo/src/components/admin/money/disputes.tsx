"use client";

/**
 * **النزاعات — مع أربعةٍ لا مع واحد.**
 *
 * # القرار الأوّل — ولماذا لا تُخصم آلياً
 *
 * **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»** (المالك،
 * ٢٠٢٦-٠٨-٠٣) و**كلُّ خصمٍ قرارُ إنسانٍ بعد أن يسمع الطرف** (٢٠٢٦-١٠-٠٤، البند ٥).
 *
 * # وقراراتُ ٢٠٢٦-١٠-٠٤ («الخسائر والنزاعات»)
 *
 * - **الماليّةُ ترى وتحسم، والدعمُ يرى ويفتح** — والأزرارُ من جدول المحرّك نفسِه
 *   (`useCanCall`) لا من قدرةٍ يظنّها الزرّ.
 * - **الحسمُ اقتراحٌ وموافقةُ شخصٍ آخر** — الاقتراحُ لا يحرّك مالاً.
 * - **رصيدٌ لا يكفي: يُخصم الموجودُ ويبقى الباقي مفتوحاً.**
 * - **«انسقط عنه قبل X مرّات» جنب الطرف** — من أعفى مرّةً يُسأل عن الثانية.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Confirm,
  Input,
  Select,
  Modal,
  PageContainer,
  PageHeader,
  Pagination,
  LoadingState,
  ReloadState,
  StatGrid,
  StatCard,
  useLiveData,
  IconStore,
  IconWallet,
  IconBalance,
  IconAdd,
  IconSearch,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import { CaseTable, CaseFilters, reasonText, type CaseRow } from "./caseTable";

const m = getMessages(defaultLocale);
const C = m.admin.claims;

type Party = "" | "merchant" | "driver" | "sales" | "customer";
type Status = "open" | "settled" | "waived" | "all";

interface Pending {
  id: string;
  action: "charge" | "waive";
  note: string;
  proposed_by: string;
  proposer_name: string;
  created_at: string;
}

interface Dispute {
  id: string;
  party_role: string;
  party_id: string;
  party_name: string;
  party_phone: string;
  reason: string;
  note: string;
  amount: number;
  recovered: number;
  open_amount: number;
  status: "open" | "settled" | "waived";
  settlement: string | null;
  order_id: string | null;
  order_number: number | null;
  waived_before: number;
  pending: Pending | null;
  created_at: string;
}

interface DisputePage {
  disputes: Dispute[];
  /** **مجموعُ ما بقي مفتوحاً بالطرف المختار** — لا عددُ الصفوف. */
  total: number;
  /** **عددُ المفتوح بالطرف المختار** — والكرتان بالترشيح نفسِه. */
  open_count: number;
  /** **عددُ الصفوف بالترشيح الحاليّ** — للترقيم. */
  count: number;
  page: number;
  per_page: number;
  open_counts: Record<string, number>;
}

const PARTIES: Party[] = ["", "merchant", "driver", "sales", "customer"];
const STATUSES: Status[] = ["open", "settled", "waived", "all"];

export function DisputesView() {
  const { user: me } = useAuth();
  const canCall = useCanCall();
  // **الأزرارُ بسياسة المحرّك نفسِها** — لا بقدرةٍ يخمّنها الزرّ (المشكلة ٣).
  const canOpen = canCall("POST", "/disputes");
  const canPropose = canCall("POST", "/disputes/{id}/propose");
  const canDecide = canCall("POST", "/dispute-resolutions/{id}/approve");

  const [party, setParty] = useState<Party>("");
  const [status, setStatus] = useState<Status>("open");
  const [page, setPage] = useState(1);
  const [proposing, setProposing] = useState<{ d: Dispute; charge: boolean } | null>(null);
  const [deciding, setDeciding] = useState<{ d: Dispute; approve: boolean } | null>(null);
  const [openNew, setOpenNew] = useState(false);
  const [notice, setNotice] = useState("");

  const { data, error, reload } = useLiveData<DisputePage>(
    () => api(`/api/v1/admin/disputes?party=${party}&status=${status}&page=${page}`),
    ["dispute", "wallet"],
    [party, status, page],
  );

  const partyLabels = C.parties as Record<string, string>;
  const statusLabels = C.statuses as Record<string, string>;
  const emptyBy = C.emptyBy as Record<string, string>;
  const counts = data?.open_counts ?? {};
  const allOpen = Object.values(counts).reduce((a, b) => a + b, 0);

  const byId = new Map((data?.disputes ?? []).map((d) => [d.id, d]));
  const rows: CaseRow[] = (data?.disputes ?? []).map((d) => ({
    key: d.id,
    amount: d.status === "open" ? d.open_amount : d.amount,
    recovered: d.recovered > 0 && d.status === "open" ? d.recovered : undefined,
    partyRole: d.party_role,
    partyId: d.party_id,
    partyName: d.party_name,
    waivedBefore: d.waived_before,
    reason: d.note ? `${reasonText(d.reason)} · ${d.note}` : d.reason,
    orderId: d.order_id,
    orderNumber: d.order_number,
    date: d.created_at,
    status: d.status,
    statusNote: d.pending
      ? `${d.pending.action === "charge" ? C.pendingCharge : C.pendingWaive} — ${C.proposedBy.replace(
          "{name}",
          d.pending.proposer_name,
        )}`
      : undefined,
  }));

  function actions(row: CaseRow) {
    const d = byId.get(row.key);
    if (!d || d.status !== "open") return null;
    if (d.pending) {
      if (!canDecide) return null;
      return (
        <span className="flex shrink-0 gap-1.5">
          {me?.id !== d.pending.proposed_by && (
            <Button className="!px-2.5" onClick={() => setDeciding({ d, approve: true })}>
              {C.approve}
            </Button>
          )}
          <Button variant="secondary" className="!px-2.5" onClick={() => setDeciding({ d, approve: false })}>
            {C.reject}
          </Button>
        </span>
      );
    }
    if (!canPropose) return null;
    return (
      <span className="flex shrink-0 gap-1.5">
        <Button className="!px-2.5" onClick={() => setProposing({ d, charge: true })}>
          {C.charge}
        </Button>
        <Button variant="secondary" className="!px-2.5" onClick={() => setProposing({ d, charge: false })}>
          {C.waive}
        </Button>
      </span>
    );
  }

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconBalance}
        title={C.title}
        actions={
          canOpen ? (
            <Button onClick={() => setOpenNew(true)} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {C.openBtn}
            </Button>
          ) : undefined
        }
      />
      <p className="mb-4 text-sm text-ink-muted">{C.hint}</p>

      {/* **شريطُ الفلاتر الواحد** — الطرفُ بعدّه والحالة، لا تبويباتٌ فوق تبويبات. */}
      <CaseFilters>
        <Select
          label={C.partyFilter}
          value={party}
          onChange={(e) => {
            setPage(1);
            setParty(e.target.value as Party);
          }}
        >
          {PARTIES.map((p) => (
            <option key={p} value={p}>
              {`${partyLabels[p || "all"]} (${fmtNum(p ? (counts[p] ?? 0) : allOpen)})`}
            </option>
          ))}
        </Select>
        <Select
          label={C.statusFilter}
          value={status}
          onChange={(e) => {
            setPage(1);
            setStatus(e.target.value as Status);
          }}
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {statusLabels[s]}
            </option>
          ))}
        </Select>
      </CaseFilters>

      {notice && (
        <div className="mb-3">
          <Alert tone="info">{notice}</Alert>
        </div>
      )}

      {/* **تحميلٌ وخطأٌ غيرُ الفراغ** — كان الفشلُ يقول «لا نزاعات مفتوحة» وأصفاراً. */}
      {error ? (
        <ReloadState label={C.loadError} onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <StatGrid>
            <StatCard label={C.openCount} value={fmtNum(data.open_count)} icon={IconStore} />
            <StatCard label={`${C.total} (${m.common.currency})`} value={fmtNum(data.total)} icon={IconWallet} />
          </StatGrid>
          <div className="mt-4">
            <CaseTable screen="disputes" rows={rows} empty={emptyBy[status] ?? C.empty} actions={actions} />
          </div>
          {data.count > data.per_page && (
            <div className="mt-4 flex justify-center">
              <Pagination page={page} total={data.count} perPage={data.per_page} onChange={setPage} />
            </div>
          )}
        </>
      )}

      {proposing && (
        <ProposeModal
          dispute={proposing.d}
          charge={proposing.charge}
          onClose={() => setProposing(null)}
          onDone={() => {
            setProposing(null);
            reload();
          }}
        />
      )}
      {deciding && (
        <DecideConfirm
          dispute={deciding.d}
          approve={deciding.approve}
          onClose={() => setDeciding(null)}
          onDone={(msg) => {
            setDeciding(null);
            setNotice(msg);
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

/** **اقتراحُ الحسم** — خصماً أو إسقاطاً، بكلمة. ولا مالَ يتحرّك هنا. */
function ProposeModal({
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
    if (busy) return;
    if (!note.trim()) {
      setError(C.noteRequired);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/disputes/${dispute.id}/propose`, {
        method: "POST",
        body: JSON.stringify({ action: charge ? "charge" : "waive", note: note.trim() }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={charge ? C.proposeChargeTitle : C.proposeWaiveTitle}>
      <form onSubmit={submit} className="space-y-3">
        <p className="text-sm text-ink-muted">{charge ? C.chargeHint : C.waiveHint}</p>
        <p className="text-sm text-ink-muted">{C.proposeHint}</p>
        <p className="rounded-control bg-field px-3 py-2 text-sm">
          {dispute.party_name} — <span dir="ltr" className="font-bold">{fmtMoney(dispute.open_amount)}</span>
        </p>
        <Input id="propose-note" label={C.note} required value={note} onChange={(e) => setNote(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={C.proposeConfirm} />
      </form>
    </Modal>
  );
}

/** **الموافقةُ أو الرفض** — من غير المقترِح، والمحرّكُ يحرس ذلك على كلّ حال. */
function DecideConfirm({
  dispute,
  approve,
  onClose,
  onDone,
}: {
  dispute: Dispute;
  approve: boolean;
  onClose: () => void;
  onDone: (msg: string) => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const pending = dispute.pending;
  if (!pending) return null;

  async function decide() {
    if (busy || !pending) return;
    setBusy(true);
    setError("");
    try {
      const r = await api<{ charged: number | null; remaining: number }>(
        `/api/v1/admin/dispute-resolutions/${pending.id}/${approve ? "approve" : "reject"}`,
        { method: "POST", body: JSON.stringify({ note: note.trim() }) },
      );
      onDone(
        approve && pending.action === "charge" && r && r.remaining > 0
          ? C.chargedPartial
              .replace("{charged}", fmtMoney(r.charged ?? 0))
              .replace("{remaining}", fmtMoney(r.remaining))
          : "",
      );
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  const body = approve
    ? (pending.action === "charge" ? C.approveChargeBody : C.approveWaiveBody)
        .replace("{party}", dispute.party_name)
        .replace("{amount}", fmtMoney(dispute.open_amount))
    : null;

  return (
    <Confirm
      open
      tone={approve ? "primary" : "danger"}
      title={approve ? C.approveTitle : C.rejectTitle}
      body={
        <div className="space-y-2">
          {body && <p>{body}</p>}
          <p className="text-sm text-ink-muted">
            {C.proposedBy.replace("{name}", pending.proposer_name)}: {pending.note}
          </p>
          {!approve && (
            <Input id="dr-reject-note" label={C.rejectNote} value={note} onChange={(e) => setNote(e.target.value)} />
          )}
          {error && <Alert>{error}</Alert>}
        </div>
      }
      confirmLabel={approve ? C.approve : C.reject}
      busy={busy}
      onConfirm={() => void decide()}
      onCancel={onClose}
    />
  );
}

interface PartyOption {
  id: string;
  name: string;
  party_phone: string;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `d-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

/**
 * فتحُ نزاعٍ بيد إنسان — **لما لا يُولد من طلب.**
 *
 * **الدورُ أوّلاً ثمّ الطرفُ ممّن يحمله** (المشكلتان ١ و١٥): المتجرُ يُختار بمعرّف
 * المتجر لا بمعرّف صاحبه، **والسائقُ لا يُسجَّل زبوناً.** **وضغطتان لا تفتحان
 * نزاعين** (المشكلة ١٤): الزرُّ يُعطَّل وقتَ الإرسال، ومفتاحُ عدمِ التكرار يُرسَل.
 */
function NewDisputeModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [role, setRole] = useState<Exclude<Party, "">>("merchant");
  const [query, setQuery] = useState("");
  const [options, setOptions] = useState<PartyOption[] | null>(null);
  const [who, setWho] = useState<PartyOption | null>(null);
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [idemKey] = useState(newKey);
  const partyOne = C.partyOne as Record<string, string>;

  useEffect(() => {
    let alive = true;
    const t = setTimeout(async () => {
      try {
        const q = new URLSearchParams({ role, q: query.trim() });
        const r = await api<{ parties: PartyOption[] }>(`/api/v1/admin/disputes/parties?${q}`);
        if (alive) setOptions(r?.parties ?? []);
      } catch (err) {
        if (alive) {
          setOptions([]);
          setError(errorText(err));
        }
      }
    }, 250);
    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [role, query]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (busy) return;
    const value = Number(amount);
    if (!who || !Number.isFinite(value) || value <= 0 || !reason.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/disputes", {
        method: "POST",
        headers: { "Idempotency-Key": idemKey },
        body: JSON.stringify({ party_role: role, party_id: who.id, amount: Math.round(value), reason: reason.trim() }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={C.openTitle}>
      <form onSubmit={submit} className="space-y-3">
        <Select
          label={C.pickRole}
          value={role}
          onChange={(e) => {
            setRole(e.target.value as Exclude<Party, "">);
            setWho(null);
            setOptions(null);
          }}
        >
          {(["merchant", "driver", "sales", "customer"] as const).map((r) => (
            <option key={r} value={r}>
              {partyOne[r]}
            </option>
          ))}
        </Select>
        {who ? (
          <div className="flex items-center justify-between gap-2 rounded-control bg-field px-3 py-2 text-sm">
            <span>
              {C.pickedParty}: <b>{who.name}</b>
            </span>
            <Button type="button" variant="secondary" className="!px-2.5" onClick={() => setWho(null)}>
              {C.pickParty}
            </Button>
          </div>
        ) : (
          <div>
            <Input
              id="dispute-party-search"
              icon={<IconSearch />}
              placeholder={C.searchParty}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            <div className="mt-2 max-h-56 overflow-y-auto">
              {options === null ? (
                <LoadingState variant="inline" />
              ) : options.length === 0 ? (
                <p className="py-2 text-sm text-ink-muted">{C.noParties}</p>
              ) : (
                <ul className="divide-y divide-line">
                  {options.map((o) => (
                    <li key={o.id}>
                      <button
                        type="button"
                        className="flex w-full items-center justify-between gap-2 px-2 py-2 text-start text-sm hover:bg-field"
                        onClick={() => setWho(o)}
                      >
                        <span className="font-medium">{o.name}</span>
                        <span dir="ltr" className="text-xs text-ink-muted">
                          {o.party_phone}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )}
        <Input
          id="dispute-amount"
          label={`${C.amount} (${m.common.currency})`}
          type="number"
          required
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <Input id="dispute-reason" label={C.reason} required value={reason} onChange={(e) => setReason(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={C.openBtn} />
      </form>
    </Modal>
  );
}
