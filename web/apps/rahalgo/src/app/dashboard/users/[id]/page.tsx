"use client";

/** صفحة تفاصيل الحساب: الملف، مؤشرات حسب أدواره، سجل المحفظة الكامل،
 *  وأدوات الأدمن (عمليات المحفظة، إعادة تعيين كلمة المرور). */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDate, fmtDateTime } from "@rahalgo/i18n";
import {
  Tabs,
  Alert,
  Badge,
  Button,
  Input,
  Modal,
  Pagination,
  FormSection,
  IconUser,
  IconPhone,
  IconDate,
  IconWallet,
  IconOrder,
  IconStore,
  IconDriver,
  IconLock,
  IconRoles,
  IconLink,
  IconPrev,
  IconLogout,
  IconStatus,
  IconStar,
  IconSupport,
  IconEdit,
  IconBlock,
  IconUnblock,
  IconBalance,
  IconLocation,
  LoadingState,
  usePlatform,
  Money,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";
import { WarningsSection } from "@/components/admin/accounts/WarningsSection";
import {
  OrdersTab,
  AddressesTab,
  CashboxTab,
  StoresTab,
} from "@/components/admin/ProfileRoleTabs";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/admin/WalletModal";
import { ManageRolesModal } from "@/components/admin/ManageRolesModal";
import StatusReasonModal from "@/components/admin/StatusReasonModal";
import { MediaThumb } from "@/components/admin/ImageUpload";
import RoleBadge from "@/components/admin/RoleBadge";

const m = getMessages(defaultLocale);
const P = m.admin.users.profile;
const KINDS: Record<string, string> = m.admin.users.txKinds;
const ACTIONS: Record<string, string> = m.admin.audit.actions;

interface FinEntry {
  ref: string;
  label: string;
  amount: number;
  status?: string;
  reason?: string;
  date: string;
}
interface FinData {
  roles: string[];
  rates: { label: string; percent: number }[];
  owed_to: { total: number; count: number; items: FinEntry[] };
  owed_by: { total: number; count: number; items: FinEntry[] };
  returns: FinEntry[];
  /** **عددُ المرتجعات كلِّها وسقفُ العرض** — (٢٠٢٦-٠٨-١٠). */
  returns_count: number;
  limit: number;
}

interface Profile {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  invite_code: string | null;
  avatar_thumb_url?: string | null;
  status_reason: string;
  admin_notes: string;
  active_sessions: number;
  roles: string[];
  created_at: string;
  referrals_count: number;
  referrals_earned: number;
  balance: number;
  orders_count: number;
  orders_spent: number;
  merchants: string[];
  rep_stores: number;
  commissions: number;
  last_seen_at: string | null;
  on_shift: boolean;
  driver_cash: number;
  deliveries: number;
  delivered_today: number;
}

interface Activity {
  action: string;
  entity: string;
  entity_id: string;
  ip: string;
  details: string;
  by_name: string | null;
  by_self: boolean;
  created_at: string;
}

interface Feedback {
  tickets: { number: number; subject: string; status: string; compensation: number; created_at: string }[];
  ratings_given: { order_number: number; merchant_name: string; platform_stars: number; driver_stars: number | null; comment: string; created_at: string }[];
  /** **أعدادُ الكلّ** — والمعروضُ صفحةٌ منه. (٢٠٢٦-٠٨-١٠.) */
  tickets_count: number;
  ratings_given_count: number;
  ratings_received_count: number;
  per_page: number;
  ratings_received: { order_number: number; merchant_name: string; stars: number; comment: string; created_at: string; as: string }[];
  avg_received: number | null;
}

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  by_name: string | null;
  order_number: number | null;
  ticket_number: number | null;
  created_at: string;
}

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    const known = (m.errors as Record<string, string>)[key];
    if (known) return known;
  }
  return m.errors.internal;
}

export default function UserProfilePage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const canWallet = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [p, setP] = useState<Profile | null>(null);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  /** **سجلُّ النشاط مُرقَّم** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **وهو أسرعُ ما ينمو في الحساب**: كلُّ دخولٍ وكلُّ تعديلٍ سطر. **ومئةٌ
      صامتةٌ تحجب ما قبل الأسبوع الماضي عمّن يراجع** — وهو ما يُبحث عنه
      بالضبط حين يُشتكى على حساب. */
  const [actPage, setActPage] = useState(1);
  /** **ولكلّ قائمةٍ صفحتُها** — **ورقمٌ واحدٌ لثلاثتها يقلّب ما لم يُطلب**:
      يبحث في تذاكره فتقفز تقييماتُه معها. */
  const [tPage, setTPage] = useState(1);
  const [gPage, setGPage] = useState(1);
  const [rPage, setRPage] = useState(1);
  const [actCount, setActCount] = useState(0);
  const [actPer, setActPer] = useState(25);
  const [feedback, setFeedback] = useState<Feedback | null>(null);
  const [phoneOpen, setPhoneOpen] = useState(false);
  const [statusModal, setStatusModal] = useState<string | null>(null);
  const [fin, setFin] = useState<FinData | null>(null);
  const [tab, setTab] = useState<
    | "overview"
    | "orders"
    | "addresses"
    | "cashbox"
    | "stores"
    | "wallet"
    | "financials"
    | "feedback"
    | "activity"
  >("overview");
  const [notice, setNotice] = useState("");
  const [walletOpen, setWalletOpen] = useState(false);
  const [rolesOpen, setRolesOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [error, setError] = useState("");

  async function setStatus(status: string, reason: string) {
    await api(`/api/v1/admin/users/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ status, status_reason: reason }),
    });
    setStatusModal(null);
    await load();
  }

  const load = useCallback(async () => {
    try {
      setP(await api<Profile>(`/api/v1/admin/users/${id}`));
      const st = await api<{ transactions: Tx[] }>(`/api/v1/admin/users/${id}/wallet`);
      setTxs(st.transactions);
      setFeedback(
        await api<Feedback>(
          `/api/v1/admin/users/${id}/feedback?t_page=${tPage}&g_page=${gPage}&r_page=${rPage}`,
        ),
      );
      setFin(await api<FinData>(`/api/v1/admin/users/${id}/financials`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [id, tPage, gPage, rPage]);

  useEffect(() => {
    void load();
  }, [load]);

  /**
   * **وسجلُّ النشاط يُجلَب وحدَه — بصفحته.**
   *
   * **ولو بقي في `load` لَما تحرّك بتقليب الصفحة**: تابعُ `useCallback`
   * معلّقٌ بـ`[id]` وحدَه، **فيُرسم الترقيمُ ويُضغط ولا يقع شيء** — وهي
   * عائلةُ العطب نفسِها التي أوقفت زرَّي الحظر والإيقاف.
   *
   * **وجلبُه وحدَه أخفُّ أيضاً**: تقليبُ صفحةٍ لا يُعيد جلبَ الحساب
   * والمحفظةِ والتقييماتِ والماليّات معه.
   */
  useEffect(() => {
    api<{ activity: Activity[]; total: number; per_page: number }>(
      `/api/v1/admin/users/${id}/activity?page=${actPage}`,
    )
      .then((act) => {
        setActivity(act?.activity ?? []);
        setActCount(act?.total ?? 0);
        setActPer(act?.per_page || 25);
      })
      // @empty-ok **وسجلٌّ لا يُجلب لا يُسقط الصفحة** — بقيّةُ الحساب تُقرأ.
      .catch(() => setActivity([]));
  }, [id, actPage]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!p) return <LoadingState variant="text" />;

  const has = (r: string) => p.roles.includes(r);

  const stats: { label: string; value: string; icon: React.ReactNode; onClick?: () => void }[] = [
    {
      label: `${m.admin.customers.balance} (${m.common.currency})`,
      value: fmtNum(p.balance),
      icon: <IconWallet className="text-primary" />,
    },
  ];
  // ══════════════════════════════════════════════════════════════════
  // **وكم دعا وكم قبض** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).
  // ══════════════════════════════════════════════════════════════════
  //
  // **والمقبولةُ من رُوفئ عنه فعلاً** لا من سجّل برمزه: المكافأةُ
  // تُقيَّد حين يُسلَّم أوّلُ طلبٍ للمدعوّ — **ومن عدّ المسجّلين وعد
  // صاحبَه بمالٍ لم يستحقّه.**
  //
  // **ويُعرضان ولو كانا صفراً** — **والسؤالُ «كم دعا؟» جوابُه «لا
  // أحد»، وهو جوابٌ لا فراغ.** (وقاعدةُ إخفاء الصفر للبطاقة التي
  // تُمسح بالعين في قائمةٍ من مئة — **والملفُّ يُفتح لشخصٍ بعينه
  // ليُسأل عنه.**)
  if (has("customer")) {
    stats.push(
      {
        label: P.referralsCount,
        value: fmtNum(p.referrals_count),
        icon: <IconLink />,
      },
      {
        label: `${P.referralsEarned} (${m.common.currency})`,
        value: fmtNum(p.referrals_earned),
        icon: <IconWallet className="text-success" />,
      }
    );
  }
  if (has("customer") || p.orders_count > 0) {
    stats.push(
      {
        label: P.ordersCount,
        value: fmtNum(p.orders_count),
        icon: <IconOrder />,
        onClick: () => router.push(`/dashboard/orders?q=${encodeURIComponent(p.phone)}`),
      },
      {
        label: `${P.spent} (${m.common.currency})`,
        value: fmtNum(p.orders_spent),
        icon: <IconOrder className="text-success" />,
      }
    );
  }
  // **وورديّتُه هنا** — (قرارُ المالك ٢٠٢٦-٠٨-١٥: حُذفت من البطاقة).
  // **ومن سأل «من يعمل الآن؟» يسأله في شاشة الطلبات**، ومن فتح ملفَّ
  // سائقٍ بعينه يريد أن يعرف حالَه.
  if (has("driver")) {
    stats.push({
      label: m.terms.onShift,
      value: p.on_shift ? m.terms.onShift : m.terms.offShift,
      icon: <IconDriver className={p.on_shift ? "text-success" : ""} />,
    });
  }
  if (has("sales")) {
    stats.push(
      { label: P.repStores, value: fmtNum(p.rep_stores), icon: <IconStore /> },
      {
        label: `${P.commissions} (${m.common.currency})`,
        value: fmtNum(p.commissions),
        icon: <IconWallet className="text-success" />,
      }
    );
  }
  if (feedback?.avg_received != null) {
    stats.push({
      label: P.avgRating,
      value: feedback.avg_received.toFixed(1),
      icon: <IconStar className="fill-accent text-accent-text" />,
    });
  }
  if (has("driver")) {
    stats.push(
      // **وسُلّم اليومَ غيرُ سُلّم كلَّه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥:
      // حُذف من البطاقة). **ذاك يقول «كم عمل في عمره» وهذا يقول
      // «أيعمل اليوم؟».**
      {
        label: m.admin.drivers.deliveredToday,
        value: fmtNum(p.delivered_today),
        icon: <IconOrder className="text-success" />,
      },
      {
        label: P.deliveries,
        value: fmtNum(p.deliveries),
        icon: <IconDriver />,
      },
      {
        label: `${P.driverCash} (${m.common.currency})`,
        value: fmtNum(p.driver_cash),
        icon: <IconWallet className="text-accent-dark" />,
      }
    );
  }

  return (
    <div>
      <button
        onClick={() => router.push("/dashboard/users")}
        className="mb-3 flex items-center gap-1 text-sm text-ink-muted hover:text-primary"
      >
        <IconPrev size={15} />
        {m.admin.users.title}
      </button>

      {/* الترويسة: البيانات في الصدارة والأزرار سطر واحد (ملاحظة مراجعة) */}
      <div className="mb-3 surface p-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <MediaThumb url={p.avatar_thumb_url} alt="" fallback={p.full_name || m.terms.avatarFallback} size={56} />
            <div className="min-w-0">
              <h1 className="heading-card truncate">{p.full_name || "—"}</h1>
              <p className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-ink-muted">
                <span className="inline-flex items-center gap-1">
                  <IconPhone size={12} />
                  <span dir="ltr">{p.phone}</span>
                </span>
                <span className="inline-flex items-center gap-1">
                  <IconDate size={12} />
                  {fmtDate(p.created_at)}
                </span>
                {p.invite_code && (
                  <Badge variant="accent" dir="ltr" className="px-1.5 font-mono">
                    {p.invite_code}
                  </Badge>
                )}
              </p>
              <div className="mt-1 flex flex-wrap items-center gap-1">
                {p.roles.map((r) => (
                  <RoleBadge key={r} role={r} />
                ))}
                <Badge
                  variant={p.status === "active" ? "success" : p.status === "suspended" ? "warning" : "danger"}
                >
                  {p.status === "active"
                    ? m.admin.users.active
                    : p.status === "suspended"
                      ? m.admin.users.suspended
                      : m.admin.users.blocked}
                </Badge>
                {p.status !== "active" && p.status_reason && (
                  <span className="text-xs text-danger">{p.status_reason}</span>
                )}
                {/* **وآخرُ ظهوره هنا** — (قرارُ المالك ٢٠٢٦-٠٨-١٥:
                    حُذف من البطاقة). **ومن فتح ملفَّ شخصٍ بعينه يسأل
                    «متى كان هنا آخرَ مرّة؟».** */}
                <span className="text-xs text-ink-muted">
                  {m.admin.users.lastSeen}:{" "}
                  {p.last_seen_at ? fmtDateTime(p.last_seen_at) : m.admin.users.neverSeen}
                </span>
              </div>
            </div>
          </div>

          {/* ══════════════════════════════════════════════════════════
              **وستّةُ أزرارٍ تلتفّ — وإلّا أزاحت الصفحةَ كلَّها**
              ══════════════════════════════════════════════════════════

              (شهده المالك ٢٠٢٦-٠٨-١١ بصورة: الصفحةُ منزاحةٌ، وبطاقاتُها
               خارجَ الشاشة، والأزرارُ وحدَها تُرى.)

              **كان `flex-nowrap` مقصوداً**: «الأزرارُ سطرٌ واحد» ملاحظةُ
              مراجعةٍ قديمة — **وهي صحيحةٌ على الحاسوب.** وستّةُ أزرارٍ
              بنصوصٍ عربيّةٍ تحتاج نحوَ ستّمئة بكسل، **والهاتفُ ثلاثمئةٌ
              وستّون.**

              **وصفٌّ لا يلتفّ لا يُقصّ — إنّما يوسّع المستندَ كلَّه.**
              فينزاح الشريطُ العلويُّ ويخرج المحتوى، **وتُقرأ الصفحةُ
              منهارةً بسبب صفٍّ فيها.** (وهو العطبُ نفسُه الذي أُصلح في
              رأس التقارير.)

              **و`whitespace-nowrap` تبقى** — تمنع كسرَ الكلمة داخلَ الزرّ،
              **ولا علاقةَ لها بالتفاف الصفّ.** */}
          <div className="flex flex-wrap items-center gap-1.5 whitespace-nowrap">
            {canWallet && (
              <Button onClick={() => setWalletOpen(true)} className="flex items-center gap-1.5 !px-2.5">
                <IconWallet size={15} />
                {m.admin.users.wallet}
              </Button>
            )}
            {isAdmin && (
              <>
                {/* ══════════════════════════════════════════════════
                    **وبابُ الأدوار هنا لا في كلّ بطاقة**
                    ══════════════════════════════════════════════════

                    (قرارُ المالك ٢٠٢٦-٠٨-١٥: «زرُّ الأدوار بلا معنًى
                     داخل الكرت — نحدّد دورَ المستخدم بداية تسجيل
                     حسابٍ جديد فقط».)

                    **ودورُ الزبون ممنوحٌ تلقائيّاً لكلّ عامل**، ودورٌ
                    ميدانيٌّ ثانٍ يرفضه المحرّك، ودورُ التاجر لا يُمنح
                    بيد — **فلا يبقى إلّا زبونٌ صار سائقاً أو مندوباً**،
                    وتقع مرّةً في العمر. **فموضعُها ملفُّه لا بطاقتُه.** */}
                <Button
                  variant="secondary"
                  onClick={() => setRolesOpen(true)}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconRoles size={15} />
                  {m.admin.users.manageRoles}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => setResetOpen(true)}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconLock size={15} />
                  {P.passwordBtn}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => setPhoneOpen(true)}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconPhone size={15} />
                  {P.changePhone}
                </Button>
                <Button
                  variant="danger"
                  onClick={async () => {
                    const res = await api<{ revoked_sessions: number }>(
                      `/api/v1/admin/users/${p.id}/logout-all`,
                      { method: "POST" }
                    );
                    setNotice(P.logoutAllDone.replace("{n}", String(res.revoked_sessions)));
                    await load();
                  }}
                  className="flex items-center gap-1.5 !px-2.5"
                >
                  <IconLogout size={15} />
                  {P.logoutAllShort} ({fmtNum(p.active_sessions)})
                </Button>
                {me?.id !== p.id &&
                  (p.status === "active" ? (
                    <>
                      <Button
                        variant="secondary"
                        onClick={() => setStatusModal("suspended")}
                        className="flex items-center gap-1.5 !px-2.5 !text-warning"
                      >
                        <IconBlock size={15} />
                        {m.admin.users.suspend}
                      </Button>
                      <Button
                        variant="danger"
                        onClick={() => setStatusModal("blocked")}
                        className="flex items-center gap-1.5 !px-2.5"
                      >
                        <IconBlock size={15} />
                        {m.admin.users.block}
                      </Button>
                    </>
                  ) : (
                    <Button
                      variant="secondary"
                      onClick={() => void setStatus("active", "")}
                      className="flex items-center gap-1.5 !px-2.5"
                    >
                      <IconUnblock size={15} />
                      {m.admin.users.activate}
                    </Button>
                  ))}
              </>
            )}
          </div>
        </div>
      </div>

      {notice && (
        <Alert tone="success" className="mb-4">{notice}</Alert>
      )}

      {/* التبويبات — كل قسم في تبويبه (ملاحظة مراجعة)

          **وتبويباتُ الدور تظهر لمن يملكه وحدَه.** «صندوق نقده» في ملفّ زبونٍ
          سطرٌ فارغٌ يُسأل عنه، **و«متاجرُ جلبها» في ملفّ سائقٍ كذلك.** فالملفُّ
          يعرض ما يخصّ صاحبَه لا ما يخصّ النظام. */}
      {/* **والفلترةُ بالأدوار تبقى** — من ليس متجراً لا يرى تبويبَ متاجره. */}
      <Tabs
        className="mb-3"
        items={([
            { key: "overview", label: P.tabs.overview, icon: IconUser, show: true },
            {
              key: "orders",
              label: P.roleTabs.orders,
              icon: IconOrder,
              show: has("customer") || has("driver"),
            },
            {
              key: "addresses",
              label: P.roleTabs.addresses,
              icon: IconLocation,
              show: has("customer"),
            },
            {
              key: "cashbox",
              label: P.roleTabs.cashbox,
              icon: IconBalance,
              show: has("driver"),
            },
            {
              key: "stores",
              label: P.roleTabs.stores,
              icon: IconStore,
              show: has("merchant") || has("sales"),
            },
            { key: "wallet", label: P.tabs.wallet, icon: IconWallet, show: true },
            { key: "financials", label: P.tabs.financials, icon: IconBalance, show: true },
            { key: "feedback", label: P.tabs.feedback, icon: IconStar, show: true },
            { key: "activity", label: P.tabs.activity, icon: IconStatus, show: true },
          ] as const)
          .filter((t) => t.show)
          .map(({ key, label, icon }) => ({ key, label, icon }))}
        value={tab}
        onChange={(k) => setTab(k as typeof tab)}
      />

      {tab === "orders" && <OrdersTab userID={id} roles={p.roles} />}
      {tab === "addresses" && <AddressesTab userID={id} />}
      {tab === "cashbox" && (
        <CashboxTab userID={id} canSettle={canWallet} onSettled={() => void load()} />
      )}
      {tab === "stores" && <StoresTab userID={id} roles={p.roles} />}

      {tab === "overview" && (
      <>
      {/* المؤشرات حسب الأدوار */}
      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
        {stats.map((s) => (
          <div
            key={s.label}
            onClick={s.onClick}
            className={`surface p-3 ${
              s.onClick ? "cursor-pointer transition-shadow hover:border-primary-edge hover:elev-1" : ""
            }`}
          >
            <div className="mb-1">{s.icon}</div>
            <p className="figure">{s.value}</p>
            <p className="text-xs text-ink-muted">{s.label}</p>
          </div>
        ))}
      </div>

      {p.merchants.length > 0 && (
        <div className="mb-3 surface p-3">
          <p className="mb-2 flex items-center gap-1.5 text-sm font-bold">
            <IconStore size={15} className="text-primary" />
            {P.merchantsOwned}
          </p>
          <div className="flex flex-wrap gap-2">
            {p.merchants.map((name) => (
              <Badge key={name} variant="primary">
                {name}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* **والإنذاراتُ في ملفّه** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «اجعله لكلّ
          الأدوار حتّى الزبون»).

          **ومن يقرّر أن يُنذر يسأل أوّلاً: كم مرّةً سبق؟** — والجوابُ هنا، مع
          رصيده وطلباته وتقييماته. */}
      <WarningsSection userID={p.id} />

      <FormSection title={P.notes} icon={<IconEdit />}>
        <NotesEditor
          userID={p.id}
          initial={p.admin_notes}
          onSaved={load}
        />
      </FormSection>
      </>
      )}

      {tab === "wallet" && (
      <FormSection title={P.statement} icon={<IconWallet />}>
        {txs.length === 0 ? (
          <p className="py-6 text-center text-sm text-ink-muted">{P.statementEmpty}</p>
        ) : (
          <ul className="space-y-1.5">
            {txs.map((t) => (
              <li
                key={t.id}
                className="flex items-center justify-between gap-3 rounded-control border border-line px-3 py-2 text-sm"
              >
                <div className="min-w-0 flex-1">
                  <span className="flex flex-wrap items-center gap-2">
                    <Badge variant={t.amount > 0 ? "success" : "danger"}>
                      {KINDS[t.kind] ?? t.kind}
                    </Badge>
                    {t.order_number != null && (
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${t.order_number}`)}
                        className="text-xs font-medium text-primary hover:underline"
                      >
                        {P.orderRef} #{fmtRef(t.order_number)}
                      </button>
                    )}
                    {t.ticket_number != null && (
                      <span className="text-xs font-medium text-primary">
                        {P.ticketRef} #{fmtRef(t.ticket_number)}
                      </span>
                    )}
                    {t.note && <span className="truncate text-xs text-ink-muted">{t.note}</span>}
                  </span>
                  <p className="mt-0.5 text-xs text-ink-muted">
                    {P.by}: {t.by_name ?? P.system}
                  </p>
                </div>
                <span className="flex shrink-0 items-center gap-3">
                  <span className={`font-bold ${t.amount > 0 ? "text-success" : "text-danger"}`} dir="ltr">
                    {t.amount > 0 ? "+" : ""}
                    {fmtNum(t.amount)}
                  </span>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(t.created_at)}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </FormSection>
      )}

      {tab === "financials" && fin && (
        <div className="space-y-4">
          {/* النِسَب المطبّقة */}
          {fin.rates.length > 0 && (
            <FormSection title={P.fin.rates} icon={<IconBalance />}>
              <ul className="space-y-1.5">
                {fin.rates.map((rt, i) => (
                  <li
                    key={i}
                    className="flex items-center justify-between rounded-control border border-line px-3 py-2 text-sm"
                  >
                    <span>{rt.label}</span>
                    <span className="font-bold text-primary-dark" dir="ltr">
                      {rt.percent}%
                    </span>
                  </li>
                ))}
              </ul>
            </FormSection>
          )}

          {/* مستحق له / مستحق عليه */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <FinBucket
              title={P.fin.owedTo}
              total={fin.owed_to.total}
              count={fin.owed_to.count}
              limit={fin.limit}
              items={fin.owed_to.items}
              positive
              onOrder={(n) => router.push(`/dashboard/orders?q=${n}`)}
            />
            <FinBucket
              title={P.fin.owedBy}
              total={fin.owed_by.total}
              count={fin.owed_by.count}
              limit={fin.limit}
              items={fin.owed_by.items}
              positive={false}
              onOrder={(n) => router.push(`/dashboard/orders?q=${n}`)}
            />
          </div>

          {/* الطلبات المرتجعة وأسبابها */}
          <FormSection title={P.fin.returns} icon={<IconBlock />}>
            {fin.returns.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.fin.returnsEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {fin.returns.map((e) => (
                  <li
                    key={e.ref}
                    className="flex items-center justify-between gap-3 rounded-control border border-line px-3 py-2 text-sm"
                  >
                    <div className="min-w-0 flex-1">
                      <span className="flex flex-wrap items-center gap-2">
                        <Badge variant="danger">
                          {(P.fin.st as Record<string, string>)[e.status ?? ""] ?? e.status}
                        </Badge>
                        <span className="font-medium">{e.label}</span>
                        {e.reason && (
                          <span className="truncate text-xs text-ink-muted">— {e.reason}</span>
                        )}
                      </span>
                    </div>
                    <span className="shrink-0 text-xs text-ink-muted" dir="ltr">
                      {fmtDate(e.date)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
            {/* **والحدُّ يُقال هنا أيضاً** — **ومن راجع مرتجعاتِ متجرٍ فرأى
                خمسين ظنّها خمسين**، وبنى عليها حكماً على أدائه. */}
            {fin.returns_count > fin.returns.length && (
              <p className="mt-2 text-center text-xs text-ink-muted">
                {P.fin.showingLatest
                  .replace("{n}", fmtNum(fin.returns.length))
                  .replace("{all}", fmtNum(fin.returns_count))}
              </p>
            )}
          </FormSection>
        </div>
      )}

      {tab === "feedback" && feedback && (
        <div className="space-y-4">
          <FormSection title={P.ticketsSection} icon={<IconSupport />}>
            {feedback.tickets.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ticketsEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.tickets.map((t) => (
                  <li
                    key={t.number}
                    onClick={() => router.push("/dashboard/tickets")}
                    className="flex cursor-pointer flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm hover:bg-row-hover"
                  >
                    <span className="flex items-center gap-2">
                      <span className="font-bold">#{fmtRef(t.number)}</span>
                      <span className="truncate">{t.subject}</span>
                    </span>
                    <span className="flex items-center gap-2">
                      {t.compensation > 0 && (
                        <span className="text-xs text-success">
                          +<Money value={t.compensation} />
                        </span>
                      )}
                      <Badge
                        variant={t.status === "resolved" ? "success" : t.status === "open" ? "warning" : "primary"}
                      >
                        {(m.admin.tickets.status as Record<string, string>)[t.status] ?? t.status}
                      </Badge>
                      <span className="text-xs text-ink-muted" dir="ltr">
                        {fmtDate(t.created_at)}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          {feedback.tickets_count > feedback.per_page && (
            <div className="mt-3 flex justify-center">
              <Pagination
                page={tPage}
                total={feedback.tickets_count}
                perPage={feedback.per_page}
                onChange={setTPage}
              />
            </div>
          )}
          </FormSection>

          <FormSection title={P.ratingsGiven} icon={<IconStar />}>
            {feedback.ratings_given.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ratingsGivenEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.ratings_given.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                  >
                    <span className="flex min-w-0 flex-wrap items-center gap-2">
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${rt.order_number}`)}
                        className="font-medium text-primary hover:underline"
                      >
                        #{fmtRef(rt.order_number)}
                      </button>
                      <span className="text-ink-muted">{rt.merchant_name}</span>
                      <span className="flex items-center gap-1 text-accent-dark"><IconStar size={12} className="fill-accent-dark" />{rt.platform_stars}</span>
                      {rt.driver_stars != null && (
                        <span className="text-xs text-ink-muted">
                          ({m.admin.ordersPage.rating.driver}: {rt.driver_stars})
                        </span>
                      )}
                      {rt.comment && (
                        <span className="truncate text-xs text-ink-muted">"{rt.comment}"</span>
                      )}
                    </span>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(rt.created_at)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          {feedback.ratings_given_count > feedback.per_page && (
            <div className="mt-3 flex justify-center">
              <Pagination
                page={gPage}
                total={feedback.ratings_given_count}
                perPage={feedback.per_page}
                onChange={setGPage}
              />
            </div>
          )}
          </FormSection>

          <FormSection title={P.ratingsRecv} icon={<IconStar />}>
            {feedback.ratings_received.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ratingsRecvEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.ratings_received.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                  >
                    <span className="flex min-w-0 flex-wrap items-center gap-2">
                      <Badge variant={rt.as === "driver" ? "primary" : "warning"}>
                        {rt.as === "driver" ? P.asDriver : P.asMerchant}
                      </Badge>
                      <span className="flex items-center gap-1 text-accent-dark"><IconStar size={12} className="fill-accent-dark" />{rt.stars}</span>
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${rt.order_number}`)}
                        className="font-medium text-primary hover:underline"
                      >
                        #{fmtRef(rt.order_number)}
                      </button>
                      <span className="text-xs text-ink-muted">{rt.merchant_name}</span>
                      {rt.comment && (
                        <span className="truncate text-xs text-ink-muted">"{rt.comment}"</span>
                      )}
                    </span>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(rt.created_at)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          {feedback.ratings_received_count > feedback.per_page && (
            <div className="mt-3 flex justify-center">
              <Pagination
                page={rPage}
                total={feedback.ratings_received_count}
                perPage={feedback.per_page}
                onChange={setRPage}
              />
            </div>
          )}
          </FormSection>
        </div>
      )}

      {tab === "activity" && (
      <div>
        <FormSection title={P.activity} icon={<IconStatus />}>
          {activity.length === 0 ? (
            <p className="py-6 text-center text-sm text-ink-muted">{P.activityEmpty}</p>
          ) : (
            <ul className="space-y-1.5">
              {activity.map((a, i) => (
                <li
                  key={i}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                >
                  <span className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="font-medium">{ACTIONS[a.action] ?? a.action}</span>
                    <span className="text-xs text-ink-muted">
                      {P.by}: {a.by_self ? P.bySelf : (a.by_name ?? P.system)}
                    </span>
                    {a.ip && (
                      <span className="text-xs text-ink-muted" dir="ltr">
                        {a.ip}
                      </span>
                    )}
                  </span>
                  <span className="shrink-0 text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(a.created_at)}
                  </span>
                </li>
              ))}
            </ul>
          )}
          {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
          {actCount > actPer && (
            <div className="mt-3 flex justify-center">
              <Pagination
                page={actPage}
                total={actCount}
                perPage={actPer}
                onChange={setActPage}
              />
            </div>
          )}
        </FormSection>
      </div>
      )}

      {statusModal && (
        <StatusReasonModal
          status={statusModal}
          onSubmit={(reason) => setStatus(statusModal, reason)}
          onClose={() => setStatusModal(null)}
        />
      )}
      {phoneOpen && (
        <ChangePhoneModal
          userID={p.id}
          current={p.phone}
          onClose={() => setPhoneOpen(false)}
          onDone={() => {
            setPhoneOpen(false);
            void load();
          }}
        />
      )}
      {walletOpen && (
        <WalletModal
          user={{ id: p.id, phone: p.phone, full_name: p.full_name }}
          isAdmin={canWallet}
          onClose={() => {
            setWalletOpen(false);
            void load();
          }}
        />
      )}
      {resetOpen && (
        <ResetPasswordModal userID={p.id} onClose={() => setResetOpen(false)} />
      )}
      <ManageRolesModal
        user={
          rolesOpen
            ? ({ id: p.id, phone: p.phone, full_name: p.full_name, roles: p.roles } as AuthUser)
            : null
        }
        onClose={() => setRolesOpen(false)}
        onChanged={load}
      />
    </div>
  );
}

function ResetPasswordModal({ userID, onClose }: { userID: string; onClose: () => void }) {
  // **والطولُ من الإعدادات** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «موحّدةً بكلّ البرنامج»).
  const { passwordMinLength: minLen } = usePlatform();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/users/${userID}/password`, {
        method: "POST",
        body: JSON.stringify({ password }),
      });
      setDone(true);
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.resetPassword}>
      {done ? (
        <div className="space-y-4 text-center">
          <Alert tone="success">{P.resetDone}</Alert>
          <Button onClick={onClose}>{m.common.confirm}</Button>
        </div>
      ) : (
        <form onSubmit={submit} className="space-y-4">
          <Input
            id="new-pw"
            label={P.newPassword}
            dir="ltr"
            required
            minLength={minLen}
            autoFocus
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="font-mono"
          />
          {error && (
            <Alert>{error}</Alert>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              {m.common.cancel}
            </Button>
            <Button type="submit" disabled={busy}>
              {m.common.save}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  );
}

function ChangePhoneModal({
  userID,
  current,
  onClose,
  onDone,
}: {
  userID: string;
  current: string;
  onClose: () => void;
  onDone: () => void;
}) {
  const [phone, setPhone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/users/${userID}`, {
        method: "PATCH",
        body: JSON.stringify({ phone }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.changePhone}>
      <form onSubmit={submit} className="space-y-4">
        <p className="text-sm text-ink-muted" dir="ltr">{current}</p>
        <Input
          id="new-phone"
          label={P.newPhone}
          icon={<IconPhone />}
          dir="ltr"
          required
          autoFocus
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="text-end"
          placeholder="09xxxxxxxx"
        />
        <p className="rounded-control bg-field px-3 py-2 text-xs leading-relaxed text-ink-muted">
          {P.phoneHint}
        </p>
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function NotesEditor({
  userID,
  initial,
  onSaved,
}: {
  userID: string;
  initial: string;
  onSaved: () => void;
}) {
  const [notes, setNotes] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);

  return (
    <div className="space-y-2">
      <p className="text-xs text-ink-muted">{P.notesHint}</p>
      <textarea
        value={notes}
        onChange={(e) => {
          setNotes(e.target.value);
          setSaved(false);
        }}
        rows={3}
        className="w-full surface-inset px-3 py-2 text-sm outline-none focus:border-primary"
      />
      <div className="flex items-center justify-end gap-2">
        {saved && <span className="text-xs text-success">{P.notesSaved}</span>}
        <Button
          disabled={busy || notes === initial}
          onClick={async () => {
            setBusy(true);
            try {
              await api(`/api/v1/admin/users/${userID}`, {
                method: "PATCH",
                body: JSON.stringify({ admin_notes: notes }),
              });
              setSaved(true);
              onSaved();
            } finally {
              setBusy(false);
            }
          }}
        >
          {m.common.save}
        </Button>
      </div>
    </div>
  );
}


function FinBucket({
  title,
  total,
  count,
  limit,
  items,
  positive,
  onOrder,
}: {
  title: string;
  total: number;
  /** **عددُ الصفوف كلِّها** — والمعروضُ أحدثُها. */
  count: number;
  limit: number;
  items: FinEntry[];
  positive: boolean;
  onOrder: (ref: string) => void;
}) {
  const tone = positive ? "text-success" : "text-danger";
  return (
    <FormSection title={title} icon={<IconWallet />}>
      {/* **و`dir` على الرقم لا على الفقرة** — وإلّا انزاحت الكتلةُ إلى اليسار
          وبقي عنوانُها يميناً، **فيُقرأ كلٌّ في اتجاه.** (شهده المالك ٢٠٢٦-٠٨-٠٩.) */}
      <p className={`figure mb-3 ${tone}`}>
        <span dir="ltr" className="inline-block">
          {positive ? "+" : ""}
          {fmtNum(total)}
        </span>{" "}
        <span className="text-sm font-normal text-ink-muted">{m.common.currency}</span>
      </p>
      {items.length === 0 ? (
        <p className="py-2 text-center text-sm text-ink-muted">{P.fin.empty}</p>
      ) : (
        <ul className="space-y-1.5">
          {items.map((e, i) => (
            <li
              key={e.ref || i}
              className="flex items-center justify-between gap-3 rounded-control border border-line px-3 py-2 text-sm"
            >
              <div className="min-w-0 flex-1">
                {e.ref ? (
                  <button
                    type="button"
                    onClick={() => onOrder(e.ref)}
                    className="truncate text-start font-medium text-primary hover:underline"
                  >
                    {e.label}
                  </button>
                ) : (
                  <span className="font-medium">{e.label}</span>
                )}
              </div>
              <span className={`shrink-0 font-bold ${tone}`} dir="ltr">
                {fmtNum(e.amount)}
              </span>
            </li>
          ))}
        </ul>
      )}
      {/* ══════════════════════════════════════════════════════════════
          **والحدُّ يُقال — والمجموعُ فوقه على الكلّ**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-١٠: «لا تنسَ إضافة الباجينيشن».)

          **والقائمةُ هنا عيّنةٌ لا جرد**: المجموعُ المعروضُ فوقها يُحسب في
          الخادم على كلّ الصفوف باستعلامٍ مستقلّ، **والأسطرُ أحدثُ خمسين.**

          **وسقفٌ صامتٌ يُقرأ «هذا كلُّ ما عليه»** — فيُصالَح المتجرُ على نصف
          دينه وهو يظنّ أنّه رأى الكشفَ كلَّه. **فيُقال العددُ صراحة.**

          **والرقمُ من الخادم لا مكتوبٌ هنا** — ورقمان لمعنًى واحدٍ يفترقان. */}
      {count > items.length && (
        <p className="mt-2 text-center text-xs text-ink-muted">
          {P.fin.showingLatest
            .replace("{n}", fmtNum(Math.min(limit, items.length)))
            .replace("{all}", fmtNum(count))}
        </p>
      )}
    </FormSection>
  );
}
