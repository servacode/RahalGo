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
  ViewToggle,
  useViewMode,
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
} from "@rahalgo/ui";
import { api, apiFile, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import WalletModal from "@/components/admin/WalletModal";
import { MerchantModal, CategoriesModal } from "@/components/admin/MerchantModal";
import StatusReasonModal from "@/components/admin/StatusReasonModal";
import RoleBadge, { ROLE_STYLES } from "@/components/admin/RoleBadge";
import { MediaThumb } from "@/components/admin/ImageUpload";
import { TempPasswordNote } from "@/components/admin/accounts/TempPasswordNote";

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
  const canCategories = canCall("POST", "/categories");
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
  const [storeOpen, setStoreOpen] = useState(false);
  const [catsOpen, setCatsOpen] = useState(false);
  const [statusModal, setStatusModal] = useState<{ users: Row[]; status: string } | null>(null);
  const [activateRows, setActivateRows] = useState<Row[] | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkMsg, setBulkMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const [view, setView] = useViewMode("users");

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
        per_page: "10",
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
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (u) => (
        <span className="flex w-full items-center justify-between gap-2">
          <span className="inline-flex min-w-0 items-center gap-2">
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
            <MediaThumb url={u.avatar_thumb_url} alt="" fallback={u.full_name || m.terms.avatarFallback} size={32} />
            <span className="min-w-0">
              <span className="flex items-center gap-1.5">
                <span className="truncate">{u.full_name || "—"}</span>
                {/* **شارةٌ صغيرةٌ للموقوف والمحظور وحدَهما** (قرار ٨). */}
                {u.status === "suspended" && <Badge variant="warning">{m.admin.users.suspended}</Badge>}
                {u.status === "blocked" && <Badge variant="danger">{m.admin.users.blocked}</Badge>}
                {u.is_system ? <Badge variant="neutral">{m.admin.users.systemAccount}</Badge> : null}
              </span>
              {/* **واسمُ متجره تحت اسمه** (قرار ٧). */}
              {u.store_names && u.store_names.length > 0 && (
                <span className="flex items-center gap-1 truncate text-xs text-ink-muted">
                  <IconStore size={11} />
                  {u.store_names.join(" · ")}
                </span>
              )}
            </span>
          </span>
          <button
            type="button"
            title={m.admin.users.viewProfile}
            onClick={(e) => {
              e.stopPropagation();
              router.push(`/dashboard/users/${u.id}`);
            }}
            className="shrink-0 rounded-control p-1 text-ink-muted transition-colors hover:bg-primary-tint hover:text-primary"
          >
            <IconView size={17} />
          </button>
        </span>
      ),
    },
    {
      id: "phone",
      header: m.admin.users.table.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (u) =>
        u.phone ? (
          <span dir="ltr" className="font-medium">
            {u.phone}
          </span>
        ) : (
          <span className="text-ink-muted">{A.phoneHidden}</span>
        ),
    },
    {
      id: "roles",
      header: m.admin.users.table.roles,
      icon: <IconRoles />,
      cell: (u) => (
        <div className="flex flex-wrap justify-end gap-1 sm:justify-start">
          {u.roles.map((r) => (
            <RoleBadge key={r} role={r} />
          ))}
        </div>
      ),
    },
    {
      id: "orders_count",
      header: m.admin.customers.ordersCount,
      icon: <IconOrder />,
      hide: (u) => !u.orders_count,
      cell: (u) => fmtNum(u.orders_count ?? 0),
    },
    ...(money
      ? ([
          {
            id: "orders_spent",
            header: `${m.admin.customers.totalSpent} (${m.common.currency})`,
            icon: <IconOrder />,
            hide: (u: Row) => !u.orders_count,
            cell: (u: Row) => fmtNum(u.orders_spent ?? 0),
          },
          {
            id: "balance",
            header: `${m.admin.customers.balance} (${m.common.currency})`,
            icon: <IconWallet />,
            cell: (u: Row) => fmtNum(u.balance ?? 0),
          },
        ] as DataColumn<Row>[])
      : []),
    {
      id: "last_order",
      header: m.admin.customers.lastOrder,
      icon: <IconStatus />,
      hide: (u) => !u.last_order_at,
      cell: (u) => (u.last_order_at ? fmtDate(u.last_order_at) : "—"),
    },
    {
      id: "rep_stores",
      header: A.storesBroughtCol,
      icon: <IconStore />,
      hide: (u) => !u.rep_stores,
      cell: (u) => fmtNum(u.rep_stores ?? 0),
    },
    ...(money
      ? ([
          {
            id: "commissions",
            header: `${m.admin.sales.totalCommissions} (${m.common.currency})`,
            icon: <IconWallet />,
            hide: (u: Row) => !u.commissions,
            cell: (u: Row) => fmtNum(u.commissions ?? 0),
          },
        ] as DataColumn<Row>[])
      : []),
    {
      id: "open_orders",
      header: m.admin.drivers.openOrders,
      icon: <IconOrder />,
      hide: (u) => !u.open_orders,
      cell: (u) => <Badge variant="primary">{fmtNum(u.open_orders ?? 0)}</Badge>,
    },
  ];

  // ── «إضافة ▾» ─────────────────────────────────────────────────────────
  const addItems: ActionMenuItem[] = [];
  if (canCreateUser) {
    addItems.push(
      { key: "driver", label: A.addDriver, icon: IconDriver, onSelect: () => setCreateKind("driver") },
      { key: "sales", label: A.addRep, icon: IconUsers, onSelect: () => setCreateKind("sales") },
    );
  }
  if (canCreateStore) addItems.push({ key: "store", label: A.addStore, icon: IconStore, onSelect: () => setStoreOpen(true) });
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
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="heading-page">{m.admin.users.title}</h1>
        <div className="flex flex-wrap gap-2">
          {canExport && (
            <Button variant="secondary" onClick={() => void exportCsv()} className="flex items-center gap-1.5">
              <IconReceipt size={16} />
              {A.exportCsv}
            </Button>
          )}
          {canCategories && (
            <Button variant="secondary" onClick={() => setCatsOpen(true)} className="flex items-center gap-1.5">
              <IconGrid size={16} />
              {A.categories}
            </Button>
          )}
          <ActionMenu label={A.add} icon={IconAdd} variant="primary" items={addItems} width={200} />
        </div>
      </div>

      {roleCounts && (
        <div className="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-8">
          <button
            onClick={() => {
              setRole("");
              setStatusFilter("");
              setOnlineOnly(false);
              setPage(1);
            }}
            className={`rounded-card border p-2.5 text-center transition-colors ${role === "" && statusFilter === "" && !onlineOnly ? "border-primary bg-primary-tint" : "border-line bg-surface hover:border-primary-edge"}`}
          >
            <p className="figure">{fmtNum(roleCounts.total)}</p>
            <p className="text-xs text-ink-muted">{m.admin.users.allRoles}</p>
          </button>
          {cards.map(({ key, label, style }) => (
            <button
              key={key}
              onClick={() => {
                setRole(role === key ? "" : key);
                setPage(1);
              }}
              className={`rounded-card border p-2.5 text-center transition-colors ${role === key ? "border-primary bg-primary-tint" : "border-line bg-surface hover:border-primary-edge"}`}
            >
              <p className={`figure inline-flex items-center gap-1 ${style ? style.text : ""}`}>
                {style && <style.Icon size={15} />}
                {fmtNum(roleCounts.roles[key] ?? 0)}
              </p>
              <p className="text-xs text-ink-muted">{label}</p>
            </button>
          ))}
          <button
            onClick={() => {
              setStatusFilter(statusFilter === "restricted" ? "" : "restricted");
              setPage(1);
            }}
            className={`rounded-card border p-2.5 text-center transition-colors ${statusFilter === "restricted" ? "border-danger bg-danger-tint" : "border-line bg-surface hover:border-danger-edge"}`}
          >
            <p className="figure inline-flex items-center gap-1 text-danger">
              <IconBlock size={15} />
              {fmtNum(roleCounts.roles.restricted ?? 0)}
            </p>
            <p className="text-xs text-ink-muted">{A.cardRestricted}</p>
          </button>
          <button
            onClick={() => {
              setOnlineOnly(!onlineOnly);
              setPage(1);
            }}
            className={`rounded-card border p-2.5 text-center transition-colors ${onlineOnly ? "border-success bg-success-tint" : "border-line bg-surface hover:border-success-edge"}`}
          >
            <p className="figure inline-flex items-center gap-1.5 text-success">
              <span className="h-2 w-2 animate-pulse rounded-badge bg-success" />
              {fmtNum(roleCounts.roles.online ?? 0)}
            </p>
            <p className="text-xs text-ink-muted">{m.admin.users.onlineCard}</p>
          </button>
        </div>
      )}

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
        <div className="ms-auto">
          <ViewToggle view={view} onChange={setView} tableLabel={m.common.viewTable} cardsLabel={m.common.viewCards} />
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
        view={view}
        // **و«لا نتائج» لا تُكتب تحت رسالة خطأ** — كانت تقول «لا أحد» والسببُ عطب.
        empty={error ? "" : m.admin.users.noResults}
        onRowClick={(u) => router.push(`/dashboard/users/${u.id}`)}
        actions={
          canWallet || canStatus
            ? (u) =>
                u.is_system ? null : (
                  <>
                    {/* **والمحفظةُ بقدرة المال وحدَها** — كانت مربوطةً بالإيقاف أيضاً فلا تراها الماليّة. */}
                    {canWallet && (
                      <Button variant="secondary" onClick={() => setWalletUser(u)} className="flex items-center gap-1.5">
                        <IconWallet size={15} />
                        {m.admin.users.wallet}
                      </Button>
                    )}
                    {canStatus &&
                      u.id !== me?.id &&
                      (u.status === "active" ? (
                        <>
                          <Button
                            variant="secondary"
                            onClick={() => setStatusModal({ users: [u], status: "suspended" })}
                            className="flex items-center gap-1.5 !text-warning"
                          >
                            <IconBlock size={15} />
                            {m.admin.users.suspend}
                          </Button>
                          <Button
                            variant="danger"
                            onClick={() => setStatusModal({ users: [u], status: "blocked" })}
                            className="flex items-center gap-1.5"
                          >
                            <IconBlock size={15} />
                            {m.admin.users.block}
                          </Button>
                        </>
                      ) : (
                        <Button variant="secondary" onClick={() => setActivateRows([u])} className="flex items-center gap-1.5">
                          <IconUnblock size={15} />
                          {m.admin.users.activate}
                        </Button>
                      ))}
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
      <CategoriesModal open={catsOpen} onClose={() => setCatsOpen(false)} />
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
