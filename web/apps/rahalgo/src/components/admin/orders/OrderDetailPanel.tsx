"use client";

/**
 * **ملفُّ الطلب في لوحةٍ جانبيّة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (سجلُّ الطلبات، البند ٤)، وصار ملفّاً
 * كاملاً ٢٠٢٦-١٠-٣٠: «لمّا افتح السجلّ لازم افهم لكلّ طلب: شو صار، كيف مشي، وحركة المال.»
 *
 *   - **شو صار** — العنوانُ والأصنافُ والطلبُ الخاصّ، والتقييمُ والتسليمُ والبضاعةُ والشكاوى.
 *   - **كيف مشي** — المسارُ بأوقاته، ومن فعل كلَّ خطوةٍ وصفتُه، والمدّةُ بين الخطوات، والمتوقَّعُ والفعليّ.
 *   - **المال** — دفعُ الزبون والخصمُ والأجرةُ وما للمتجر والسائق والمندوب والمنصّة، لكلّ من يفتح السجلّ.
 *   - **والقيودُ الماليّةُ من الدفتر** — لمن يملك `finance.read` وحدَه، كما كانت.
 *   - **والمحادثةُ قراءةً فقط** — لمن يملك `orders.communications.read`.
 *
 * **وكلُّ قسمٍ يُحمَّل وحدَه ويفشل وحدَه** — المحادثةُ الساقطةُ لا تُخفي المسار.
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  getMessages,
  defaultLocale,
  fmtDateTime,
  fmtNum,
  fmtMoney,
  fmtSpan,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Drawer,
  InvoiceLines,
  LoadingState,
  Money,
  statusMeta,
  Stars,
  Timeline,
  type TimelineNode,
  IconUser,
  IconStore,
  IconMoto,
  IconWallet,
  IconLocation,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { roleLabelByCode } from "@/lib/rolemeta";
import { DriverMoneyBox } from "./DriverMoneyBox";
import { orderKindLabel, payerLabel } from "./payer";

const m = getMessages(defaultLocale);
const H = m.admin.ordersPage.history;
const F = H.file;
const OFFICE_STATUS: Record<string, string> = m.admin.ordersPage.board.status;
const STATUS_LABELS: Record<string, string> = m.orders.status;
const TX_KINDS: Record<string, string> = m.shared.txKinds;
const PARTIES: Record<string, string> = H.parties;
const ROLES: Record<string, string> = F.roles;
const BORNE: Record<string, string> = F.borne;
const CASE_KINDS: Record<string, string> = F.caseKinds;
const CASE_STATUS: Record<string, string> = F.caseStatus;
const ENDED_BY: Record<string, string> = m.admin.ordersPage.endedBy;
const FAULTS: Record<string, string> = m.admin.ordersPage.door.faults;
const FAILS: Record<string, string> = m.common.failReasons;
const DOOR_REPORTS: Record<string, string> = m.admin.ordersPage.door.reports;
/** **ترتيبُ الأطراف في القيود** — الزبونُ يدفع، ثمّ من يقبض، ثمّ المنصّة، ثمّ ما سواها. */
const PARTY_ORDER = ["customer", "merchant", "driver", "sales", "platform", "other"];

/** **صفحةُ كلّ قضيّة** — الشكاوى والمطالبات في «الدعم»، والطارئُ في صفحته. */
function caseHref(kind: string, id: string): string {
  if (kind === "emergency") return `/dashboard/emergencies/${id}`;
  return kind === "dispute" ? "/dashboard/tickets?tab=disputes" : "/dashboard/tickets?tab=tickets";
}

/** **اسمُ المرسِل** — وحين يغيب يُقال دورُه بالعربيّ لا رمزُه (`driver`). */
function senderLabel(l: { sender: string; role: string }): string {
  return l.sender || PARTIES[l.role] || roleLabelByCode(l.role);
}

/** **الفرقُ بين وقتين بالثواني** — و`null` إن غاب أحدُهما. */
function secsBetween(from?: string | null, to?: string | null): number | null {
  if (!from || !to) return null;
  return Math.max(0, (Date.parse(to) - Date.parse(from)) / 1000);
}

interface Step {
  status: string;
  at: string;
  actor?: string | null;
  actor_role?: string;
  note?: string;
}

interface Item {
  id: string;
  name: string;
  unit_price: number;
  qty: number;
  note: string;
  options: { group: string; name: string; price_delta: number }[];
  drink_for?: string | null;
}

interface OrderFile {
  item_offers: { item_id: string; offer_cut: number; offer_borne_by: string }[];
  commission_waiver: { amount: number; by_name: string; at: string | null; reason: string } | null;
  cases: { kind: string; id: string; number?: number; status: string; title: string; amount?: number; created_at: string }[];
  ledger: { merchant: number; driver: number; rep: number; platform: number; lines: number };
  late_after_min: number;
  late: boolean;
}

interface Detail {
  id: string;
  number: number;
  status: string;
  kind?: string;
  delivery_mode?: string;
  custom_mode?: string;
  customer_name: string;
  customer_phone?: string;
  customer_phone_masked?: string;
  merchant_name: string;
  driver_name: string | null;
  driver_phone?: string | null;
  total: number;
  subtotal: number;
  delivery_fee: number;
  discount: number;
  free_waived?: number;
  wallet_paid: number;
  cash_due: number;
  payment_method: string;
  fee_payer?: string;
  promo_code: string | null;
  merchant_net?: number;
  platform_commission?: number;
  commission_taken?: number;
  commission_waived?: number;
  commission_percent_exact?: number;
  pay_merchant?: number;
  driver_fee?: number;
  address_text: string;
  zone_name?: string | null;
  notes: string;
  cancel_reason: string;
  ended_by?: string;
  fail_reason?: string;
  fault?: string;
  custom_request?: string;
  parcel_note?: string;
  holder_name?: string;
  holder_phone?: string;
  custom_goods_amount?: number | null;
  custom_store_paid?: number | null;
  fetch_amount?: number | null;
  partner_merchant_name?: string;
  partner_commission_percent_exact?: number | null;
  ext_store_merchant_name?: string;
  ext_store_reason?: string;
  prep_minutes?: number | null;
  delivery_estimate_min?: number;
  to_door_eta_sec?: number | null;
  accepted_at?: string | null;
  ready_at?: string | null;
  picked_up_at?: string | null;
  delivered_at?: string | null;
  closed_at?: string | null;
  proof_url?: string | null;
  proof_skip_reason?: string;
  returned_at?: string | null;
  return_to?: "office" | "store" | null;
  goods_handed_at?: string | null;
  rating?: { platform_stars: number; driver_stars: number | null } | null;
  board?: { delivered_by_code?: boolean };
  ops_stage_times?: (string | null)[];
  ops_stage_late?: (boolean | null)[];
  items?: Item[];
  created_at: string;
  timeline: Step[];
  file?: OrderFile;
  /**
   * **موقعُ الزبون المباشر إن شاركه وهو حديث** (قرارُ المالك ٢٠٢٦-١٠-١٦) — يصل لمن يرى
   * العنوانَ وحدَه (`admin_orders_handlers.go`).
   */
  customer_live?: {
    pin_gap_m: number;
    at: string;
    accuracy_m: number | null;
    precise: boolean;
  } | null;
}

/**
 * **سطرُ موقع الزبون المباشر** — «يبعد X م عن العنوان · قبل Y ث»، **والتقريبيُّ يُقال عنه**:
 * يُرى ولا يتحوّل إليه طريقُ الكابتن.
 */
function liveLine(live: NonNullable<Detail["customer_live"]>, nowMs: number): string {
  const ageSec = Math.max(0, Math.round((nowMs - new Date(live.at).getTime()) / 1000));
  const age =
    ageSec < 60
      ? H.liveAgeSec.replace("{n}", fmtNum(ageSec))
      : H.liveAgeMin.replace("{n}", fmtNum(Math.floor(ageSec / 60)));
  const gap = live.pin_gap_m < 0 ? "—" : fmtNum(Math.round(live.pin_gap_m));
  const line = H.liveLine.replace("{m}", gap).replace("{age}", age);
  return live.precise ? line : line + H.liveApprox;
}

interface ChatLine {
  id: string;
  body: string;
  role: string;
  sender: string;
  created_at: string;
}

interface MoneyBreakdown {
  lines: { party: string; name: string; kind: string; amount: number; note: string }[];
}

type Load<T> = T | null | "failed";

function useLoad<T>(path: string | null): Load<T> {
  const [v, setV] = useState<Load<T>>(null);
  useEffect(() => {
    if (!path) return;
    let alive = true;
    setV(null);
    api<T>(path)
      .then((r) => alive && setV(r))
      .catch(() => alive && setV("failed"));
    return () => {
      alive = false;
    };
  }, [path]);
  return v;
}

export function OrderDetailPanel({
  orderId,
  onClose,
}: {
  orderId: string;
  onClose: () => void;
}) {
  const { can } = useAuth();
  const order = useLoad<Detail>(`/api/v1/admin/orders/${orderId}`);
  const chat = useLoad<{ lines: ChatLine[] }>(
    can("orders.communications.read") ? `/api/v1/admin/orders/${orderId}/chat` : null,
  );
  const money = useLoad<MoneyBreakdown>(
    can("finance.read") ? `/api/v1/admin/orders/${orderId}/breakdown` : null,
  );
  const [copied, setCopied] = useState(false);
  const [copyErr, setCopyErr] = useState("");

  const title =
    order && order !== "failed" ? F.title.replace("{n}", `#${order.number}`) : H.openDetails;

  const copy = () => {
    void navigator.clipboard
      .writeText(window.location.href)
      .then(() => {
        setCopied(true);
        setCopyErr("");
      })
      .catch((err) => setCopyErr(errorText(err)));
  };

  return (
    <Drawer
      open
      onClose={onClose}
      title={title}
      actions={
        <Button variant="secondary" onClick={copy}>
          {copied ? H.copied : H.copyLink}
        </Button>
      }
    >
      {copyErr && <Alert tone="warning" className="mb-3">{copyErr}</Alert>}
      {order === null && <LoadingState variant="text" />}
      {order === "failed" && <Alert tone="warning">{H.panelFailed}</Alert>}
      {order && order !== "failed" && (
        <div className="space-y-5">
          {/* **كلُّ سطرٍ بعنوانه وأيقونته** (تدقيقُ السجلّ ٢٠٢٦-١٠-٢٧). */}
          <div className="space-y-2 text-sm">
            <p className="flex flex-wrap items-center gap-2">
              <Badge variant={statusMeta("order", order.status).badge}>
                {OFFICE_STATUS[order.status] ?? order.status}
              </Badge>
              <Badge variant="neutral">{orderKindLabel(order)}</Badge>
              <span className="text-ink-muted" dir="ltr">
                {fmtDateTime(order.created_at)}
              </span>
            </p>
            <InfoRow icon={<IconUser size={15} />} label={H.infoCustomer}>
              {order.customer_name || "—"}
              <span dir="ltr" className="ms-2 text-xs text-ink-muted">
                {order.customer_phone || order.customer_phone_masked || ""}
              </span>
            </InfoRow>
            {order.merchant_name && (
              <InfoRow icon={<IconStore size={15} />} label={H.infoStore}>
                {order.merchant_name}
              </InfoRow>
            )}
            {order.driver_name && (
              <InfoRow icon={<IconMoto size={15} />} label={H.infoDriver}>
                {order.driver_name}
                {order.driver_phone && (
                  <span dir="ltr" className="ms-2 text-xs text-ink-muted">
                    {order.driver_phone}
                  </span>
                )}
              </InfoRow>
            )}
            {order.customer_live && (
              <p className="font-bold text-success">{liveLine(order.customer_live, Date.now())}</p>
            )}
            <InfoRow icon={<IconWallet size={15} />} label={H.infoTotal}>
              <Money value={order.total} className="font-bold" />
            </InfoRow>
          </div>

          <WhatSection o={order} />
          <HowSection o={order} />
          <MoneySection o={order} />

          {can("orders.communications.read") && (
            <section>
              <h3 className="mb-2 text-sm font-bold">{H.chat}</h3>
              {chat === null && <LoadingState variant="text" />}
              {chat === "failed" && <Alert tone="warning">{m.errors.offline}</Alert>}
              {chat && chat !== "failed" && chat.lines.length === 0 && (
                <p className="text-sm text-ink-muted">{H.chatEmpty}</p>
              )}
              {chat && chat !== "failed" && chat.lines.length > 0 && (
                <ul className="space-y-1.5">
                  {chat.lines.map((l) => (
                    <li key={l.id} className="flex items-start gap-2 text-sm">
                      <Badge variant={l.role === "driver" ? "primary" : "neutral"}>
                        {senderLabel(l)}
                      </Badge>
                      <span className="min-w-0 flex-1">{l.body}</span>
                      <span dir="ltr" className="shrink-0 text-xs text-ink-muted">
                        {fmtDateTime(l.created_at)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}

          {can("finance.read") && (
            <section>
              <h3 className="mb-2 text-sm font-bold">{F.ledger}</h3>
              {money === null && <LoadingState variant="text" />}
              {money === "failed" && <Alert tone="warning">{m.errors.offline}</Alert>}
              {money && money !== "failed" && money.lines.length === 0 && (
                <p className="text-sm text-ink-muted">{H.moneyEmpty}</p>
              )}
              {/* **مجمّعةٌ بالطرف، و«إلو/عليه» بدل إشارة الناقص** (تدقيقُ السجلّ ٢٠٢٦-١٠-٢٧) —
                  **والأرقامُ من الدفتر كما هي — عرضٌ لا حساب.** */}
              {money && money !== "failed" && money.lines.length > 0 && (
                <div className="space-y-3 text-sm">
                  {PARTY_ORDER.map((party) => {
                    const lines = money.lines.filter(
                      (l) => (PARTY_ORDER.includes(l.party) ? l.party : "other") === party,
                    );
                    if (lines.length === 0) return null;
                    return (
                      <div key={party} className="rounded-control border border-line-soft p-2.5">
                        <p className="mb-1.5 text-xs font-bold">{PARTIES[party] ?? party}</p>
                        <ul className="space-y-1.5">
                          {lines.map((l, i) => (
                            <li key={i} className="flex items-start justify-between gap-2">
                              <span className="min-w-0">
                                {TX_KINDS[l.kind] ?? l.kind}
                                {party !== "platform" && party !== "other" && l.name && (
                                  <span className="text-ink-muted"> — {l.name}</span>
                                )}
                                {l.note && <span className="block text-xs text-ink-muted">{l.note}</span>}
                              </span>
                              <span
                                className={`flex shrink-0 items-center gap-1.5 ${
                                  l.amount < 0 ? "text-danger" : "text-success"
                                }`}
                              >
                                <span className="text-xs">{l.amount < 0 ? H.debit : H.credit}</span>
                                <Money value={Math.abs(l.amount)} small />
                              </span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    );
                  })}
                </div>
              )}
            </section>
          )}
        </div>
      )}
    </Drawer>
  );
}

/** **عنوانُ قسمٍ من أقسام الملفّ الثلاثة.** */
function SectionBox({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-control border border-line-soft p-3">
      <h3 className="mb-2.5 text-sm font-bold">{title}</h3>
      <div className="space-y-3 text-sm">{children}</div>
    </section>
  );
}

/** **سطرٌ بعنوانه** — العنوانُ يميناً والقيمةُ يساراً. */
function Row({ label, children, strong = false }: { label: React.ReactNode; children: React.ReactNode; strong?: boolean }) {
  return (
    <div className={`flex items-start justify-between gap-3 ${strong ? "font-bold" : ""}`}>
      <span className={`shrink-0 ${strong ? "" : "text-ink-muted"}`}>{label}</span>
      <span className="min-w-0 text-end">{children}</span>
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════
// **شو صار**
// ══════════════════════════════════════════════════════════════════
function WhatSection({ o }: { o: Detail }) {
  const items = o.items ?? [];
  const offers = o.file?.item_offers ?? [];
  const cases = o.file?.cases ?? [];
  const ended = ["cancelled", "rejected", "failed", "refunded"].includes(o.status);
  const reason =
    o.cancel_reason || (o.fail_reason ? (DOOR_REPORTS[o.fail_reason] ?? FAILS[o.fail_reason] ?? o.fail_reason) : "");
  const partnerPct = o.partner_commission_percent_exact;
  return (
    <SectionBox title={F.what}>
      {ended && (
        <div className="space-y-1 rounded-control bg-field px-3 py-2">
          <Row label={H.endedBy}>{o.ended_by ? (ENDED_BY[o.ended_by] ?? o.ended_by) : H.endedBySystem}</Row>
          {reason && <Row label={H.reason}>{reason}</Row>}
          {o.fault && <Row label={H.fault}>{FAULTS[o.fault] ?? o.fault}</Row>}
        </div>
      )}

      {(o.address_text || o.zone_name) && (
        <Row
          label={
            <span className="inline-flex items-center gap-1.5">
              <IconLocation size={15} />
              {F.address}
            </span>
          }
        >
          {[o.zone_name, o.address_text].filter(Boolean).join(m.common.listSeparator)}
        </Row>
      )}
      {o.notes && <Row label={F.notes}>{o.notes}</Row>}

      {items.length > 0 && (
        <div>
          <p className="mb-1.5 text-xs font-bold">{F.items}</p>
          <InvoiceLines items={items} />
          {offers.map((of) => {
            const it = items.find((x) => x.id === of.item_id);
            return (
              <p key={of.item_id} className="mt-1 text-xs text-success">
                {F.offerLine
                  .replace("{n}", fmtMoney(of.offer_cut * (it?.qty ?? 1)))
                  .replace("{item}", it?.name ?? "")
                  .replace("{who}", BORNE[of.offer_borne_by] ?? of.offer_borne_by)}
              </p>
            );
          })}
        </div>
      )}

      {(o.custom_request || o.parcel_note || o.holder_name || o.custom_goods_amount != null ||
        o.custom_store_paid != null || o.fetch_amount != null || o.partner_merchant_name || o.ext_store_merchant_name) && (
        <div className="space-y-1">
          {o.custom_request && <Row label={F.customRequest}>{o.custom_request}</Row>}
          {o.parcel_note && <Row label={F.parcel}>{o.parcel_note}</Row>}
          {o.holder_name && (
            <Row label={F.holder}>
              {o.holder_name}
              {o.holder_phone && (
                <span dir="ltr" className="ms-2 text-xs text-ink-muted">
                  {o.holder_phone}
                </span>
              )}
            </Row>
          )}
          {o.custom_goods_amount != null && (
            <Row label={F.goodsAmount}>
              <Money value={o.custom_goods_amount} small />
            </Row>
          )}
          {o.custom_store_paid != null && (
            <Row label={F.storePaid}>
              <Money value={o.custom_store_paid} small />
            </Row>
          )}
          {o.fetch_amount != null && (
            <Row label={F.fetchAmount}>
              <Money value={o.fetch_amount} small />
            </Row>
          )}
          {o.partner_merchant_name && (
            <Row label={F.partner}>
              {partnerPct != null
                ? F.partnerPct.replace("{name}", o.partner_merchant_name).replace("{p}", fmtNum(partnerPct))
                : o.partner_merchant_name}
            </Row>
          )}
          {o.ext_store_merchant_name && (
            <Row label={F.extStore}>
              {o.ext_store_merchant_name}
              {o.ext_store_reason && <span className="block text-xs text-ink-muted">{o.ext_store_reason}</span>}
            </Row>
          )}
        </div>
      )}

      {o.rating && (
        <div className="space-y-1">
          <p className="text-xs font-bold">{F.rating}</p>
          <Row label={F.ratingPlatform}>
            <Stars value={o.rating.platform_stars} size="sm" />
          </Row>
          {o.rating.driver_stars != null && (
            <Row label={F.ratingDriver}>
              <Stars value={o.rating.driver_stars} size="sm" />
            </Row>
          )}
        </div>
      )}

      {(o.board?.delivered_by_code || o.proof_url || o.proof_skip_reason) && (
        <Row label={F.proof}>
          {[
            o.board?.delivered_by_code ? F.byCode : "",
            o.proof_url ? F.byPhoto : "",
            o.proof_skip_reason ? F.noPhoto.replace("{r}", o.proof_skip_reason) : "",
          ]
            .filter(Boolean)
            .join(m.common.listSeparator)}
        </Row>
      )}

      {(o.returned_at || o.return_to || o.goods_handed_at) && (
        <div className="space-y-1">
          <p className="text-xs font-bold">{F.goods}</p>
          {o.return_to && <Row label={F.goods}>{F.returnTo[o.return_to]}</Row>}
          {o.goods_handed_at && (
            <Row label={F.goodsHanded}>
              <span dir="ltr">{fmtDateTime(o.goods_handed_at)}</span>
            </Row>
          )}
          {o.returned_at && (
            <Row label={F.returnedAt}>
              <span dir="ltr">{fmtDateTime(o.returned_at)}</span>
            </Row>
          )}
        </div>
      )}

      {cases.length > 0 && (
        <div>
          <p className="mb-1.5 text-xs font-bold">{F.cases}</p>
          <ul className="space-y-1.5">
            {cases.map((c) => (
              <li key={`${c.kind}-${c.id}`} className="flex flex-wrap items-center justify-between gap-2">
                <span className="flex min-w-0 flex-wrap items-center gap-1.5">
                  <Badge variant={c.kind === "emergency" ? "danger" : "warning"}>
                    {CASE_KINDS[c.kind] ?? c.kind}
                    {c.number ? ` #${c.number}` : ""}
                  </Badge>
                  <span className="min-w-0">{c.title}</span>
                  <span className="text-xs text-ink-muted">{CASE_STATUS[c.status] ?? c.status}</span>
                </span>
                <Link href={caseHref(c.kind, c.id)} className="text-xs font-medium text-accent-text underline">
                  {F.caseOpen}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}
    </SectionBox>
  );
}

// ══════════════════════════════════════════════════════════════════
// **كيف مشي**
// ══════════════════════════════════════════════════════════════════
function HowSection({ o }: { o: Detail }) {
  // **علامةُ التأخير من المحرّك** — مرحلةٌ تأخّرت تُعلَّم على الخطوة التي دخلتها بوقتها نفسِه.
  const lateTimes = new Set<number>();
  (o.ops_stage_late ?? []).forEach((late, i) => {
    const at = o.ops_stage_times?.[i];
    if (late && at) lateTimes.add(Date.parse(at));
  });
  const total = secsBetween(o.created_at, o.closed_at);
  const prepActual = secsBetween(o.accepted_at, o.ready_at);
  const roadActual = secsBetween(o.picked_up_at, o.delivered_at);
  const roadPlannedSec = o.to_door_eta_sec ?? (o.delivery_estimate_min ? o.delivery_estimate_min * 60 : null);
  const file = o.file;
  return (
    <SectionBox title={F.how}>
      <Timeline
        nodes={o.timeline.map((s, i): TimelineNode => {
          const prev = i > 0 ? o.timeline[i - 1] : null;
          const gap = prev ? secsBetween(prev.at, s.at) : null;
          const late = lateTimes.has(Date.parse(s.at));
          const role = s.actor_role ? (ROLES[s.actor_role] ?? s.actor_role) : "";
          const who = s.actor ? (role ? `${s.actor} (${role})` : s.actor) : role || H.bySystem;
          return {
            id: `${i}`,
            state: "done",
            tone: late ? "warning" : undefined,
            title: (
              <span className="inline-flex flex-wrap items-center gap-1.5">
                {s.status === "ready" ? F.readyStep : (OFFICE_STATUS[s.status] ?? STATUS_LABELS[s.status] ?? s.status)}
                {late && <Badge variant="warning">{F.late}</Badge>}
              </span>
            ),
            detail: (
              <span className="text-xs text-ink-muted">
                {who}
                {gap != null && gap > 0 ? ` · ${F.afterPrev.replace("{d}", fmtSpan(gap))}` : ""}
                {s.note ? ` — ${s.note}` : ""}
              </span>
            ),
            trailing: (
              <span dir="ltr" className="text-xs">
                {fmtDateTime(s.at)}
              </span>
            ),
          };
        })}
      />

      <div className="space-y-1 rounded-control bg-field px-3 py-2">
        <p className="text-xs font-bold">{F.times}</p>
        {total != null && (
          <Row label={F.totalTook}>
            <span className="inline-flex items-center gap-1.5">
              {fmtSpan(total)}
              {file?.late && <Badge variant="warning">{F.late}</Badge>}
            </span>
          </Row>
        )}
        {file && file.late_after_min > 0 && (
          <Row label={F.lateAfter}>{F.minutes.replace("{n}", fmtNum(file.late_after_min))}</Row>
        )}
        {o.prep_minutes != null && o.prep_minutes > 0 && (
          <Row label={F.prepPlanned}>{F.minutes.replace("{n}", fmtNum(o.prep_minutes))}</Row>
        )}
        {prepActual != null && <Row label={F.prepActual}>{fmtSpan(prepActual)}</Row>}
        {roadPlannedSec != null && roadPlannedSec > 0 && (
          <Row label={F.roadPlanned}>{fmtSpan(roadPlannedSec)}</Row>
        )}
        {roadActual != null && <Row label={F.roadActual}>{fmtSpan(roadActual)}</Row>}
      </div>
    </SectionBox>
  );
}

// ══════════════════════════════════════════════════════════════════
// **المال** — لكلّ من يفتح السجلّ (ملفُّ الطلب ٢٠٢٦-١٠-٣٠). **عرضٌ لا حساب**: أرقامُ الخادم كما هي.
// ══════════════════════════════════════════════════════════════════
function MoneySection({ o }: { o: Detail }) {
  const file = o.file;
  const waiver = file?.commission_waiver ?? null;
  const commission = o.commission_taken ?? o.platform_commission ?? 0;
  // **خصمُ العروض بمن تحمّله** — من لقطة كلّ صنفٍ لحظةَ الطلب.
  const offerBy: Record<string, number> = {};
  for (const of of file?.item_offers ?? []) {
    const it = (o.items ?? []).find((x) => x.id === of.item_id);
    offerBy[of.offer_borne_by] = (offerBy[of.offer_borne_by] ?? 0) + of.offer_cut * (it?.qty ?? 1);
  }
  const ledger = file?.ledger;
  return (
    <SectionBox title={F.money}>
      <div className="space-y-1">
        <Row label={F.customerPaid} strong>
          <Money value={o.total} small />
        </Row>
        <p className="text-end text-xs text-ink-muted">{payerLabel(o)}</p>
        {o.wallet_paid > 0 && o.cash_due > 0 && (
          <>
            <Row label={F.fromWallet}>
              <Money value={o.wallet_paid} small />
            </Row>
            <Row label={F.inCash}>
              <Money value={o.cash_due} small />
            </Row>
          </>
        )}
      </div>

      <div className="space-y-1 border-t border-line-soft pt-2">
        <Row label={F.subtotal}>
          <Money value={o.subtotal} small />
        </Row>
        {o.discount > 0 && (
          <Row label={o.promo_code ? `${F.discount} · ${F.promo.replace("{code}", o.promo_code)}` : F.discount}>
            <span className="flex items-center gap-1">
              −<Money value={o.discount} small />
            </span>
          </Row>
        )}
        {Object.entries(offerBy).map(([who, amount]) => (
          <Row key={who} label={`${F.offerDiscount} · ${F.offerOn.replace("{who}", BORNE[who] ?? who)}`}>
            <Money value={amount} small />
          </Row>
        ))}
        <Row label={F.deliveryFee}>
          <Money value={o.delivery_fee} small />
        </Row>
        {(o.free_waived ?? 0) > 0 && (
          <Row label={F.freeWaived}>
            <Money value={o.free_waived ?? 0} small />
          </Row>
        )}
      </div>

      <div className="space-y-1 border-t border-line-soft pt-2">
        {(o.merchant_net ?? 0) > 0 && (
          <Row label={F.merchantNet}>
            <Money value={o.merchant_net ?? 0} small />
          </Row>
        )}
        {waiver ? (
          <div className="rounded-control bg-field px-3 py-2">
            <Row label={F.commissionWaived} strong>
              <Money value={waiver.amount} small />
            </Row>
            <p className="text-xs text-ink-muted">
              {F.waivedBy
                .replace("{name}", waiver.by_name || "—")
                .replace("{at}", waiver.at ? fmtDateTime(waiver.at) : "—")}
            </p>
            {waiver.reason && <p className="text-xs">{F.waivedReason.replace("{r}", waiver.reason)}</p>}
          </div>
        ) : (
          commission > 0 && (
            <Row label={F.commission}>
              <Money value={commission} small />
            </Row>
          )
        )}
        {(o.driver_fee ?? 0) > 0 && (
          <Row label={F.driverFee}>
            <Money value={o.driver_fee ?? 0} small />
          </Row>
        )}
        {ledger && ledger.rep !== 0 && (
          <Row label={F.repCommission}>
            <Money value={ledger.rep} small />
          </Row>
        )}
        {ledger && ledger.lines > 0 && (
          <Row label={F.platformProfit} strong>
            <Money value={ledger.platform} small />
          </Row>
        )}
      </div>

      <ul>
        <DriverMoneyBox o={o} />
      </ul>
    </SectionBox>
  );
}

/** **سطرٌ بعنوانه وأيقونته** — العنوانُ يميناً والقيمةُ يساراً. */
function InfoRow({ icon, label, children }: { icon: React.ReactNode; label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <span className="inline-flex shrink-0 items-center gap-1.5 text-ink-muted">
        {icon}
        {label}
      </span>
      <span className="min-w-0 text-end">{children}</span>
    </div>
  );
}
