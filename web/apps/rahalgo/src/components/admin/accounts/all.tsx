"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Pagination,
  Alert,
  useLiveRefresh,
  Button,
  Input,
  Select,
  Badge,
  Chips,
  Modal,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconUser,
  IconPhone,
  IconRoles,
  IconStatus,
  IconSearch,
  IconAdd,
  IconBlock,
  IconUnblock,
  IconLock,
  IconWallet,
  IconOrder,
  IconLink,
  IconStore,
  IconDriver,
  IconGrid,
  CopyCode,
  IconView,
  usePlatform,
} from "@rahalgo/ui";
import { api, ApiError, tokenStore, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/admin/WalletModal";
import { MerchantModal, CategoriesModal } from "@/components/admin/MerchantModal";
import StatusReasonModal from "@/components/admin/StatusReasonModal";
import RoleBadge, { ROLE_STYLES } from "@/components/admin/RoleBadge";
import { MediaThumb } from "@/components/admin/ImageUpload";

const m = getMessages(defaultLocale);

const ROLE_LABELS: Record<string, string> = m.terms.roleNames;
const ALL_ROLES = Object.keys(ROLE_LABELS);

interface UserPage {
  users: AuthUser[];
  total: number;
  page: number;
  per_page: number;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

export default function AllAccountsTable() {
  const { user: me } = useAuth();
  const router = useRouter();
  const isAdmin = !!me?.roles.includes("admin");

  const [data, setData] = useState<UserPage | null>(null);
  const [query, setQuery] = useState("");
  const [role, setRole] = useState("");
  const [onlineOnly, setOnlineOnly] = useState(false);
  const [statusFilter, setStatusFilter] = useState("");
  const [roleCounts, setRoleCounts] = useState<{ total: number; roles: Record<string, number> } | null>(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [walletUser, setWalletUser] = useState<AuthUser | null>(null);
  // **ونافذتا السائق** — نُقلتا من شاشتهم كما هما.
  // **ونافذةُ المتجر** — صاحبُه أعلاها وبياناتُه أسفلَها.
  const [storeOpen, setStoreOpen] = useState(false);
  const [catsOpen, setCatsOpen] = useState(false);
  const [statusModal, setStatusModal] = useState<{ user: AuthUser; status: string } | null>(null);
  const [view, setView] = useViewMode("users");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({ query, role, status: statusFilter, online: onlineOnly ? "true" : "", page: String(page), per_page: "10" });
      api<{ total: number; roles: Record<string, number> }>("/api/v1/admin/users/stats")
        .then(setRoleCounts)
        .catch(() => undefined);
      setData(await api<UserPage>(`/api/v1/admin/users?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [query, role, statusFilter, onlineOnly, page]);

  useEffect(() => {
    const t = setTimeout(load, 250); // تهدئة البحث
    return () => clearTimeout(t);
  }, [load]);

  useLiveRefresh(["account"], load);

  async function exportCsv() {
    const params = new URLSearchParams({ query, role, status: statusFilter, online: onlineOnly ? "true" : "" });
    const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    const res = await fetch(`${base}/api/v1/admin/users/export?${params}`, {
      headers: { Authorization: `Bearer ${tokenStore.access ?? ""}` },
    });
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "accounts.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  async function setStatus(u: AuthUser, status: string, reason = "") {
    try {
      await api(`/api/v1/admin/users/${u.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status, status_reason: reason }),
      });
      setStatusModal(null);
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<AuthUser>[] = [
    {
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (u) => (
        <span className="flex w-full items-center justify-between gap-2">
          <span className="inline-flex min-w-0 items-center gap-2">
            <MediaThumb url={u.avatar_thumb_url} alt="" fallback={u.full_name || m.terms.avatarFallback} size={32} />
            <span className="truncate">{u.full_name || "—"}</span>
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
      cell: (u) => (
        <span dir="ltr" className="font-medium">
          {u.phone}
        </span>
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
    // ══════════════════════════════════════════════════════════════
    // **وأرقامُه كزبون — هنا لا في تبويبٍ ثانٍ**
    // ══════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «تبويبٌ منفصلٌ باسم الزبائن لا يلزم
    //  أساساً — كلُّ شيءٍ نريده موجودٌ بكلّ الحسابات».)
    //
    // **ولا نقدَ ولا «سلّم اليوم» هنا** — (قرارُ المالك ٢٠٢٦-٠٨-١٥):
    // **رقمان يخصّان يومَ السائق لا هُويّتَه**، وموضعُهما ملفُّه
    // وشاشةُ الصندوق. **والبطاقةُ تقول من هو.**
    //
    // **ولا تُعرض لمن لا طلبَ له**: موظّفٌ وسائقٌ ومتجرٌ أصفارُهم
    // صادقةٌ ولا تعني شيئاً — **وحقلٌ يظهر فارغاً دائماً يُتعلَّم
    // تجاهلُه، ثمّ يمتلئ يوماً فلا يُنظر إليه.**
    //
    // **والرصيدُ للجميع** — كلُّ حسابٍ له محفظة.
    {
      id: "orders_count",
      header: m.admin.customers.ordersCount,
      icon: <IconOrder />,
      hide: (u) => !u.orders_count,
      cell: (u) => fmtNum(u.orders_count ?? 0),
    },
    {
      id: "orders_spent",
      header: `${m.admin.customers.totalSpent} (${m.common.currency})`,
      icon: <IconOrder />,
      hide: (u) => !u.orders_count,
      cell: (u) => fmtNum(u.orders_spent ?? 0),
    },
    {
      id: "balance",
      header: `${m.admin.customers.balance} (${m.common.currency})`,
      icon: <IconWallet />,
      cell: (u) => fmtNum(u.balance ?? 0),
    },
    {
      id: "last_order",
      header: m.admin.customers.lastOrder,
      icon: <IconStatus />,
      hide: (u) => !u.last_order_at,
      cell: (u) => (u.last_order_at ? fmtDate(u.last_order_at) : "—"),
    },
    // ══════════════════════════════════════════════════════════════
    // **وأرقامُه كمندوب — بلا رمز دعوته**
    // ══════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «احذفه من كرت المندوب، لا يلزم أصلاً
    //  أن يكون بالكرت».)
    //
    // **ورمزُه في ملفّه** — ومن أراد أن يمليه على أحدٍ يفتحه، **ولا
    // يُملى رمزٌ من قائمةٍ يُمسح فيها بالعين.**
    {
      id: "rep_stores",
      header: m.admin.sales.merchantsCount,
      icon: <IconStore />,
      hide: (u) => !u.rep_stores,
      cell: (u) => fmtNum(u.rep_stores ?? 0),
    },
    {
      id: "commissions",
      header: `${m.admin.sales.totalCommissions} (${m.common.currency})`,
      icon: <IconWallet />,
      hide: (u) => !u.commissions,
      cell: (u) => fmtNum(u.commissions ?? 0),
    },
    // ══════════════════════════════════════════════════════════════
    // **وحالُه كسائق — بلا ورديّته**
    // ══════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥.) **وموضعُها ملفُّه** — ومن سأل «من
    // يعمل الآن؟» يسأله في شاشة الطلبات لا في جدول الحسابات.
    {
      id: "open_orders",
      header: m.admin.drivers.openOrders,
      icon: <IconOrder />,
      hide: (u) => !u.open_orders,
      cell: (u) => <Badge variant="primary">{fmtNum(u.open_orders ?? 0)}</Badge>,
    },
    // ══════════════════════════════════════════════════════════════
    // **ولا حالةٌ ولا آخرُ ظهورٍ في البطاقة**
    // ══════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
    //
    // **والحالةُ تقولها الأزرارُ نفسُها**: من كان فعّالاً ظهر له
    // «إيقاف» و«حظر»، ومن أُوقف ظهر له «تفعيل». **ورقاقةٌ تقول ما
    // يقوله الزرُّ تحتها زينةٌ لا خبر.**
    //
    // **و«آخرُ ظهور» يقولها ما فوق**: بطاقةُ «المتّصلون الآن»
    // وتصفيةُ الاتّصال. **و«لم يظهر بعد» في كلّ صفٍّ عمودٌ من نصٍّ
    // واحد.**
    //
    // **وكلتاهما في ملفّه** — والبطاقةُ تقول من هو، والملفُّ يفصّل.
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="heading-page">{m.admin.users.title}</h1>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            onClick={exportCsv}
            className="flex items-center gap-1.5"
          >
            <IconView size={16} />
            {m.admin.users.export}
          </Button>
          {isAdmin && (
            <>
              {/* ══════════════════════════════════════════════════════
                  **زرّان لا واحد — والمتجرُ ليس حساباً**
                  ══════════════════════════════════════════════════════

                  (قرارُ المالك ٢٠٢٦-٠٨-١٥: «يصبح لدينا إضافةُ متجرٍ
                   وإضافةُ مستخدم — المستخدمُ لباقي المستخدمين، أمّا
                   المتجرُ فهو لحساب صاحب المتجر والمتجرِ نفسِه».)

                  **و«مستخدم جديد» لا يمنح دورَ التاجر أصلاً**
                  (`checkGrantable`) — **فمن أراد تاجراً وجد البابَ
                  مغلقاً ولا يعرف أين يفتحه.** فصار البابُ هنا. */}
              <Button onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
                <IconAdd size={16} />
                {m.admin.users.create}
              </Button>
              <Button
                variant="secondary"
                onClick={() => setStoreOpen(true)}
                className="flex items-center gap-1.5"
              >
                <IconStore size={16} />
                {m.admin.merchants.create}
              </Button>
              {/* **وإدارةُ التصنيفات من شؤون المنصّة لا من شؤون متجر**
                  — (قرارُ المالك ٢٠٢٦-٠٨-١٥). **وكانت مدفونةً في
                  تبويب المتاجر**: من أراد تصنيفاً جديداً فتح المتاجر
                  ليصل إلى ما لا يخصّها. */}
              <Button
                variant="secondary"
                onClick={() => setCatsOpen(true)}
                className="flex items-center gap-1.5"
              >
                <IconGrid size={16} />
                {m.admin.merchants.categoriesTitle}
              </Button>
            </>
          )}
        </div>
      </div>

      {roleCounts && (
        <div className="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7">
          <button
            onClick={() => setRole("")}
            className={`rounded-card border p-2.5 text-center transition-colors ${role === "" ? "border-primary bg-primary-tint" : "border-line bg-surface hover:border-primary-edge"}`}
          >
            <p className="figure">{roleCounts.total}</p>
            <p className="text-xs text-ink-muted">{m.admin.users.allRoles}</p>
          </button>
          {([
            { key: "staff", label: m.admin.users.staffCard, style: ROLE_STYLES.ops },
            { key: "sales", label: ROLE_LABELS.sales, style: ROLE_STYLES.sales },
            { key: "driver", label: ROLE_LABELS.driver, style: ROLE_STYLES.driver },
            { key: "merchant", label: ROLE_LABELS.merchant, style: ROLE_STYLES.merchant },
            { key: "customer", label: ROLE_LABELS.customer, style: ROLE_STYLES.customer },
          ] as const).map(({ key, label, style }) => (
            <button
              key={key}
              onClick={() => { setRole(role === key ? "" : key); setPage(1); }}
              className={`rounded-card border p-2.5 text-center transition-colors ${role === key ? "border-primary bg-primary-tint" : "border-line bg-surface hover:border-primary-edge"}`}
            >
              <p className={`figure inline-flex items-center gap-1 ${style ? style.text : ""}`}>
                {style && <style.Icon size={15} />}
                {roleCounts.roles[key] ?? 0}
              </p>
              <p className="text-xs text-ink-muted">{label}</p>
            </button>
          ))}
          <button
            onClick={() => { setOnlineOnly(!onlineOnly); setPage(1); }}
            className={`rounded-card border p-2.5 text-center transition-colors ${onlineOnly ? "border-success bg-success-tint" : "border-line bg-surface hover:border-success-edge"}`}
          >
            <p className="figure inline-flex items-center gap-1.5 text-success">
              <span className="h-2 w-2 animate-pulse rounded-badge bg-success" />
              {roleCounts.roles.online ?? 0}
            </p>
            <p className="text-xs text-ink-muted">{m.admin.users.onlineCard}</p>
          </button>
        </div>
      )}

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.users.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            value={role}
            onChange={(e) => {
              setRole(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.users.allRoles}</option>
            {ALL_ROLES.map((r) => (
              <option key={r} value={r}>
                {ROLE_LABELS[r]}
              </option>
            ))}
          </Select>
        </div>
        <div className="w-36">
          <Select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.users.statusFilter.all}</option>
            <option value="active">{m.admin.users.statusFilter.active}</option>
            <option value="suspended">{m.admin.users.statusFilter.suspended}</option>
            <option value="blocked">{m.admin.users.statusFilter.blocked}</option>
          </Select>
        </div>
        <div className="ms-auto">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <DataView
        items={data?.users ?? []}
        getKey={(u) => u.id}
        columns={columns}
        view={view}
        empty={m.admin.users.noResults}
        onRowClick={(u) => router.push(`/dashboard/users/${u.id}`)}
        actions={
          isAdmin
            ? (u) => (
                <>
                  <Button
                    variant="secondary"
                    onClick={() => setWalletUser(u)}
                    className="flex items-center gap-1.5"
                  >
                    <IconWallet size={15} />
                    {m.admin.users.wallet}
                  </Button>
                  {/* **وأفعالُ السائق مع سائقٍ وحدَه** — (قرارُ المالك
                      ٢٠٢٦-٠٨-١٥). **و«إنهاءُ الورديّة» و«التسوية»
                      يُفعلان على عجل**: من فتح ملفَّه ليضغط زرّاً
                      واحداً دفع ثمنَ صفحةٍ كاملة. */}
                  {/* **ولا أفعالَ سائقٍ في البطاقة** — (قرارُ المالك
                      ٢٠٢٦-٠٨-١٥): كشفُ الصندوق وإنهاءُ الورديّة في
                      ملفّه. **والبطاقةُ تقول من هو، والملفُّ يفعل
                      به.** */}
                  {u.id !== me?.id &&
                    (u.status === "active" ? (
                      <>
                        <Button
                          variant="secondary"
                          onClick={() => setStatusModal({ user: u, status: "suspended" })}
                          className="flex items-center gap-1.5 !text-warning"
                        >
                          <IconBlock size={15} />
                          {m.admin.users.suspend}
                        </Button>
                        <Button
                          variant="danger"
                          onClick={() => setStatusModal({ user: u, status: "blocked" })}
                          className="flex items-center gap-1.5"
                        >
                          <IconBlock size={15} />
                          {m.admin.users.block}
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="secondary"
                        onClick={() => setStatus(u, "active")}
                        className="flex items-center gap-1.5"
                      >
                        <IconUnblock size={15} />
                        {m.admin.users.activate}
                      </Button>
                    ))}
                </>
              )
            : undefined
        }
      />

      {/* **ولا «إجمالي» أسفلَ الجدول** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).

          **وبطاقاتُ الأعلى تقول العدَّ أصلاً**، وهذا يقول عددَ ما
          طابق التصفية — **فيُقرأ رقمان مختلفان في شاشةٍ واحدة**
          (`4` فوق و`1` تحت) فلا يُصدَّق أيُّهما. */}
      {data && (
        <div className="mt-4 flex flex-wrap items-center justify-end gap-2 text-sm text-ink-muted">
          {/* **والترقيمُ من المكوّن المشترك.**

              كان مكتوباً هنا وفي ثلاثة ملفّاتٍ أخرى بالشكل نفسِه، **وأرقامُه
              لاتينيّةٌ في واجهةٍ عربية** (`{page} / {totalPages}`) لأنّها لم
              تمرّ بـ`fmtNum`. */}
          <Pagination
            page={page}
            total={data.total}
            perPage={data.per_page}
            onChange={setPage}
          />
        </div>
      )}

      <CreateUserModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => {
          setCreateOpen(false);
          void load();
        }}
      />
      <CategoriesModal open={catsOpen} onClose={() => setCatsOpen(false)} />
      {storeOpen && (
        <MerchantModal
          merchant={null}
          categories={[]}
          onClose={() => setStoreOpen(false)}
          onSaved={() => {
            setStoreOpen(false);
            void load();
          }}
        />
      )}
      {walletUser && <WalletModal user={walletUser} onClose={() => setWalletUser(null)} isAdmin={isAdmin || !!me?.roles.includes("finance")} />}

      {/* **ونافذةُ السبب تُرسَم** — (شهده المالك ٢٠٢٦-٠٨-١٠: «زرُّ إيقاف
          حساب لا يعمل… وزرُّ الحظر لا يعمل»).

          **كانت مكتوبةً في هذا الملفّ كاملةً ولا تُنادى**: الزرُّ يضع
          الاختيارَ في `statusModal` **ولا يقرؤه أحد.** فيضغط الموظّفُ فلا يقع
          شيء — **ولا خطأ ولا سطرٌ في سجلّ** — فيضغط ثانيةً وثالثة، **ثمّ
          يظنّ الحسابَ محظوراً وهو يعمل.** */}
      {statusModal && (
        <StatusReasonModal
          status={statusModal.status}
          onSubmit={(reason) => void setStatus(statusModal.user, statusModal.status, reason)}
          onClose={() => setStatusModal(null)}
        />
      )}
    </div>
  );
}

function CreateUserModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}) {
  // **والطولُ من الإعدادات** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «موحّدةً بكلّ البرنامج»).
  const { passwordMinLength: minLen } = usePlatform();
  const [phone, setPhone] = useState("");
  const [fullName, setFullName] = useState("");
  const [roles, setRoles] = useState<string[]>(["driver"]);
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function toggleRole(r: string) {
    setRoles((prev) => (prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r]));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (password !== password2) {
      setError(m.errors.password_mismatch);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/users", {
        method: "POST",
        body: JSON.stringify({ phone, full_name: fullName, roles, password }),
      });
      setPhone("");
      setFullName("");
      setRoles(["driver"]);
      setPassword("");
      onCreated();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={m.admin.users.createTitle}>
      {/* ══════════════════════════════════════════════════════════════
          **ولا تعبئةَ تلقائيّةً في نموذجٍ يُنشئ حسابَ غيرك**
          ══════════════════════════════════════════════════════════════

          (شهده المالك ٢٠٢٦-٠٨-٠٨: فتح النافذةَ فوجد **رقمَه هو في حقل
           الاسم الكامل** ورقمُ الهاتف فارغ.)

          **كروم يرى نموذجاً فيه كلمتا مرورٍ فيَعُدّه تسجيلَ دخول**، فيبحث
          عن حقل «اسم المستخدم» — **ويقع على أوّل حقلٍ نصّيٍّ يجده**، وهو
          هنا الاسمُ لا الهاتف. فيحشو فيه ما حفظه لهذا الموقع.

          **والحسابُ المُنشَأ حسابُ غيره**: فلا شيءَ محفوظٌ يصلح له —
          **وكلُّ ما يُحشى خطأٌ يُحفظ في القاعدة إن لم يُنتبَه.**

          `off` على النصوص، **و`new-password` على الكلمتين**: هي التي تقول
          لكروم «هذه كلمةٌ تُنشأ لا تُستعاد»، فلا يعرض المحفوظةَ ولا يعرض
          الحفظ. */}
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
        <div>
          <span className="mb-1 block text-sm font-medium">{m.admin.users.rolesLabel}</span>
          {/* **والحبّاتُ مركزيّة** — كانت هنا بـ`px-3 py-1` وفي نافذة الأدوار
              بـ`px-3 py-1.5` وفي صفحة الصنف بثالث. (طلبُ المالك ٢٠٢٦-٠٨-٠٧.) */}
          {/* **وسبعةُ أدوارٍ تلتفّ ولا تنزلق** — من لم يرَ «مدير المنصة»
              لأنّها خارج الإطار لا يعرف أنّها موجودة. (٢٠٢٦-٠٨-٠٨.) */}
          <Chips
            items={ALL_ROLES.map((r) => ({ id: r, label: ROLE_LABELS[r] ?? r }))}
            value={roles}
            onChange={toggleRole}
            wrap
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Input
            id="new-password"
            label={m.admin.users.passwordRequired}
            icon={<IconLock />}
            type="password"
            required
            autoComplete="new-password"
            minLength={minLen}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Input
            id="new-password2"
            label={m.admin.users.confirmPassword}
            icon={<IconLock />}
            type="password"
            required
            autoComplete="new-password"
            minLength={minLen}
            value={password2}
            onChange={(e) => setPassword2(e.target.value)}
          />
        </div>
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy || roles.length === 0}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
