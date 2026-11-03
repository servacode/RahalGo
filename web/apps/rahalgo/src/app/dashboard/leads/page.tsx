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
  AdminPage,
  AdminAsync,
  FilterBar,
  FilterSelect,
  PeriodPicker,
  ContactButtons,
  IconStore,
  IconUser,
  IconPromos,
  IconLink,
  IconLocation,
  IconSuccess,
  IconBlock,
  IconUnblock,
  IconSearch,
  IconHourglass,
  IconWarning,
  IconDate,
  Modal,
  Input,
  Alert,
  Confirm,
  FormActions,
  Chips,
  Switch,
} from "@rahalgo/ui";
import { TempPasswordNote } from "@/components/admin/accounts/TempPasswordNote";
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

/** مفاتيحُ الرابط — **الفلاترُ تُحفظ فيه** فيُشارَك ويُعاد إليه. */
const KEYS = ["status", "q", "rep_id", "governorate_id", "category_id", "from", "to", "page"] as const;
type Key = (typeof KEYS)[number];

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
  const allCount = STATUSES.reduce((s, k) => s + (counts[k] ?? 0), 0);

  function clearFilters() {
    setQ("");
    setParams({ q: "", rep_id: "", governorate_id: "", category_id: "", from: "", to: "" });
  }

  const repOptions = reps.length > 1;

  return (
    <AdminPage<string>
      icon={IconLink}
      title={L.title}
      count={data?.counts ? allCount : null}
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
          key: s as string,
          label: L.st[s],
          count: data?.counts ? (counts[s] ?? 0) : null,
          attention: s === "new",
        })),
        { key: "all", label: L.filterAll, count: data?.counts ? allCount : null },
      ]}
      tab={status}
      onTab={(k) => setParams({ status: k === "new" ? "" : k })}
      notices={
        <>
          {/* **وهو شغّالٌ يُقال بوضوح** — لا تلميحٌ مخفيّ. */}
          {auto.row?.on && <Alert tone="warning">{L.autoOnBanner}</Alert>}
          {auto.error && <Alert onDismiss={auto.clearError}>{auto.error}</Alert>}
          {/* **وسببُ الخادم يُعرض بنصّه** — جوابٌ يُصلَح به. */}
          {error && data && <Alert onDismiss={() => setError("")}>{error}</Alert>}
          {notice && (
            <Alert tone={notice.tone} onDismiss={() => setNotice(null)}>
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
        </>
      }
      toolbar={
        <FilterBar
          search={{ id: "lead-search", value: q, onChange: setQ, placeholder: L.search }}
          filtered={filtered}
          onClear={clearFilters}
          end={L.resultsCount.replace("{n}", fmtNum(count))}
        >
          {repOptions && (
            <FilterSelect id="lead-rep" label={L.rep} value={get("rep_id")} onChange={(v) => setParams({ rep_id: v })}>
              <option value="">{L.allReps}</option>
              {reps.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name}
                </option>
              ))}
            </FilterSelect>
          )}
          <PeriodPicker
            id="lead-period"
            from={get("from")}
            to={get("to")}
            today={today}
            onChange={(f, t) => setParams({ from: f, to: t })}
          />
        </FilterBar>
      }
      pagination={{ page, total: count, perPage, onChange: (p) => setParams({ page: p > 1 ? String(p) : "" }) }}
    >
      {/* **والمحافظةُ والتصنيفُ ممّا في الطلبات نفسِها** — ويُخفى المرشِّحُ إن لم يبقَ فيه إلّا خيارٌ واحد. */}
      {(govs.length > 1 || cats.length > 1) && (
        <div className="flex flex-wrap items-center gap-2">
          {govs.length > 1 && (
            <FilterSelect
              id="lead-gov"
              label={L.allGovs}
              value={get("governorate_id")}
              onChange={(v) => setParams({ governorate_id: v })}
            >
              <option value="">{L.allGovs}</option>
              {govs.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </FilterSelect>
          )}
          {cats.length > 1 && (
            <FilterSelect
              id="lead-cat"
              label={L.category}
              value={get("category_id")}
              onChange={(v) => setParams({ category_id: v })}
            >
              <option value="">{L.allCats}</option>
              {cats.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </FilterSelect>
          )}
        </div>
      )}

      <AdminAsync
        loading={!loaded}
        error={!!error && !data}
        onRetry={() => void load()}
        hasData={!!data}
        empty={leads.length === 0}
        emptyTitle={filtered ? L.emptyFiltered : (L.emptyByStatus[status as LeadStatus | "all"] ?? L.empty)}
        emptyIcon={filtered ? IconSearch : IconStore}
        emptyAction={
          filtered ? (
            <Button variant="secondary" onClick={clearFilters}>
              {L.clear}
            </Button>
          ) : undefined
        }
      >
        <ul className="admin-grid">
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
      </AdminAsync>

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
    </AdminPage>
  );
}

function NoteLine({ label, text }: { label: string; text: string }) {
  return (
    <p className="border-s-2 border-line ps-2 text-sm">
      <span className="text-xs text-ink-muted">{label}: </span>
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
          {l.category_name && <span className="block text-xs text-ink-muted">{l.category_name}</span>}
        </span>
        <Badge variant={STATUS_VARIANT[l.status]}>{L.st[l.status]}</Badge>
      </div>

      {/* **«ناطر من…» هو الرقمُ الأهمّ للمعلَّق** — وتاريخُ التسجيل تحته لكلّ طلب
          (قرارُ المالك ٢٠٢٦-١٠-٢٦: «لازم هذا تاريخ التسجيل والانضمام»). */}
      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span className="min-w-0">
          <span className="block text-xs text-ink-muted">{pending ? L.waiting : L.registered}</span>
          {pending ? (
            <span className="figure block text-warning">{waitingFor(l.created_at)}</span>
          ) : (
            <span className="block font-bold" dir="ltr">
              {fmtDateTime(l.created_at)}
            </span>
          )}
        </span>
        {pending && (
          <span className="text-end text-xs text-ink-muted">
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
          <ContactButtons phone={l.phone} name={l.owner_name} />
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
