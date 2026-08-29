"use client";

/**
 * **أقسامُ الملفّ الخاصّةُ بالدور** — طلباتُه وعناوينُه وصندوقُه ومتاجرُه.
 *
 * # لماذا هنا لا في الأقسام الأربعة
 *
 * كانت المنصةُ تحمل خمسةَ أبوابٍ للشيء الواحد: «الحسابات» فيها الجميع،
 * **و«الزبائن» و«السائقون» و«المتاجر» و«المندوبون» كلٌّ يعيد عرضَ فئةٍ منهم**
 * بأفعالٍ لا توجد في الآخر. **فمن أراد أن يعرف زبوناً فتح بابين ولم يجد صورتَه
 * في أيّهما كاملة.**
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «الزبائن له ملفٌّ خاصّ يجمع طلباته وشكاويه وكلَّ
 * شيءٍ يخصّه · وأيضاً السائقين والمتاجر والمندوب بنفس الأسلوب» ثمّ «يمكننا حذفُ
 * أقسام المندوب والسائقين والزبائن والمتاجر بما أنّها مجموعةٌ كلُّها بقسم
 * الحسابات — **ولا نريد خسارةَ أيّ ميزة**».
 *
 * **والشرطُ هو الأهمّ**: الحذفُ لا يجوز قبل أن يستوعب الملفُّ كلَّ فعلٍ كان في
 * الأقسام الأربعة. **فبُنيت هذه أوّلاً، ويُحذف بعدها.**
 *
 * # ولماذا تُحمَّل عند فتح التبويب
 *
 * أربعةُ نداءاتٍ إضافيةٍ في كلّ فتحِ ملفٍّ **حملٌ على من يريد رقمَ هاتفٍ فقط**.
 * فلا تُنادى نقطةٌ حتّى يُطلب تبويبُها.
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Alert,
  LoadingState,
  Badge,
  Button,
  FormSection,
  EmptyState,
  Pagination,
  IconOrder,
  IconChat,
  IconLocation,
  IconBalance,
  IconStore,
  IconCheck,
  IconLogout,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { StoreActions, type StoreTarget } from "@/components/admin/StoreActions";

const m = getMessages(defaultLocale);
const R = m.admin.users.profile.roleTabs;
// **ومفاتيحُ السائق من بابها** — كانت مترجَمةً ولا زرَّ يستعملها.
const D = m.admin.drivers;
const STATUS_LABELS: Record<string, string> = m.orders.status;
/** حالُ المتجر — **ثلاثةٌ لا خارطةَ لها في القاموس**، فتُجمع هنا مرّةً. */
const MERCHANT_STATUS: Record<string, string> = {
  active: m.admin.merchants.active,
  inactive: m.admin.merchants.inactive,
  suspended: m.admin.merchants.suspended,
};

const STATUS_VARIANT: Record<string, "warning" | "primary" | "success" | "danger" | "neutral"> = {
  pending: "warning",
  accepted: "primary",
  preparing: "primary",
  dispatching: "warning",
  assigned: "primary",
  at_pickup: "primary",
  picked_up: "primary",
  on_the_way: "primary",
  at_dropoff: "primary",
  delivered: "success",
  rejected: "danger",
  cancelled: "danger",
  failed: "danger",
  refunded: "neutral",
};

interface OrderRow {
  id: string;
  number: number;
  status: string;
  total: number;
  created_at: string;
  merchant_name: string;
  /** **نوعُ الطلب** — `custom` طلبٌ خاصٌّ بلا متجر. */
  kind?: string;
  /** ما طلبه الزبونُ بلفظه — في الخاصّ وحدَه. */
  custom_request?: string;
  /** **لماذا لم يصل** — في الملغى والمرفوض والمتعذّر. */
  cancel_reason?: string;
}

/** **رأسُ حديثٍ في ملفّ صاحبه** — سطورُه تُطلب عند فتحه. */
interface ChatThread {
  order_id: string;
  order_number: number;
  peer: string;
  peer_role: string;
  count: number;
  flagged: number;
  last: string;
  last_role: string;
  last_at: string;
}

interface ChatAudit {
  customer: string;
  driver: string;
  lines: {
    id: string;
    body: string;
    role: string;
    sender: string;
    created_at: string;
    flagged: boolean;
    flag_word?: string;
  }[];
}

interface AddressRow {
  id: string;
  label: string;
  address_text: string;
  is_default: boolean;
}

interface CashEntry {
  kind: string;
  amount: number;
  note: string;
  order_number: number | null;
  created_at: string;
}

/** **عميلٌ محتملٌ رشّحه المندوب** — بحاله. */
interface LeadRow {
  id: string;
  store_name: string;
  phone: string;
  area: string;
  status: string;
}

/** **وصفُّ المتجر كاملاً** — **نافذةُ التعديل تحتاجه كلَّه**، ولأنّ
 *  أزرارَه انتقلت إلى هنا (قرارُ المالك ٢٠٢٦-٠٨-١٦). */
type StoreRow = StoreTarget;

/**
 * **طلباتُه** — زبوناً كان أو سائقاً.
 *
 * والمرشِّحُ يختلف بالدور: **`customer_id` لمن طلب، و`driver_id` لمن أوصل** —
 * ومن يحمل الدورين يُعرض له الاثنان في قائمتين، **فخلطُهما يجعل «طلباته» تعني
 * شيئين في سطرٍ واحد.**
 */
export function OrdersTab({ userID, roles }: { userID: string; roles: string[] }) {
  const router = useRouter();
  const open = useCallback(
    (o: OrderRow) => router.push(`/dashboard/orders?q=${o.number}`),
    [router],
  );
  return (
    <div className="space-y-4">
      {roles.includes("customer") && (
        <OrderList title={R.ordersAsCustomer} filter={`customer_id=${userID}`} onOpen={open} />
      )}
      {roles.includes("driver") && (
        <OrderList title={R.ordersAsDriver} filter={`driver_id=${userID}`} onOpen={open} />
      )}
      {/* ══════════════════════════════════════════════════════════════
          **وطلباتُ متجره — وهي كلُّ عمله**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-١٦: «ابدأ بملفّ صاحب المتجر».)

          **كان الملفُّ يعرض ما اشتراه لنفسه** ولا يعرض **طلباً واحداً
          وصل متجرَه.** والمُرشِّحُ في المحرّك موجودٌ ولا يُنادى من هنا.

          **و`owner_id` لا `merchant_id`**: الأوّلُ يجمع متاجرَه كلَّها،
          **والثاني يخاطب متجراً بعينه** — والملفُّ يخاطب إنسانا. */}
      {roles.includes("merchant") && (
        <OrderList title={R.ordersAtStore} filter={`owner_id=${userID}`} onOpen={open} />
      )}
    </div>
  );
}

/**
 * **عشرون في الصفحة** — والباقي بالتنقّل.
 *
 * **وكانت خمسين بلا تنقّل** (قرارُ المالك ٢٠٢٦-٠٨-١٥: «اجعلها كاملةً وليس
 * ٥٠ فقط، واجعل باجينيشن أيضاً»). **فمن طلب ستّين يُعرض له خمسون** —
 * **والعشرةُ الباقيةُ لا بابَ إليها**، ولا شيءَ يقول إنّها حُجبت.
 */
const ORDERS_PER_PAGE = 20;

/**
 * **قائمةٌ تجلب صفحتَها بنفسها.**
 *
 * **ولكلٍّ صفحتُها**: من حمل الدورين له قائمتان — **وصفحةٌ واحدةٌ تحكمهما
 * تُقلّب ما لم يُطلب تقليبُه.**
 */
function OrderList({
  title,
  filter,
  onOpen,
}: {
  title: string;
  filter: string;
  onOpen: (o: OrderRow) => void;
}) {
  const [rows, setRows] = useState<OrderRow[] | null | "failed">(null);
  // **والمجموعُ من المحرّك لا من طول الصفحة** — طولُها عشرون دائماً،
  // **ولو كُتب في العنوان لقال «طلباته (٢٠)» لمن طلب مئة.**
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);

  useEffect(() => {
    let alive = true;
    api<{ orders: OrderRow[]; total: number }>(
      `/api/v1/admin/orders?${filter}&page=${page}&per_page=${ORDERS_PER_PAGE}`,
    )
      .then((r) => {
        if (!alive) return;
        setRows(r.orders ?? []);
        setTotal(r.total ?? 0);
      })
      .catch(() => alive && setRows("failed"));
    return () => {
      alive = false;
    };
  }, [filter, page]);

  if (rows === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (rows === null) return <LoadingState variant="text" />;

  return (
    <FormSection title={`${title} (${fmtNum(total)})`} icon={<IconOrder />}>
      {rows.length === 0 ? (
        <EmptyState icon={IconOrder} title={R.ordersEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((o) => (
            <li key={o.id}>
              <button
                onClick={() => onOpen(o)}
                className="flex w-full flex-col gap-0.5 py-2 text-start hover:bg-row-hover"
              >
                <span className="flex w-full items-center gap-3">
                <span dir="ltr" className="w-16 shrink-0 font-bold tabular-nums">
                  #{fmtRef(o.number)}
                </span>
                <Badge variant={STATUS_VARIANT[o.status] ?? "neutral"}>
                  {STATUS_LABELS[o.status] ?? o.status}
                </Badge>
                {/* ══════════════════════════════════════════════════════
                    **والخاصُّ يقول ما هو — لا فراغاً مكانَ متجر**
                    ══════════════════════════════════════════════════════

                    (قرارُ المالك ٢٠٢٦-٠٨-١٥.)

                    **الطلبُ الخاصُّ لا متجرَ له** — والعمودُ كان يُعرض
                    فارغاً، **فيُقرأ عطباً في القراءة** لا «طلبٌ بلا
                    متجر»: أضاع الاسمُ أم لم يكن؟

                    **فتحلّ الشارةُ محلَّ الاسم ويتبعها نصُّ طلبه** —
                    وهو ما يقوم مقام اسم المتجر في هذا النوع. */}
                {o.kind === "custom" ? (
                  <span className="flex min-w-0 flex-1 items-center gap-2">
                    <Badge variant="warning">{R.orderCustom}</Badge>
                    <span className="truncate text-sm text-ink-muted">
                      {o.custom_request}
                    </span>
                  </span>
                ) : (
                  <span className="min-w-0 flex-1 truncate text-sm text-ink-muted">
                    {o.merchant_name}
                  </span>
                )}
                <span dir="ltr" className="shrink-0 tabular-nums">
                  {fmtNum(o.total)}
                </span>
                <span dir="ltr" className="hidden shrink-0 text-xs text-ink-muted sm:inline">
                  {fmtDateTime(o.created_at)}
                </span>
                </span>
                {/* ══════════════════════════════════════════════════════
                    **ولماذا لم يصل — في سطره**
                    ══════════════════════════════════════════════════════

                    (قرارُ المالك ٢٠٢٦-٠٨-١٥: نُقل مع حذف «الماليّة» من
                    ملفّ الزبون.)

                    **كان السببُ في تبويب «الماليّة» وحدَه** — وهو تبويبٌ
                    كلُّ ما فيه للزبون بطاقتا صفر، **فيُفتح لأجل سطرٍ
                    واحدٍ ولا يُعرف أنّه فيه.**

                    **وموضعُه سطرُ الطلب**: من رأى «ملغى» سأل «لماذا»
                    في اللحظة نفسِها — **وجوابٌ في شاشةٍ أخرى لا يُقرأ.** */}
                {o.cancel_reason && (
                  <span className="w-full truncate ps-16 text-xs text-danger">
                    {o.cancel_reason}
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
      <Pagination
        page={page}
        total={total}
        perPage={ORDERS_PER_PAGE}
        onChange={setPage}
      />
    </FormSection>
  );
}

/**
 * **أحاديثُه — كلُّها في ملفّه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «نسينا سجلَّ الدردشات… مهمّةٌ في حال حدوث أيّ
 *  مشكلةٍ أو نزاع».)
 *
 * # ولماذا في ملفّه لا في الطلب
 *
 * **الحديثُ كان يُقرأ من الطلب وحدَه** — ومن يحكم في نزاعٍ **لا يعرف رقمَ
 * الطلب بعد**: يعرف اسمَ الإنسان، **فيمشي طلباته واحداً واحداً يفتح كلَّ
 * حديثٍ يبحث عن سطر.**
 *
 * **والسؤالُ عند الخلاف عن شخصٍ لا عن طلب**: «هل أساء هذا السائقُ من
 * قبل؟» — وجوابُه في عشرين حديثاً متفرّقاً.
 *
 * # والسطورُ تُطلب بالطلب
 *
 * **رأسُ الحديث وحدَه يُجلب** — **ومئاتُ الأسطر مع كلّ فتحةِ ملفٍّ حملٌ
 * لسؤالٍ لم يُسأل بعد.**
 */
export function ChatsTab({ userID }: { userID: string }) {
  const [rows, setRows] = useState<ChatThread[] | null | "failed">(null);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState("");

  useEffect(() => {
    let alive = true;
    api<{ threads: ChatThread[]; total: number }>(
      `/api/v1/admin/users/${userID}/chats?page=${page}&per_page=${CHATS_PER_PAGE}`,
    )
      .then((r) => {
        if (!alive) return;
        setRows(r.threads ?? []);
        setTotal(r.total ?? 0);
      })
      .catch(() => alive && setRows("failed"));
    return () => {
      alive = false;
    };
  }, [userID, page]);

  if (rows === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (rows === null) return <LoadingState variant="text" />;

  return (
    <FormSection title={`${R.chats} (${fmtNum(total)})`} icon={<IconChat />}>
      {rows.length === 0 ? (
        <EmptyState icon={IconChat} title={R.chatsEmpty} />
      ) : (
        <ul className="space-y-2">
          {rows.map((t) => (
            <li key={t.order_id} className="rounded-control border border-line p-3">
              <button
                type="button"
                onClick={() => setOpen(open === t.order_id ? "" : t.order_id)}
                className="flex w-full items-center gap-2 text-start"
              >
                <span dir="ltr" className="shrink-0 font-bold tabular-nums">
                  #{fmtRef(t.order_number)}
                </span>
                {/* **والموسومُ يُعلَن في الرأس** — **وهو ما يُفتح الملفُّ
                    لأجله**، ولا يُعرف بلا فتحِ كلِّ حديث. */}
                {t.flagged > 0 && (
                  <Badge variant="danger">
                    {R.chatFlagged.replace("{n}", fmtNum(t.flagged))}
                  </Badge>
                )}
                <span className="min-w-0 flex-1 truncate text-sm text-ink-muted">
                  {t.peer} — {t.last}
                </span>
                <span className="shrink-0 text-2xs text-ink-muted">
                  {R.chatLines.replace("{n}", fmtNum(t.count))}
                </span>
                <span dir="ltr" className="shrink-0 text-2xs text-ink-muted">
                  {fmtDateTime(t.last_at)}
                </span>
              </button>
              {open === t.order_id && <ChatLines orderID={t.order_id} />}
            </li>
          ))}
        </ul>
      )}
      <Pagination page={page} total={total} perPage={CHATS_PER_PAGE} onChange={setPage} />
    </FormSection>
  );
}

const CHATS_PER_PAGE = 10;

/**
 * **سطورُ الحديث — بقائلها لا بـ«لي/له».**
 *
 * **والإدارةُ ليست طرفاً**، فلا `mine` لها. **ولا وسمَ قراءةٍ يُكتب**:
 * علامةٌ زرقاءُ يضعها طرفٌ ثالثٌ كذبٌ يُحتجّ به.
 */
function ChatLines({ orderID }: { orderID: string }) {
  const [th, setTh] = useState<ChatAudit | null | "failed">(null);

  useEffect(() => {
    let alive = true;
    api<ChatAudit>(`/api/v1/admin/orders/${orderID}/chat`)
      .then((r) => alive && setTh(r))
      .catch(() => alive && setTh("failed"));
    return () => {
      alive = false;
    };
  }, [orderID]);

  if (th === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (th === null) return <LoadingState variant="text" />;

  return (
    <ul className="mt-3 space-y-1.5 border-t border-line-soft pt-3">
      {th.lines.map((l) => (
        <li key={l.id} className="flex items-start gap-2 text-sm">
          <Badge variant={l.role === "driver" ? "primary" : "neutral"}>
            {l.sender || (l.role === "driver" ? R.chatDriver : R.chatCustomer)}
          </Badge>
          <span className="min-w-0 flex-1">
            <span className={l.flagged ? "text-danger" : ""}>{l.body}</span>
            {/* **واللفظُ الذي أوقعها يُقال** — **ولمراجعة الحارس نفسِه**
                حين يَسِم بريئا. */}
            {l.flagged && l.flag_word && (
              <span className="ms-2 text-2xs text-danger">({l.flag_word})</span>
            )}
          </span>
          <span dir="ltr" className="shrink-0 text-2xs text-ink-muted">
            {fmtDateTime(l.created_at)}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** **عناوينُه** — قراءةً لا كتابة: عنوانُ بيتِ إنسانٍ يكتبه هو. */
export function AddressesTab({ userID }: { userID: string }) {
  const [rows, setRows] = useState<AddressRow[] | null | "failed">(null);

  useEffect(() => {
    api<AddressRow[]>(`/api/v1/admin/users/${userID}/addresses`)
      .then(setRows)
      .catch(() => setRows("failed"));
  }, [userID]);

  // **والفشلُ يُقال** — كان يُعرض «لا شيء» فيُقرأ حكماً على المستخدم.
  if (rows === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (!rows) return <LoadingState variant="text" />;

  return (
    <FormSection title={R.addresses} icon={<IconLocation />}>
      {rows.length === 0 ? (
        <EmptyState icon={IconLocation} title={R.addressesEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((a) => (
            <li key={a.id} className="flex items-start gap-3 py-2">
              <IconLocation size={16} className="mt-0.5 shrink-0 text-ink-muted" />
              <span className="min-w-0 flex-1">
                <span className="block font-medium">
                  {a.label}
                  {a.is_default && (
                    <Badge variant="primary" className="ms-2">
                      {R.defaultAddress}
                    </Badge>
                  )}
                </span>
                <span className="block text-sm text-ink-muted">{a.address_text}</span>
              </span>
            </li>
          ))}
        </ul>
      )}
    </FormSection>
  );
}

/**
 * **صندوقُ نقده** — ما قبض وما سلّم، **وزرُّ التسوية معه.**
 *
 * كان في قسمٍ منفصلٍ (`/dashboard/cash`) يجمع كلَّ السائقين. **ومن فتح ملفَّ
 * سائقٍ ليعرف ما في ذمّته اضطُرّ إلى قسمٍ آخرَ ثمّ بحثٍ عن اسمه** — ومالٌ في
 * ذمّة إنسانٍ يُقرأ في ملفّه.
 */
export function CashboxTab({
  userID,
  canSettle,
  onSettled,
}: {
  userID: string;
  canSettle: boolean;
  onSettled: () => void;
}) {
  const [held, setHeld] = useState(0);
  const [rows, setRows] = useState<CashEntry[] | null | "failed">(null);
  const [busy, setBusy] = useState(false);
  // **وإغلاقُ الدوام خلف تأكيد** — فعلٌ يُخرج إنساناً من عمله، **وضغطةٌ
  // بلا رجعةٍ على زرٍّ يجاور «تسليم الصندوق» تقع سهواً.**
  const [ending, setEnding] = useState(false);
  const [shiftNote, setShiftNote] = useState("");

  const load = useCallback(() => {
    api<{ held: number; entries: CashEntry[] }>(`/api/v1/admin/drivers/${userID}/cash`)
      .then((r) => {
        setHeld(r.held ?? 0);
        setRows(r.entries ?? []);
      })
      .catch(() => setRows("failed"));
  }, [userID]);

  useEffect(load, [load]);

  async function settle() {
    setBusy(true);
    try {
      await api(`/api/v1/admin/drivers/${userID}/settle`, {
        method: "POST",
        body: JSON.stringify({ amount: held, note: R.settleNote }),
      });
      load();
      onSettled();
    } finally {
      setBusy(false);
    }
  }

  // ══════════════════════════════════════════════════════════════════
  // **وإغلاقُ الدوام من هنا** — (بُني ٢٠٢٦-٠٨-٢٣)
  // ══════════════════════════════════════════════════════════════════
  //
  // **كانت النقطةُ في المحرّك واسمُها مترجَماً منذ زمنٍ ولا زرَّ
  // يستعملهما** — قِيس بمقارنة أبواب المحرّك بما تناديه اللوحة.
  //
  // **ومن ذهب ونسي علمَه يبقى في الدور، فيتأخّر كلُّ طلبٍ بمقدار
  // غيابه** — والعملياتُ تراه متاحاً ولا تملك أن تُخرجه.
  //
  // **وموضعُه ملفُّ السائق لا شاشةٌ ثانية**: من فتحه ليعرف لماذا لا
  // تصل طلباتُه هو من يحتاج الزرّ.
  async function endShift() {
    setBusy(true);
    try {
      await api(`/api/v1/admin/drivers/${userID}/end-shift`, {
        method: "POST",
        body: JSON.stringify({ note: shiftNote.trim() }),
      });
      setShiftNote("");
      setEnding(false);
      onSettled();
    } finally {
      setBusy(false);
    }
  }

  // **والفشلُ يُقال** — كان يُعرض «لا شيء» فيُقرأ حكماً على المستخدم.
  if (rows === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (!rows) return <LoadingState variant="text" />;

  return (
    <FormSection title={R.cashbox} icon={<IconBalance />}>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-control bg-field px-3 py-2">
        <span className="text-sm text-ink-muted">{R.cashHeld}</span>
        <span dir="ltr" className={`figure ${held > 0 ? "text-warning" : ""}`}>
          <Money value={held} small />
        </span>
        {canSettle && held > 0 && (
          <Button disabled={busy} onClick={() => void settle()}>
            <span className="flex items-center gap-1.5">
              <IconCheck size={14} />
              {R.settle}
            </span>
          </Button>
        )}
      </div>
      {/* ══════════════════════════════════════════════════════════
          **إغلاقُ دوامه** — انظر `endShift` أعلاه
          ══════════════════════════════════════════════════════════ */}
      <div className="mb-3 rounded-control border border-line px-3 py-2">
        {!ending ? (
          <>
            <button
              type="button"
              className="flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
              onClick={() => setEnding(true)}
            >
              <IconLogout size={14} />
              {D.endShift}
            </button>
            <p className="mt-1 text-xs text-ink-muted">{D.endShiftHint}</p>
          </>
        ) : (
          <div className="space-y-2">
            <p className="text-xs text-ink-muted">{D.endShiftHint}</p>
            <input
              className="w-full rounded-control border border-line bg-field px-2 py-1.5 text-sm"
              placeholder={D.endShiftNote}
              value={shiftNote}
              onChange={(e) => setShiftNote(e.target.value)}
            />
            <FormActions onSave={() => void endShift()} onCancel={() => setEnding(false)} busy={busy} saveLabel={D.endShift} />
          </div>
        )}
      </div>

      {rows.length === 0 ? (
        <EmptyState icon={IconBalance} title={R.cashEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((e, i) => (
            <li key={i} className="flex items-center gap-3 py-2 text-sm">
              <span className="min-w-0 flex-1">
                {e.order_number ? (
                  <span dir="ltr" className="font-bold tabular-nums">
                    #{fmtRef(e.order_number)}
                  </span>
                ) : (
                  <span className="text-ink-muted">{R.cashKinds[e.kind as keyof typeof R.cashKinds] ?? e.kind}</span>
                )}
                {e.note && <span className="block text-xs text-ink-muted">{e.note}</span>}
              </span>
              <span
                dir="ltr"
                className={`shrink-0 tabular-nums ${e.amount < 0 ? "text-success" : "text-warning"}`}
              >
                {e.amount > 0 ? "+" : ""}
                {fmtNum(e.amount)}
              </span>
              <span dir="ltr" className="hidden shrink-0 text-xs text-ink-muted sm:inline">
                {fmtDateTime(e.created_at)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </FormSection>
  );
}

/**
 * **متاجرُه** — لصاحب المتجر متجرُه، **وللمندوب ما جلب.**
 *
 * والسؤالان مختلفان: **«ماذا أملك؟» و«ماذا جلبتُ؟»** — والاستعلامُ واحدٌ
 * بمرشِّحٍ مختلف (`rep_id`).
 */
export function StoresTab({ userID, roles }: { userID: string; roles: string[] }) {
  const router = useRouter();
  const [rows, setRows] = useState<StoreRow[] | null | "failed">(null);
  const isRep = roles.includes("sales");

  // ══════════════════════════════════════════════════════════════════
  // **ومالكُه في الشرط — لا قائمةٌ بلا شرط**
  // ══════════════════════════════════════════════════════════════════
  //
  // (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
  //
  // **كان يُنادى `query=` لصاحب المتجر** — بلا مُرشِّحٍ أصلاً، **فيعرض
  // متاجرَ المنصّة كلَّها** تحت عنوان «متاجرُ يملكها». **ولم يُرَ لأنّ في
  // القاعدة متجراً واحداً** — وحين تصير عشرين يراها كلُّ صاحبِ متجرٍ في
  // ملفّه.
  //
  // **والمحرّكُ لم يكن يعرف المالكَ أصلاً**: أربعةُ مُرشِّحاتٍ ولا واحدَ
  // منها له. **فأُضيف هناك أوّلاً** — وشرطٌ في الشاشة بلا شرطٍ في المحرّك
  // وعدٌ بحجب.
  // **ويُعاد الجلبُ بعد كلّ فعل** — وإلّا بقي السطرُ يقول ما بطل: يُحظر
  // المتجرُ ويبقى «فعّال» أمام من حظره.
  const reload = useCallback(() => {
    api<{ merchants: StoreRow[] }>(
      `/api/v1/admin/merchants?${isRep ? `rep_id=${userID}` : `owner_id=${userID}`}&per_page=100`,
    )
      .then((r) => setRows(r.merchants ?? []))
      .catch(() => setRows("failed"));
  }, [userID, isRep]);

  useEffect(reload, [reload]);

  // **والفشلُ يُقال** — كان يُعرض «لا شيء» فيُقرأ حكماً على المستخدم.
  if (rows === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (!rows) return <LoadingState variant="text" />;

  return (
    <>
    <FormSection title={isRep ? R.storesBrought : R.storesOwned} icon={<IconStore />}>
      {rows.length === 0 ? (
        <EmptyState icon={IconStore} title={R.storesEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((s) => (
            <li key={s.id}>
              <div className="flex w-full flex-wrap items-center gap-3 py-2">
                <IconStore size={16} className="shrink-0 text-ink-muted" />
                <span className="min-w-0 flex-1 truncate font-medium">{s.name}</span>
                <Badge variant={s.status === "active" ? "success" : "danger"}>
                  {MERCHANT_STATUS[s.status] ?? s.status}
                </Badge>
                {/* **وعدّادُ مخالفاته معه** — (قرارُ المالك ٢٠٢٦-٠٨-١٦).

                    **المحرّكُ يرسله والتبويبُ يُسقطه** — **وهو الرقمُ
                    الذي يقرّر الحظر**، فمن راجع صاحبَه ليقرّر قرأ اسماً
                    وحالاً وعمولةً ولم يقرأ ما يحكم. */}
                {s.violations > 0 && (
                  <Badge variant="danger">
                    {R.violations.replace("{n}", fmtNum(s.violations))}
                  </Badge>
                )}
                <span dir="ltr" className="shrink-0 text-sm tabular-nums text-ink-muted">
                  {fmtNum(s.commission_percent)}%
                </span>
                {/* ══════════════════════════════════════════════════════
                    **وأفعالُ المتجر معه — لا في شاشةٍ أخرى**
                    ══════════════════════════════════════════════════════

                    (قرارُ المالك ٢٠٢٦-٠٨-١٦.)

                    **والشرطُ قبل حذف تبويب المتاجر**: أن يستوعب الملفُّ
                    **كلَّ فعلٍ** كان فيه — «ولا نريد خسارةَ أيّ ميزة».

                    **وهي في سطر المتجر لا أعلى الصفحة**: هناك زرّا
                    إيقافٍ وحظرٍ **للحساب**، **وزرّان متشابهان لمعنيين
                    يُخلطان** — فيُوقَف إنسانٌ وقُصد متجرُه. */}
                <StoreActions store={s} onChanged={reload} />
              </div>
            </li>
          ))}
        </ul>
      )}
    </FormSection>
    {/* **وعملاؤه المحتملون معهم** — (قرارُ المالك ٢٠٢٦-٠٨-١٦).

        **وهي عملُ المندوب الأوّل**: الملفُّ كان يقول «كم متجراً جلب»
        **ولا يقول كم رشّح وكم رُفض له.** وشاشتُها منفصلة، **وهو داءُ
        الأربعة الذي عولج أمس.** */}
    {isRep && <LeadsList userID={userID} />}
    </>
  );
}

/**
 * **عملاؤه المحتملون** — ما رشّحه المندوبُ ولم يُفتح بعد.
 *
 * **وحالاتُه ثلاث**: جديدٌ · تحوّل إلى متجر · رُفض. **والمرفوضُ يُقال** —
 * **وقائمةٌ تعرض ما نجح وحدَه تُري المندوبَ أفضلَ ممّا هو.**
 */
function LeadsList({ userID }: { userID: string }) {
  const [rows, setRows] = useState<LeadRow[] | null | "failed">(null);

  useEffect(() => {
    api<{ leads: LeadRow[] }>(`/api/v1/admin/leads?rep_id=${userID}&per_page=50`)
      .then((r) => setRows(r.leads ?? []))
      .catch(() => setRows("failed"));
  }, [userID]);

  if (rows === "failed") return <Alert tone="warning">{m.errors.offline}</Alert>;
  if (!rows) return <LoadingState variant="text" />;
  // **ولا يُرسم فارغاً** — عنوانٌ صفريٌّ يُقرأ عطباً.
  if (rows.length === 0) return null;

  return (
    <FormSection title={`${R.leads} (${fmtNum(rows.length)})`} icon={<IconStore />}>
      <ul className="divide-y divide-line">
        {rows.map((l) => (
          <li key={l.id} className="flex items-center gap-3 py-2">
            <Badge variant={LEAD_TONE[l.status] ?? "neutral"}>
              {(R.leadSt as Record<string, string>)[l.status] ?? l.status}
            </Badge>
            <span className="min-w-0 flex-1 truncate font-medium">{l.store_name}</span>
            {l.area && <span className="shrink-0 text-xs text-ink-muted">{l.area}</span>}
            <span dir="ltr" className="shrink-0 text-xs text-ink-muted">{l.phone}</span>
          </li>
        ))}
      </ul>
    </FormSection>
  );
}

/** **ولونُ الحال يُقرأ قبل حرفه.** */
const LEAD_TONE: Record<string, "success" | "danger" | "warning"> = {
  new: "warning",
  converted: "success",
  rejected: "danger",
};
