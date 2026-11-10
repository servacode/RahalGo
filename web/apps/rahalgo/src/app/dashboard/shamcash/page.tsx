"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/dashboard/shamcash — سجلّ شام كاش اليدويّ** (قرارُ المالك ٢٠٢٦-١١-١٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * الزبونُ أحياناً يحوّل على حساب شام كاش تبع المالك بدل ما يدفع للسائق نقداً — **والمالكُ يعطي
 * السائقَ المبلغَ نقداً من جيبه**، والسائقُ يسلّم المكتبَ كاملاً كالعادة. **فلا شيء هنا يمسّ
 * الخزينةَ ولا حسابَ السائقين** — سجلٌّ جانبيٌّ (`internal/shamcash`):
 *
 *  - رصيدٌ لكلّ عملة = البداية + التحويلات (غير «ما وصل») + الإيداعات − السحوبات
 *  - «ما سلّمته للسائقين لسّا» مجموعاً ولكلّ سائق
 *  - الحركاتُ بمرشّحاتها، و«سلّمت السائق بدالها» و«ما وصل» وتراجعُهما، وإلغاءُ إيداعٍ أو سحب
 *  - «أضف حركة»: إيداعُ المالك أو سحبُه بعملته وتاريخه وملاحظته
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDate, fmtDateTime, fmtMoney, fmtNum, fmtUsd } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  DataView,
  FormActions,
  IconAdd,
  IconCash,
  IconDate,
  IconEdit,
  IconMoto,
  IconOrder,
  IconUser,
  IconWallet,
  Input,
  LoadingState,
  Modal,
  PageContainer,
  Pagination,
  ReloadState,
  Select,
  Switch,
  Textarea,
  ViewToggle,
  useLiveData,
  useViewMode,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const S = m.admin.shamcash;

type Currency = "SYP" | "USD";
type Kind = "driver_transfer" | "deposit" | "withdrawal";

interface Movement {
  id: string;
  kind: Kind;
  order_id: string | null;
  order_number: number | null;
  driver_id: string | null;
  driver_name: string | null;
  currency: Currency;
  amount: number;
  occurred_at: string;
  note: string;
  status: "claimed" | "not_arrived";
  not_arrived_at: string | null;
  compensated_at: string | null;
  compensated_by_name: string | null;
  voided_at: string | null;
  void_reason: string;
  created_by_name: string | null;
  before_opening: boolean;
  /** السائق كتب مبلغاً أكبر من المطلوب من الزبون (قبله الخادم وعلّمه). */
  exceeds_due: boolean;
  due_at_entry: number | null;
}

interface Balance {
  currency: Currency;
  opening: number;
  opening_as_of: string | null;
  transfers: number;
  deposits: number;
  withdrawals: number;
  balance: number;
  not_arrived: number;
  owed_to_drivers: number;
}

interface Summary {
  balances: Balance[];
  owed_by_driver: { driver_id: string; driver_name: string; currency: Currency; amount: number; count: number }[];
  drivers: { id: string; name: string }[];
}

/** **المبلغُ بعملته** — الدولارُ بالسنتات، ولا يُجمع مع الليرة ولا يُحوَّل. */
function fmtAmount(amount: number, currency: Currency | string): string {
  return currency === "USD" ? fmtUsd(amount) : fmtMoney(amount);
}

/** **مدخلُ المبلغ** — ليرةٌ صحيحة أو دولارٌ بمنزلتين بالأكثر؛ و`null` لما لا يصلح. */
function parseAmount(text: string, currency: Currency, allowZero = false): number | null {
  const t = text.trim();
  const ok = currency === "USD" ? /^\d+(\.\d{1,2})?$/.test(t) : /^\d+$/.test(t);
  if (!ok) return null;
  const v = Number(t);
  if (!Number.isFinite(v) || (allowZero ? v < 0 : v <= 0)) return null;
  return v;
}

/** **السنتاتُ أو الليرةُ نصّاً للتعديل** — عكسُ `parseAmount`. */
function amountText(units: number, currency: Currency): string {
  return currency === "USD" ? String(units / 100) : String(units);
}

const MINUS = String.fromCharCode(0x2212);

export default function ShamcashPage() {
  const { can } = useAuth();
  const canManage = can("finance.manage");
  const canOrder = useCanOpen().order;
  const [view, setView, { narrow }] = useViewMode("admin-shamcash", "table");
  const [kind, setKind] = useState<"" | Kind>("");
  const [driver, setDriver] = useState("");
  const [status, setStatus] = useState("");
  const [comp, setComp] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [voided, setVoided] = useState(false);
  const [page, setPage] = useState(1);
  const [notice, setNotice] = useState("");
  const [adding, setAdding] = useState(false);
  const [opening, setOpening] = useState<Balance | null>(null);
  const [acting, setActing] = useState<{ row: Movement; act: Act } | null>(null);

  const sum = useLiveData<Summary>(() => api("/api/v1/admin/shamcash/summary"), ["wallet", "order"]);
  const qs = new URLSearchParams();
  if (kind) qs.set("kind", kind);
  if (driver) qs.set("driver", driver);
  if (status) qs.set("status", status);
  if (comp) qs.set("compensated", comp);
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  if (voided) qs.set("voided", "1");
  qs.set("page", String(page));
  const list = useLiveData<{ movements: Movement[]; total: number; per_page: number }>(
    () => api(`/api/v1/admin/shamcash?${qs.toString()}`),
    ["wallet", "order"],
    [qs.toString()],
  );

  function reload() {
    sum.reload();
    list.reload();
  }
  function filter(fn: () => void) {
    fn();
    setPage(1);
  }

  const s = sum.data;
  const owedSyp = s?.balances.find((b) => b.currency === "SYP")?.owed_to_drivers ?? 0;
  const owedUsd = s?.balances.find((b) => b.currency === "USD")?.owed_to_drivers ?? 0;

  return (
    <PageContainer>
      <PageHero
        icon={IconCash}
        title={S.title}
        actions={
          canManage ? (
            <Button onClick={() => setAdding(true)} className="flex min-h-11 items-center gap-1.5">
              <IconAdd size={16} />
              {S.addMovement}
            </Button>
          ) : undefined
        }
      />
      {notice && (
        <Alert tone="success" className="mb-3">
          {notice}
        </Alert>
      )}

      {sum.error && !s && <ReloadState onRetry={reload} />}
      {!s && !sum.error && <LoadingState variant="list" rows={2} />}
      {s && (
        <>
          {/* ── الرصيدان ثمّ ما للسائقين: أرقامُ الخادم كما هي ── */}
          <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {s.balances.map((b) => (
              <Tile
                key={b.currency}
                icon={IconWallet}
                label={b.currency === "USD" ? S.balanceUsd : S.balanceSyp}
                value={fmtAmount(b.balance, b.currency)}
                sub={[
                  S.balanceSub
                    .replace("{o}", fmtAmount(b.opening, b.currency))
                    .replace("{t}", fmtAmount(b.transfers, b.currency))
                    .replace("{d}", fmtAmount(b.deposits, b.currency))
                    .replace("{w}", fmtAmount(b.withdrawals, b.currency)),
                  b.not_arrived > 0 ? S.notArrivedSub.replace("{n}", fmtAmount(b.not_arrived, b.currency)) : "",
                ]
                  .filter(Boolean)
                  .join(m.common.listSeparator)}
                tone={b.balance < 0 ? "danger" : "default"}
                extra={
                  canManage ? (
                    <Button variant="secondary" className="min-h-11" onClick={() => setOpening(b)}>
                      <IconEdit size={14} />
                      {S.openingEdit}
                    </Button>
                  ) : undefined
                }
              />
            ))}
            <Tile
              icon={IconMoto}
              label={S.owedTitle}
              value={
                owedSyp > 0 || owedUsd > 0
                  ? [owedSyp > 0 ? fmtMoney(owedSyp) : "", owedUsd > 0 ? fmtUsd(owedUsd) : ""]
                      .filter(Boolean)
                      .join(" + ")
                  : fmtNum(0)
              }
              sub={S.owedHint}
              tone={owedSyp > 0 || owedUsd > 0 ? "warning" : "muted"}
              onClick={() =>
                filter(() => {
                  setKind("driver_transfer");
                  setComp("no");
                  setStatus("claimed");
                })
              }
            />
          </div>

          <Panel title={S.owedTitle} className="mb-4">
            {s.owed_by_driver.length === 0 ? (
              <p className="text-sm text-ink-muted">{S.owedNone}</p>
            ) : (
              <ul className="divide-y divide-line-soft">
                {s.owed_by_driver.map((o) => (
                  <li key={o.driver_id + o.currency} className="flex flex-wrap items-center justify-between gap-2 py-2">
                    <span className="min-w-0">
                      <span className="font-bold">{o.driver_name}</span>
                      <span className="text-xs text-ink-muted"> · {S.owedCount.replace("{n}", fmtNum(o.count))}</span>
                    </span>
                    <span className="flex items-center gap-2">
                      <span className="font-bold text-warning" dir="ltr">
                        {fmtAmount(o.amount, o.currency)}
                      </span>
                      <Button
                        variant="secondary"
                        className="min-h-11"
                        onClick={() =>
                          filter(() => {
                            setKind("driver_transfer");
                            setDriver(o.driver_id);
                            setComp("no");
                            setStatus("claimed");
                          })
                        }
                      >
                        {S.owedShow}
                      </Button>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </Panel>
        </>
      )}

      {/* ── المرشّحات ── */}
      <Panel title={S.movements} className="mb-3">
        <div className="space-y-3">
          <Chips
            value={kind}
            onChange={(v) => filter(() => setKind(v as "" | Kind))}
            items={[
              { id: "", label: S.kindAll },
              { id: "driver_transfer", label: S.kinds.driver_transfer },
              { id: "deposit", label: S.kinds.deposit },
              { id: "withdrawal", label: S.kinds.withdrawal },
            ]}
          />
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
            <Select aria-label={S.colDriver} value={driver} onChange={(e) => filter(() => setDriver(e.target.value))}>
              <option value="">{S.driverAll}</option>
              {(s?.drivers ?? []).map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </Select>
            <Select aria-label={S.colStatus} value={status} onChange={(e) => filter(() => setStatus(e.target.value))}>
              <option value="">{S.statusAll}</option>
              <option value="claimed">{S.statusClaimed}</option>
              <option value="not_arrived">{S.statusNotArrived}</option>
            </Select>
            <Select aria-label={S.compAll} value={comp} onChange={(e) => filter(() => setComp(e.target.value))}>
              <option value="">{S.compAll}</option>
              <option value="no">{S.compNo}</option>
              <option value="yes">{S.compYes}</option>
            </Select>
            <Switch checked={voided} onChange={(v) => filter(() => setVoided(v))} label={S.showVoided} />
            <Input id="sc-from" type="date" label={S.from} value={from} onChange={(e) => filter(() => setFrom(e.target.value))} />
            <Input id="sc-to" type="date" label={S.to} value={to} onChange={(e) => filter(() => setTo(e.target.value))} />
          </div>
        </div>
      </Panel>

      {!narrow && (
        <div className="mb-2 flex justify-end">
          <ViewToggle view={view} onChange={setView} tableLabel={m.common.viewTable} cardsLabel={m.common.viewCards} />
        </div>
      )}
      {list.error && !list.data && <ReloadState onRetry={reload} />}
      {!list.data && !list.error && <LoadingState variant="list" rows={4} />}
      {list.data && (
        <>
          <DataView<Movement>
            view={narrow ? "cards" : view}
            items={list.data.movements}
            getKey={(r) => r.id}
            empty={S.empty}
            columns={[
              {
                id: "amount",
                header: S.colAmount,
                primary: true,
                icon: <IconCash size={14} />,
                cell: (r) => (
                  <span
                    dir="ltr"
                    className={`font-bold ${r.voided_at || r.status === "not_arrived" ? "text-ink-muted line-through" : r.exceeds_due ? "rounded-control bg-danger-tint px-1.5 text-danger" : r.kind === "withdrawal" ? "text-danger" : "text-accent-text"}`}
                  >
                    {r.kind === "withdrawal" ? MINUS : ""}
                    {fmtAmount(r.amount, r.currency)}
                  </span>
                ),
              },
              {
                id: "kind",
                header: S.colKind,
                cell: (r) => <KindBadges r={r} />,
              },
              {
                id: "date",
                header: S.colDate,
                icon: <IconDate size={14} />,
                cell: (r) => <span dir="ltr">{fmtDateTime(r.occurred_at)}</span>,
              },
              {
                id: "order",
                header: S.colOrder,
                icon: <IconOrder size={14} />,
                hide: (r) => !r.order_id,
                cell: (r) =>
                  r.order_id ? (
                    <OpenLink
                      allowed={canOrder}
                      href={`/dashboard/orders?id=${r.order_id}`}
                      className="font-bold text-primary hover:underline"
                    >
                      {S.orderNo.replace("{n}", String(r.order_number ?? ""))}
                    </OpenLink>
                  ) : (
                    <span className="text-ink-muted">—</span>
                  ),
              },
              {
                id: "driver",
                header: S.colDriver,
                icon: <IconUser size={14} />,
                hide: (r) => !r.driver_id,
                cell: (r) => <span>{r.driver_name ?? "—"}</span>,
              },
              {
                id: "note",
                header: S.colNote,
                hide: (r) => !r.note && !r.voided_at && !r.compensated_at && !r.exceeds_due,
                cell: (r) => (
                  <span className="text-xs text-ink-muted">
                    {[
                      r.note,
                      r.exceeds_due && r.due_at_entry != null
                        ? S.exceedsNote.replace("{due}", fmtAmount(r.due_at_entry, r.currency))
                        : "",
                      r.compensated_at
                        ? S.compensatedBy
                            .replace("{name}", r.compensated_by_name ?? "")
                            .replace("{date}", fmtDate(r.compensated_at))
                        : "",
                      r.voided_at ? S.voidedWhy.replace("{r}", r.void_reason) : "",
                    ]
                      .filter(Boolean)
                      .join(m.common.listSeparator) || "—"}
                  </span>
                ),
              },
            ]}
            actions={canManage ? (r) => <RowActions r={r} onAct={(act) => setActing({ row: r, act })} /> : undefined}
          />
          <Pagination page={page} total={list.data.total} perPage={list.data.per_page} onChange={setPage} />
        </>
      )}

      {adding && (
        <AddModal
          onClose={() => setAdding(false)}
          onDone={() => {
            setAdding(false);
            setNotice(S.added);
            reload();
          }}
        />
      )}
      {opening && (
        <OpeningModal
          b={opening}
          onClose={() => setOpening(null)}
          onDone={() => {
            setOpening(null);
            setNotice(S.openingSaved);
            reload();
          }}
        />
      )}
      {acting && (
        <ActModal
          row={acting.row}
          act={acting.act}
          onClose={() => setActing(null)}
          onDone={() => {
            setActing(null);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

function KindBadges({ r }: { r: Movement }) {
  return (
    <span className="flex flex-wrap gap-1">
      <Badge variant={r.kind === "driver_transfer" ? "info" : r.kind === "deposit" ? "success" : "warning"}>
        {S.kinds[r.kind]}
      </Badge>
      {r.voided_at && <Badge variant="neutral">{S.badgeVoided}</Badge>}
      {r.kind === "driver_transfer" && r.status === "not_arrived" && <Badge variant="danger">{S.badgeNotArrived}</Badge>}
      {r.kind === "driver_transfer" && r.status === "claimed" && r.compensated_at && (
        <Badge variant="success">{S.badgeCompensated}</Badge>
      )}
      {r.kind === "driver_transfer" && r.status === "claimed" && !r.compensated_at && (
        <Badge variant="warning">{S.badgeOwed}</Badge>
      )}
      {r.before_opening && !r.voided_at && <Badge variant="neutral">{S.badgeBeforeOpening}</Badge>}
      {r.exceeds_due && !r.voided_at && <Badge variant="danger">{S.badgeExceeds}</Badge>}
    </span>
  );
}

type Act = "compensate" | "uncompensate" | "not-arrived" | "arrived" | "void";
/** **ما يلزمه سببٌ** — التراجعُ والإلغاء (والخادمُ يحكم أيضاً). */
const NEEDS_REASON: Record<Act, boolean> = {
  compensate: false,
  uncompensate: true,
  "not-arrived": false,
  arrived: true,
  void: true,
};
const ACT_LABEL: Record<Act, string> = {
  compensate: S.actCompensate,
  uncompensate: S.actUncompensate,
  "not-arrived": S.actNotArrived,
  arrived: S.actArrived,
  void: S.actVoid,
};

function RowActions({ r, onAct }: { r: Movement; onAct: (a: Act) => void }) {
  if (r.voided_at) return null;
  const acts: Act[] = [];
  if (r.kind === "driver_transfer") {
    if (r.status === "claimed") acts.push(r.compensated_at ? "uncompensate" : "compensate");
    if (!r.compensated_at) acts.push(r.status === "not_arrived" ? "arrived" : "not-arrived");
  } else {
    acts.push("void");
  }
  return (
    <>
      {acts.map((a) => (
        <Button
          key={a}
          variant={a === "compensate" ? "primary" : a === "not-arrived" || a === "void" ? "danger-outline" : "secondary"}
          className="min-h-11"
          onClick={() => onAct(a)}
        >
          {ACT_LABEL[a]}
        </Button>
      ))}
    </>
  );
}

function ActModal({ row, act, onClose, onDone }: { row: Movement; act: Act; onClose: () => void; onDone: () => void }) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const need = NEEDS_REASON[act];
  const amount = fmtAmount(row.amount, row.currency);
  async function submit() {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/shamcash/${row.id}/${act}`, {
        method: "POST",
        body: JSON.stringify(need ? { reason: reason.trim() } : {}),
      });
      onDone();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open onClose={onClose} title={ACT_LABEL[act]}>
      <div className="space-y-3">
        <p className="text-sm">
          {act === "compensate"
            ? S.confirmCompensate.replace("{amount}", amount)
            : act === "not-arrived"
              ? S.confirmNotArrived.replace("{amount}", amount)
              : `${S.kinds[row.kind]} · ${amount}`}
        </p>
        {need && (
          <Textarea
            id="sc-reason"
            label={S.reasonLabel}
            required
            rows={2}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        )}
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy || (need && reason.trim() === "")}
          saveLabel={S.confirm}
          cancelLabel={S.back}
        />
      </div>
    </Modal>
  );
}

function CurrencyChips({ value, onChange }: { value: Currency; onChange: (c: Currency) => void }) {
  return (
    <div>
      <p className="mb-1 text-sm font-medium">{S.currency}</p>
      <Chips
        value={value}
        onChange={(v) => onChange(v as Currency)}
        items={[
          { id: "SYP", label: S.syp },
          { id: "USD", label: S.usd },
        ]}
      />
    </div>
  );
}

function AddModal({ onClose, onDone }: { onClose: () => void; onDone: () => void }) {
  const [kind, setKind] = useState<"deposit" | "withdrawal">("deposit");
  const [currency, setCurrency] = useState<Currency>("SYP");
  const [amount, setAmount] = useState("");
  const [when, setWhen] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const v = parseAmount(amount, currency);
  async function submit() {
    if (busy || v == null) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/shamcash/movements", {
        method: "POST",
        body: JSON.stringify({ kind, currency, amount: v, occurred_at: when, note: note.trim() }),
      });
      onDone();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open onClose={onClose} title={S.addTitle}>
      <div className="space-y-3">
        <Chips
          value={kind}
          onChange={(k) => setKind(k as "deposit" | "withdrawal")}
          items={[
            { id: "deposit", label: S.kinds.deposit },
            { id: "withdrawal", label: S.kinds.withdrawal },
          ]}
        />
        <CurrencyChips value={currency} onChange={setCurrency} />
        <Input
          id="sc-amount"
          label={S.amount}
          inputMode={currency === "USD" ? "decimal" : "numeric"}
          dir="ltr"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          error={amount !== "" && v == null ? S.amountInvalid : undefined}
        />
        <Input
          id="sc-when"
          type="datetime-local"
          label={S.when}
          value={when}
          onChange={(e) => setWhen(e.target.value)}
        />
        <p className="text-xs text-ink-muted">{S.whenHelp}</p>
        <Textarea id="sc-note" label={S.note} rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy || v == null}
          saveLabel={S.addMovement}
          cancelLabel={S.back}
        />
      </div>
    </Modal>
  );
}

function OpeningModal({ b, onClose, onDone }: { b: Balance; onClose: () => void; onDone: () => void }) {
  const [amount, setAmount] = useState(b.opening_as_of ? amountText(b.opening, b.currency) : "");
  const [asOf, setAsOf] = useState(b.opening_as_of ? b.opening_as_of.slice(0, 10) : "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const v = parseAmount(amount, b.currency, true);
  async function submit() {
    if (busy || v == null || !asOf) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/shamcash/opening", {
        method: "PUT",
        body: JSON.stringify({ currency: b.currency, amount: v, as_of: asOf }),
      });
      onDone();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open onClose={onClose} title={`${S.openingEdit} · ${b.currency === "USD" ? S.usd : S.syp}`}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">
          {S.opening}:{" "}
          {b.opening_as_of
            ? S.openingOf.replace("{amount}", fmtAmount(b.opening, b.currency)).replace("{date}", fmtDate(b.opening_as_of))
            : S.openingNone}
        </p>
        <Input
          id="sc-open-amount"
          label={S.amount}
          inputMode={b.currency === "USD" ? "decimal" : "numeric"}
          dir="ltr"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          error={amount !== "" && v == null ? S.amountInvalid : undefined}
        />
        <Input
          id="sc-open-asof"
          type="date"
          label={S.openingAsOf}
          value={asOf}
          onChange={(e) => setAsOf(e.target.value)}
        />
        <p className="text-xs text-ink-muted">{S.openingHelp}</p>
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy || v == null || !asOf}
          saveLabel={S.confirm}
          cancelLabel={S.back}
        />
      </div>
    </Modal>
  );
}
