"use client";

/**
 * **طلبات انضمام المتاجر** — متاجرُ أضافها المندوبون وتنتظر قرارَ المكتب.
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤.) **للمتاجر التي يضيفها المندوبون وحدَها** —
 * والخادمُ يرشّح ذلك. بحثٌ وفلاتر محفوظةٌ في الرابط · «ينتظر منذ» · تحذيرُ الرقم
 * المكرّر والمتجر القريب بالاسم نفسه · **تأكيدٌ قبل الموافقة** · «بحاجة معلومات»
 * تعود للمندوب بملاحظة · أسبابُ رفضٍ جاهزة.
 */

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { getMessages, defaultLocale, fmtDate, fmtNum, fmtSpan, errorText } from "@rahalgo/i18n";
import {
  useLiveRefresh,
  PageHeader,
  Button,
  Badge,
  CategoryIcon,
  DataView,
  Pagination,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconStore,
  IconPhone,
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
  Modal,
  Input,
  Select,
  Alert,
  Confirm,
  FormActions,
  Chips,
} from "@rahalgo/ui";
import { TempPasswordNote } from "@/components/admin/accounts/TempPasswordNote";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { AutoModeToggle } from "@/components/admin/AutoModeToggle";

const m = getMessages(defaultLocale);
const L = m.admin.leads;

type LeadStatus = "new" | "needs_info" | "converted" | "rejected";

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
}

interface Named {
  id: string;
  name: string;
}

const STATUS_VARIANT: Record<LeadStatus, "warning" | "success" | "danger" | "info"> = {
  new: "warning",
  needs_info: "info",
  converted: "success",
  rejected: "danger",
};

const STATUS_CHIPS: { id: string; label: string }[] = [
  { id: "new", label: L.st.new },
  { id: "needs_info", label: L.st.needs_info },
  { id: "converted", label: L.st.converted },
  { id: "rejected", label: L.st.rejected },
  { id: "all", label: L.filterAll },
];

/** مفاتيحُ الرابط — **الفلاترُ تُحفظ فيه** فيُشارَك ويُعاد إليه. */
const KEYS = ["status", "q", "rep_id", "governorate_id", "category_id", "from", "to", "page"] as const;
type Key = (typeof KEYS)[number];

/** «ينتظر منذ» — أيّامٌ إن بلغ يوماً، وإلّا ساعاتٌ ودقائق. */
function waitingFor(iso: string): string {
  const secs = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  const days = Math.floor(secs / 86400);
  return days >= 1 ? L.days.replace("{n}", fmtNum(days)) : fmtSpan(secs);
}

function LeadsScreen() {
  const params = useSearchParams();
  // **وملفُّ الحساب لمن يقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥): موظّفُ العمليّات
  // لا يملك `users.read` — **فلا يُرسَم له رابطٌ يُردّ ٤٠٣.**
  const canProfile = useCanCall()("GET", "/users/{id}");
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

  const [leads, setLeads] = useState<Lead[]>([]);
  const [reps, setReps] = useState<Named[]>([]);
  const [govs, setGovs] = useState<Named[]>([]);
  const [cats, setCats] = useState<Named[]>([]);
  // **ولا «لا طلبات» قبل أن يصل الردّ** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣).
  const [loaded, setLoaded] = useState(false);
  const [view, setView] = useViewMode("leads", "cards");
  const [count, setCount] = useState(0);
  const [perPage, setPerPage] = useState(20);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState<{ tone: "success" | "warning"; text: string; owner?: string; temp?: string } | null>(null);
  const [busy, setBusy] = useState("");
  /** حقلُ البحث يُكتب محلّيّاً ثمّ يُرسَل للرابط بعد توقّفٍ قصير. */
  const [q, setQ] = useState(get("q"));

  /** الطلبُ الذي يُوافَق عليه الآن — **لا موافقةَ بلا تأكيد.** */
  const [approving, setApproving] = useState<Lead | null>(null);
  /** الطلبُ الذي يُردّ أو يُعاد للمندوب — **ولا يُرسَل حتى تُكتب كلمة.** */
  const [deciding, setDeciding] = useState<{ lead: Lead; status: "rejected" | "needs_info" } | null>(null);
  const [note, setNote] = useState("");

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
      const res = await api<{ leads: Lead[]; total: number; per_page: number; reps: Named[] }>(
        `/api/v1/admin/leads?${query}`,
      );
      setLeads(res?.leads ?? []);
      setReps(res?.reps ?? []);
      setCount(res?.total ?? 0);
      setPerPage(res?.per_page || 20);
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

  useEffect(() => {
    void api<{ governorates: Named[] }>("/api/v1/admin/governorates")
      .then((r) => setGovs(r.governorates ?? []))
      .catch(() => {});
    void api<Named[]>("/api/v1/admin/categories")
      .then((r) => setCats(r ?? []))
      .catch(() => {});
  }, []);

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
      setNote("");
    } catch (err) {
      setError(errorText(err));
      setDeciding(null);
      setApproving(null);
    } finally {
      setBusy("");
      // **وتُعاد القراءةُ في الحالين** — قائمةٌ لا تُحدَّث بعد فشلٍ تُري حالاً قد تبدّل.
      await load();
    }
  }

  const warnings = (l: Lead) => (
    <>
      {l.duplicate_phone.length > 0 && (
        <span className="flex items-center gap-1 text-xs text-warning">
          <IconWarning size={13} />
          {L.warnDupPhone.replace("{names}", l.duplicate_phone.join(m.common.listSep))}
        </span>
      )}
      {l.nearby_same_name.length > 0 && (
        <span className="flex items-center gap-1 text-xs text-warning">
          <IconWarning size={13} />
          {L.warnNearby.replace("{names}", l.nearby_same_name.join(m.common.listSep))}
        </span>
      )}
    </>
  );

  const columns: DataColumn<Lead>[] = [
    {
      id: "store",
      header: L.store,
      icon: <IconStore />,
      primary: true,
      cell: (l) => (
        <span className="flex flex-col gap-0.5">
          <span className="font-medium">{l.store_name}</span>
          {warnings(l)}
        </span>
      ),
    },
    {
      id: "owner",
      header: L.owner,
      icon: <IconUser />,
      cell: (l) => l.owner_name || "—",
    },
    {
      id: "phone",
      header: L.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (l) => (
        <a
          href={`https://wa.me/${l.phone.replace(/^\+/, "").replace(/^0/, "963")}`}
          target="_blank"
          rel="noreferrer"
          dir="ltr"
          className="font-medium text-primary hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {l.phone}
        </a>
      ),
    },
    {
      id: "category",
      header: L.category,
      icon: <IconStore />,
      cell: (l) =>
        l.category_name ? (
          <span className="inline-flex items-center gap-1.5">
            {/* **والأيقونةُ تُرسم لا تُطبع** — `category_icon` مفتاحٌ لاتينيّ. */}
            <CategoryIcon name={l.category_icon} size={15} />
            {l.category_name}
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "area",
      header: L.area,
      cell: (l) => (
        <span className="inline-flex items-center gap-1.5">
          {[l.district, l.area].filter(Boolean).join(" — ") || "—"}
          {l.lat != null && l.lng != null && (
            <a
              href={`https://www.google.com/maps?q=${l.lat},${l.lng}`}
              target="_blank"
              rel="noreferrer"
              title={L.onMap}
              onClick={(e) => e.stopPropagation()}
              className="text-primary hover:text-primary-dark"
            >
              <IconLocation size={15} />
            </a>
          )}
        </span>
      ),
    },
    {
      id: "rep",
      header: L.rep,
      icon: <IconPromos />,
      cell: (l) => (
        <span className="flex items-center gap-1.5">
          {l.rep_name ?? "—"}
          {l.rep_code && (
            <Badge variant="accent" dir="ltr" className="px-1.5 font-mono">
              {l.rep_code}
            </Badge>
          )}
        </span>
      ),
    },
    {
      id: "status",
      header: L.status,
      icon: <IconStatus />,
      cell: (l) => (
        <span className="flex flex-col gap-0.5">
          <Badge variant={STATUS_VARIANT[l.status]}>{L.st[l.status]}</Badge>
          {l.decision_note && l.status !== "converted" && (
            <span className="text-xs text-ink-muted">
              {L.decisionNote}: {l.decision_note}
            </span>
          )}
        </span>
      ),
    },
    {
      id: "waiting",
      header: L.waiting,
      icon: <IconHourglass />,
      cell: (l) =>
        l.status === "new" || l.status === "needs_info" ? (
          <span className="text-sm font-medium text-warning">{waitingFor(l.created_at)}</span>
        ) : (
          <span dir="ltr" className="text-xs text-ink-muted">
            {fmtDate(l.created_at)}
          </span>
        ),
    },
  ];

  const filtered = KEYS.some((k) => k !== "status" && k !== "page" && params.get(k));

  return (
    <div>
      <div className="mb-2 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconLink} title={L.title} />
        <ViewToggle view={view} onChange={setView} tableLabel={m.common.viewTable} cardsLabel={m.common.viewCards} />
      </div>
      <p className="mb-4 text-sm text-ink-muted">{L.subtitle}</p>

      {/* **قبولٌ تلقائيٌّ بضغطة** (قرارُ المالك ٢٠٢٦-١٠-٠٨) — يظهر لمن يملك كتابتَه. */}
      <AutoModeToggle
        settingKey="leads.auto_approve"
        label={m.admin.autoMode.leadsLabel}
        hint={m.admin.autoMode.leadsHint}
      />

      {/* **وسببُ الخادم يُعرض بنصّه** — جوابٌ يُصلَح به. */}
      {error && (
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

      <Chips items={STATUS_CHIPS} value={status} onChange={(id) => setParams({ status: id === "new" ? "" : id })} className="mb-3" />

      <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <Input
          id="lead-search"
          icon={<IconSearch size={16} />}
          placeholder={L.search}
          aria-label={L.search}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <Select aria-label={L.rep} value={get("rep_id")} onChange={(e) => setParams({ rep_id: e.target.value })}>
          <option value="">{L.allReps}</option>
          {reps.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </Select>
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
        <Select aria-label={L.category} value={get("category_id")} onChange={(e) => setParams({ category_id: e.target.value })}>
          <option value="">{L.allCats}</option>
          {cats.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </Select>
        <Input id="lead-from" type="date" label={L.from} value={get("from")} onChange={(e) => setParams({ from: e.target.value })} />
        <Input id="lead-to" type="date" label={L.to} value={get("to")} onChange={(e) => setParams({ to: e.target.value })} />
        {filtered && (
          <div className="flex items-end">
            <Button
              variant="secondary"
              onClick={() => {
                setQ("");
                setParams({ q: "", rep_id: "", governorate_id: "", category_id: "", from: "", to: "" });
              }}
            >
              {L.clear}
            </Button>
          </div>
        )}
      </div>

      <DataView
        items={leads}
        loading={!loaded && !error}
        getKey={(l) => l.id}
        columns={columns}
        view={view}
        empty={L.empty}
        actions={(l) =>
          /* **وما حُوِّل لا يُردّ** — المتجرُ قائمٌ يبيع، والخادمُ يمنعه أيضاً. */
          l.status === "converted" ? null : (
            <>
              <Button
                variant="secondary"
                disabled={busy === l.id}
                onClick={() => setApproving(l)}
                className="flex items-center gap-1.5 !text-success"
              >
                <IconSuccess size={15} />
                {L.markConverted}
              </Button>
              {l.status === "new" && (
                <Button
                  variant="secondary"
                  onClick={() => setDeciding({ lead: l, status: "needs_info" })}
                  className="flex items-center gap-1.5 !text-info"
                >
                  <IconHourglass size={15} />
                  {L.markNeedsInfo}
                </Button>
              )}
              {l.status !== "rejected" ? (
                /* **والردُّ يلزمه كلمة** — المندوبُ يقرؤها في إشعاره. */
                <Button
                  variant="secondary"
                  onClick={() => setDeciding({ lead: l, status: "rejected" })}
                  className="flex items-center gap-1.5 !text-danger"
                >
                  <IconBlock size={15} />
                  {L.markRejected}
                </Button>
              ) : null}
              {l.status !== "new" && (
                <Button variant="secondary" onClick={() => void setStatus(l.id, "new")} className="flex items-center gap-1.5">
                  <IconUnblock size={15} />
                  {L.reopen}
                </Button>
              )}
            </>
          )
        }
      />

      {/* **تأكيدٌ قبل الموافقة** — يقول ما سيقع والتحذيرات قبل الضغط. */}
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
              {(approving.duplicate_phone.length > 0 || approving.nearby_same_name.length > 0) && (
                <div className="space-y-1">
                  <p className="font-medium text-warning">{L.confirmWarnings}</p>
                  {warnings(approving)}
                </div>
              )}
            </div>
          )
        }
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
            {deciding.lead.note && (
              <p className="rounded-control bg-field px-3 py-2 text-sm">
                <span className="text-ink-muted">{L.repNote}: </span>
                {deciding.lead.note}
              </p>
            )}
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

      {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
      {count > perPage && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={count} perPage={perPage} onChange={(p) => setParams({ page: p > 1 ? String(p) : "" })} />
        </div>
      )}
    </div>
  );
}

export default function LeadsPage() {
  return (
    <Suspense>
      <LeadsScreen />
    </Suspense>
  );
}
