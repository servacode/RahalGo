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
  Badge,
  Button,
  Chips,
  Confirm,
  Input,
  Modal,
  PageContainer,
  Pagination,
  LoadingState,
  ReloadState,
  useLiveData,
  IconBalance,
  IconAdd,
  IconSearch,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { ApproveButton, RejectButton } from "@/components/admin/WalletRequests";
import { CaseList, reasonText, type CaseRow } from "./caseTable";

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

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — أزرارٌ بعرض البطاقة: الموافقةُ خضراء والرفضُ بإطارٍ أحمر.
  function actions(row: CaseRow) {
    const d = byId.get(row.key);
    if (!d || d.status !== "open") return null;
    if (d.pending) {
      if (!canDecide) return null;
      return (
        <>
          <ApproveButton label={C.approve} onClick={() => setDeciding({ d, approve: true })} />
          <RejectButton label={C.reject} onClick={() => setDeciding({ d, approve: false })} />
        </>
      );
    }
    if (!canPropose) return null;
    return (
      <>
        <Button className="w-full" onClick={() => setProposing({ d, charge: true })}>
          {C.charge}
        </Button>
        <Button variant="secondary" className="w-full" onClick={() => setProposing({ d, charge: false })}>
          {C.waive}
        </Button>
      </>
    );
  }

  return (
    <PageContainer width="full">
      {/* **رأسٌ واحدٌ كرأس الشكاوى** (فحصُ ٢٠٢٦-١٠-٢٧) — العنوانُ وعددُ المفتوح ومجموعُه وزرُّ «افتح نزاع»
          بسطر، وتحته الطرفُ والحالةُ أزرارٌ بأعدادها بدل القوائم المنسدلة. **ولا مربّعاتٌ لا تُضغط.** */}
      <section className="surface overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 p-4">
          <span
            className={`inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-full ${
              (data?.open_count ?? 0) > 0 ? "bg-warning-solid text-on-solid" : "bg-accent text-on-bright"
            }`}
          >
            <IconBalance size={22} />
          </span>
          <div className="min-w-0 flex-1">
            <h2 className="heading-card flex flex-wrap items-center gap-2">
              {C.title}
              {data && (
                <button
                  type="button"
                  onClick={() => {
                    setPage(1);
                    setStatus("open");
                  }}
                >
                  <Badge variant={data.open_count > 0 ? "warning" : "neutral"}>
                    {C.headCount.replace("{n}", fmtNum(data.open_count))}
                  </Badge>
                </button>
              )}
            </h2>
            <p className="text-sm text-ink-muted">{C.hint}</p>
          </div>
          {data && (
            <div className="shrink-0 text-end">
              <span className="block text-2xs text-ink-muted">{C.total}</span>
              <span className={`figure block leading-none ${data.total > 0 ? "text-warning" : "text-ink-muted"}`}>
                {fmtMoney(data.total)}
              </span>
            </div>
          )}
          {canOpen && (
            <Button
              onClick={() => setOpenNew(true)}
              className="flex w-full items-center justify-center gap-1.5 sm:w-auto"
            >
              <IconAdd size={16} />
              {C.openBtn}
            </Button>
          )}
        </div>
        <div className="space-y-2 border-t border-line-soft bg-field px-4 py-3">
          <Chips
            wrap
            items={PARTIES.map((p) => ({
              id: p || "all",
              label: partyLabels[p || "all"],
              count: p ? (counts[p] ?? 0) : allOpen,
            }))}
            value={party || "all"}
            onChange={(k) => {
              setPage(1);
              setParty((k === "all" ? "" : k) as Party);
            }}
          />
          <Chips
            wrap
            items={STATUSES.map((s) => ({ id: s, label: statusLabels[s] }))}
            value={status}
            onChange={(k) => {
              setPage(1);
              setStatus(k as Status);
            }}
          />
        </div>
      </section>

      {notice && (
        <Alert tone="success" onDismiss={() => setNotice("")}>
          {notice}
        </Alert>
      )}

      {/* **تحميلٌ وخطأٌ غيرُ الفراغ** — كان الفشلُ يقول «لا نزاعات مفتوحة» وأصفاراً. */}
      {error ? (
        <ReloadState label={C.loadError} onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <CaseList rows={rows} empty={emptyBy[status] ?? C.empty} emptyHint={C.emptyHint} actions={actions} />
          {data.count > data.per_page && (
            <div className="flex justify-center">
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
          onDone={(msg) => {
            setProposing(null);
            setNotice(msg);
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

/**
 * **الحسم** — خصماً أو مسامحةً، بكلمة. **حركةُ المالك أو المدير تنفذ فوراً وطلبُ الموظّف ينتظر**
 * (قرارُ المالك ٢٠٢٦-١٠-٢٧) — والرسالةُ تُقرأ من الردّ (`applied`) لا تُفترض.
 */
function ProposeModal({
  dispute,
  charge,
  onClose,
  onDone,
}: {
  dispute: Dispute;
  charge: boolean;
  onClose: () => void;
  onDone: (msg: string) => void;
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
      const r = await api<{ applied?: boolean; charged?: number | null; remaining?: number }>(
        `/api/v1/admin/disputes/${dispute.id}/propose`,
        {
          method: "POST",
          body: JSON.stringify({ action: charge ? "charge" : "waive", note: note.trim() }),
        },
      );
      if (!r?.applied) onDone(C.sent);
      else if (!charge) onDone(C.appliedWaive);
      else if ((r.remaining ?? 0) > 0)
        onDone(
          C.chargedPartial
            .replace("{charged}", fmtMoney(r.charged ?? 0))
            .replace("{remaining}", fmtMoney(r.remaining ?? 0)),
        );
      else onDone(C.appliedCharge.replace("{charged}", fmtMoney(r.charged ?? 0)));
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
          : approve
            ? C.applied
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
    // **ولا رجوعٌ صامت** (فحصُ ٢٠٢٦-١٠-٢٧) — كان الزرُّ يُضغط فلا يحدث شيءٌ ولا يُقال ما الناقص.
    if (!who) {
      setError(C.pickPartyFirst);
      return;
    }
    if (!Number.isFinite(value) || value <= 0) {
      setError(C.amountRequired);
      return;
    }
    if (!reason.trim()) {
      setError(C.reasonRequired);
      return;
    }
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
        <div className="space-y-1.5">
          <p className="text-sm font-medium">{C.pickRole}</p>
          <Chips
            wrap
            items={(["merchant", "driver", "sales", "customer"] as const).map((r) => ({ id: r, label: partyOne[r] }))}
            value={role}
            onChange={(k) => {
              setRole(k as Exclude<Party, "">);
              setWho(null);
              setOptions(null);
              setError("");
            }}
          />
        </div>
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
