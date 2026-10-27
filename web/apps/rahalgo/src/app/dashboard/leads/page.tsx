"use client";

/**
 * **طلبات انضمام المتاجر** — متاجرُ أضافها المندوبون وتنتظر قرارَ المكتب.
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤.) **للمتاجر التي يضيفها المندوبون وحدَها** —
 * والخادمُ يرشّح ذلك. بحثٌ وفلاتر محفوظةٌ في الرابط · «ينتظر منذ» · تحذيرُ الرقم
 * المكرّر والمتجر القريب بالاسم نفسه · **تأكيدٌ قبل الموافقة** · «بحاجة معلومات»
 * تُشعر المندوبَ بملاحظة · أسبابُ رفضٍ جاهزة.
 *
 * **والتصميمُ الاحترافيّ** (المالك ٢٠٢٦-١٠-٢٧، تدقيقُ اللوحة): رأسٌ موحّد · أربعةُ أرقامٍ تُضغط
 * بعدِّ كلّ حالة · الفلاترُ ببطاقةٍ واحدةٍ بحبّات · كرتٌ لكلّ طلبٍ بأفعالٍ بعرضه · فراغٌ لكلّ حالة.
 */

import { Suspense, useCallback, useEffect, useMemo, useState, type ComponentType } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { getMessages, defaultLocale, fmtDateTime, fmtNum, fmtSpan, errorText } from "@rahalgo/i18n";
import {
  useLiveRefresh,
  Button,
  ButtonLink,
  Badge,
  CategoryIcon,
  Pagination,
  PageContainer,
  LoadingState,
  ReloadState,
  IconStore,
  IconUser,
  IconPromos,
  IconStatus,
  IconLink,
  IconLocation,
  IconSuccess,
  IconBlock,
  IconUnblock,
  IconSearch,
  IconHourglass,
  IconWarning,
  IconWhatsApp,
  IconDate,
  Modal,
  Input,
  Select,
  Alert,
  Confirm,
  FormActions,
  Chips,
  Switch,
} from "@rahalgo/ui";
import { TempPasswordNote } from "@/components/admin/accounts/TempPasswordNote";
import { PageHero } from "@/components/admin/PageHero";
import { Tile, type TileTone } from "@/components/admin/DashParts";
import { api } from "@/lib/api";
import { useCanCall, useCanOpen } from "@/lib/policy";

const m = getMessages(defaultLocale);
const L = m.admin.leads;

type LeadStatus = "new" | "needs_info" | "converted" | "rejected";
const STATUSES: LeadStatus[] = ["new", "needs_info", "converted", "rejected"];

interface Lead {
  id: string;
  store_name: string;
  owner_name: string;
  phone: string;
  area: string;
  district: string;
  category_name: string | null;
  category_icon: string | null;
  lat: number | null;
  lng: number | null;
  rep_name: string | null;
  rep_code: string | null;
  status: LeadStatus;
  /** ما كتبه المندوبُ حين أرسل — **يُقرأ قبل الردّ، فقد يكون فيه جوابُ سؤالك.** */
  note?: string;
  /** سببُ الرفض أو ما ينقص — صوتُ الإدارة. */
  decision_note?: string;
  created_at: string;
  duplicate_phone: string[];
  nearby_same_name: string[];
  existing_account: boolean;
  /** **المتجرُ الذي صار منه** — للمقبول وحدَه. */
  merchant_id?: string | null;
}

interface Named {
  id: string;
  name: string;
}

interface LeadsRes {
  leads: Lead[];
  total: number;
  per_page: number;
  reps: Named[];
  governorates?: Named[];
  categories?: Named[];
  counts?: Partial<Record<LeadStatus, number>>;
  /** **يومُ المنصّة** — يبدأ بساعة فتح الدوام كفلتر التاريخ في الخادم. */
  today?: string;
  day_start_hour?: number;
}

const STATUS_VARIANT: Record<LeadStatus, "warning" | "success" | "danger" | "info"> = {
  new: "warning",
  needs_info: "info",
  converted: "success",
  rejected: "danger",
};

const STATUS_TINT: Record<LeadStatus, string> = {
  new: "bg-warning-solid text-on-solid",
  needs_info: "bg-accent text-on-bright",
  converted: "bg-success-solid text-on-solid",
  rejected: "bg-field text-ink-muted",
};

const STATUS_ICON: Record<LeadStatus | "all", ComponentType<{ size?: number; className?: string }>> = {
  new: IconHourglass,
  needs_info: IconWarning,
  converted: IconSuccess,
  rejected: IconBlock,
  all: IconStatus,
};

const TILE_TONE: Record<LeadStatus, TileTone> = {
  new: "warning",
  needs_info: "accent",
  converted: "success",
  rejected: "danger",
};

/** مفاتيحُ الرابط — **الفلاترُ تُحفظ فيه** فيُشارَك ويُعاد إليه. */
const KEYS = ["status", "q", "rep_id", "governorate_id", "category_id", "from", "to", "page"] as const;
type Key = (typeof KEYS)[number];

type Period = "all" | "today" | "week" | "month" | "custom";

/** **حسابُ التواريخ نصّاً** — `YYYY-MM-DD` من يوم المنصّة الذي يرسله الخادم، بلا منطقةٍ زمنيّة. */
function shiftDay(day: string, delta: number): string {
  const d = new Date(`${day}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + delta);
  return d.toISOString().slice(0, 10);
}

function periodRange(p: Period, today: string): { from: string; to: string } {
  switch (p) {
    case "today":
      return { from: today, to: today };
    case "week":
      return { from: shiftDay(today, -6), to: today };
    case "month":
      return { from: `${today.slice(0, 8)}01`, to: today };
    default:
      return { from: "", to: "" };
  }
}

/** **أيُّ حبّةٍ تطابق الرابط** — ومدىً لا يطابق أيّاً منها «تواريخ مخصّصة». */
function periodOf(from: string, to: string, today: string): Period {
  if (!from && !to) return "all";
  if (today) {
    for (const p of ["today", "week", "month"] as const) {
      const r = periodRange(p, today);
      if (r.from === from && r.to === to) return p;
    }
  }
  return "custom";
}

/** «ينتظر منذ» — أيّامٌ إن بلغ يوماً، وإلّا ساعاتٌ ودقائق. */
function waitingFor(iso: string): string {
  const secs = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  const days = Math.floor(secs / 86400);
  return days >= 1 ? L.days.replace("{n}", fmtNum(days)) : fmtSpan(secs);
}

/** **الرقمُ مقروءاً** — `0900 555 106` بدل `+963900555106`. */
function prettyPhone(raw: string): string {
  let d = raw.replace(/[^\d]/g, "");
  if (d.startsWith("963")) d = `0${d.slice(3)}`;
  if (d.length === 10) return `${d.slice(0, 4)} ${d.slice(4, 7)} ${d.slice(7)}`;
  return raw;
}

function waLink(raw: string): string {
  return `https://wa.me/${raw.replace(/[^\d]/g, "").replace(/^0/, "963")}`;
}

/**
 * **مفتاحُ القبول التلقائي في الرأس** — البابُ نفسُه (`/admin/auto-mode/leads`) الذي يقرؤه
 * `AutoModeToggle` المشترك، **لكن بتأكيدٍ قبل التشغيل وشريطٍ ظاهرٍ وهو شغّال** (تدقيقُ اللوحة
 * ٢٠٢٦-١٠-٢٧): كان الشرحُ تلميحاً لا يُرى على الجوّال، والتشغيلُ بضغطةٍ يفتح متاجرَ بلا مراجعة.
 */
function useLeadsAutoMode() {
  const canRead = useCanCall()("GET", "/auto-mode/leads");
  const [row, setRow] = useState<{ on: boolean; editable: boolean } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!canRead) return;
    try {
      const res = await api<{ on: boolean; editable: boolean }>("/api/v1/admin/auto-mode/leads");
      setRow({ on: res.on === true, editable: res.editable === true });
    } catch {
      setRow(null);
    }
  }, [canRead]);

  useEffect(() => {
    void load();
  }, [load]);
  useLiveRefresh(["settings"], () => void load());

  async function set(next: boolean) {
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/auto-mode/leads", { method: "PUT", body: JSON.stringify({ on: next }) });
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return { row, busy, error, set, clearError: () => setError("") };
}

function LeadsScreen() {
  const params = useSearchParams();
  // **وملفُّ الحساب لمن يقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥): موظّفُ العمليّات
  // لا يملك `users.read` — **فلا يُرسَم له رابطٌ يُردّ ٤٠٣.**
  const open = useCanOpen();
  const canProfile = open.user;
  const router = useRouter();
  const pathname = usePathname();
  const get = (k: Key) => params.get(k) ?? "";
  /** **والقسمُ يفتح على «جديد»** — المعلَّقُ لا المنتهي (قرارُ المالك ٢٠٢٦-٠٨-٠٣). */
  const status = get("status") || "new";
  const page = Number(get("page")) || 1;

  const setParams = useCallback(
    (patch: Partial<Record<Key, string>>) => {
      const next = new URLSearchParams(params.toString());
      for (const [k, v] of Object.entries(patch)) {
        if (v) next.set(k, v);
        else next.delete(k);
      }
      // **ومن بدّل فلتراً يعود للصفحة الأولى** — رابعةٌ قد لا توجد.
      if (!("page" in patch)) next.delete("page");
      const qs = next.toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [params, pathname, router],
  );

  const [data, setData] = useState<LeadsRes | null>(null);
  // **ولا «لا طلبات» قبل أن يصل الردّ** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣).
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState<{ tone: "success" | "warning"; text: string; owner?: string; temp?: string } | null>(null);
  const [busy, setBusy] = useState("");
  /** حقلُ البحث يُكتب محلّيّاً ثمّ يُرسَل للرابط بعد توقّفٍ قصير. */
  const [q, setQ] = useState(get("q"));
  /** **«تواريخ مخصّصة» تُفتح قبل أن يُختار تاريخ** — وإلّا لا تظهر الخانتان. */
  const [customOpen, setCustomOpen] = useState(false);

  /** الطلبُ الذي يُوافَق عليه الآن — **لا موافقةَ بلا تأكيد.** */
  const [approving, setApproving] = useState<Lead | null>(null);
  /** **وإعادةُ الفتح تمحو ملاحظةَ الإدارة** — فلا تقع بلا سؤال. */
  const [reopening, setReopening] = useState<Lead | null>(null);
  /** الطلبُ الذي يُردّ أو يُعاد للمندوب — **ولا يُرسَل حتى تُكتب كلمة.** */
  const [deciding, setDeciding] = useState<{ lead: Lead; status: "rejected" | "needs_info" } | null>(null);
  const [note, setNote] = useState("");
  const [autoConfirm, setAutoConfirm] = useState(false);
  const auto = useLeadsAutoMode();

  useEffect(() => {
    const t = setTimeout(() => {
      if (q.trim() !== get("q")) setParams({ q: q.trim() });
    }, 400);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q]);

  const query = useMemo(() => {
    const p = new URLSearchParams();
    p.set("status", status === "all" ? "" : status);
    for (const k of ["q", "rep_id", "governorate_id", "category_id", "from", "to"] as const) {
      const v = params.get(k);
      if (v) p.set(k, v);
    }
    p.set("page", String(page));
    return p.toString();
  }, [params, status, page]);

  const load = useCallback(async () => {
    try {
      const res = await api<LeadsRes>(`/api/v1/admin/leads?${query}`);
      setData(res ?? null);
      setError("");
    } catch (err) {
      setError(errorText(err));
    } finally {
      setLoaded(true);
    }
  }, [query]);

  useEffect(() => {
    void load();
  }, [load]);

  useLiveRefresh(["lead"], load);

  async function setStatus(id: string, next: LeadStatus, text = "") {
    if (busy) return;
    setBusy(id);
    setError("");
    setNotice(null);
    try {
      const res = await api<{ welcome?: { sent?: boolean; user_id?: string; temp_password?: string } }>(
        `/api/v1/admin/leads/${id}/status`,
        { method: "POST", body: JSON.stringify({ status: next, note: text }) },
      );
      if (next === "converted") {
        const w = res?.welcome;
        setNotice(
          w?.sent
            ? { tone: "success", text: L.approvedSent }
            : { tone: "warning", text: L.approvedNotSent, owner: w?.user_id, temp: w?.temp_password },
        );
      }
      setDeciding(null);
      setApproving(null);
      setReopening(null);
      setNote("");
    } catch (err) {
      setError(errorText(err));
      setDeciding(null);
      setApproving(null);
      setReopening(null);
    } finally {
      setBusy("");
      // **وتُعاد القراءةُ في الحالين** — قائمةٌ لا تُحدَّث بعد فشلٍ تُري حالاً قد تبدّل.
      await load();
    }
  }

  const leads = data?.leads ?? [];
  const reps = data?.reps ?? [];
  const govs = data?.governorates ?? [];
  const cats = data?.categories ?? [];
  const counts = data?.counts ?? {};
  const today = data?.today ?? "";
  const count = data?.total ?? 0;
  const perPage = data?.per_page || 20;
  const filtered = KEYS.some((k) => k !== "status" && k !== "page" && params.get(k));
  const period = customOpen ? "custom" : periodOf(get("from"), get("to"), today);
  const allCount = STATUSES.reduce((s, k) => s + (counts[k] ?? 0), 0);

  function clearFilters() {
    setQ("");
    setCustomOpen(false);
    setParams({ q: "", rep_id: "", governorate_id: "", category_id: "", from: "", to: "" });
  }

  function pickPeriod(p: Period) {
    if (p === "custom") {
      setCustomOpen(true);
      return;
    }
    setCustomOpen(false);
    setParams(periodRange(p, today));
  }

  if (!loaded) return <LoadingState />;

  const repChips = [{ id: "", label: L.allReps }, ...reps.map((r) => ({ id: r.id, label: r.name }))];
  const periodChips = (["all", "today", "week", "month", "custom"] as const)
    // **ولا حبّاتُ مدىً قبل أن يُعرف يومُ المنصّة** — «اليوم» بيوم الجهاز يخالف فلترَ الخادم.
    .filter((p) => p === "all" || p === "custom" || today !== "")
    .map((p) => ({ id: p, label: L.periods[p] }));

  return (
    <PageContainer>
      <PageHero<string>
        icon={IconLink}
        title={L.title}
        subtitle={L.subtitle}
        actions={
          auto.row?.editable ? (
            <div className="rounded-full border border-line px-3 py-1.5">
              <Switch
                checked={auto.row.on}
                disabled={auto.busy}
                label={`${L.autoTitle}: ${auto.row.on ? L.autoOn : L.autoOff}`}
                // **التشغيلُ بتأكيد، والإطفاءُ فوراً** — الإطفاءُ يُرجع المراجعة ولا يفتح شيئاً.
                onChange={(v) => (v ? setAutoConfirm(true) : void auto.set(false))}
              />
            </div>
          ) : undefined
        }
        tabs={[
          ...STATUSES.map((s) => ({
            key: s,
            label: L.st[s],
            icon: STATUS_ICON[s],
            count: data?.counts ? (counts[s] ?? 0) : null,
            warn: s === "new",
          })),
          { key: "all", label: L.filterAll, icon: STATUS_ICON.all, count: data?.counts ? allCount : null },
        ]}
        value={status}
        onChange={(k) => setParams({ status: k === "new" ? "" : k })}
      />

      {/* **وهو شغّالٌ يُقال بوضوح** — لا تلميحٌ مخفيّ. */}
      {auto.row?.on && (
        <Alert tone="warning" className="mb-4">
          {L.autoOnBanner}
        </Alert>
      )}
      {auto.error && (
        <Alert className="mb-4" onDismiss={auto.clearError}>
          {auto.error}
        </Alert>
      )}

      {/* **أربعةُ أرقامٍ تُضغط** — عددُ كلّ حالةٍ بالفلاتر نفسِها، و«جديدة» هي الأهمّ. */}
      {data?.counts && (
        <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
          {STATUSES.map((s) => (
            <Tile
              key={s}
              icon={STATUS_ICON[s]}
              label={L.tiles[s]}
              value={fmtNum(counts[s] ?? 0)}
              tone={(counts[s] ?? 0) > 0 ? TILE_TONE[s] : "muted"}
              onClick={() => setParams({ status: s === "new" ? "" : s })}
            />
          ))}
        </div>
      )}

      {/* **وسببُ الخادم يُعرض بنصّه** — جوابٌ يُصلَح به. */}
      {error && data && (
        <Alert className="mb-4" onDismiss={() => setError("")}>
          {error}
        </Alert>
      )}
      {notice && (
        <Alert tone={notice.tone} className="mb-4" onDismiss={() => setNotice(null)}>
          {notice.text}
          <TempPasswordNote value={notice.temp} />
          {canProfile && notice.owner && (
            <>
              {" — "}
              <Link href={`/dashboard/users/${notice.owner}`} className="font-medium underline">
                {L.openOwner}
              </Link>
            </>
          )}
        </Alert>
      )}

      {/* **الفلاترُ ببطاقةٍ واحدة** — عنوانٌ وعددُ النتائج، بحثٌ بعرضها، والمندوبُ والمدّةُ حبّات. */}
      <section className="surface mb-4 space-y-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="heading-card">
            {L.filtersTitle}
            <span className="ms-2 text-xs font-normal text-ink-muted">{L.resultsCount.replace("{n}", fmtNum(count))}</span>
          </h2>
          {(filtered || customOpen) && (
            <Button variant="ghost" onClick={clearFilters}>
              {L.clear}
            </Button>
          )}
        </div>
        <Input
          id="lead-search"
          icon={<IconSearch size={16} />}
          placeholder={L.search}
          aria-label={L.search}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {reps.length > 1 && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm text-ink-muted">{L.rep}:</span>
            <Chips wrap items={repChips} value={get("rep_id")} onChange={(id) => setParams({ rep_id: id })} />
          </div>
        )}
        {/* **والمحافظةُ والتصنيفُ ممّا في الطلبات نفسِها** — ويُخفى المرشِّحُ إن لم يبقَ فيه إلّا خيارٌ واحد. */}
        {(govs.length > 1 || cats.length > 1) && (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {govs.length > 1 && (
              <Select
                aria-label={L.allGovs}
                value={get("governorate_id")}
                onChange={(e) => setParams({ governorate_id: e.target.value })}
              >
                <option value="">{L.allGovs}</option>
                {govs.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.name}
                  </option>
                ))}
              </Select>
            )}
            {cats.length > 1 && (
              <Select aria-label={L.category} value={get("category_id")} onChange={(e) => setParams({ category_id: e.target.value })}>
                <option value="">{L.allCats}</option>
                {cats.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </Select>
            )}
          </div>
        )}
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-ink-muted">{L.period}:</span>
          <Chips wrap items={periodChips} value={period} onChange={pickPeriod} />
        </div>
        {period === "custom" && (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Input id="lead-from" type="date" label={L.from} value={get("from")} onChange={(e) => setParams({ from: e.target.value })} />
            <Input id="lead-to" type="date" label={L.to} value={get("to")} onChange={(e) => setParams({ to: e.target.value })} />
          </div>
        )}
        {/* **والتاريخُ بيوم المنصّة** — طلبُ الثالثة فجراً يُحسب على اليوم السابق، فيُقال ذلك. */}
        {period !== "all" && data?.day_start_hour != null && (
          <p className="text-2xs text-ink-muted">{L.dayStartNote.replace("{h}", fmtNum(data.day_start_hour))}</p>
        )}
      </section>

      {error && !data ? (
        <ReloadState onRetry={() => void load()} label={L.loadError} />
      ) : leads.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span
            className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${
              filtered ? "bg-field text-ink-muted" : "bg-success-solid text-on-solid"
            }`}
          >
            {filtered ? <IconSearch size={28} /> : <IconStore size={28} />}
          </span>
          <h2 className="heading-section">
            {filtered ? L.emptyFiltered : L.emptyByStatus[status as LeadStatus | "all"] ?? L.empty}
          </h2>
          <p className="max-w-md text-sm text-ink-muted">{filtered ? L.emptyFilteredHint : L.emptyHint}</p>
          {filtered && (
            <Button variant="secondary" onClick={clearFilters}>
              {L.clear}
            </Button>
          )}
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {leads.map((l) => (
            <LeadCard
              key={l.id}
              l={l}
              busy={busy === l.id}
              canMerchant={open.merchant}
              onApprove={() => setApproving(l)}
              onNeedsInfo={() => setDeciding({ lead: l, status: "needs_info" })}
              onReject={() => setDeciding({ lead: l, status: "rejected" })}
              onReopen={() => setReopening(l)}
            />
          ))}
        </ul>
      )}

      {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
      {count > perPage && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={count} perPage={perPage} onChange={(p) => setParams({ page: p > 1 ? String(p) : "" })} />
        </div>
      )}

      {/* **تأكيدٌ قبل الموافقة** — يقول ما سيقع والتحذيرات وملاحظةَ المندوب قبل الضغط. */}
      <Confirm
        open={approving != null}
        tone="primary"
        busy={busy !== ""}
        title={L.confirmTitle.replace("{name}", approving?.store_name ?? "")}
        confirmLabel={L.markConverted}
        onCancel={() => setApproving(null)}
        onConfirm={() => approving && void setStatus(approving.id, "converted")}
        body={
          approving && (
            <div className="space-y-2">
              <p>{L.confirmBody}</p>
              <p>{approving.existing_account ? L.existingAccount : L.confirmNewOwner}</p>
              {approving.note && <NoteLine label={L.repNote} text={approving.note} />}
              {(approving.duplicate_phone.length > 0 || approving.nearby_same_name.length > 0) && (
                <div className="space-y-1">
                  <p className="font-medium text-warning">{L.confirmWarnings}</p>
                  <Warnings l={approving} />
                </div>
              )}
            </div>
          )
        }
      />

      <Confirm
        open={reopening != null}
        tone="primary"
        busy={busy !== ""}
        title={L.reopenConfirmTitle.replace("{name}", reopening?.store_name ?? "")}
        confirmLabel={L.reopen}
        onCancel={() => setReopening(null)}
        onConfirm={() => reopening && void setStatus(reopening.id, "new")}
        body={<p>{L.reopenConfirmBody}</p>}
      />

      <Confirm
        open={autoConfirm}
        tone="danger"
        busy={auto.busy}
        title={L.autoConfirmTitle}
        confirmLabel={L.autoConfirmLabel}
        onCancel={() => setAutoConfirm(false)}
        onConfirm={() => {
          setAutoConfirm(false);
          void auto.set(true);
        }}
        body={<p>{L.autoConfirmBody}</p>}
      />

      {deciding && (
        <Modal
          open
          onClose={() => {
            setDeciding(null);
            setNote("");
          }}
          title={`${deciding.status === "rejected" ? L.markRejected : L.markNeedsInfo}: ${deciding.lead.store_name}`}
        >
          <div className="space-y-3">
            <p className="text-sm text-ink-muted">{deciding.status === "rejected" ? L.rejectHint : L.needsInfoHint}</p>
            {/* **وما كتبه المندوبُ يُقرأ قبل الردّ** — قد يكون فيه جوابُ سؤالك. */}
            {deciding.lead.note && <NoteLine label={L.repNote} text={deciding.lead.note} />}
            {deciding.status === "rejected" && (
              <div>
                <p className="mb-1 text-sm font-medium">{L.reasonsLabel}</p>
                <Chips
                  wrap
                  items={L.rejectReasons.map((r) => ({ id: r, label: r }))}
                  value={note}
                  onChange={(r) => setNote(r)}
                />
              </div>
            )}
            <Input
              id="lead-decision-note"
              label={deciding.status === "rejected" ? L.rejectNote : L.needsInfoNote}
              required
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <FormActions
              busy={busy !== ""}
              onSave={() => note.trim() && void setStatus(deciding.lead.id, deciding.status, note.trim())}
              onCancel={() => {
                setDeciding(null);
                setNote("");
              }}
              saveLabel={deciding.status === "rejected" ? L.markRejected : L.markNeedsInfo}
              tone={deciding.status === "rejected" ? "danger" : undefined}
            />
          </div>
        </Modal>
      )}
    </PageContainer>
  );
}

function NoteLine({ label, text }: { label: string; text: string }) {
  return (
    <p className="border-s-2 border-line ps-2 text-sm">
      <span className="text-2xs text-ink-muted">{label}: </span>
      {text}
    </p>
  );
}

/** **تحذيرا الرقم المكرّر والمتجر القريب** — سطرٌ لكلٍّ منهما. */
function Warnings({ l }: { l: Lead }) {
  return (
    <>
      {l.duplicate_phone.length > 0 && (
        <span className="flex items-start gap-1.5 text-xs text-warning">
          <IconWarning size={14} className="mt-0.5 shrink-0" />
          {L.warnDupPhone.replace("{names}", l.duplicate_phone.join(m.common.listSep))}
        </span>
      )}
      {l.nearby_same_name.length > 0 && (
        <span className="flex items-start gap-1.5 text-xs text-warning">
          <IconWarning size={14} className="mt-0.5 shrink-0" />
          {L.warnNearby.replace("{names}", l.nearby_same_name.join(m.common.listSep))}
        </span>
      )}
    </>
  );
}

/** **كرتُ طلبٍ واحد** — اسمُه وحالُه، «ناطر من…» كبيراً، صاحبُه ورقمُه، عنوانُه، مندوبُه، ملاحظاتُه، وأفعالُه بعرضه. */
function LeadCard({
  l,
  busy,
  canMerchant,
  onApprove,
  onNeedsInfo,
  onReject,
  onReopen,
}: {
  l: Lead;
  busy: boolean;
  canMerchant: boolean;
  onApprove: () => void;
  onNeedsInfo: () => void;
  onReject: () => void;
  onReopen: () => void;
}) {
  const router = useRouter();
  const pending = l.status === "new" || l.status === "needs_info";
  const hasWarn = l.duplicate_phone.length > 0 || l.nearby_same_name.length > 0;
  // **والعنوانُ لا يتكرّر** — «مركز الرقة، الرقة — مركز الرقة»: التفصيلُ يُكتب إن زاد شيئاً.
  const detail = l.area && !l.district.includes(l.area) ? l.area : "";
  const hasMap = l.lat != null && l.lng != null;

  return (
    <li className="surface flex flex-col gap-3 p-4">
      {hasWarn && (
        <div className="space-y-1 rounded-control border border-warning bg-warning-tint px-3 py-2">
          <Warnings l={l} />
        </div>
      )}

      <div className="flex items-start gap-3">
        <span className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${STATUS_TINT[l.status]}`}>
          {/* **والأيقونةُ تُرسم لا تُطبع** — `category_icon` مفتاحٌ لاتينيّ. */}
          {l.category_icon ? <CategoryIcon name={l.category_icon} size={20} /> : <IconStore size={20} />}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{l.store_name}</span>
          {l.category_name && <span className="block text-2xs text-ink-muted">{l.category_name}</span>}
        </span>
        <Badge variant={STATUS_VARIANT[l.status]}>{L.st[l.status]}</Badge>
      </div>

      {/* **«ناطر من…» هو الرقمُ الأهمّ للمعلَّق** — وتاريخُ التسجيل تحته لكلّ طلب
          (قرارُ المالك ٢٠٢٦-١٠-٢٦: «لازم هذا تاريخ التسجيل والانضمام»). */}
      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span className="min-w-0">
          <span className="block text-2xs text-ink-muted">{pending ? L.waiting : L.registered}</span>
          {pending ? (
            <span className="figure block text-warning">{waitingFor(l.created_at)}</span>
          ) : (
            <span className="block font-bold" dir="ltr">
              {fmtDateTime(l.created_at)}
            </span>
          )}
        </span>
        {pending && (
          <span className="text-end text-2xs text-ink-muted">
            <span className="flex items-center justify-end gap-1">
              <IconDate size={12} />
              {L.registered}
            </span>
            <span dir="ltr">{fmtDateTime(l.created_at)}</span>
          </span>
        )}
      </div>

      <div className="space-y-2 text-sm">
        <div className="flex items-center justify-between gap-2">
          <span className="flex min-w-0 items-center gap-2">
            <IconUser size={15} className="shrink-0 text-ink-muted" />
            <span className="min-w-0">
              <span className="block truncate">{l.owner_name || "—"}</span>
              <span className="block text-xs tabular-nums text-ink-muted" dir="ltr">
                {prettyPhone(l.phone)}
              </span>
            </span>
          </span>
          <ButtonLink href={waLink(l.phone)} target="_blank" rel="noreferrer" variant="secondary" className="shrink-0 !px-3 !py-1.5">
            <IconWhatsApp size={15} className="text-success" />
            {L.whatsapp}
          </ButtonLink>
        </div>

        <div className="flex items-center justify-between gap-2">
          <span className="flex min-w-0 items-center gap-2">
            <IconLocation size={15} className="shrink-0 text-ink-muted" />
            <span className="min-w-0">
              <span className="block">{l.district || l.area || "—"}</span>
              {detail && <span className="block text-xs text-ink-muted">{detail}</span>}
            </span>
          </span>
          {hasMap && (
            <ButtonLink
              href={`https://www.google.com/maps?q=${l.lat},${l.lng}`}
              target="_blank"
              rel="noreferrer"
              variant="secondary"
              className="shrink-0 !px-3 !py-1.5"
            >
              <IconLocation size={15} />
              {L.onMap}
            </ButtonLink>
          )}
        </div>

        <div className="flex items-center gap-2">
          <IconPromos size={15} className="shrink-0 text-ink-muted" />
          <span className="text-ink-muted">{L.rep}:</span>
          <span className="truncate">{l.rep_name ?? "—"}</span>
          {l.rep_code && (
            <Badge variant="accent" className="shrink-0">
              {L.code}{" "}
              <span dir="ltr" className="font-mono">
                {l.rep_code}
              </span>
            </Badge>
          )}
        </div>
      </div>

      {(l.note || (l.decision_note && l.status !== "converted")) && (
        <div className="space-y-1">
          {l.note && <NoteLine label={L.repNote} text={l.note} />}
          {l.decision_note && l.status !== "converted" && <NoteLine label={L.decisionNote} text={l.decision_note} />}
        </div>
      )}

      {/* **«بحاجة معلومات» تُشعر المندوبَ ولا تُرجع الطلبَ لتطبيقه** (تدقيقُ اللوحة ٢٠٢٦-١٠-٢٧) —
          فيُقال للمكتب أنّ الخطوةَ التالية عنده. */}
      {l.status === "needs_info" && (
        <p className="rounded-control bg-info-tint px-3 py-2 text-xs text-info">{L.awaitingRep}</p>
      )}

      {l.status === "converted" ? (
        /* **وما حُوِّل لا يُردّ** — المتجرُ قائمٌ يبيع، ورابطُه بدل الأفعال. */
        l.merchant_id && canMerchant ? (
          <div className="mt-auto border-t border-line-soft pt-3">
            <Button
              variant="secondary"
              className="flex w-full items-center justify-center gap-1.5"
              onClick={() => router.push(`/dashboard/merchants/${l.merchant_id}`)}
            >
              <IconStore size={15} />
              {L.openStore}
            </Button>
          </div>
        ) : null
      ) : (
        <div className="mt-auto space-y-2 border-t border-line-soft pt-3">
          <Button className="flex w-full items-center justify-center gap-1.5" disabled={busy} onClick={onApprove}>
            <IconSuccess size={15} />
            {L.markConverted}
          </Button>
          {/* **والمرفوضُ له زرٌّ واحد** («إعادة فتح») — بعرض البطاقة لا نصفِها. */}
          <div className={`grid gap-2 ${l.status === "rejected" ? "grid-cols-1" : "grid-cols-2"}`}>
            {l.status === "new" && (
              <Button variant="secondary" disabled={busy} onClick={onNeedsInfo} className="flex w-full items-center justify-center gap-1.5">
                <IconHourglass size={15} />
                {L.markNeedsInfo}
              </Button>
            )}
            {l.status !== "new" && (
              <Button variant="secondary" disabled={busy} onClick={onReopen} className="flex w-full items-center justify-center gap-1.5">
                <IconUnblock size={15} />
                {L.reopen}
              </Button>
            )}
            {l.status !== "rejected" ? (
              /* **والردُّ يلزمه كلمة** — المندوبُ يقرؤها في إشعاره. */
              <Button variant="danger" disabled={busy} onClick={onReject} className="flex w-full items-center justify-center gap-1.5">
                <IconBlock size={15} />
                {L.markRejected}
              </Button>
            ) : null}
          </div>
        </div>
      )}
    </li>
  );
}

export default function LeadsPage() {
  return (
    <Suspense>
      <LeadsScreen />
    </Suspense>
  );
}
