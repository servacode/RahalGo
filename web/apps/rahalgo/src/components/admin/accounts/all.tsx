"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ الحسابات — بابٌ واحدٌ لكلّ من في المنصّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ الحسابات.)
 *
 *	«إضافة ▾»         زرٌّ واحدٌ بقائمة أنواع (سائق · مندوب · متجر · موظّف) بحسب الصلاحيّة،
 *	                   ودورٌ واحدٌ للحساب، **ولا كلمةَ سرٍّ يكتبها الموظّف**
 *	البحث             بالاسم والهاتف وكود الدعوة **واسمِ المتجر**، واسمُ المتجر تحت صاحبه
 *	الشارة            للموقوف والمحظور وحدَهما
 *	التحديدُ الجماعيّ  إيقافٌ وتفعيلٌ وتصديرٌ فقط — **ولا فعلَ ماليّاً**
 *	التصدير           لمديرِ المنصّة وحدَه (`users.export`)، عربيٌّ ومحميٌّ من صيغ إكسل
 *	الأرصدة           تُحجب في المحرّك عمّن لا يملك المالَ ولا خدمةَ العملاء
 *
 * **وكلُّ زرٍّ بقدرة بابه من جدول المحرّك** (`useCanCall`).
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDate, errorText } from "@rahalgo/i18n";
import { roleLabelByCode, signupGroups, ACCOUNT_TYPE_ROLE_CODES } from "@/lib/rolemeta";
import { listRoles, roleLabel, type Role } from "@/lib/rbac";
import {
  Pagination,
  Alert,
  useLiveRefresh,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  Confirm,
  Checkbox,
  DataView,
  ActionMenu,
  type ActionMenuItem,
  type DataColumn,
  IconUser,
  IconPhone,
  IconRoles,
  IconStatus,
  IconSearch,
  IconAdd,
  IconBlock,
  IconDelete,
  IconUnblock,
  IconWallet,
  IconOrder,
  IconStore,
  IconDriver,
  IconGrid,
  IconView,
  IconReceipt,
  IconUsers,
  FormActions,
  IconLock,
  IconHourglass,
  IconHandshake,
} from "@rahalgo/ui";
import { api, apiFile, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import WalletModal, { AdvanceModal } from "@/components/admin/WalletModal";
import { MerchantModal } from "@/components/admin/MerchantModal";
import StatusReasonModal from "@/components/admin/StatusReasonModal";
import RoleBadge, { ROLE_STYLES } from "@/components/admin/RoleBadge";
import { MediaThumb } from "@/components/admin/ImageUpload";
import { PageHero } from "@/components/admin/PageHero";
import { TempPasswordNote } from "@/components/admin/accounts/TempPasswordNote";
import { ResetPasswordModal, ChangePhoneModal } from "@/components/admin/accounts/ProfileParts";

const m = getMessages(defaultLocale);
const A = m.admin.acc;

type Row = AuthUser & { store_names?: string[] };

interface UserPage {
  users: Row[];
  total: number;
  page: number;
  per_page: number;
  money_hidden: boolean;
}

type NewKind = "driver" | "sales" | "staff";

export default function AllAccountsTable() {
  const { user: me, can } = useAuth();
  const router = useRouter();
  const canCall = useCanCall();
  const canCreateUser = canCall("POST", "/users");
  const canGrant = canCall("POST", "/users/{id}/roles");
  const canCreateStore = canCall("POST", "/merchants");
  // **وكلمةُ المرور زرٌّ سريعٌ في البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — بلا فتح الملف.
  const canReset = canCall("POST", "/users/{id}/password");
  const [resetUser, setResetUser] = useState<{ id: string; hours: number } | null>(null);
  /** **تبديلُ الرقم من البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — النافذةُ نفسُها التي في الملفّ. */
  const [phoneUser, setPhoneUser] = useState<Row | null>(null);
  const openReset = (u: Row) => {
    // **مهلةُ الكلمة المؤقّتة من المحرّك** — تُقرأ من ملفّه، والافتراضُ إن تعذّر.
    api<{ temp_password_hours?: number }>(`/api/v1/admin/users/${u.id}`)
      .then((p) => setResetUser({ id: u.id, hours: p.temp_password_hours ?? 24 }))
      .catch(() => setResetUser({ id: u.id, hours: 24 }));
  };
  const canStatus = canCall("PATCH", "/users/{id}");
  const canWallet = canCall("POST", "/users/{id}/wallet");
  // **والتصديرُ لمديرِ المنصّة وحدَه** — الزرُّ يُخفى عن غيره (قرار ١١).
  const canExport = can("users.export");
  const canPhone = can("users.contact.read");

  const urlParam = (k: string) =>
    typeof window === "undefined" ? "" : (new URLSearchParams(window.location.search).get(k) ?? "");
  const [data, setData] = useState<UserPage | null>(null);
  const [query, setQuery] = useState("");
  const [role, setRole] = useState(() => urlParam("role"));
  const [onlineOnly, setOnlineOnly] = useState(false);
  const [statusFilter, setStatusFilter] = useState(() => urlParam("status"));
  const [roleCounts, setRoleCounts] = useState<{ total: number; roles: Record<string, number> } | null>(null);
  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [createKind, setCreateKind] = useState<NewKind | null>(null);
  const [walletUser, setWalletUser] = useState<Row | null>(null);
  const [advanceUser, setAdvanceUser] = useState<Row | null>(null);
  const [storeOpen, setStoreOpen] = useState(false);
  const [statusModal, setStatusModal] = useState<{ users: Row[]; status: string } | null>(null);
  const [activateRows, setActivateRows] = useState<Row[] | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkMsg, setBulkMsg] = useState("");
  const [busy, setBusy] = useState(false);

  // **والإحصاءاتُ مرّةً لا مع كلّ حرف** — كانت تُعاد مع كلّ ضغطةٍ في البحث.
  const loadStats = useCallback(() => {
    api<{ total: number; roles: Record<string, number> }>("/api/v1/admin/users/stats")
      .then(setRoleCounts)
      // @empty-ok **والبطاقاتُ تلميح** — القائمةُ تعمل بدونها.
      .catch(() => undefined);
  }, []);

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        query,
        role,
        status: statusFilter,
        online: onlineOnly ? "true" : "",
        page: String(page),
        per_page: "50",
      });
      setData(await api<UserPage>(`/api/v1/admin/users?${params}`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [query, role, statusFilter, onlineOnly, page]);

  useEffect(() => {
    const t = setTimeout(load, 250); // تهدئة البحث
    return () => clearTimeout(t);
  }, [load]);
  useEffect(loadStats, [loadStats]);

  const canRolesList = canCall("GET", "/roles");
  useEffect(() => {
    let alive = true;
    // **ومن لا يقرأ الأدوار يُرشِّح بصفات الحساب وحدَها** — كان يُنادى فيُردّ ٤٠٣.
    if (!canRolesList) {
      setAllRoles(ACCOUNT_TYPE_ROLE_CODES.map((code) => ({ code, name_key: `roles.${code}` }) as Role));
      return;
    }
    void listRoles()
      .then((r) => alive && setAllRoles(r))
      .catch(() => alive && setAllRoles([]));
    return () => {
      alive = false;
    };
  }, [canRolesList]);

  const refresh = useCallback(() => {
    void load();
    loadStats();
  }, [load, loadStats]);
  useLiveRefresh(["account"], refresh);

  async function exportCsv(ids?: string[]) {
    const params = new URLSearchParams({ query, role, status: statusFilter, online: onlineOnly ? "true" : "" });
    if (ids && ids.length) params.set("ids", ids.join(","));
    try {
      const res = await apiFile(`/api/v1/admin/users/export?${params}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "accounts.csv";
      a.click();
      URL.revokeObjectURL(url);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }

  // **جهاتُ الاتصال لجوال المالك** (٢٠٢٦-١٠-١٢) — ملفٌّ يُفتح على الجوال فتُحفظ كلُّها.
  const [contactsSince, setContactsSince] = useState("");
  async function exportContacts() {
    const params = new URLSearchParams({ query, role, status: statusFilter, format: "vcf", since: contactsSince });
    try {
      const res = await apiFile(`/api/v1/admin/users/export?${params}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "rahalgo-contacts.vcf";
      a.click();
      URL.revokeObjectURL(url);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }

  /** **فعلٌ على حسابٍ أو أكثر** — والخطأُ يُعدّ ويُقال لا يُبتلع. */
  async function applyStatus(rows: Row[], status: string, reason = "") {
    setBusy(true);
    let ok = 0;
    let lastErr = "";
    for (const u of rows) {
      try {
        await api(`/api/v1/admin/users/${u.id}`, {
          method: "PATCH",
          body: JSON.stringify({ status, status_reason: reason }),
        });
        ok++;
      } catch (err) {
        lastErr = errorText(err);
      }
    }
    setBusy(false);
    const failed = rows.length - ok;
    if (rows.length > 1 || failed > 0) {
      setBulkMsg(
        [A.bulkDone.replace("{n}", fmtNum(ok)), failed > 0 ? `${A.bulkFailed.replace("{n}", fmtNum(failed))} — ${lastErr}` : ""]
          .filter(Boolean)
          .join(" · "),
      );
    }
    setSelected(new Set());
    refresh();
    if (failed > 0 && rows.length === 1) throw new Error(lastErr);
  }

  const rows = useMemo(() => data?.users ?? [], [data]);
  const money = data ? !data.money_hidden : true;
  const selectable = canStatus || canExport;
  const pageIds = rows.filter((u) => !u.is_system && u.id !== me?.id).map((u) => u.id);
  const allOnPage = pageIds.length > 0 && pageIds.every((i) => selected.has(i));
  const selRows = rows.filter((u) => selected.has(u.id));

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  const columns: DataColumn<Row>[] = [
    {
      // ══════════════════════════════════════════════════════════════
      // **رأسُ البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٢٥): الدورُ فوق، والصورةُ دائرةٌ في
      // الوسط، وتحتها الاسمُ يميناً والرقمُ يساراً.
      // ══════════════════════════════════════════════════════════════
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (u) => (
        <span className="flex w-full flex-col gap-3">
          {/* **مربّعُ التحديد يميناً قبل الدور، ولا عينَ** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — الكبسةُ على
              البطاقة تفتح الملفَّ أصلاً. */}
          <span className="flex items-center gap-2">
            {selectable && !u.is_system && u.id !== me?.id && (
              <span onClick={(e) => e.stopPropagation()}>
                <Checkbox
                  id={`sel-${u.id}`}
                  label={<span className="sr-only">{A.selectRow}</span>}
                  checked={selected.has(u.id)}
                  onChange={() => toggle(u.id)}
                />
              </span>
            )}
            <span className="flex flex-wrap gap-1">
              {u.roles.map((r) => (
                <RoleBadge key={r} role={r} />
              ))}
              {/* **شارةٌ صغيرةٌ للموقوف والمحظور وحدَهما** (قرار ٨). */}
              {u.status === "suspended" && <Badge variant="warning">{m.admin.users.suspended}</Badge>}
              {u.status === "blocked" && <Badge variant="danger">{m.admin.users.blocked}</Badge>}
              {u.is_system ? <Badge variant="neutral">{m.admin.users.systemAccount}</Badge> : null}
            </span>
          </span>
          <span className="flex justify-center">
            <MediaThumb
              url={u.avatar_thumb_url}
              alt=""
              fallback={u.full_name || m.terms.avatarFallback}
              size={72}
              round
            />
          </span>
          <span className="flex items-center justify-between gap-3">
            <span className="min-w-0 truncate text-base font-bold">{u.full_name || "—"}</span>
            {u.phone ? (
              <span dir="ltr" className="shrink-0 font-medium">
                {u.phone}
              </span>
            ) : (
              <span className="shrink-0 text-xs text-ink-muted">{A.phoneHidden}</span>
            )}
          </span>
          {/* **واسمُ متجره بسطرٍ ثانٍ في الوسط** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — لصاحب المتجر وحدَه. */}
          {/* **وسطرُه محجوزٌ ولو فارغاً** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — فيبقى الرصيدُ والأزرارُ على مستوًى
              واحدٍ في كلّ البطاقات؛ الفراغُ مكانُ المتجر لا تحته. */}
          <span
            aria-hidden={!(u.store_names && u.store_names.length > 0)}
            className={`flex h-5 items-center justify-center gap-1.5 truncate text-sm font-medium text-ink-muted ${
              u.store_names && u.store_names.length > 0 ? "" : "invisible"
            }`}
          >
            <IconStore size={14} />
            {u.store_names && u.store_names.length > 0 ? u.store_names.join(" · ") : "—"}
          </span>
        </span>
      ),
    },
    // **لا عددَ طلباتٍ ولا إنفاقَ ولا آخرَ طلبٍ في البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — في ملفّ الحساب.
    ...(money
      ? ([
          {
            id: "balance",
            header: `${m.admin.customers.balance} (${m.common.currency})`,
            icon: <IconWallet />,
            cell: (u: Row) => fmtNum(u.balance ?? 0),
          },
        ] as DataColumn<Row>[])
      : []),
    // **ولا «طلبات جارية» ولا متاجرُ المندوب وعمولاتُه في البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٢٥) —
    // فيبقى شكلُ البطاقات موحّداً؛ وكلُّها في ملفّ الحساب.
  ];

  // ── «إضافة ▾» ─────────────────────────────────────────────────────────
  const addItems: ActionMenuItem[] = [];
  if (canCreateUser) {
    addItems.push(
      { key: "driver", label: A.addDriver, icon: IconDriver, onSelect: () => setCreateKind("driver") },
      { key: "sales", label: A.addRep, icon: IconUsers, onSelect: () => setCreateKind("sales") },
    );
  }
  // **ولا «متجر جديد» هنا** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — المتاجرُ تُضاف وتُدار من السوق (تبويبُ المتاجر).
  if (canCreateUser && canGrant) addItems.push({ key: "staff", label: A.addStaff, icon: IconRoles, onSelect: () => setCreateKind("staff") });

  const cards = [
    { key: "staff", label: m.admin.users.staffCard, style: ROLE_STYLES.ops },
    { key: "sales", label: roleLabelByCode("sales"), style: ROLE_STYLES.sales },
    { key: "driver", label: roleLabelByCode("driver"), style: ROLE_STYLES.driver },
    { key: "merchant", label: roleLabelByCode("merchant"), style: ROLE_STYLES.merchant },
    { key: "customer", label: A.cardCustomers, style: ROLE_STYLES.customer },
  ] as const;

  return (
    <div>
      {/* **رأسٌ موحّد** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «تصميم احترافي») — الأدوارُ تبويباتٌ بعدّها بسطرٍ واحد،
          والحالاتُ الثلاث (موقوف أو محظور · المحذوفة · المتّصلون الآن) أزرارٌ صغيرةٌ تحتها. */}
      <PageHero<string>
        icon={IconUsers}
        title={m.admin.users.title}
        actions={
          <div className="flex flex-wrap items-center gap-2">
          {canExport && (
            <Button variant="secondary" onClick={() => void exportCsv()} className="flex items-center gap-1.5">
              <IconReceipt size={16} />
              {A.exportCsv}
            </Button>
          )}
          {canExport && (
            <div className="flex items-center gap-1.5">
              <Input
                type="date"
                value={contactsSince}
                onChange={(e) => setContactsSince(e.target.value)}
                title={A.contactsSince}
                aria-label={A.contactsSince}
              />
              <Button variant="secondary" onClick={() => void exportContacts()} className="flex items-center gap-1.5">
                <IconReceipt size={16} />
                {A.exportContacts}
              </Button>
            </div>
          )}
          {/* **ولا زرَّ تصنيفاتٍ هنا** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — بابُها السوق (تبويبُ التصنيفات). */}
          <ActionMenu label={A.add} icon={IconAdd} variant="primary" items={addItems} width={200} />
          </div>
        }
        tabs={
          roleCounts
            ? [
                { key: "all", label: m.admin.users.allRoles, count: roleCounts.total },
                ...cards.map(({ key, label, style }) => ({
                  key,
                  label,
                  icon: style?.Icon,
                  count: roleCounts.roles[key] ?? 0,
                })),
              ]
            : undefined
        }
        value={role === "" ? "all" : role}
        onChange={(k) => {
          setRole(k === "all" ? "" : k);
          setPage(1);
        }}
      >
        {roleCounts && (
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              aria-pressed={statusFilter === "restricted"}
              onClick={() => {
                setStatusFilter(statusFilter === "restricted" ? "" : "restricted");
                setPage(1);
              }}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-bold transition-colors ${
                statusFilter === "restricted" ? "border-danger bg-danger-solid text-on-solid" : "border-line text-danger hover:bg-row-hover"
              }`}
            >
              <IconBlock size={15} />
              {A.cardRestricted}
              <span className="tabular-nums">{fmtNum(roleCounts.roles.restricted ?? 0)}</span>
            </button>
            {/* **والمحذوفةُ زرُّها** (قرارُ المالك ٢٠٢٦-١٠-٠٦) — ولا تُخلط بالأحياء. */}
            <button
              type="button"
              aria-pressed={statusFilter === "deleted"}
              onClick={() => {
                setStatusFilter(statusFilter === "deleted" ? "" : "deleted");
                setPage(1);
              }}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-bold transition-colors ${
                statusFilter === "deleted" ? "border-ink-muted bg-field text-ink" : "border-line text-ink-muted hover:bg-row-hover"
              }`}
            >
              <IconDelete size={15} />
              {A.cardDeleted}
              <span className="tabular-nums">{fmtNum(roleCounts.roles.deleted ?? 0)}</span>
            </button>
            <button
              type="button"
              aria-pressed={onlineOnly}
              onClick={() => {
                setOnlineOnly(!onlineOnly);
                setPage(1);
              }}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-sm font-bold transition-colors ${
                onlineOnly ? "border-success bg-success-solid text-on-solid" : "border-line text-success hover:bg-row-hover"
              }`}
            >
              <span className={`h-2 w-2 animate-pulse rounded-badge ${onlineOnly ? "bg-on-solid" : "bg-success"}`} />
              {m.admin.users.onlineCard}
              <span className="tabular-nums">{fmtNum(roleCounts.roles.online ?? 0)}</span>
            </button>
          </div>
        )}
      </PageHero>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-full sm:w-80">
          <Input
            icon={<IconSearch />}
            placeholder={canPhone ? A.searchPlaceholder : A.searchPlaceholderNoPhone}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-[calc(50%-0.375rem)] sm:w-44">
          <Select
            value={role}
            onChange={(e) => {
              setRole(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.users.allRoles}</option>
            {/* **و«موظّفو المنصّة» خيارٌ هنا أيضاً** — كانت البطاقةُ تُرشّح والقائمةُ تقول «كلّ الأدوار». */}
            <option value="staff">{m.admin.users.staffCard}</option>
            {[...allRoles]
              .sort((a, b) => roleLabel(a).localeCompare(roleLabel(b), "ar"))
              .map((r) => (
                <option key={r.code} value={r.code}>
                  {r.code === "customer" ? A.cardCustomers : roleLabel(r)}
                </option>
              ))}
          </Select>
        </div>
        <div className="w-[calc(50%-0.375rem)] sm:w-40">
          <Select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.users.statusFilter.all}</option>
            <option value="active">{m.admin.users.statusFilter.active}</option>
            <option value="restricted">{A.restricted}</option>
            <option value="suspended">{m.admin.users.statusFilter.suspended}</option>
            <option value="blocked">{m.admin.users.statusFilter.blocked}</option>
          </Select>
        </div>
      </div>

      {/* ══════════════════════════════════════════════════════════════
          **التحديدُ الجماعيّ — إيقافٌ وتفعيلٌ وتصديرٌ فقط** (قرار ١٠)
          ══════════════════════════════════════════════════════════════ */}
      {selectable && rows.length > 0 && (
        <div className="mb-3 flex flex-wrap items-center gap-2 rounded-control bg-field px-3 py-2 text-sm">
          <Checkbox
            id="sel-all"
            label={A.selectAll}
            checked={allOnPage}
            onChange={() =>
              setSelected((prev) => {
                const next = new Set(prev);
                if (allOnPage) pageIds.forEach((i) => next.delete(i));
                else pageIds.forEach((i) => next.add(i));
                return next;
              })
            }
          />
          {selected.size > 0 && (
            <>
              <span className="text-ink-muted">{A.selectedN.replace("{n}", fmtNum(selected.size))}</span>
              {canStatus && (
                <>
                  <Button
                    variant="secondary"
                    className="flex items-center gap-1.5 !px-2.5"
                    disabled={busy}
                    onClick={() => setStatusModal({ users: selRows, status: "suspended" })}
                  >
                    <IconBlock size={14} />
                    {A.bulkSuspend}
                  </Button>
                  <Button
                    variant="secondary"
                    className="flex items-center gap-1.5 !px-2.5"
                    disabled={busy}
                    onClick={() => setActivateRows(selRows)}
                  >
                    <IconUnblock size={14} />
                    {A.bulkActivate}
                  </Button>
                </>
              )}
              {canExport && (
                <Button variant="secondary" className="!px-2.5" onClick={() => void exportCsv([...selected])}>
                  {A.exportSelected}
                </Button>
              )}
              <Button variant="ghost" className="!px-2.5" onClick={() => setSelected(new Set())}>
                {A.bulkClear}
              </Button>
            </>
          )}
        </div>
      )}
      {bulkMsg && (
        <Alert tone="info" className="mb-3">
          {bulkMsg}
        </Alert>
      )}
      {data?.money_hidden && <p className="mb-2 text-xs text-ink-muted">{A.moneyHidden}</p>}

      {error && (
        <Alert className="mb-4">
          <span>{error}</span>{" "}
          <button type="button" onClick={() => void load()} className="font-bold underline">
            {A.retry}
          </button>
        </Alert>
      )}

      <DataView
        items={rows}
        loading={data === null && !error}
        getKey={(u) => u.id}
        columns={columns}
        // **بطاقاتٌ وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «الغي نظام الجداول من الصفحة نخليه بس بطاقات»).
        view="cards"
        // **و«لا نتائج» لا تُكتب تحت رسالة خطأ** — كانت تقول «لا أحد» والسببُ عطب.
        empty={error ? "" : m.admin.users.noResults}
        onRowClick={(u) => router.push(`/dashboard/users/${u.id}`)}
        actions={
          canWallet || canStatus || canReset
            ? (u) =>
                u.is_system ? null : (
                  <>
                  <div className="flex w-full items-center justify-around">
                    {/* **أيقوناتٌ عاريةٌ بلا مربّعات، بصفٍّ واحد** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — شفّافةٌ بلون فعلها — `!min-w-0` يلغي حدَّ
                        عرض أزرار البطاقة (٦٫٥rem) الذي كان يُنزلها سطرين؛ والاسمُ في التلميح ولقارئ الشاشة.
                        **والمحفظةُ بقدرة المال وحدَها** — كانت مربوطةً بالإيقاف أيضاً فلا تراها الماليّة. */}
                    {canWallet && (
                      <Button
                        variant="ghost"
                        onClick={() => setWalletUser(u)}
                        title={m.admin.users.wallet}
                        aria-label={m.admin.users.wallet}
                        className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full text-ink-muted hover:text-ink"
                      >
                        <IconWallet size={20} />
                      </Button>
                    )}
                    {canWallet && u.roles.some((r) => r !== "customer" && r !== "merchant") && (
                      <Button
                        variant="ghost"
                        onClick={() => setAdvanceUser(u)}
                        title={A.advance.button}
                        aria-label={A.advance.button}
                        className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full text-ink-muted hover:text-ink"
                      >
                        <IconHandshake size={20} />
                      </Button>
                    )}
                    {canReset && u.id !== me?.id && (
                      <Button
                        variant="ghost"
                        onClick={() => openReset(u)}
                        title={m.admin.users.profile.passwordBtn}
                        aria-label={m.admin.users.profile.passwordBtn}
                        className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full text-ink-muted hover:text-ink"
                      >
                        <IconLock size={20} />
                      </Button>
                    )}
                    {canStatus && u.id !== me?.id && (
                      <Button
                        variant="ghost"
                        onClick={() => setPhoneUser(u)}
                        title={m.admin.users.profile.changePhone}
                        aria-label={m.admin.users.profile.changePhone}
                        className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full text-ink-muted hover:text-ink"
                      >
                        <IconPhone size={20} />
                      </Button>
                    )}
                    {canStatus &&
                      u.id !== me?.id &&
                      (u.status === "active" ? (
                        <>
                          <Button
                            variant="ghost"
                            onClick={() => setStatusModal({ users: [u], status: "suspended" })}
                            title={m.admin.users.suspend}
                            aria-label={m.admin.users.suspend}
                            className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full !text-warning"
                          >
                            <IconHourglass size={20} />
                          </Button>
                          <Button
                            variant="ghost"
                            onClick={() => setStatusModal({ users: [u], status: "blocked" })}
                            title={m.admin.users.block}
                            aria-label={m.admin.users.block}
                            className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full !text-danger"
                          >
                            <IconBlock size={20} />
                          </Button>
                        </>
                      ) : (
                        <Button
                          variant="ghost"
                          onClick={() => setActivateRows([u])}
                          title={m.admin.users.activate}
                          aria-label={m.admin.users.activate}
                          className="!h-11 !w-11 inline-flex items-center justify-center !min-h-0 !min-w-0 !flex-none !p-0 rounded-full !text-success"
                        >
                          <IconUnblock size={20} />
                        </Button>
                      ))}
                  </div>
                  </>
                )
            : undefined
        }
      />

      {data && (
        <div className="mt-4 flex flex-wrap items-center justify-end gap-2 text-sm text-ink-muted">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {createKind && (
        <CreateAccountModal
          kind={createKind}
          allRoles={allRoles}
          onClose={() => setCreateKind(null)}
          onCreated={() => refresh()}
        />
      )}
      {phoneUser && (
        <ChangePhoneModal
          userID={phoneUser.id}
          current={phoneUser.phone ?? ""}
          onClose={() => setPhoneUser(null)}
          onDone={(msg) => {
            setPhoneUser(null);
            setBulkMsg(msg);
            void load();
          }}
        />
      )}
      {resetUser && (
        <ResetPasswordModal
          userID={resetUser.id}
          hours={resetUser.hours}
          onClose={() => setResetUser(null)}
          onDone={() => void load()}
        />
      )}
      {storeOpen && (
        <MerchantModal
          merchant={null}
          categories={[]}
          onClose={() => setStoreOpen(false)}
          onSaved={() => {
            setStoreOpen(false);
            refresh();
          }}
        />
      )}
      {walletUser && <WalletModal user={walletUser} onClose={() => setWalletUser(null)} isAdmin={canWallet} />}
      {advanceUser && <AdvanceModal user={advanceUser} onClose={() => setAdvanceUser(null)} />}

      {statusModal && (
        <StatusReasonModal
          status={statusModal.status}
          holds={
            statusModal.users.length === 1 && statusModal.users[0]?.roles.includes("driver")
              ? { orders: statusModal.users[0].open_orders ?? 0, cash: statusModal.users[0].driver_cash ?? 0 }
              : undefined
          }
          onSubmit={async (reason) => {
            await applyStatus(statusModal.users, statusModal.status, reason);
            setStatusModal(null);
          }}
          onClose={() => setStatusModal(null)}
        />
      )}
      {/* **والتفعيلُ خلف تأكيد** — كان يقع من أوّل كبسة. */}
      <Confirm
        open={activateRows !== null}
        tone="primary"
        title={activateRows && activateRows.length > 1 ? A.bulkActivateTitle : A.activateTitle}
        body={activateRows && activateRows.length > 1 ? A.bulkActivateBody : A.activateBody}
        confirmLabel={m.admin.users.activate}
        busy={busy}
        onCancel={() => setActivateRows(null)}
        onConfirm={async () => {
          const list = activateRows ?? [];
          setActivateRows(null);
          try {
            await applyStatus(list, "active");
          } catch (err) {
            setError(errorText(err));
          }
        }}
      />
    </div>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابٌ جديد — سائقٌ أو مندوبٌ أو موظّف، بدورٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا خانةَ كلمةِ سرّ**: النظامُ يولّدها ويرسلها برسالةٍ إلى الرقم مع رابط التطبيق
 * (أو باب اللوحة للموظّف)، تنتهي بعد مهلتها وتُبدَّل عند أوّل دخول.
 *
 * **والموظّفُ خطوتان في المحرّك بقصد**: حسابٌ يُخلَق ثمّ دورٌ يُمنَح بمساره المخوَّل
 * (`roles.manage` وتأكيد). **ونصفُ عملٍ يُقال كما هو.**
 */
function CreateAccountModal({
  kind,
  allRoles,
  onClose,
  onCreated,
}: {
  kind: NewKind;
  allRoles: readonly Role[];
  onClose: () => void;
  onCreated: () => void;
}) {
  const { user: me } = useAuth();
  const router = useRouter();
  const [phone, setPhone] = useState("");
  const [fullName, setFullName] = useState("");
  const [staffRole, setStaffRole] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState<{ id: string; sent: boolean; temp?: string } | null>(null);
  const [partial, setPartial] = useState<{ id: string; role: string; why: string } | null>(null);

  // **أدوارُ الموظّفين التي يملك المشغّلُ منحَها** — ولا متجرَ ولا صفةَ حساب.
  const staffOptions = signupGroups(allRoles, me?.roles ?? [])
    .filter((g) => g.viaGrant)
    .flatMap((g) => g.roles.filter((r) => !r.locked));

  async function grant(userID: string, role: string) {
    await api(`/api/v1/admin/users/${userID}/roles`, {
      method: "POST",
      body: JSON.stringify({ role, reason: A.createTitle.staff }),
    });
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setPartial(null);
    let made: { id: string; welcome?: { sent: boolean; temp_password?: string } };
    try {
      made = await api<{ id: string; welcome?: { sent: boolean; temp_password?: string } }>("/api/v1/admin/users", {
        method: "POST",
        body: JSON.stringify({
          phone,
          full_name: fullName,
          roles: [kind === "staff" ? "customer" : kind],
          welcome_app: kind === "staff" ? "panel" : "",
        }),
      });
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
      return;
    }
    if (kind === "staff") {
      try {
        await grant(made.id, staffRole);
      } catch (err) {
        setPartial({ id: made.id, role: staffRole, why: errorText(err) });
        setBusy(false);
        onCreated();
        return;
      }
    }
    setBusy(false);
    setDone({ id: made.id, sent: !!made.welcome?.sent, temp: made.welcome?.temp_password });
    onCreated();
  }

  return (
    <Modal open onClose={onClose} title={A.createTitle[kind]}>
      {done ? (
        <div className="space-y-4">
          <Alert tone={done.sent ? "success" : "warning"}>{done.sent ? A.welcomeSent : A.welcomeNotSent}</Alert>
          <TempPasswordNote value={done.temp} />
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => {
                onClose();
                router.push(`/dashboard/users/${done.id}`);
              }}
            >
              {A.openProfile}
            </Button>
            <Button onClick={onClose}>{m.common.confirm}</Button>
          </div>
        </div>
      ) : (
        <form onSubmit={submit} className="space-y-4" autoComplete="off">
          <Input
            id="new-phone"
            label={m.auth.phone}
            icon={<IconPhone />}
            dir="ltr"
            required
            autoComplete="off"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="09xxxxxxxx"
            className="text-end"
          />
          <Input
            id="new-name"
            label={m.admin.users.fullName}
            icon={<IconUser />}
            autoComplete="off"
            value={fullName}
            onChange={(e) => setFullName(e.target.value)}
          />
          {kind === "staff" && (
            <Select id="new-staff-role" label={A.staffRole} required value={staffRole} onChange={(e) => setStaffRole(e.target.value)}>
              <option value="">{A.pickStaffRole}</option>
              {staffOptions.map((r) => (
                <option key={r.code} value={r.code}>
                  {r.label}
                </option>
              ))}
            </Select>
          )}
          <p className="rounded-control bg-field px-3 py-2 text-xs leading-relaxed text-ink-muted">
            {A.noPasswordHint.replace("{h}", fmtNum(72))}
          </p>
          {partial && (
            <Alert className="mb-3">
              <span className="block font-bold">
                {m.admin.users.signupPartial}: {roleLabelByCode(partial.role)}
              </span>
              <span className="mt-1 block text-xs">{m.admin.users.signupPartialHelp}</span>
              <span className="mt-1 block text-xs opacity-80">{partial.why}</span>
              <span className="mt-2 flex flex-wrap gap-2">
                <Button
                  variant="secondary"
                  disabled={busy}
                  onClick={async () => {
                    setBusy(true);
                    try {
                      await grant(partial.id, partial.role);
                      setDone({ id: partial.id, sent: false });
                      setPartial(null);
                    } catch (err) {
                      setPartial({ ...partial, why: errorText(err) });
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  {m.admin.users.signupRetryGrant}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => {
                    onClose();
                    router.push(`/dashboard/users/${partial.id}`);
                  }}
                >
                  {m.admin.users.signupOpenProfile}
                </Button>
              </span>
            </Alert>
          )}
          {error && <Alert>{error}</Alert>}
          <FormActions submit onCancel={onClose} busy={busy} />
        </form>
      )}
    </Modal>
  );
}
