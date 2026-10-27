"use client";

/**
 * **الأهداف والمكافآت** — من وصل لأيّ مرحلة، وكم انصرف، ومن ينتظر قراراً.
 *
 * # مكافأةُ الهدف تنصرف لحالها
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩ ثمّ ٢٠٢٦-٠٨-٣١: ثلاثُ مراحل تُدفع آليّاً عند
 * بلوغها.) **كان العنوانُ يقول «والمكافأةُ بيدك»** فيفهم الموظّفُ أنّه يدفع
 * بيده فيدفع مرّتين. هذه الصفحةُ لا تدفع مكافأةَ الهدف — تُريها.
 *
 * # واليدويُّ بموافقة (قراراتُ المالك ٢٠٢٦-١٠-٠٤) — إلّا من المالك والمدير
 *
 *  ١ · المكافأةُ والعقوبةُ اليدويّة طلبٌ يوافق عليه موظّفٌ آخر — لا مالَ قبلها.
 *      **وحركةُ المالك أو المدير تنفُذ فوراً** (قرارُ المالك ٢٠٢٦-١٠-٢٧) —
 *      **والرسالةُ تُقرأ من الردّ (`applied`) لا تُفترض.**
 *  ٣ و٤ · طلبٌ مسترجَعٌ أو متجرٌ حُذف/تجريبيّ تحت مرحلةٍ قُبضت ⇒ تنبيهٌ هنا
 *  ٥ · اختيارُ شهرٍ وتصدير
 *  ٦ · اليدويُّ «تقدير» دائماً — لا «عن الهدف»
 *
 * # والعقوبةُ الأكبرُ من الرصيد تصير ديناً (قرارُ المالك ٢٠٢٦-١٠-١٥)
 *
 * **كانت الصفحةُ تردّها قبل الخادم** — فمن رصيدُه سالبٌ لا يُعاقَب أبداً.
 * والآن يُقال الرصيدُ للمعلومة وحدَها، ويُقال إنّ الباقي دين.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtDateTime,
  fmtLongDate,
  westernDigits,
  errorText,
} from "@rahalgo/i18n";
import {
  Chips,
  Alert,
  PageContainer,
  Button,
  ButtonLink,
  Badge,
  Modal,
  Drawer,
  Confirm,
  Input,
  Textarea,
  LoadingState,
  ReloadState,
  IconStar,
  IconDriver,
  IconUser,
  IconTarget,
  IconWallet,
  IconWarning,
  IconSettings,
  IconArrowOut,
  IconSearch,
  IconClose,
  FormActions,
  useLiveRefresh,
} from "@rahalgo/ui";
import { api, apiFile, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const P = m.admin.incentives;
const S = m.shared.incentives;

interface Standing {
  user_id: string;
  name: string;
  phone: string;
  done: number;
  target: number;
  reached: boolean;
  level: number;
  levels: number;
  auto_paid: number;
  manual_paid: number;
  penalized: number;
  balance: number;
  /** **متاجرُ فتحها المندوبُ ولم تكتمل شروطُها** — لا تُعدّ بعد (٢٠٢٦-١٠-١٣). */
  pending?: number;
}

interface Level {
  n: number;
  target: number;
  reward: number;
}

interface IncAlert {
  kind: "count_dropped" | "grant_failed";
  incentive_id?: string;
  failure_id?: string;
  user_id: string;
  name: string;
  month: string;
  level: number;
  target_count: number;
  current: number;
  amount: number;
  refunded: number;
  fake_stores: string[];
  attempts?: number;
}

interface PageData {
  month: string;
  standings: Standing[];
  summary: { reached_per_level: number[]; auto_paid: number; manual_paid: number; penalties: number };
  levels: Level[];
  alerts: IncAlert[];
  cap: number;
}

interface IncRequest {
  id: string;
  user_id: string;
  user_name: string;
  kind: "reward" | "penalty";
  amount: number;
  note: string;
  source: string;
  proposed_by: string;
  proposer_name: string;
  created_at: string;
  /** **الدورُ الذي اقتُرح له** — ويُعرض في تبويبه وحدَه (قرارُ المالك ٢٠٢٦-١٠-١٥). */
  role?: Role | "";
}

interface Entry {
  id: string;
  kind: "reward" | "penalty";
  amount: number;
  reason: string;
  for_target: boolean;
  by: string;
  created_at: string;
}

type Role = "driver" | "sales";
type Kind = "reward" | "penalty";
/** **فلترُ الكروت** — حبّةٌ أو مربّعٌ مضغوط؛ و`stageN` من مربّع «وصلوا للمرحلة n». */
type Show = "all" | "reached" | "notyet" | "manual" | "penalized" | `stage${number}`;

/** **من تنفُذ حركتُه فوراً** — المالكُ والمدير (`approval.AutoApplies`)؛ سؤالُ عرضٍ والحدُّ في المحرّك. */
const AUTO_APPLY_ROLES = ["owner_super_admin", "admin"];

/** **موضوعُ الإعدادات لكلّ تبويب** — زرُّ المراحل يفتح قسمَ الأهداف مباشرةً (`?topic=`). */
const SETTINGS_TOPIC: Record<Role, string> = { driver: "drivers", sales: "reps" };

/** **ما يغيّر التقدّمَ دون أن يمسّ الحوافز** — تسليمُ طلبٍ للسائق، وتأهيلُ متجرٍ للمندوب. */
const PROGRESS_TOPIC: Record<Role, string> = { driver: "order", sales: "merchant" };

const fill = (s: string, v: Record<string, string | number>) =>
  Object.entries(v).reduce((acc, [k, x]) => acc.split(`{${k}}`).join(String(x)), s);

const DAMASCUS_MS = 3 * 3600 * 1000;

/** شهرُ لحظةٍ بتوقيت دمشق — «2026-10». */
function monthOf(t: number): string {
  const d = new Date(t + DAMASCUS_MS);
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}`;
}

/** الشهرُ الجاري بتوقيت دمشق. */
const thisMonth = () => monthOf(Date.now());

/** **شهرٌ قبل أو بعد** — «2026-01» − ١ = «2025-12». */
function shiftMonth(month: string, by: number): string {
  const [y = 0, mo = 1] = month.split("-").map(Number);
  const i = y * 12 + (mo - 1) + by;
  return `${Math.floor(i / 12)}-${String((i % 12) + 1).padStart(2, "0")}`;
}

/** **اسمُ الشهر بالعربي** — «تشرين الأول 2026» بدل «2026-10». من `fmtLongDate` نفسِه فلا تتعدّد الأسماء. */
function monthName(month: string): string {
  const [y, mo] = month.split("-").map(Number);
  if (!y || !mo) return month;
  return fmtLongDate(new Date(Date.UTC(y, mo - 1, 15, 9))).replace(/^\S+\s+/, "");
}

/** **مفتاحُ الرقم للبحث** — 0900 555 103 و+963900555103 و٠٩٠٠… كلُّها رقمٌ واحد. */
function phoneKey(s: string): string {
  let d = westernDigits(s).replace(/\D/g, "");
  if (d.startsWith("00")) d = d.slice(2);
  if (d.startsWith("0")) d = `963${d.slice(1)}`;
  return d;
}

const kindLabel = (k: Kind) => (k === "reward" ? S.reward : S.penalty);

function matchesShow(x: Standing, show: Show): boolean {
  switch (show) {
    case "all":
      return true;
    case "reached":
      return x.level > 0;
    case "notyet":
      return x.level === 0;
    case "manual":
      return x.manual_paid > 0;
    case "penalized":
      return x.penalized > 0;
    default:
      return x.level >= Number(show.slice(5));
  }
}

export default function IncentivesPage() {
  const [role, setRole] = useState<Role>("driver");
  const [month, setMonth] = useState(thisMonth());
  const [query, setQuery] = useState("");
  const [show, setShow] = useState<Show>("all");
  const canCall = useCanCall();
  const canGrant = canCall("POST", "/users/{id}/incentive");
  const canDecide = canCall("POST", "/incentive-requests/{id}/approve");
  const canSettings = useCanOpen().settings;
  const { user } = useAuth();
  // **ولفظُ الزرّ بحسب الشخص** — «نفّذ هلق» لمن تنفُذ حركتُه، و«ابعت للموافقة» لغيره.
  const autoApplies = (user?.roles ?? []).some((r) => AUTO_APPLY_ROLES.includes(r));

  /** **والفشلُ ليس فراغاً** — «لا أحدَ بلغ» على قراءةٍ فشلت قرارُ مالٍ على زور. */
  const [data, setData] = useState<PageData | null | "failed" | "nostages">(null);
  const [requests, setRequests] = useState<IncRequest[]>([]);
  const [pageError, setPageError] = useState("");
  const [notice, setNoticeRaw] = useState("");
  const noticeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  /** **ورسالةُ النجاح تختفي لحالها** — كانت تبقى وتنتقل مع التبويب. */
  const setNotice = useCallback((s: string) => {
    if (noticeTimer.current) clearTimeout(noticeTimer.current);
    setNoticeRaw(s);
    if (s) noticeTimer.current = setTimeout(() => setNoticeRaw(""), 6000);
  }, []);
  useEffect(() => () => {
    if (noticeTimer.current) clearTimeout(noticeTimer.current);
  }, []);

  const load = useCallback(() => {
    setPageError("");
    api<PageData>(`/api/v1/admin/incentives/${role}?month=${month}`)
      .then((r) => setData(r))
      .catch((err) => {
        // **ولا مراحلَ ليس عطلاً** — الخادمُ يردّ ٤٠٤ حين تُمسح مراحلُ الدور.
        if (err instanceof ApiError && err.status === 404) {
          setData("nostages");
          return;
        }
        setData("failed");
        setPageError(errorText(err));
      });
    api<{ requests: IncRequest[] }>(`/api/v1/admin/incentive-requests?status=pending`)
      .then((r) => setRequests(r.requests ?? []))
      .catch((err) => setPageError(errorText(err)));
  }, [role, month]);

  useEffect(() => {
    setData(null);
    setShow("all");
    setNotice("");
    load();
  }, [load, setNotice]);
  useLiveRefresh(["incentive", "wallet", "settings"], load);

  // **والتقدّمُ يتحدّث مع التسليم** — بنبضةٍ مخفّفة: لا أكثرَ من جلبةٍ كلّ ٢٠ ثانية.
  const lastLive = useRef(0);
  const liveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const loadThrottled = useCallback(() => {
    if (liveTimer.current) return;
    const wait = Math.max(0, 20000 - (Date.now() - lastLive.current));
    liveTimer.current = setTimeout(() => {
      liveTimer.current = null;
      lastLive.current = Date.now();
      load();
    }, wait);
  }, [load]);
  useEffect(() => () => {
    if (liveTimer.current) clearTimeout(liveTimer.current);
  }, []);
  useLiveRefresh([PROGRESS_TOPIC[role]], loadThrottled);

  // ── نافذةُ الطلب ─────────────────────────────────────────────────
  const [granting, setGranting] = useState<Standing | null>(null);
  const [kind, setKind] = useState<Kind>("reward");
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [formError, setFormError] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const ready = data && typeof data === "object" ? data : null;
  const cap = ready ? ready.cap : 0;

  function openGrant(x: Standing) {
    setKind("reward");
    setAmount("");
    setReason("");
    setFormError("");
    setNotice("");
    setGranting(x);
  }

  /**
   * **التحقّقُ قبل الإرسال** — كان الحقلُ يقبل كسوراً وسالباً ويردّه الخادمُ برسالةٍ عامّة.
   * **ولا شرطَ على الرصيد** — العقوبةُ الأكبرُ منه تصير ديناً في الخادم.
   */
  function validate(): string {
    const n = Number(amount);
    if (!/^\d+$/.test(amount.trim()) || !Number.isSafeInteger(n) || n <= 0) return P.errAmount;
    if (cap > 0 && n > cap) return fill(P.errOverCap, { cap: fmtMoney(cap) });
    if (!reason.trim()) return P.errReason;
    return "";
  }

  function askConfirm() {
    const err = validate();
    setFormError(err);
    if (!err) setConfirming(true);
  }

  async function submit() {
    const who = granting;
    if (!who) return;
    setConfirming(false);
    setBusy(true);
    setFormError("");
    try {
      const r = await api<{ applied?: boolean }>(`/api/v1/admin/users/${who.user_id}/incentive`, {
        method: "POST",
        // **والدورُ مع الطلب** — من له الدوران يُعدّ في التبويب الذي اقتُرح منه وحدَه.
        body: JSON.stringify({ kind, amount: Number(amount), reason: reason.trim(), role }),
      });
      setGranting(null);
      setNotice(r?.applied ? P.appliedNow : P.requestSent);
      load();
    } catch (e) {
      // **والخطأُ داخلَ النافذة** — كان يُكتب خلفها فلا يُرى.
      setFormError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  // ── الطلباتُ المعلّقة ─────────────────────────────────────────────
  const [approving, setApproving] = useState<IncRequest | null>(null);
  const [rejecting, setRejecting] = useState<IncRequest | null>(null);
  const [rejectNote, setRejectNote] = useState("");

  async function decide(r: IncRequest, ok: boolean) {
    setApproving(null);
    setRejecting(null);
    setBusy(true);
    try {
      await api(`/api/v1/admin/incentive-requests/${r.id}/${ok ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({ note: ok ? "" : rejectNote.trim() }),
      });
      setRejectNote("");
      setNotice(ok ? P.approvedDone : P.rejectedDone);
      load();
    } catch (err) {
      setPageError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  // ── التنبيهات ─────────────────────────────────────────────────────
  const [deciding, setDeciding] = useState<{ a: IncAlert; d: "keep" | "clawback" } | null>(null);
  const [alertNote, setAlertNote] = useState("");
  const [alertNoteErr, setAlertNoteErr] = useState(false);

  /** **وكلُّ فتحةٍ تبدأ فارغة** — كان سببُ التنبيه السابق يبقى في الخانة. */
  function openDecide(a: IncAlert, d: "keep" | "clawback") {
    setAlertNote("");
    setAlertNoteErr(false);
    setDeciding({ a, d });
  }

  async function decideAlert() {
    const x = deciding;
    if (!x?.a.incentive_id) return;
    // **والزرُّ لا يتجمّد على خانةٍ فارغة** — كان يكتب «جارٍ التحميل» فيُظنّ عالقاً.
    if (!alertNote.trim()) {
      setAlertNoteErr(true);
      return;
    }
    setDeciding(null);
    setBusy(true);
    try {
      const r = await api<{ applied?: boolean }>(`/api/v1/admin/incentive-alerts/${x.a.incentive_id}/decide`, {
        method: "POST",
        body: JSON.stringify({ decision: x.d, note: alertNote.trim() }),
      });
      setAlertNote("");
      setNotice(x.d === "keep" ? P.keepDone : r?.applied ? P.clawbackApplied : P.clawbackSent);
      load();
    } catch (err) {
      setPageError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function retry(a: IncAlert) {
    setBusy(true);
    try {
      await api(`/api/v1/admin/incentive-failures/${a.failure_id}/retry`, { method: "POST", body: "{}" });
      setNotice(P.retryDone);
      load();
    } catch (err) {
      setPageError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  // ── التصدير ───────────────────────────────────────────────────────
  async function exportCsv() {
    try {
      const res = await apiFile(`/api/v1/admin/incentives/${role}/export?month=${month}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `incentives-${role}-${month}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setPageError(errorText(err));
    }
  }

  // ── كشفُ الشخص ────────────────────────────────────────────────────
  const [ledgerOf, setLedgerOf] = useState<Standing | null>(null);
  const [ledger, setLedger] = useState<Entry[] | null | "failed">(null);
  useEffect(() => {
    if (!ledgerOf) return;
    setLedger(null);
    api<{ entries: Entry[] }>(`/api/v1/admin/users/${ledgerOf.user_id}/incentives?limit=200`)
      .then((r) => setLedger(r.entries ?? []))
      .catch(() => setLedger("failed"));
  }, [ledgerOf]);

  /** **الكشفُ تحت عناوين الشهور** — الشهرُ المختارُ أوّلاً ثمّ الأحدث فالأقدم، ولكلّ شهرٍ مجموعُه. */
  const ledgerGroups = useMemo(() => {
    if (!Array.isArray(ledger)) return [];
    const by = new Map<string, Entry[]>();
    for (const e of ledger) {
      const k = monthOf(Date.parse(e.created_at));
      by.set(k, [...(by.get(k) ?? []), e]);
    }
    return [...by.entries()]
      .sort(([a], [b]) => (a === month ? -1 : b === month ? 1 : b.localeCompare(a)))
      .map(([k, list]) => ({
        month: k,
        list,
        auto: list.filter((e) => e.kind === "reward" && e.for_target).reduce((s, e) => s + e.amount, 0),
        manual: list.filter((e) => e.kind === "reward" && !e.for_target).reduce((s, e) => s + e.amount, 0),
        penalty: list.filter((e) => e.kind === "penalty").reduce((s, e) => s + e.amount, 0),
      }));
  }, [ledger, month]);

  const TABS: HeroTab<Role>[] = [
    { key: "driver", label: P.tabDrivers, icon: IconDriver },
    { key: "sales", label: P.tabSales, icon: IconUser },
  ];
  const unit = role === "driver" ? P.unitDriver : P.unitSales;
  const settingsHref = `/dashboard/settings?topic=${SETTINGS_TOPIC[role]}`;

  const searched = useMemo(() => {
    if (!ready) return [];
    const q = query.trim();
    if (!q) return ready.standings;
    const qp = phoneKey(q);
    return ready.standings.filter(
      (x) => x.name.includes(q) || x.phone.includes(q) || (qp.length >= 3 && phoneKey(x.phone).includes(qp)),
    );
  }, [ready, query]);

  const rows = useMemo(() => searched.filter((x) => matchesShow(x, show)), [searched, show]);

  const counts = useMemo(() => {
    const all = ready?.standings ?? [];
    return {
      all: all.length,
      reached: all.filter((x) => x.level > 0).length,
      notyet: all.filter((x) => x.level === 0).length,
      manual: all.filter((x) => x.manual_paid > 0).length,
      penalized: all.filter((x) => x.penalized > 0).length,
    };
  }, [ready]);

  /**
   * **الطلباتُ بدورها لا بمن في القائمة** — كان طلبُ حسابٍ أُوقف يختفي من التبويبين فلا يُقرَّر.
   * وما لا دورَ عليه يُعرض في تبويب السائقين وحدَه، لا في الاثنين.
   */
  const listed = useMemo(() => new Set((ready?.standings ?? []).map((x) => x.user_id)), [ready]);
  const roleRequests = useMemo(
    () => requests.filter((r) => (r.role ? r.role === role : role === "driver")),
    [requests, role],
  );

  const showChips = [
    { id: "all" as Show, label: P.fAll, count: counts.all },
    { id: "reached" as Show, label: P.fReached, count: counts.reached },
    { id: "notyet" as Show, label: P.fNotYet, count: counts.notyet },
    { id: "manual" as Show, label: P.fManual, count: counts.manual },
    { id: "penalized" as Show, label: P.fPenalized, count: counts.penalized },
  ];
  const stageShow = show.startsWith("stage") ? Number(show.slice(5)) : 0;
  const filtered = show !== "all" || query.trim() !== "";
  const toggleShow = (s: Show) => setShow((cur) => (cur === s ? "all" : s));

  const now = thisMonth();
  const prev = shiftMonth(now, -1);

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦ ثمّ ٢٠٢٦-١٠-٢٧) — رأسٌ موحّدٌ بتبويبَي الدور، أرقامٌ تُضغط
  // فتفلتر، وفلاترُ ببطاقةٍ واحدة، وكرتٌ لكلّ شخصٍ ولكلّ تنبيهٍ ولكلّ طلب.
  return (
    <PageContainer>
      <PageHero<Role>
        icon={IconStar}
        title={P.title}
        subtitle={P.subtitle}
        actions={
          <Button variant="secondary" onClick={() => void exportCsv()} className="flex items-center gap-1.5">
            <IconArrowOut size={16} />
            {P.export}
          </Button>
        }
        tabs={TABS}
        value={role}
        onChange={setRole}
      />

      {/* ── الفلاتر ببطاقةٍ واحدة: الشهر، والبحث، ومن يطلع ── */}
      <section className="surface flex flex-col gap-3 p-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-ink-muted">{P.month}:</span>
          <Chips
            items={[
              { id: now, label: P.monthThis },
              { id: prev, label: P.monthPrev },
            ]}
            value={month}
            onChange={setMonth}
            wrap
          />
          {/* **وأيُّ شهرٍ بأسهم** — بدل خانة الشهر التي يختلف شكلُها من متصفّحٍ لآخر. */}
          <span className="ms-auto inline-flex items-center gap-1">
            <Button variant="secondary" aria-label={P.monthBack} onClick={() => setMonth(shiftMonth(month, -1))}>
              {"→"}
            </Button>
            <span className="min-w-[8.5rem] text-center text-sm font-bold">{monthName(month)}</span>
            <Button
              variant="secondary"
              aria-label={P.monthForward}
              disabled={month >= now}
              onClick={() => setMonth(shiftMonth(month, 1))}
            >
              {"←"}
            </Button>
          </span>
        </div>
        <Input
          id="incentives-search"
          label={P.search}
          icon={<IconSearch size={16} />}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        {ready && (
          <div className="flex flex-wrap items-center gap-2 border-t border-line-soft pt-3">
            <span className="text-sm text-ink-muted">{P.show}:</span>
            <Chips items={showChips} value={show} onChange={setShow} wrap />
            {stageShow > 0 && (
              <button
                type="button"
                onClick={() => setShow("all")}
                className="inline-flex items-center gap-1.5 rounded-full bg-accent px-3 py-1 text-sm font-medium text-on-bright"
              >
                {fill(P.fStage, { n: fmtNum(stageShow) })}
                <IconClose size={14} />
                <span className="sr-only">{P.clearFilter}</span>
              </button>
            )}
            {filtered && (
              <span className="ms-auto text-xs text-ink-muted">
                {fill(P.shown, { n: fmtNum(rows.length), total: fmtNum(counts.all) })}
              </span>
            )}
          </div>
        )}
      </section>

      {pageError && <Alert>{pageError}</Alert>}
      {notice && <Alert tone="success">{notice}</Alert>}

      {data === null ? (
        <LoadingState variant="stats" />
      ) : data === "failed" ? (
        <ReloadState onRetry={load} />
      ) : data === "nostages" ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-warning-solid text-on-solid">
            <IconTarget size={28} />
          </span>
          <h2 className="heading-section">{P.noStagesTitle}</h2>
          <p className="max-w-md text-sm text-ink-muted">{canSettings ? P.noStagesHint : P.noStagesHintAsk}</p>
          {canSettings && (
            <ButtonLink href={settingsHref} variant="secondary">
              <IconSettings size={14} />
              {P.editInSettings}
            </ButtonLink>
          )}
        </section>
      ) : (
        <>
          {/* ── الأرقام: كلُّ مرحلة · الآليّ · اليدويّ · العقوبات — وكلُّ مربّعٍ يفلتر ── */}
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-3 2xl:grid-cols-6">
            {data.summary.reached_per_level.map((n, i) => (
              <Tile
                key={i}
                icon={IconTarget}
                label={fill(P.cardStage, { n: fmtNum(i + 1) })}
                value={fmtNum(n)}
                tone={n > 0 ? "accent" : "muted"}
                onClick={() => toggleShow(`stage${i + 1}`)}
              />
            ))}
            <Tile
              icon={IconWallet}
              tone={data.summary.auto_paid > 0 ? "success" : "muted"}
              label={P.cardAuto}
              value={fmtMoney(data.summary.auto_paid)}
              onClick={() => toggleShow("reached")}
            />
            <Tile
              icon={IconStar}
              tone={data.summary.manual_paid > 0 ? "success" : "muted"}
              label={P.cardManual}
              value={fmtMoney(data.summary.manual_paid)}
              onClick={() => toggleShow("manual")}
            />
            <Tile
              icon={IconWarning}
              tone={data.summary.penalties > 0 ? "danger" : "muted"}
              label={P.cardPenalties}
              value={fmtMoney(data.summary.penalties)}
              onClick={() => toggleShow("penalized")}
            />
          </div>

          {/* ── مراحلُ الهدف — الوعدُ كما هو، وتعديلُه من الإعدادات ── */}
          <section className="surface p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div className="min-w-0">
                <h2 className="heading-card">{P.stagesTitle}</h2>
                <p className="text-xs text-ink-muted">{P.stagesHint}</p>
              </div>
              {/* **ولمن يفتح الإعدادات وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ لا تراها. */}
              {canSettings && (
                <ButtonLink href={settingsHref} variant="secondary">
                  <IconSettings size={14} />
                  {P.editInSettings}
                </ButtonLink>
              )}
            </div>
            {data.levels.length === 0 ? (
              <p className="text-sm text-ink-muted">{P.noStages}</p>
            ) : (
              <ol className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                {data.levels.map((l) => (
                  <li key={l.n} className="flex items-center gap-3 rounded-control bg-field px-3 py-2.5">
                    <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-accent text-sm font-bold text-on-bright tabular-nums">
                      {fmtNum(l.n)}
                    </span>
                    <span className="min-w-0">
                      <span className="block text-2xs text-ink-muted">{fill(P.stageN, { n: fmtNum(l.n) })}</span>
                      <span className="block font-bold tabular-nums">
                        {fmtNum(l.target)} {unit}
                      </span>
                      <span className="block text-xs text-success">
                        {fill(P.stageReward, { reward: fmtMoney(l.reward) })}
                      </span>
                    </span>
                  </li>
                ))}
              </ol>
            )}
          </section>

          {/* ── التنبيهات — كرتٌ لكلّ واحد، والماليّةُ تقرّر ── */}
          {data.alerts.length > 0 && (
            <section className="surface p-4">
              <div className="mb-3">
                <h2 className="heading-card flex items-center gap-2 text-danger">
                  <IconWarning size={18} />
                  {P.alertsTitle}
                  <Badge variant="danger">{fmtNum(data.alerts.length)}</Badge>
                </h2>
                <p className="text-xs text-ink-muted">{P.alertsHint}</p>
              </div>
              <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2">
                {data.alerts.map((a) => (
                  <AlertCard
                    key={a.incentive_id ?? a.failure_id}
                    a={a}
                    canDecide={canDecide}
                    busy={busy}
                    onDecide={(d) => openDecide(a, d)}
                    onRetry={() => void retry(a)}
                  />
                ))}
              </ul>
            </section>
          )}

          {/* ── الطلباتُ المعلّقة — كرتٌ لكلّ طلب، والمبلغُ كبير ── */}
          {roleRequests.length > 0 && (
            <section className="surface p-4">
              <div className="mb-3">
                <h2 className="heading-card flex items-center gap-2">
                  <IconWallet size={18} />
                  {P.pendingTitle}
                  <Badge variant="warning">{fmtNum(roleRequests.length)}</Badge>
                </h2>
                <p className="text-xs text-ink-muted">{P.pendingHint}</p>
              </div>
              <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2">
                {roleRequests.map((r) => (
                  <li key={r.id} className="flex flex-col gap-2 rounded-card border border-line-soft p-3 text-sm">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant={r.kind === "penalty" ? "danger" : "success"}>{kindLabel(r.kind)}</Badge>
                      {!listed.has(r.user_id) && <Badge variant="warning">{P.notListed}</Badge>}
                    </div>
                    <p
                      className={`figure tabular-nums ${r.kind === "penalty" ? "text-danger" : "text-success"}`}
                      dir="ltr"
                    >
                      {fmtMoney(r.amount)}
                    </p>
                    <p className="font-bold">{r.user_name}</p>
                    {r.note && <p className="border-s-2 border-line ps-2">{r.note}</p>}
                    <p className="text-2xs text-ink-muted">
                      {fill(P.proposedBy, { name: r.proposer_name, when: fmtDateTime(r.created_at) })}
                    </p>
                    {canDecide && (
                      <div className="mt-auto grid grid-cols-2 gap-2 border-t border-line-soft pt-2">
                        <Button className="w-full" disabled={busy} onClick={() => setApproving(r)}>
                          {P.approve}
                        </Button>
                        <Button
                          variant="secondary"
                          className="w-full"
                          disabled={busy}
                          onClick={() => {
                            setRejectNote("");
                            setRejecting(r);
                          }}
                        >
                          {P.reject}
                        </Button>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* ── الأشخاص — كرتٌ لكلّ واحد ── */}
          {rows.length === 0 ? (
            <section className="surface flex flex-col items-center gap-3 p-8 text-center">
              <span
                className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${
                  data.standings.length === 0 ? "bg-field text-ink-muted" : "bg-accent text-on-bright"
                }`}
              >
                {data.standings.length === 0 ? <IconUser size={28} /> : <IconSearch size={28} />}
              </span>
              <h2 className="heading-section">{data.standings.length === 0 ? P.empty : P.noResults}</h2>
              <p className="max-w-md text-sm text-ink-muted">
                {data.standings.length === 0 ? P.emptyHint : P.noResultsHint}
              </p>
              {data.standings.length > 0 && filtered && (
                <Button
                  variant="secondary"
                  onClick={() => {
                    setShow("all");
                    setQuery("");
                  }}
                >
                  {P.clearFilter}
                </Button>
              )}
            </section>
          ) : (
            <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
              {rows.map((x) => (
                <PersonCard
                  key={x.user_id}
                  x={x}
                  levels={data.levels}
                  unit={unit}
                  canGrant={canGrant}
                  onLedger={() => setLedgerOf(x)}
                  onGrant={() => openGrant(x)}
                />
              ))}
            </ul>
          )}
        </>
      )}

      {/* ── نافذةُ الطلب ── */}
      <Modal
        open={granting !== null}
        onClose={() => setGranting(null)}
        title={`${P.grantTitle} — ${granting?.name || granting?.phone || ""}`}
      >
        <div className="space-y-3">
          {formError && <Alert>{formError}</Alert>}
          <p className="text-sm font-medium">{P.kind}</p>
          {/* **النوعُ بلونه** — المكافأةُ خضراء والعقوبةُ حمراء، وتحت كلٍّ ما تفعله. */}
          <div className="grid grid-cols-2 gap-2">
            {(["reward", "penalty"] as Kind[]).map((k) => {
              const on = kind === k;
              const tint =
                k === "reward" ? "border-success bg-success-tint text-success" : "border-danger bg-danger-tint text-danger";
              return (
                <button
                  key={k}
                  type="button"
                  aria-pressed={on}
                  onClick={() => {
                    setKind(k);
                    setFormError("");
                  }}
                  className={`rounded-control border-2 p-3 text-start transition-colors ${
                    on ? tint : "border-line bg-surface text-ink hover:bg-row-hover"
                  }`}
                >
                  <span className="block font-bold">{kindLabel(k)}</span>
                  <span className="block text-2xs text-ink-muted">
                    {k === "reward" ? P.kindRewardHint : P.kindPenaltyHint}
                  </span>
                </button>
              );
            })}
          </div>
          {kind === "penalty" && granting && (
            <div className="rounded-control bg-field px-3 py-2 text-sm">
              <p className="text-ink-muted">
                {P.balance}:{" "}
                {granting.balance < 0 ? (
                  <span className="font-bold text-danger">
                    {fill(P.balanceDebt, { b: fmtMoney(-granting.balance) })}
                  </span>
                ) : (
                  <span className="font-bold text-ink" dir="ltr">
                    {fmtMoney(granting.balance)}
                  </span>
                )}
              </p>
              <p className="mt-1 text-xs text-ink-muted">{P.debtHint}</p>
            </div>
          )}
          <Input
            id="incentive-amount"
            label={P.amount}
            type="number"
            inputMode="numeric"
            min={1}
            step={1}
            max={cap > 0 ? cap : undefined}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          {cap > 0 && <p className="text-xs text-ink-muted">{fill(P.amountHint, { cap: fmtMoney(cap) })}</p>}
          {/* **والسببُ إلزاميّ ويصل صاحبَه في الإشعار** — فمن عوقب يعرف بماذا. */}
          <Textarea
            id="incentive-reason"
            label={P.reason}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <FormActions
            onSave={askConfirm}
            onCancel={() => setGranting(null)}
            busy={busy}
            saveLabel={autoApplies ? P.applyNow : P.sendForApproval}
          />
        </div>
      </Modal>

      <Confirm
        open={confirming && granting !== null}
        tone={kind === "penalty" ? "danger" : "primary"}
        title={P.confirmTitle}
        body={
          granting &&
          fill(autoApplies ? P.confirmBodyNow : P.confirmBodyRequest, {
            kind: kindLabel(kind),
            amount: fmtMoney(Number(amount) || 0),
            name: granting.name || granting.phone,
          })
        }
        confirmLabel={autoApplies ? P.applyNow : P.sendForApproval}
        busy={busy}
        onConfirm={() => void submit()}
        onCancel={() => setConfirming(false)}
      />

      <Confirm
        open={approving !== null}
        tone="primary"
        title={P.approveTitle}
        body={
          approving &&
          fill(P.approveBody, {
            kind: kindLabel(approving.kind),
            amount: fmtMoney(approving.amount),
            name: approving.user_name,
            reason: approving.note,
          })
        }
        confirmLabel={P.approve}
        busy={busy}
        onConfirm={() => approving && void decide(approving, true)}
        onCancel={() => setApproving(null)}
      />
      <Confirm
        open={rejecting !== null}
        title={P.rejectTitle}
        body={
          <Input id="inc-reject-note" label={P.rejectNote} value={rejectNote} onChange={(e) => setRejectNote(e.target.value)} />
        }
        confirmLabel={P.reject}
        busy={busy}
        onConfirm={() => rejecting && void decide(rejecting, false)}
        onCancel={() => setRejecting(null)}
      />
      <Confirm
        open={deciding !== null}
        tone={deciding?.d === "clawback" ? "danger" : "primary"}
        title={deciding?.d === "clawback" ? P.decideClawTitle : P.decideKeepTitle}
        body={
          <div className="space-y-2">
            {deciding?.d === "clawback" && (
              <p className="text-sm">{autoApplies ? P.clawbackHintNow : P.clawbackHint}</p>
            )}
            <Input
              id="inc-alert-note"
              label={P.decideNote}
              value={alertNote}
              onChange={(e) => {
                setAlertNote(e.target.value);
                if (e.target.value.trim()) setAlertNoteErr(false);
              }}
            />
            {alertNoteErr && <p className="text-xs text-danger">{P.decideNoteNeeded}</p>}
          </div>
        }
        confirmLabel={deciding?.d === "clawback" ? P.clawback : P.keep}
        busy={busy}
        onConfirm={() => void decideAlert()}
        onCancel={() => setDeciding(null)}
      />

      {/* ── كشفُ الشخص — مَن ومتى ولماذا، آليّاً أو يدويّاً، تحت عناوين الشهور ── */}
      <Drawer
        open={ledgerOf !== null}
        onClose={() => setLedgerOf(null)}
        title={`${P.ledgerTitle} — ${ledgerOf?.name || ledgerOf?.phone || ""}`}
      >
        {ledger === null ? (
          <LoadingState variant="text" />
        ) : ledger === "failed" ? (
          <ReloadState onRetry={() => ledgerOf && setLedgerOf({ ...ledgerOf })} />
        ) : ledger.length === 0 ? (
          <div className="flex flex-col items-center gap-3 p-6 text-center">
            <span className="inline-flex h-14 w-14 items-center justify-center rounded-full bg-field text-ink-muted">
              <IconWallet size={24} />
            </span>
            <h3 className="heading-card">{P.ledgerEmpty}</h3>
            <p className="text-sm text-ink-muted">{P.ledgerEmptyHint}</p>
          </div>
        ) : (
          <div className="space-y-4">
            {ledgerGroups.map((g) => (
              <section key={g.month}>
                <div className="sticky top-0 z-10 rounded-control bg-field px-3 py-2">
                  <p className="flex items-center gap-2 text-sm font-bold">
                    {monthName(g.month)}
                    {g.month === month && <Badge variant="primary">{P.ledgerSelected}</Badge>}
                  </p>
                  <p className="mt-0.5 flex flex-wrap gap-x-3 text-2xs text-ink-muted">
                    <span>
                      {P.auto}: <span className="font-bold text-success">{fmtMoney(g.auto)}</span>
                    </span>
                    <span>
                      {P.manual}: <span className="font-bold text-success">{fmtMoney(g.manual)}</span>
                    </span>
                    <span>
                      {P.penalized}: <span className="font-bold text-danger">{fmtMoney(g.penalty)}</span>
                    </span>
                  </p>
                </div>
                <ul className="divide-y divide-line-soft">
                  {g.list.map((e) => (
                    <li key={e.id} className="space-y-1 py-2 text-sm">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={e.kind === "penalty" ? "danger" : "success"}>{kindLabel(e.kind)}</Badge>
                        {e.kind === "reward" && (
                          <Badge variant={e.for_target ? "primary" : "accent"}>
                            {e.for_target ? P.targetBonus : P.appreciation}
                          </Badge>
                        )}
                        <span
                          className={`font-bold ${e.kind === "penalty" ? "text-danger" : "text-success"}`}
                          dir="ltr"
                        >
                          {fmtMoney(e.amount)}
                        </span>
                        <span className="ms-auto text-xs text-ink-muted" dir="ltr">
                          {fmtDateTime(e.created_at)}
                        </span>
                      </div>
                      <p>{e.reason}</p>
                      <p className="text-xs text-ink-muted">{e.by || P.system}</p>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
        )}
      </Drawer>
    </PageContainer>
  );
}

/**
 * **كرتُ التنبيه** — الاسمُ، والمبلغُ كبيراً، وجملةٌ تقول ما وقع بالشهر باسمه،
 * وزرّان بعرض الكرت. **ومن لا يقرّر يقرأ «القرار للمالية»** بدل فراغ.
 */
function AlertCard({
  a,
  canDecide,
  busy,
  onDecide,
  onRetry,
}: {
  a: IncAlert;
  canDecide: boolean;
  busy: boolean;
  onDecide: (d: "keep" | "clawback") => void;
  onRetry: () => void;
}) {
  const failed = a.kind === "grant_failed";
  const why =
    a.refunded > 0
      ? fill(P.alertRefunded, { n: fmtNum(a.refunded) })
      : a.fake_stores.length > 0
        ? fill(P.alertFake, { names: a.fake_stores.join(P.listSep) })
        : P.alertRemoved;
  return (
    <li className="flex flex-col gap-2 rounded-card border border-danger p-3 text-sm">
      <div className="flex items-start gap-2">
        <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-danger-solid text-on-solid">
          <IconWarning size={16} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{a.name}</span>
          <span className="block text-2xs text-ink-muted">{failed ? P.alertDue : P.alertPaid}</span>
        </span>
        <span className="figure shrink-0 tabular-nums" dir="ltr">
          {fmtMoney(a.amount)}
        </span>
      </div>
      <p>
        {failed
          ? fill(P.alertFailed, { month: monthName(a.month), attempts: fmtNum(a.attempts ?? 1) })
          : fill(P.alertDropped, {
              level: fmtNum(a.level),
              month: monthName(a.month),
              target: fmtNum(a.target_count),
              current: fmtNum(a.current),
            })}
      </p>
      {!failed && <p className="text-xs text-ink-muted">{why}</p>}
      <div className="mt-auto border-t border-line-soft pt-2">
        {!canDecide ? (
          <p className="text-center text-xs text-ink-muted">{P.decideOnlyFinance}</p>
        ) : failed ? (
          <Button className="w-full" disabled={busy} onClick={onRetry}>
            {P.retry}
          </Button>
        ) : (
          <div className="grid grid-cols-2 gap-2">
            <Button variant="secondary" className="w-full" disabled={busy} onClick={() => onDecide("keep")}>
              {P.keep}
            </Button>
            <Button variant="danger" className="w-full" disabled={busy} onClick={() => onDecide("clawback")}>
              {P.clawback}
            </Button>
          </div>
        )}
      </div>
    </li>
  );
}

/**
 * **كرتُ الشخص** — تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) بدل صفّ الجدول: الاسمُ والمرحلة،
 * والإنجازُ بشريطٍ وكم باقي للمرحلة الجاية، والآليُّ واليدويُّ والعقوباتُ بثلاث مربّعات، والزرّان.
 */
function PersonCard({
  x,
  levels,
  unit,
  canGrant,
  onLedger,
  onGrant,
}: {
  x: Standing;
  levels: Level[];
  unit: string;
  canGrant: boolean;
  onLedger: () => void;
  onGrant: () => void;
}) {
  const name = x.name || x.phone;
  const pct = x.target > 0 ? Math.min(100, Math.round((x.done / x.target) * 100)) : 0;
  // **المرحلةُ الجاية** — للعرض وحدَه؛ المكافأةُ يحسبها الخادم.
  const next = levels.find((l) => l.n > x.level);
  const remaining = next ? Math.max(0, next.target - x.done) : 0;
  return (
    <li className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <span
          className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full font-bold ${
            x.level > 0 ? "bg-success-solid text-on-solid" : "bg-field text-ink"
          }`}
          aria-hidden
        >
          {name.trim().charAt(0)}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{name}</span>
          {x.name && (
            <span className="block text-2xs text-ink-muted" dir="ltr">
              {x.phone}
            </span>
          )}
        </span>
        {x.levels > 0 && (
          <Badge variant={x.level > 0 ? "success" : "neutral"} className="shrink-0">
            {x.level > 0 ? fill(P.stageReached, { n: fmtNum(x.level), total: fmtNum(x.levels) }) : P.stageNone}
          </Badge>
        )}
      </div>

      <div className="rounded-control bg-field px-3 py-2.5">
        {x.target > 0 ? (
          <>
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-2xs text-ink-muted">{P.progress}</span>
              {/* **والوحدةُ بوحدة الدور** — عملاءُ للمندوب لا متاجر (٢٠٢٦-١٠-١٥). */}
              <span className="font-bold tabular-nums">
                {fill(P.progressOf, { done: fmtNum(x.done), target: fmtNum(x.target), unit })}
              </span>
            </div>
            <div className="mt-2 h-2 overflow-hidden rounded-full bg-surface" aria-hidden>
              <div
                className={`h-full rounded-full ${pct >= 100 ? "bg-success-solid" : "bg-accent"}`}
                style={{ width: `${pct}%` }}
              />
            </div>
            {x.levels > 0 && (
              <p className="mt-1.5 text-2xs text-ink-muted">
                {next
                  ? remaining > 0
                    ? fill(P.nextStage, { n: fmtNum(remaining), unit, stage: fmtNum(next.n) })
                    : ""
                  : P.allStages}
              </p>
            )}
          </>
        ) : (
          <span className="text-xs text-ink-muted">{P.noTarget}</span>
        )}
        {(x.pending ?? 0) > 0 && (
          <p className="mt-1 text-2xs text-warning">{fill(P.pendingStores, { n: fmtNum(x.pending ?? 0) })}</p>
        )}
      </div>

      <div className="flex gap-2 text-center">
        <span className="min-w-0 flex-1 rounded-control border border-line-soft px-2 py-2">
          <span className="block text-2xs text-ink-muted">{P.auto}</span>
          <span className="block font-bold tabular-nums text-success" dir="ltr">
            {fmtMoney(x.auto_paid)}
          </span>
        </span>
        <span className="min-w-0 flex-1 rounded-control border border-line-soft px-2 py-2">
          <span className="block text-2xs text-ink-muted">{P.manual}</span>
          <span className="block font-bold tabular-nums text-success" dir="ltr">
            {fmtMoney(x.manual_paid)}
          </span>
        </span>
        <span className="min-w-0 flex-1 rounded-control border border-line-soft px-2 py-2">
          <span className="block text-2xs text-ink-muted">{P.penalized}</span>
          <span className="block font-bold tabular-nums text-danger" dir="ltr">
            {fmtMoney(x.penalized)}
          </span>
        </span>
      </div>

      <div className={`mt-auto grid gap-2 border-t border-line-soft pt-3 ${canGrant ? "grid-cols-2" : "grid-cols-1"}`}>
        <Button variant="secondary" className="w-full" onClick={onLedger}>
          {P.openLedger}
        </Button>
        {canGrant && (
          <Button className="w-full" onClick={onGrant}>
            {P.grant}
          </Button>
        )}
      </div>
    </li>
  );
}
