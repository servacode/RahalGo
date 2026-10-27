"use client";

/** صفحة تفاصيل الحساب: الملف، مؤشرات حسب أدواره، سجل المحفظة الكامل،
 *  وأدوات الأدمن (عمليات المحفظة، إعادة تعيين كلمة المرور). */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDate, fmtDateTime, errorText} from "@rahalgo/i18n";
import { roleLabelByCode } from "@/lib/rolemeta";
import {
  Tabs,
  Alert,
  Badge,
  Button,
  Select,
  Checkbox,
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
  IconBlock,
  IconUnblock,
  IconBalance,
  IconLocation,
  IconChat,
  LoadingState,
  ReloadState,
  Money,
  Confirm,
  ActionMenu,
  IconChevronDown,
  type ActionMenuItem,
} from "@rahalgo/ui";
import { api, type AuthUser } from "@/lib/api";
import { WarningsSection } from "@/components/admin/accounts/WarningsSection";
import {
  OrdersTab,
  ChatsTab,
  AddressesTab,
  CashboxTab,
  StoresTab,
  EndShiftButton,
} from "@/components/admin/ProfileRoleTabs";
import { useCanCall } from "@/lib/policy";
import {
  NotAllowed,
  NotesLog,
  CashBanCard,
  DriverPanel,
  TempPasswordBadge,
  PendingPhoneBanner,
  ChangePhoneModal,
  ResetPasswordModal,
  type PendingPhoneChange,
} from "@/components/admin/accounts/ProfileParts";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/admin/WalletModal";
import { ManageRolesModal } from "@/components/admin/ManageRolesModal";
import StatusReasonModal from "@/components/admin/StatusReasonModal";
import { MediaThumb } from "@/components/admin/ImageUpload";
import RoleBadge from "@/components/admin/RoleBadge";
import { Tile, type TileTone } from "@/components/admin/DashParts";
import { isDeletedAccount } from "@/components/admin/accounts/all";

type IconT = React.ComponentType<{ size?: number; className?: string }>;

const m = getMessages(defaultLocale);
const P = m.admin.users.profile;
// **وأنواعُ الحركة من المعجم المشترك أوّلاً** — كان `driver_earning` يظهر إنكليزيّاً ٦٨ مرّةً
// في محفظة سائق (قرارُ المالك ٢٠٢٦-١٠-٠٤).
const KINDS: Record<string, string> = { ...m.admin.users.txKinds, ...m.shared.txKinds };
const A = m.admin.acc;
const U = A.ux;
/** **والرصيدان اسمُهما واحدٌ في المنصّة** — معجمُ الكشف المشترك. */
const ST = m.shared.statement;
/**
 * **أسبابُ الشكوى — من المعجمين معاً.**
 *
 * **الزبونُ يشتكي بأسبابه والسائقُ يبلّغ بأسبابه** — **ومعجمٌ واحدٌ يترك
 * نصفَ الشكاوى بلا اسم**: تُعرض بعنوانها المحفوظ («بلاغُ سائق») ولا يُعرف
 * أعلى المتجر هي أم على الزبون.
 */
const REASONS: Record<string, string> = {
  ...m.site.complaint.reasons,
  ...m.driver.history.reportReasons,
};
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
  /** **`percent_exact` بمنزلتين** (١٫٥) — و`percent` مقرَّبة (٢٠٢٦-١٠-١٨). */
  rates: { label: string; percent: number; percent_exact?: number }[];
  owed_to: { total: number; count: number; items: FinEntry[] };
  owed_by: { total: number; count: number; items: FinEntry[] };
  returns: FinEntry[];
  /** **عددُ المرتجعات كلِّها وسقفُ العرض** — (٢٠٢٦-٠٨-١٠). */
  returns_count: number;
  /** **أربعةُ سجلّاتٍ كانت خارجَ الملفّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٥). */
  incentives: FinEntry[];
  payouts: FinEntry[];
  disputes: FinEntry[];
  emergencies: FinEntry[];
  incentives_count: number;
  payouts_count: number;
  disputes_count: number;
  emergencies_count: number;
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
  // ── قسمُ الحسابات (قراراتُ المالك ٢٠٢٦-١٠-٠٤) ──
  money_hidden: boolean;
  open_orders: number;
  temp_password_pending: boolean;
  temp_password_expires_at: string | null;
  temp_password_hours: number;
  ever_logged_in: boolean;
  vehicle_type: string;
  vehicle_plate: string;
  vehicle_color: string;
  cash_limit: number;
  cash_limit_general: number;
  cash_limit_override: number | null;
  accident_locked: boolean;
  accident_lock_at: string | null;
  accident_cleared_at: string | null;
  accident_cleared_by: string | null;
  held_commission: number;
  held_commissions_count: number;
  pending_phone_change: PendingPhoneChange | null;
  stores: { id: string; name: string; status: string }[];
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

/** **صفُّ شكوى** — ومعه من فتحها ولماذا، **والعنوانُ وحدَه لا يقولهما.** */
interface TicketRow {
  id: string;
  number: number;
  subject: string;
  status: string;
  compensation: number;
  reason: string;
  by_name: string;
  created_at: string;
}

interface Feedback {
  tickets: TicketRow[];
  /** **الشكاوى عليه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥). */
  tickets_against: TicketRow[];
  tickets_against_count: number;
  ratings_given: { order_number: number; merchant_name: string; platform_stars: number; driver_stars: number | null; created_at: string }[];
  /** **أعدادُ الكلّ** — والمعروضُ صفحةٌ منه. (٢٠٢٦-٠٨-١٠.) */
  tickets_count: number;
  ratings_given_count: number;
  ratings_received_count: number;
  per_page: number;
  ratings_received: { order_number: number; merchant_name: string; stars: number; created_at: string; as: string }[];
  avg_received: number | null;
  /** **ما قاله السائقون عن متجره** — (قرارُ المالك ٢٠٢٦-٠٨-١٦). */
  by_drivers: { order_number: number; merchant_name: string; speed_stars: number; conduct_stars: number; created_at: string; driver: string }[];
  by_drivers_count: number;
  avg_speed: number | null;
  avg_conduct: number | null;
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


export default function UserProfilePage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { user: me, can } = useAuth();
  // ══════════════════════════════════════════════════════════════════
  // **وكلُّ زرٍّ بقدرة بابه من جدول المحرّك** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
  // ══════════════════════════════════════════════════════════════════
  //
  // **كانت أزرارُ الإيقاف والحظر والرقم والكلمة والجلسات تفحص «إدارة الأدوار»**
  // والمحرّكُ يطلب «إدارة حالة الحساب» — فدورُ «الثقة والسلامة» لا يرى زرَّه، ومن
  // معه «إدارة الأدوار» وحدَها يراه ويُردّ.
  const canCall = useCanCall();
  const canStatus = canCall("PATCH", "/users/{id}");
  const canRoles = canCall("POST", "/users/{id}/roles");
  const canReset = canCall("POST", "/users/{id}/password");
  const canLogout = canCall("POST", "/users/{id}/logout-all");
  const canResend = canCall("POST", "/users/{id}/resend-welcome");
  const canWalletRead = canCall("GET", "/users/{id}/wallet");
  const canWalletPropose = canCall("POST", "/users/{id}/wallet");

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
  /** **وللشكاوى عليه صفحتُها** — قائمتان لا قائمة. */
  const [aPage, setAPage] = useState(1);
  const [gPage, setGPage] = useState(1);
  const [rPage, setRPage] = useState(1);
  /** **ولتقييمات السائقين لمتجره صفحتُها** — كانت تتبع `rPage` بلا أزرار. */
  const [dPage, setDPage] = useState(1);
  const [actCount, setActCount] = useState(0);
  const [actPer, setActPer] = useState(25);
  /** **وأفعالُه المتاحةُ للترشيح** — من المحرّك، **وما ليس عنده لا يُعرض عليه.** */
  const [actKinds, setActKinds] = useState<{ action: string; count: number }[]>([]);
  const [actFilter, setActFilter] = useState("");
  /**
   * **وتجديدُ الجلسة مخفيٌّ افتراضاً** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).
   *
   * **تكتبه الساعةُ لا الإنسان**: هاتفٌ تُرك مفتوحاً يكتب سطراً كلَّ دورة
   * تجديد، **فيدفع الأفعالَ الحقيقيّةَ خارجَ الصفحة الأولى.**
   */
  const [actRefresh, setActRefresh] = useState(false);
  // ══════════════════════════════════════════════════════════════════
  // **ودفترُ المحفظة يُقلَّب ولا يُقصّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٥)
  // ══════════════════════════════════════════════════════════════════
  //
  // **كان يُطلب بلا حدٍّ فيقصّه المحرّكُ عند خمسين ويقول `truncated`** —
  // **وهذه الشاشةُ لا تقرأ الحقل.** فمن راجع دفترَ زبونٍ اشتكى «رصيدي
  // ناقص» رأى خمسين حركةً **وظنّها كلَّ ما وقع.**
  const [wPage, setWPage] = useState(1);
  const [txTotal, setTxTotal] = useState(0);
  const [txPer, setTxPer] = useState(50);
  // **والرصيدان يُقالان** — دفترٌ بلا افتتاحيٍّ وختاميّ لا يُراجَع: **من
  // يجمع الأسطر لا تساوي رصيدَه فيظنّ الخللَ في المنصّة.**
  const [txOpening, setTxOpening] = useState(0);
  const [txClosing, setTxClosing] = useState(0);
  /** **ويُعاد جلبُه بعد حركةٍ يدويّة** — وإلّا شُحن الرصيدُ ولم يظهر قيدُه. */
  const [wReload, setWReload] = useState(0);
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
    | "chats"
    | "activity"
  >("overview");
  const [notice, setNotice] = useState("");
  const [feedbackError, setFeedbackError] = useState("");
  const [walletError, setWalletError] = useState("");
  const [finError, setFinError] = useState("");
  const [confirmLogout, setConfirmLogout] = useState(false);
  const [actionError, setActionError] = useState("");
  const [confirmActivate, setConfirmActivate] = useState(false);
  const [busyAction, setBusyAction] = useState(false);
  const [walletOpen, setWalletOpen] = useState(false);
  const [rolesOpen, setRolesOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [error, setError] = useState("");

  // **والخطأُ يُرمى لنافذة السبب فتعرضه** — كانت تبقى مفتوحةً بلا تفسير.
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
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [id]);

  // **والشكاوى وحدَها** — فشلُها لا يُسقط الملفَّ كلَّه (قرارُ المالك ٢٠٢٦-١٠-٠٤).
  const loadFeedback = useCallback(async () => {
    try {
      setFeedback(
        await api<Feedback>(
          `/api/v1/admin/users/${id}/feedback?t_page=${tPage}&a_page=${aPage}&g_page=${gPage}&r_page=${rPage}&d_page=${dPage}`,
        ),
      );
      setFeedbackError("");
    } catch (err) {
      setFeedbackError(errorText(err));
    }
  }, [id, tPage, aPage, gPage, rPage, dPage]);

  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    void loadFeedback();
  }, [loadFeedback]);

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
  // **والسجلُّ لمن يملك قراءته** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣) — `users.sensitive.read`: كان يُنادى لكلّ
  // أحدٍ فيُردّ ٤٠٣ ويُبتلع فيُقرأ «لا نشاط»، **والجوابُ الصحيحُ «غيرُ مسموح».**
  const canSensitive = can("users.sensitive.read");
  const canFinance = can("finance.read");
  const canChats = can("orders.communications.read");
  useEffect(() => {
    if (!canSensitive) return;
    api<{
      activity: Activity[];
      total: number;
      per_page: number;
      kinds: { action: string; count: number }[];
    }>(
      `/api/v1/admin/users/${id}/activity?page=${actPage}` +
        `&action=${actFilter}&refresh=${actRefresh}`,
    )
      .then((act) => {
        setActivity(act?.activity ?? []);
        setActCount(act?.total ?? 0);
        setActPer(act?.per_page || 25);
        setActKinds(act?.kinds ?? []);
      })
      // @empty-ok **وسجلٌّ لا يُجلب لا يُسقط الصفحة** — بقيّةُ الحساب تُقرأ.
      .catch(() => setActivity([]));
  }, [id, actPage, actFilter, actRefresh, canSensitive]);

  /**
   * **ودفترُ المحفظة يُجلَب وحدَه — بصفحته.**
   *
   * **ولو بقي في `load` لَما تحرّك بتقليب الصفحة**: تابعُه معلّقٌ بغير
   * `wPage`، **فيُرسم الترقيمُ ويُضغط ولا يقع شيء** — وهي عائلةُ العطب
   * نفسِها التي أوقفت سجلَّ النشاط.
   */
  useEffect(() => {
    // **والدفترُ لمن يملك المال** — `finance.read`.
    if (!canFinance) return;
    api<{
      transactions: Tx[];
      total: number;
      per_page: number;
      opening: number;
      closing: number;
    }>(`/api/v1/admin/users/${id}/wallet?page=${wPage}`)
      .then((st) => {
        setTxs(st?.transactions ?? []);
        setTxTotal(st?.total ?? 0);
        setTxPer(st?.per_page || 50);
        setTxOpening(st?.opening ?? 0);
        setTxClosing(st?.closing ?? 0);
        setWalletError("");
      })
      // **والفشلُ يُقال** — كان يُقرأ «لا حركات».
      .catch((err) => {
        setTxs([]);
        setWalletError(errorText(err));
      });
  }, [id, wPage, wReload, canFinance]);

  /**
   * **والكشفُ الماليُّ يُجلَب حين يُفتح تبويبُه.**
   *
   * **وكان يُنادى مع كلّ فتحةِ ملفّ** — **ولزبونٍ يردّ أصفاراً حُسبت
   * بأربعة نداءاتٍ للقاعدة**، وتبويبُه محجوبٌ عنه أصلاً فلا يراها أحد.
   */
  useEffect(() => {
    if (tab !== "financials" || !canFinance) return;
    api<FinData>(`/api/v1/admin/users/${id}/financials`)
      .then((f) => {
        setFin(f);
        setFinError("");
      })
      // **والفشلُ يُقال** — كان صفحةً فارغةً بلا مؤشّر.
      .catch((err) => {
        setFin(null);
        setFinError(errorText(err));
      });
  }, [id, tab, canFinance]);

  if (error)
    return (
      <section className="surface">
        <ReloadState onRetry={() => void load()} label={`${U.profileLoadError} — ${error}`} />
      </section>
    );
  if (!p) return <LoadingState variant="text" />;

  const has = (r: string) => p.roles.includes(r);

  const isMe = me?.id === p.id;
  const money = !p.money_hidden;
  /** **المحذوفُ بلا أيّ فعل** (تدقيقُ ٢٠٢٦-١٠-٢٧) — والمحرّكُ يردّ تبديلَ حاله. */
  const gone = isDeletedAccount(p);
  const showOrdersTab = has("customer") || has("driver");
  const toTab = (t: typeof tab, allowed = true) => (allowed ? () => setTab(t) : undefined);
  // ══════════════════════════════════════════════════════════════════
  // **الأرقامُ بطاقاتٌ تُضغط بألوانٍ لها معنى** (التصميمُ الاحترافيّ ٢٠٢٦-١٠-٢٧)
  // ══════════════════════════════════════════════════════════════════
  //
  // الرصيدُ أخضرُ «إله» أو أحمرُ «عليه»، والمحجوزُ برتقاليّ — **وكلُّ بطاقةٍ تفتح تبويبَها
  // داخلَ الملفّ** (لا تُخرجه إلى شاشةٍ أخرى — قرارُ المالك ٢٠٢٦-٠٨-١٥).
  const stats: { key: string; icon: IconT; label: string; value: string; sub?: string; tone?: TileTone; onClick?: () => void }[] = [];
  if (money) {
    stats.push({
      key: "balance",
      icon: IconWallet,
      label: `${m.admin.customers.balance} (${m.common.currency})`,
      value: fmtNum(p.balance),
      sub: p.balance < 0 ? U.balOwes : p.balance > 0 ? U.balOwed : undefined,
      tone: p.balance < 0 ? "danger" : p.balance > 0 ? "success" : "muted",
      onClick: toTab("wallet", canFinance),
    });
  }
  // **وكم دعا وكم قبض** (قرارُ المالك ٢٠٢٦-٠٨-١٥) — ويُعرضان ولو كانا صفراً: «لا أحد» جوابٌ لا فراغ.
  if (has("customer")) {
    stats.push({
      key: "refs",
      icon: IconLink,
      label: P.referralsCount,
      value: fmtNum(p.referrals_count),
      tone: p.referrals_count > 0 ? "accent" : "muted",
    });
    if (money) {
      stats.push({
        key: "refsEarned",
        icon: IconWallet,
        label: `${P.referralsEarned} (${m.common.currency})`,
        value: fmtNum(p.referrals_earned),
        tone: p.referrals_earned > 0 ? "success" : "muted",
        onClick: toTab("wallet", canFinance),
      });
    }
  }
  if (has("customer") || p.orders_count > 0) {
    stats.push({
      key: "orders",
      icon: IconOrder,
      label: P.ordersCount,
      value: fmtNum(p.orders_count),
      tone: p.orders_count > 0 ? "default" : "muted",
      onClick: toTab("orders", showOrdersTab),
    });
    if (money) {
      stats.push({
        key: "spent",
        icon: IconOrder,
        label: `${P.spent} (${m.common.currency})`,
        value: fmtNum(p.orders_spent),
        tone: p.orders_spent > 0 ? "default" : "muted",
        onClick: toTab("orders", showOrdersTab),
      });
    }
  }
  if (has("sales")) {
    stats.push({
      key: "repStores",
      icon: IconStore,
      label: P.repStores,
      value: fmtNum(p.rep_stores),
      tone: p.rep_stores > 0 ? "accent" : "muted",
      onClick: toTab("stores"),
    });
    if (money) {
      stats.push({
        key: "commissions",
        icon: IconWallet,
        label: `${P.commissions} (${m.common.currency})`,
        value: fmtNum(p.commissions),
        tone: p.commissions > 0 ? "success" : "muted",
        onClick: toTab("financials", canFinance),
      });
    }
    if (money && p.held_commission > 0) {
      stats.push({
        key: "held",
        icon: IconWallet,
        label: `${A.heldCommission} (${m.common.currency})`,
        value: fmtNum(p.held_commission),
        tone: "warning",
        onClick: toTab("financials", canFinance),
      });
    }
  }
  if (feedback?.avg_received != null) {
    stats.push({
      key: "rating",
      icon: IconStar,
      label: P.avgRating,
      value: fmtNum(Math.round(feedback.avg_received * 10) / 10),
      tone: feedback.avg_received < 3 ? "danger" : feedback.avg_received < 4 ? "warning" : "success",
      onClick: toTab("feedback"),
    });
  }
  if (has("driver")) {
    // **وسُلّم اليومَ غيرُ سُلّم كلَّه** — اليومُ يومُ المنصّة من ساعة الفتح (يحسبه المحرّك).
    stats.push(
      {
        key: "today",
        icon: IconOrder,
        label: m.admin.drivers.deliveredToday,
        value: fmtNum(p.delivered_today),
        tone: p.delivered_today > 0 ? "success" : "muted",
        onClick: toTab("orders", showOrdersTab),
      },
      {
        key: "deliveries",
        icon: IconDriver,
        label: P.deliveries,
        value: fmtNum(p.deliveries),
        tone: p.deliveries > 0 ? "default" : "muted",
        onClick: toTab("orders", showOrdersTab),
      },
    );
  }

  // ══════════════════════════════════════════════════════════════════
  // **إجراءاتُ الحساب — قائمةٌ واحدة، والخطرُ بالأحمر في آخرها**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-١٠-٠٤.) **ولا يعيد الموظّفُ كلمتَه ولا يُخرج نفسَه ولا يوقفها
  // من ملفّه** — بابُه «حسابي».
  const actions: ActionMenuItem[] = [];
  if (canRoles && !gone) actions.push({ key: "roles", label: m.admin.users.manageRoles, icon: IconRoles, onSelect: () => setRolesOpen(true) });
  if (canReset && !isMe && !gone) actions.push({ key: "reset", label: P.passwordBtn, icon: IconLock, onSelect: () => setResetOpen(true) });
  if (canStatus && !isMe && !gone) actions.push({ key: "phone", label: P.changePhone, icon: IconPhone, onSelect: () => setPhoneOpen(true) });
  if (canLogout && !isMe && !gone)
    actions.push({
      key: "logout",
      label: `${P.logoutAllShort} (${fmtNum(p.active_sessions)})`,
      icon: IconLogout,
      danger: true,
      onSelect: () => setConfirmLogout(true),
    });
  if (canStatus && !isMe && !gone) {
    if (p.status === "active") {
      actions.push(
        { key: "suspend", label: m.admin.users.suspend, icon: IconBlock, danger: true, hint: A.statusExplain.suspended, onSelect: () => setStatusModal("suspended") },
        { key: "block", label: m.admin.users.block, icon: IconBlock, danger: true, hint: A.statusExplain.blocked, onSelect: () => setStatusModal("blocked") },
      );
    } else {
      actions.push({ key: "activate", label: m.admin.users.activate, icon: IconUnblock, onSelect: () => setConfirmActivate(true) });
    }
  }

  const statusBadge =
    gone ? (
      <Badge variant="neutral">{U.deletedBadge}</Badge>
    ) : p.status === "active" ? (
      <Badge variant="success">{m.admin.users.active}</Badge>
    ) : p.status === "suspended" ? (
      <Badge variant="warning">{m.admin.users.suspended}</Badge>
    ) : (
      <Badge variant="danger">{m.admin.users.blocked}</Badge>
    );

  return (
    <div>
      <button
        type="button"
        onClick={() => router.push("/dashboard/users")}
        className="mb-3 flex items-center gap-1 text-sm text-ink-muted hover:text-primary"
      >
        <IconPrev size={15} />
        {U.back}
      </button>

      {/* ══════════════════════════════════════════════════════════════
          **رأسُ الملفّ على نمط رأس الصفحة الموحّد** (التصميمُ الاحترافيّ ٢٠٢٦-١٠-٢٧)
          ══════════════════════════════════════════════════════════════
          صورةٌ دائريّةٌ كبيرة، والاسم، وسطرٌ للرقم والتسجيل وآخر ظهور، وشاراتُ الدور والحالة بسطرٍ
          لحالها، وزرّا «المحفظة» و«إجراءات» على الطرف — **يلتفّان على الهاتف ولا يزيحان الصفحة**
          (شهده المالك ٢٠٢٦-٠٨-١١). */}
      <section className="surface mb-3 p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <MediaThumb url={p.avatar_thumb_url} alt="" fallback={p.full_name || m.terms.avatarFallback} size={72} round />
            <div className="min-w-0 space-y-1">
              <h1 className="heading-page truncate">{p.full_name || "—"}</h1>
              <p className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-ink-muted">
                <span className="inline-flex items-center gap-1">
                  <IconPhone size={12} />
                  {/* **والرقمُ المحجوبُ يُقال، والمحذوفُ «رقم محذوف»** — لا `deleted-…` إنكليزيّة. */}
                  {gone ? (
                    <span>{U.deletedPhone}</span>
                  ) : p.phone ? (
                    <span dir="ltr">{p.phone}</span>
                  ) : (
                    <span>{A.phoneHidden}</span>
                  )}
                </span>
                <span className="inline-flex items-center gap-1">
                  <IconDate size={12} />
                  {U.joined}: {fmtDate(p.created_at)}
                </span>
                {/* **وآخرُ ظهوره هنا** (قرارُ المالك ٢٠٢٦-٠٨-١٥). */}
                <span>
                  {m.admin.users.lastSeen}: {p.last_seen_at ? fmtDateTime(p.last_seen_at) : m.admin.users.neverSeen}
                </span>
              </p>
              <div className="flex flex-wrap items-center gap-1">
                {p.roles.map((r) => (
                  <RoleBadge key={r} role={r} />
                ))}
                {statusBadge}
                {p.invite_code && (
                  <Badge variant="accent" dir="ltr" className="px-1.5 font-mono">
                    {p.invite_code}
                  </Badge>
                )}
                {/* **«لم يدخل بعد · تنتهي بعد …»** و«إعادة إرسال» (قرارُ المالك ٢٠٢٦-١٠-٠٤). */}
                {!gone && p.temp_password_pending && p.temp_password_expires_at && (
                  <TempPasswordBadge
                    userID={p.id}
                    expiresAt={p.temp_password_expires_at}
                    canResend={canResend && !isMe}
                    onDone={() => void load()}
                  />
                )}
              </div>
            </div>
          </div>
          {!gone && ((canWalletRead || canWalletPropose) || actions.length > 0) && (
            <div className="flex flex-wrap items-center gap-2 whitespace-nowrap">
              {(canWalletRead || canWalletPropose) && (
                <Button onClick={() => setWalletOpen(true)} className="flex items-center gap-1.5">
                  <IconWallet size={16} />
                  {m.admin.users.wallet}
                </Button>
              )}
              {actions.length > 0 && <ActionMenu label={A.actions} items={actions} />}
            </div>
          )}
        </div>
      </section>

      {/* **وسببُ الإيقاف في صندوق تنبيهٍ تحت الرأس** — لا كلمةً حمراءَ صغيرةً بين الشارات. */}
      {gone ? (
        <Alert tone="info" className="mb-3">
          {U.profileDeleted}
        </Alert>
      ) : (
        p.status !== "active" && (
          <Alert tone={p.status === "suspended" ? "warning" : "error"} className="mb-3">
            <span className="block font-bold">{p.status === "suspended" ? U.profileSuspended : U.profileBlocked}</span>
            <span className="block text-sm">
              {U.reasonLabel}: {p.status_reason || U.reasonNone}
            </span>
          </Alert>
        )
      )}

      {actionError && <Alert className="mb-3">{actionError}</Alert>}
      {p.pending_phone_change && (
        <PendingPhoneBanner req={p.pending_phone_change} onDone={() => void load()} />
      )}

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
              show: showOrdersTab,
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
              // ══════════════════════════════════════════════════════════
              // **والمتجرُ عميلٌ عند المندوب لا مِلك**
              // ══════════════════════════════════════════════════════════
              //
              // (قرارُ المالك ٢٠٢٦-٠٨-١٦: «المتاجرُ لصاحب المتجر وليس
              //  للمندوب، ولكنّ المتجرَ عميلٌ لدى المندوب».)
              //
              // **و«متاجره» في ملفّ المندوب تقول ما ليس صحيحاً**: يملكها
              // أصحابُها، **وهو جلبهم وحدَه.**
              //
              // **ولا يجتمع الدوران** (قرارُ المالك ٢٠٢٦-٠٨-١٦: «ممنوعٌ
              // منعاً باتاً») — **فالاسمُ واحدٌ لا اثنان.**
              label: has("sales") ? P.roleTabs.clients : P.roleTabs.stores,
              icon: IconStore,
              show: has("merchant") || has("sales"),
            },
            // **والتبويبُ لمن يملك قراءته** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان يظهر لكلّ أحدٍ فيقول «غير مسموح».
            { key: "wallet", label: P.tabs.wallet, icon: IconWallet, show: canFinance },
            {
              key: "financials",
              // ══════════════════════════════════════════════════════════
              // **ولا ماليّةَ للزبون** — (قرارُ المالك ٢٠٢٦-٠٨-١٥)
              // ══════════════════════════════════════════════════════════
              //
              // **الكشفُ يُبنى بشروطٍ على الدور**: النِسَبُ لمندوبٍ ومتجر،
              // و«مستحقٌّ له» لمندوبٍ وسائق، و«مستحقٌّ عليه» لمتجرٍ وسائق.
              // **ولا فرعَ يضيف للزبون شيئاً** — فيرى بطاقتَي صفرٍ لن
              // تصيرا شيئاً أبداً.
              //
              // **وبطاقةُ صفرٍ ليست خبراً «لا شيءَ عليه»** — هي سؤالٌ
              // يُطرح كلَّ مرّة: أحقٌّ هذا أم عطبٌ في القراءة؟
              //
              // **وما كان ينفرد به منقولٌ**: سببُ الإلغاء صار في سطر
              // الطلب بتبويب «طلباته». **ومالُه محسوبٌ في مواضعه**: ما
              // أنفق في النظرة العامّة، وما استُرجع في سجلّ المحفظة.
              //
              // **ومن حمل دورين يبقى له** — سائقٌ يطلب لنفسه يرى كشفَه.
              label: P.tabs.financials,
              icon: IconBalance,
              show: canFinance && (has("driver") || has("merchant") || has("sales")),
            },
            { key: "feedback", label: P.tabs.feedback, icon: IconStar, show: true },
            {
              key: "chats",
              // **وأحاديثُه في ملفّه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥).
              //
              // **ومن يحكم في نزاعٍ يعرف اسمَ الإنسان لا رقمَ الطلب** —
              // وكان الحديثُ يُقرأ من الطلب وحدَه، **فيمشي طلباته واحداً
              // واحداً يفتح كلَّ حديثٍ يبحث عن سطر.**
              //
              // **والقناةُ بين الزبون والسائق** — فمن ليس أحدَهما لا حديثَ له.
              label: P.tabs.chats,
              icon: IconChat,
              show: canChats && (has("customer") || has("driver")),
            },
            { key: "activity", label: P.tabs.activity, icon: IconStatus, show: canSensitive },
          ] as const)
          .filter((t) => t.show)
          .map(({ key, label, icon }) => ({ key, label, icon }))}
        value={tab}
        onChange={(k) => setTab(k as typeof tab)}
      />

      {tab === "orders" && (canCall("GET", "/orders") ? <OrdersTab userID={id} roles={p.roles} /> : <NotAllowed />)}
      {tab === "addresses" && (canSensitive ? <AddressesTab userID={id} /> : <NotAllowed />)}
      {tab === "cashbox" && <CashboxTab userID={id} onSettled={() => void load()} />}
      {tab === "stores" && (canCall("GET", "/merchants") ? <StoresTab userID={id} roles={p.roles} /> : <NotAllowed />)}
      {tab === "wallet" && !canFinance && <NotAllowed />}
      {tab === "financials" && !canFinance && <NotAllowed />}
      {tab === "chats" && !canChats && <NotAllowed />}
      {tab === "activity" && !canSensitive && <NotAllowed />}

      {tab === "overview" && (
      <>
      {p.money_hidden && <p className="mb-2 text-xs text-ink-muted">{A.moneyHidden}</p>}
      {has("driver") && (
        <DriverPanel
          d={p}
          onChanged={() => void load()}
          endShift={p.on_shift ? <EndShiftButton userID={p.id} onDone={() => void load()} /> : null}
        />
      )}
      {has("customer") && <CashBanCard userID={p.id} />}
      {has("sales") && p.held_commissions_count > 0 && (
        <Alert tone="warning" className="mb-3">
          {A.heldCommissionHint}
        </Alert>
      )}
      {/* **الأرقامُ بطاقاتٌ تُضغط** — كلٌّ تفتح تبويبَها. */}
      {stats.length > 0 && (
        <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4">
          {stats.map((s) => (
            <Tile key={s.key} icon={s.icon} label={s.label} value={s.value} sub={s.sub} tone={s.tone} onClick={s.onClick} />
          ))}
        </div>
      )}

      {p.stores.length > 0 && (
        <div className="mb-3 surface p-3">
          <p className="mb-2 flex items-center gap-1.5 text-sm font-bold">
            <IconStore size={15} className="text-primary" />
            {P.merchantsOwned}
          </p>
          {/* **ومتجرُه يُفتح** — ملفُّ المتجر بابُه. */}
          <div className="flex flex-wrap gap-2">
            {p.stores.map((st) => (
              <button
                key={st.id}
                type="button"
                onClick={() => router.push(`/dashboard/merchants/${st.id}`)}
                className="rounded-badge"
              >
                <Badge variant={st.status === "active" ? "primary" : st.status === "suspended" ? "danger" : "neutral"}>
                  {st.name}
                </Badge>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* **والإنذاراتُ في ملفّه** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «اجعله لكلّ
          الأدوار حتّى الزبون»).

          **ومن يقرّر أن يُنذر يسأل أوّلاً: كم مرّةً سبق؟** — والجوابُ هنا، مع
          رصيده وطلباته وتقييماته. */}
      <WarningsSection userID={p.id} />

      <NotesLog userID={p.id} />
      </>
      )}

      {tab === "wallet" && canFinance && (
      <FormSection
        title={`${P.statement} (${fmtNum(txTotal)})`}
        icon={<IconWallet />}
      >
        {walletError && <Alert className="mb-2">{walletError}</Alert>}
        {/* ══════════════════════════════════════════════════════════════
            **والرصيدان يُقفلان الصفحة — وفوقها** (قرارُ المالك ٢٠٢٦-٠٨-١٥)
            ══════════════════════════════════════════════════════════════
            الافتتاحيُّ + مجموعُ المعروض = الختاميّ، **والصفحةُ الثانيةُ تُقفل برصيدها هي.**
            ومعهما ما دخل وما طلع في هذه الصفحة. */}
        {txs.length > 0 && (
          <div className="mb-3 grid grid-cols-2 gap-2 lg:grid-cols-4">
            <div className="rounded-card border border-line-soft p-3">
              <span className="block text-2xs text-ink-muted">{ST.opening}</span>
              <span className="figure block" dir="ltr">{fmtNum(txOpening)}</span>
            </div>
            <div className="rounded-card border border-line-soft p-3">
              <span className="block text-2xs text-ink-muted">{U.pageIn}</span>
              <span className="figure block text-success" dir="ltr">
                +{fmtNum(txs.reduce((a, t) => a + (t.amount > 0 ? t.amount : 0), 0))}
              </span>
            </div>
            <div className="rounded-card border border-line-soft p-3">
              <span className="block text-2xs text-ink-muted">{U.pageOut}</span>
              <span className="figure block text-danger" dir="ltr">
                {fmtNum(txs.reduce((a, t) => a + (t.amount < 0 ? t.amount : 0), 0))}
              </span>
            </div>
            <div className="rounded-card border border-line-soft p-3">
              <span className="block text-2xs text-ink-muted">{ST.closing}</span>
              <span className={`figure block ${txClosing < 0 ? "text-danger" : ""}`} dir="ltr">{fmtNum(txClosing)}</span>
            </div>
          </div>
        )}
        {txs.length === 0 ? (
          !walletError && (
            <div className="flex flex-col items-center gap-2 py-6 text-center">
              <span className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-field text-ink-muted">
                <IconWallet size={22} />
              </span>
              <p className="text-sm text-ink-muted">{P.statementEmpty}</p>
            </div>
          )
        ) : (
          <ul className="grid grid-cols-1 gap-2 lg:grid-cols-2">
            {txs.map((t) => (
              <li key={t.id} className="flex flex-col gap-2 rounded-card border border-line-soft p-3 text-sm">
                <div className="flex items-start justify-between gap-3">
                  <Badge variant={t.amount > 0 ? "success" : "danger"}>{KINDS[t.kind] ?? t.kind}</Badge>
                  <span className={`figure ${t.amount > 0 ? "text-success" : "text-danger"}`} dir="ltr">
                    {t.amount > 0 ? "+" : ""}
                    {fmtNum(t.amount)}
                  </span>
                </div>
                {(t.order_number != null || t.ticket_number != null || t.note) && (
                  <span className="flex flex-wrap items-center gap-2">
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
                    {t.note && <span className="text-xs text-ink-muted">{t.note}</span>}
                  </span>
                )}
                <span className="flex flex-wrap items-center justify-between gap-2 text-2xs text-ink-muted">
                  <span>
                    {P.by}: {t.by_name ?? P.system}
                  </span>
                  <span dir="ltr">{fmtDateTime(t.created_at)}</span>
                </span>
              </li>
            ))}
          </ul>
        )}
        {txTotal > txPer && (
          <div className="mt-3 flex justify-center">
            <Pagination page={wPage} total={txTotal} perPage={txPer} onChange={setWPage} />
          </div>
        )}
      </FormSection>
      )}

      {tab === "financials" && canFinance && finError && <Alert>{finError}</Alert>}
      {tab === "financials" && canFinance && !fin && !finError && <LoadingState variant="text" />}
      {tab === "financials" && canFinance && fin && (
        <div className="space-y-4">
          {/* النِسَب المطبّقة */}
          {fin.rates.length > 0 && (
            <FormSection title={P.fin.rates} icon={<IconBalance />}>
              <ul className="space-y-1.5">
                {fin.rates.map((rt, i) => (
                  <li
                    key={i}
                    className="flex items-center justify-between rounded-card border border-line-soft p-3 text-sm"
                  >
                    <span>{rt.label}</span>
                    <span className="font-bold text-primary-dark" dir="ltr">
                      {fmtNum(rt.percent_exact ?? rt.percent)}%
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

          {/* ══════════════════════════════════════════════════════════
              **وأربعةُ سجلّاتٍ كانت خارجَ ملفّه**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٥ بعد فحصٍ طلبه: «نعم ضمن الماليّة».)

              **لكلٍّ منها شاشةٌ مستقلّةٌ في اللوحة** — فمن أراد أن يعرف
              سائقاً **خرج من ملفّه وبحث عن اسمه في أربعة أماكن.**

              **وهذا ما بُني الملفُّ ليمنعه**: «يجمع كلَّ شيءٍ يخصّه —
              ولا نريد خسارةَ أيّ ميزة» (٢٠٢٦-٠٨-٠٣).

              **وتُعرض إن كان فيها شيء** — **وأربعةُ عناوينَ فارغةٍ
              تُقرأ عطباً**، وهي تظهر لمن لا يكون له منها شيءٌ أبداً. */}
          <FinLog
            title={P.fin.incentives}
            rows={fin.incentives}
            count={fin.incentives_count}
            limit={fin.limit}
            icon={<IconStar />}
            money
          />
          <FinLog
            title={P.fin.payouts}
            rows={fin.payouts}
            count={fin.payouts_count}
            limit={fin.limit}
            icon={<IconWallet />}
            money
          />
          <FinLog
            title={P.fin.disputes}
            rows={fin.disputes}
            count={fin.disputes_count}
            limit={fin.limit}
            icon={<IconBalance />}
            money
            labelIsRole
            onOrder={(n) => router.push(`/dashboard/orders?q=${n}`)}
          />
          <FinLog
            title={P.fin.emergencies}
            rows={fin.emergencies}
            count={fin.emergencies_count}
            limit={fin.limit}
            icon={<IconStatus />}
            onOrder={(n) => router.push(`/dashboard/orders?q=${n}`)}
          />

          {/* الطلبات المرتجعة وأسبابها */}
          <FormSection title={P.fin.returns} icon={<IconBlock />}>
            {fin.returns.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.fin.returnsEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {fin.returns.map((e) => (
                  <li
                    key={e.ref}
                    className="flex items-center justify-between gap-3 rounded-card border border-line-soft p-3 text-sm"
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

      {tab === "feedback" && feedbackError && <Alert>{feedbackError}</Alert>}
      {tab === "feedback" && !feedback && !feedbackError && <LoadingState variant="text" />}
      {tab === "feedback" && feedback && (
        <div className="space-y-4">
          {/* ══════════════════════════════════════════════════════════
              **وشكاواه وشكاوى عليه قائمتان لا قائمة**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٥.)

              **كانتا مخلوطتين**: بلاغُ السائق على الزبون يقع جنبَ شكوى
              الزبون، **ولا يفرّقهما إلّا نصُّ العنوان.** **والحكمان
              متناقضان** — «عليه ثلاثُ شكاوى» و«اشتكى ثلاثاً» — **ومن
              يقرأ هذه الشاشةَ هو من يقرّر الإنذارَ أو الحظر.** */}
          <TicketList
            title={P.ticketsSection}
            rows={feedback.tickets}
            total={feedback.tickets_count}
            perPage={feedback.per_page}
            page={tPage}
            onPage={setTPage}
            onOpen={(t) => router.push(`/dashboard/tickets?t=${t.id}`)}
          />
          {/* **وشكاوى عليه تُعرض ولو كانت صفراً** — **وغيابُ القسم يُقرأ
              «لا شكاوى» تخميناً**، ووجودُه فارغاً يقوله يقينا. */}
          <TicketList
            title={P.ticketsAgainst}
            rows={feedback.tickets_against}
            total={feedback.tickets_against_count}
            perPage={feedback.per_page}
            page={aPage}
            onPage={setAPage}
            onOpen={(t) => router.push(`/dashboard/tickets?t=${t.id}`)}
            danger
          />

          <FormSection title={P.ratingsGiven} icon={<IconStar />}>
            {feedback.ratings_given.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ratingsGivenEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.ratings_given.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-card border border-line-soft p-3 text-sm"
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

          {/* ══════════════════════════════════════════════════════════
              **ولا تقييمَ يرد على زبون**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٥: «الزبونُ يقيّم ولا يُقيَّم».)

              **وجدولُ التقييمات فيه عمودان لا ثالثَ لهما**: نجومُ المنصّة
              ونجومُ السائق — **ولا عمودَ يُقيّم الزبون**، ولا شاشةَ في
              المنصّة تفعل.

              **والقسمُ كان يُرسَم فارغاً دائماً** فيُقرأ «لم يقيّمه أحد»
              **لا «لا يُقيَّم أصلاً»** — وهو داءُ «الماليّة» نفسُه. */}
          {/* ══════════════════════════════════════════════════════════
              **وما قاله السائقون عن متجره — يُقرأ لأوّل مرّة**
              ══════════════════════════════════════════════════════════

              (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)

              **جدولُ `merchant_ratings` يُكتب فيه بعد كلّ تسليم** —
              السائقُ يقيّم المتجرَ بنجمتين: سرعةُ التجهيز وحُسنُ
              التعامل. **ولا شاشةَ في المنصّة كلِّها تقرؤه.**

              **والسائقُ يُسأل بعد كلّ تسليم** فيُنفَق وقتُه على رأيٍ لا
              يبلغ أحداً — **وهذا أسوأُ من غياب الميزة**: غيابُها يُعرف،
              **وهذه تبدو موجودةً وهي معطّلة.** */}
          {has("merchant") && feedback.by_drivers_count > 0 && (
            <FormSection
              title={`${P.byDrivers} (${fmtNum(feedback.by_drivers_count)})`}
              icon={<IconStar />}
            >
              <div className="mb-3 flex flex-wrap gap-4 text-sm">
                <span className="text-ink-muted">
                  {P.avgSpeed}:{" "}
                  <span className="font-bold text-accent-dark" dir="ltr">
                    {fmtNum(Math.round((feedback.avg_speed ?? 0) * 10) / 10)}
                  </span>
                </span>
                <span className="text-ink-muted">
                  {P.avgConduct}:{" "}
                  <span className="font-bold text-accent-dark" dir="ltr">
                    {fmtNum(Math.round((feedback.avg_conduct ?? 0) * 10) / 10)}
                  </span>
                </span>
              </div>
              <ul className="space-y-1.5">
                {feedback.by_drivers.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-card border border-line-soft p-3 text-sm"
                  >
                    <span className="flex min-w-0 flex-wrap items-center gap-2">
                      <button
                        type="button"
                        onClick={() => router.push(`/dashboard/orders?q=${rt.order_number}`)}
                        className="font-medium text-primary hover:underline"
                      >
                        #{fmtRef(rt.order_number)}
                      </button>
                      <span className="text-xs text-ink-muted">{rt.merchant_name}</span>
                      <span className="text-xs text-ink-muted">
                        {P.avgSpeed}: {rt.speed_stars} · {P.avgConduct}: {rt.conduct_stars}
                      </span>
                      {/* **وقائلُه باسمه** — **وتقييمٌ بلا قائلٍ لا يُراجَع**،
                          ولا يُعرف أسائقٌ واحدٌ كرّرها أم عشرة. */}
                      {rt.driver && (
                        <span className="text-xs text-ink-muted">{P.by}: {rt.driver}</span>
                      )}
                    </span>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(rt.created_at)}
                    </span>
                  </li>
                ))}
              </ul>
              {feedback.by_drivers_count > feedback.per_page && (
                <div className="mt-3 flex justify-center">
                  <Pagination
                    page={dPage}
                    total={feedback.by_drivers_count}
                    perPage={feedback.per_page}
                    onChange={setDPage}
                  />
                </div>
              )}
            </FormSection>
          )}

          {(has("driver") || has("merchant")) && (
          <FormSection title={P.ratingsRecv} icon={<IconStar />}>
            {feedback.ratings_received.length === 0 ? (
              <p className="py-4 text-center text-sm text-ink-muted">{P.ratingsRecvEmpty}</p>
            ) : (
              <ul className="space-y-1.5">
                {feedback.ratings_received.map((rt, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-card border border-line-soft p-3 text-sm"
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
          )}
        </div>
      )}

      {tab === "chats" && canChats && <ChatsTab userID={id} />}

      {tab === "activity" && canSensitive && (
      <div>
        <FormSection title={`${P.activity} (${fmtNum(actCount)})`} icon={<IconStatus />}>
          {/* ══════════════════════════════════════════════════════════
              **ومُرشِّحٌ بالفعل — وأفعالُه وحدَها في قائمته**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٥.)

              **وهو أطولُ قائمةٍ في الملفّ وأسرعُها نموّاً، ويُفتح للبحث
              عن واقعةٍ بعينها** — وكان يُقلَّب صفحةً صفحة.

              **وقائمةٌ بثمانين فعلاً لحسابٍ فيه ثلاثةٌ تُبحث ولا تُقرأ**
              — فتُبنى ممّا عنده فعلاً، ومعها عددُ كلٍّ. */}
          <div className="mb-3 flex flex-wrap items-center gap-3">
            <Select
              id="act-filter"
              label=""
              value={actFilter}
              onChange={(e) => {
                setActFilter(e.target.value);
                setActPage(1);
              }}
            >
              <option value="">{P.activityAll}</option>
              {actKinds.map((k) => (
                <option key={k.action} value={k.action}>
                  {(ACTIONS[k.action] ?? k.action) + ` (${fmtNum(k.count)})`}
                </option>
              ))}
            </Select>
            {/* **وتجديدُ الجلسة يُطلب ولا يُفرض** — أثرُ أمانٍ بعنوانٍ ووقت،
                **يُخفى ولا يُحذف.** */}
            <Checkbox
              id="act-refresh"
              label={P.activityRefresh}
              checked={actRefresh}
              onChange={(e) => {
                setActRefresh(e.target.checked);
                setActPage(1);
              }}
            />
          </div>
          {activity.length === 0 ? (
            <p className="py-6 text-center text-sm text-ink-muted">{P.activityEmpty}</p>
          ) : (
            <ul className="space-y-1.5">
              {activity.map((a, i) => (
                <ActivityRow key={i} a={a} selfID={id} onOpen={(to) => router.push(to)} />
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
          holds={has("driver") ? { orders: p.open_orders, cash: p.driver_cash } : undefined}
          onSubmit={(reason) => setStatus(statusModal, reason)}
          onClose={() => setStatusModal(null)}
        />
      )}
      {phoneOpen && (
        <ChangePhoneModal
          userID={p.id}
          current={p.phone}
          onClose={() => setPhoneOpen(false)}
          onDone={(msg) => {
            setPhoneOpen(false);
            setNotice(msg);
            void load();
          }}
        />
      )}
      {walletOpen && (
        <WalletModal
          user={{ id: p.id, phone: p.phone, full_name: p.full_name }}
          isAdmin={canWalletPropose}
          onClose={() => {
            setWalletOpen(false);
            // **والدفترُ يُعاد جلبُه** — وإلّا شُحن الرصيدُ ولم يظهر قيدُه.
            setWReload((n) => n + 1);
            void load();
          }}
        />
      )}
      {resetOpen && (
        <ResetPasswordModal
          userID={p.id}
          hours={p.temp_password_hours}
          onClose={() => setResetOpen(false)}
          onDone={() => void load()}
        />
      )}
      {/* **وإخراجٌ من كلّ الأجهزة خلف تأكيد** — كان يقع من أوّل كبسة. */}
      <Confirm
        open={confirmLogout}
        title={A.logoutTitle}
        body={A.logoutBody}
        confirmLabel={A.logoutConfirm}
        busy={busyAction}
        onCancel={() => setConfirmLogout(false)}
        onConfirm={async () => {
          setBusyAction(true);
          try {
            const res = await api<{ revoked_sessions: number }>(`/api/v1/admin/users/${p.id}/logout-all`, {
              method: "POST",
            });
            setNotice(P.logoutAllDone.replace("{n}", fmtNum(res.revoked_sessions)));
            await load();
          } catch (err) {
            setActionError(errorText(err));
          } finally {
            setBusyAction(false);
            setConfirmLogout(false);
          }
        }}
      />
      <Confirm
        open={confirmActivate}
        tone="primary"
        title={A.activateTitle}
        body={A.activateBody}
        confirmLabel={m.admin.users.activate}
        busy={busyAction}
        onCancel={() => setConfirmActivate(false)}
        onConfirm={async () => {
          setBusyAction(true);
          try {
            await setStatus("active", "");
          } catch (err) {
            setActionError(errorText(err));
          } finally {
            setBusyAction(false);
            setConfirmActivate(false);
          }
        }}
      />
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

/**
 * **سطرُ نشاطٍ بجملةٍ بسيطة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — الفعلُ ومن عمله ومتى، **والتفاصيلُ وعنوانُ الجهاز
 * وراء زرّ «التفاصيل»**: كانت `مفتاح: قيمة` وعنوانُ IP تزحم السطرَ على المالك.
 */
function ActivityRow({ a, selfID, onOpen }: { a: Activity; selfID: string; onOpen: (to: string) => void }) {
  const [open, setOpen] = useState(false);
  const pairs = detailPairs(a.details);
  // **ويُربط بما يخصّه — حين تكون له وجهةٌ تُفتح.** وحسابٌ آخرُ وحدَه (`entity='user'`): **ورابطُ
  // الطلب يحتاج رقمَه والمحفوظُ معرّفُه** — وبحثُ الطلبات يطابق الرقمَ فيردّ «لا نتائج».
  const to = a.entity === "user" && a.entity_id && a.entity_id !== selfID ? `/dashboard/users/${a.entity_id}` : null;
  const label = ACTIONS[a.action] ?? a.action;
  const who = a.by_self ? P.bySelf : (a.by_name ?? P.system);
  return (
    <li className="rounded-card border border-line-soft p-3 text-sm">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <span className="min-w-0">
          {to ? (
            <button type="button" onClick={() => onOpen(to)} className="font-bold text-primary hover:underline">
              {label}
            </button>
          ) : (
            <span className="font-bold">{label}</span>
          )}
          <span className="block text-xs text-ink-muted">
            {P.by}: {who}
          </span>
        </span>
        <span className="shrink-0 text-2xs text-ink-muted" dir="ltr">
          {fmtDateTime(a.created_at)}
        </span>
      </div>
      {(pairs.length > 0 || a.ip) && (
        <>
          <button
            type="button"
            onClick={() => setOpen(!open)}
            aria-expanded={open}
            className="mt-2 inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
          >
            <IconChevronDown size={12} className={open ? "rotate-180" : ""} />
            {open ? U.hideDetails : U.details}
          </button>
          {open && (
            <ul className="mt-2 space-y-1 rounded-control bg-field px-3 py-2 text-xs text-ink-muted">
              {pairs.map((d, j) => (
                <li key={j}>
                  {d.k ? `${d.k}: ` : ""}
                  <span className="text-ink">{d.v}</span>
                </li>
              ))}
              {a.ip && (
                <li>
                  {U.device}: <span dir="ltr">{a.ip}</span>
                </li>
              )}
            </ul>
          )}
        </>
      )}
    </li>
  );
}

const DETAIL_KEYS: Record<string, string> = P.detailKeys;
// **والاسمُ من `rolemeta`** — مصدرٌ واحدٌ لكلّ شاشة.
/** **وأسبابُ الإنذار معجمٌ ثانٍ** — يُقرأ منه ما لا تعرفه أسبابُ الشكوى. */
const WARN_REASONS: Record<string, string> = m.admin.warnings.reasons;

/**
 * **تفاصيلُ السطر مقروءةً — لا `{"reason":"abuse_driver"}`.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
 *
 * **كان المحرّكُ يرسلها والشاشةُ تُهملها** — فتقرأ «تعديل إعداد» ولا تعرف
 * أيَّ إعدادٍ ولا من ماذا إلى ماذا، **والجوابُ وصل وسقط.** وتقرأ «توجيه
 * إنذار» ولا تعرف سببَه **وهو محفوظٌ في السطر نفسِه.**
 *
 * **والقيمُ تُترجَم لا مفاتيحُها وحدَها**: `abuse_driver` رمزٌ للآلة —
 * **ومن عرضه كما هو أرى المكتبَ إنكليزيّةً في لوحةٍ عربيّة.**
 */
function detailPairs(raw: string): { k: string; v: string }[] {
  if (!raw) return [];
  let obj: Record<string, unknown>;
  try {
    obj = JSON.parse(raw) as Record<string, unknown>;
  } catch {
    // **ونصٌّ لا يُفكّ يُعرض كما هو** — **وإخفاؤه يخفي ما جاء لأجله.**
    return [{ k: "", v: raw }];
  }
  const out: { k: string; v: string }[] = [];
  for (const [k, val] of Object.entries(obj)) {
    if (val === null || val === "" || (Array.isArray(val) && val.length === 0)) continue;
    let v: string;
    if (Array.isArray(val)) {
      v = val.map((x) => roleLabelByCode(String(x))).join(m.common.listSep);
    } else if (k === "reason") {
      v = REASONS[String(val)] ?? WARN_REASONS[String(val)] ?? String(val);
    } else if (k === "role") {
      v = roleLabelByCode(String(val));
    } else if (typeof val === "boolean") {
      // **و«true/false» إنكليزيّةٌ في لوحةٍ عربيّة** (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥).
      v = val ? P.detailYes : P.detailNo;
    } else if (typeof val === "object") {
      v = JSON.stringify(val);
    } else {
      v = String(val);
    }
    out.push({ k: DETAIL_KEYS[k] ?? k, v });
  }
  return out;
}

/**
 * **قائمةُ شكاوى — تقول من فتحها ولماذا.**
 *
 * **والعنوانُ نصٌّ محفوظٌ لا علامة** («شكوى على طلب» أو «بلاغُ سائق») —
 * **ومن بنى عليه حكماً بنى على نصٍّ يُبدَّل يوماً.** فيُقال السببُ شارةً
 * ويُقال فاتحُها باسمه.
 */
function TicketList({
  title,
  rows,
  total,
  perPage,
  page,
  onPage,
  onOpen,
  danger = false,
}: {
  title: string;
  rows: TicketRow[];
  total: number;
  perPage: number;
  page: number;
  onPage: (p: number) => void;
  onOpen: (t: TicketRow) => void;
  /** **وشكاوى عليه تُقرأ بلونها** — سجلٌّ عليه لا خدمةٌ طلبها. */
  danger?: boolean;
}) {
  return (
    <FormSection title={`${title} (${fmtNum(total)})`} icon={<IconSupport />}>
      {rows.length === 0 ? (
        <p className="py-4 text-center text-sm text-ink-muted">{P.ticketsEmpty}</p>
      ) : (
        <ul className="space-y-1.5">
          {rows.map((t) => (
            <li
              key={t.id}
              onClick={() => onOpen(t)}
              className="flex cursor-pointer flex-wrap items-center justify-between gap-2 rounded-card border border-line-soft p-3 text-sm hover:bg-row-hover"
            >
              <span className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="font-bold">#{fmtRef(t.number)}</span>
                {/* **والسببُ شارةٌ — وهو ما تعرضه شاشةُ التذاكر أصلاً.**
                    **وكان الملفُّ لا يجلبه**: تقرأ «شكوى على طلب» ولا
                    تعرف أهي «لم أستلم طلبي» أم «سلوك السائق» **حتّى تفتح
                    شاشةً أخرى.** */}
                {t.reason && REASONS[t.reason] ? (
                  <Badge variant={danger ? "danger" : "warning"}>{REASONS[t.reason]}</Badge>
                ) : (
                  <span className="truncate">{t.subject}</span>
                )}
                {/* **ومن فتحها** — **وبلاغُ سائقٍ في قائمة «شكاواه» يُقرأ
                    شكواه** ما لم يُقل فاتحُه. */}
                {t.by_name && (
                  <span className="truncate text-xs text-ink-muted">
                    {P.ticketBy}: {t.by_name}
                  </span>
                )}
              </span>
              <span className="flex items-center gap-2">
                {t.compensation > 0 && (
                  <span className="text-xs text-success">
                    +<Money value={t.compensation} />
                  </span>
                )}
                <Badge
                  variant={
                    t.status === "resolved" ? "success" : t.status === "open" ? "warning" : "primary"
                  }
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
      {total > perPage && (
        <div className="mt-3 flex justify-center">
          <Pagination page={page} total={total} perPage={perPage} onChange={onPage} />
        </div>
      )}
    </FormSection>
  );
}

/**
 * **سجلٌّ بسيطٌ في الكشف** — حوافزُ أو سحبٌ أو نزاعٌ أو بلاغُ طوارئ.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
 *
 * **ولا يُرسم إن كان فارغاً** — **وأربعةُ عناوينَ فارغةٍ تُقرأ عطباً**، وهي
 * تظهر لمن لا يكون له منها شيءٌ أبداً. (وهو داءُ «الماليّة» عند الزبون
 * الذي حُذف اليومَ نفسِه — **فلا يُعاد بأربع نسخ.**)
 */
function FinLog({
  title,
  rows,
  count,
  limit,
  icon,
  money = false,
  labelIsRole = false,
  onOrder,
}: {
  title: string;
  rows: FinEntry[];
  count: number;
  limit: number;
  icon: React.ReactNode;
  /** **اللفظُ رمزُ دورٍ** (`party_role` في النزاع) — يُترجَم: كان يظهر driver وmerchant. */
  labelIsRole?: boolean;
  /** **أفيه مبلغ؟** — بلاغُ الطوارئ لا مالَ فيه، **وصفرٌ معروضٌ يُقرأ مبلغا.** */
  money?: boolean;
  onOrder?: (ref: string) => void;
}) {
  if (rows.length === 0) return null;
  return (
    <FormSection title={`${title} (${fmtNum(count)})`} icon={icon}>
      <ul className="space-y-1.5">
        {rows.map((e, i) => (
          <li
            key={i}
            className="flex flex-wrap items-center justify-between gap-2 rounded-card border border-line-soft p-3 text-sm"
          >
            <span className="flex min-w-0 flex-wrap items-center gap-2">
              {e.status && (
                <Badge variant={FIN_TONE[e.status] ?? "neutral"}>
                  {(P.fin.st as Record<string, string>)[e.status] ?? e.status}
                </Badge>
              )}
              {/* **ورقمُ الطلب يُفتح** — والمرجعُ رقمٌ لا معرّف. */}
              {e.ref && onOrder && (
                <button
                  type="button"
                  onClick={() => onOrder(e.ref)}
                  className="font-medium text-primary hover:underline"
                >
                  #{fmtRef(Number(e.ref))}
                </button>
              )}
              {e.reason && <span className="truncate">{e.reason}</span>}
              {e.label && (
                <span className="truncate text-xs text-ink-muted">
                  {labelIsRole ? `${U.partyAs}: ${roleLabelByCode(e.label)}` : e.label}
                </span>
              )}
            </span>
            <span className="flex shrink-0 items-center gap-3">
              {/* **والعقوبةُ سالبةٌ بلونها** — **وموجبان في عمودٍ واحدٍ
                  يُقرأ من عوقب كمن كوفئ.** */}
              {money && e.amount !== 0 && (
                <span
                  className={`text-base font-bold tabular-nums ${e.amount < 0 ? "text-danger" : "text-success"}`}
                  dir="ltr"
                >
                  {fmtNum(e.amount)}
                </span>
              )}
              <span className="text-xs text-ink-muted" dir="ltr">
                {fmtDate(e.date)}
              </span>
            </span>
          </li>
        ))}
      </ul>
      {/* **والحدُّ يُقال** — **وسقفٌ صامتٌ يُقرأ «هذا كلُّ ما عليه».** */}
      {count > rows.length && (
        <p className="mt-2 text-center text-xs text-ink-muted">
          {P.fin.showingLatest
            .replace("{n}", fmtNum(rows.length))
            .replace("{all}", fmtNum(count))}
        </p>
      )}
      {limit > 0 && null}
    </FormSection>
  );
}

/** **ولونُ الحال يقول ما يقوله الاسم** — يُقرأ بلمحةٍ قبل أن يُقرأ حرفا. */
const FIN_TONE: Record<string, "success" | "danger" | "warning" | "neutral"> = {
  reward: "success",
  penalty: "danger",
  paid: "success",
  approved: "success",
  settled: "success",
  resolved: "success",
  rejected: "danger",
  open: "warning",
  pending: "warning",
  waived: "neutral",
};

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
              className="flex items-center justify-between gap-3 rounded-card border border-line-soft p-3 text-sm"
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
              <span className={`shrink-0 text-base font-bold tabular-nums ${tone}`} dir="ltr">
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
