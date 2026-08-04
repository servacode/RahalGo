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
  Badge,
  Button,
  FormSection,
  EmptyState,
  IconOrder,
  IconLocation,
  IconBalance,
  IconStore,
  IconCheck,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const R = m.admin.users.profile.roleTabs;
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

interface StoreRow {
  id: string;
  name: string;
  status: string;
  commission_percent: number;
}

/**
 * **طلباتُه** — زبوناً كان أو سائقاً.
 *
 * والمرشِّحُ يختلف بالدور: **`customer_id` لمن طلب، و`driver_id` لمن أوصل** —
 * ومن يحمل الدورين يُعرض له الاثنان في قائمتين، **فخلطُهما يجعل «طلباته» تعني
 * شيئين في سطرٍ واحد.**
 */
export function OrdersTab({ userID, roles }: { userID: string; roles: string[] }) {
  const router = useRouter();
  const [asCustomer, setAsCustomer] = useState<OrderRow[]>([]);
  const [asDriver, setAsDriver] = useState<OrderRow[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    const get = (q: string) =>
      api<{ orders: OrderRow[] }>(`/api/v1/admin/orders?${q}&per_page=50`)
        .then((r) => r.orders ?? [])
        .catch(() => []);
    const [c, d] = await Promise.all([
      roles.includes("customer") ? get(`customer_id=${userID}`) : Promise.resolve([]),
      roles.includes("driver") ? get(`driver_id=${userID}`) : Promise.resolve([]),
    ]);
    setAsCustomer(c);
    setAsDriver(d);
    setLoading(false);
  }, [userID, roles]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) return <p className="py-8 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div className="space-y-4">
      {roles.includes("customer") && (
        <OrderList title={R.ordersAsCustomer} rows={asCustomer} onOpen={(o) =>
          router.push(`/dashboard/orders?q=${o.number}`)} />
      )}
      {roles.includes("driver") && (
        <OrderList title={R.ordersAsDriver} rows={asDriver} onOpen={(o) =>
          router.push(`/dashboard/orders?q=${o.number}`)} />
      )}
    </div>
  );
}

function OrderList({
  title,
  rows,
  onOpen,
}: {
  title: string;
  rows: OrderRow[];
  onOpen: (o: OrderRow) => void;
}) {
  return (
    <FormSection title={`${title} (${fmtNum(rows.length)})`} icon={<IconOrder />}>
      {rows.length === 0 ? (
        <EmptyState icon={IconOrder} title={R.ordersEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((o) => (
            <li key={o.id}>
              <button
                onClick={() => onOpen(o)}
                className="flex w-full items-center gap-3 py-2 text-start hover:bg-page/60"
              >
                <span dir="ltr" className="w-16 shrink-0 font-bold tabular-nums">
                  #{fmtRef(o.number)}
                </span>
                <Badge variant={STATUS_VARIANT[o.status] ?? "neutral"}>
                  {STATUS_LABELS[o.status] ?? o.status}
                </Badge>
                <span className="min-w-0 flex-1 truncate text-sm text-ink-muted">
                  {o.merchant_name}
                </span>
                <span dir="ltr" className="shrink-0 tabular-nums">
                  {fmtNum(o.total)}
                </span>
                <span dir="ltr" className="hidden shrink-0 text-xs text-ink-muted sm:inline">
                  {fmtDateTime(o.created_at)}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </FormSection>
  );
}

/** **عناوينُه** — قراءةً لا كتابة: عنوانُ بيتِ إنسانٍ يكتبه هو. */
export function AddressesTab({ userID }: { userID: string }) {
  const [rows, setRows] = useState<AddressRow[] | null>(null);

  useEffect(() => {
    api<AddressRow[]>(`/api/v1/admin/users/${userID}/addresses`)
      .then(setRows)
      .catch(() => setRows([]));
  }, [userID]);

  if (!rows) return <p className="py-8 text-center text-ink-muted">{m.common.loading}</p>;

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
  const [rows, setRows] = useState<CashEntry[] | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api<{ held: number; entries: CashEntry[] }>(`/api/v1/admin/drivers/${userID}/cash`)
      .then((r) => {
        setHeld(r.held ?? 0);
        setRows(r.entries ?? []);
      })
      .catch(() => setRows([]));
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

  if (!rows) return <p className="py-8 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <FormSection title={R.cashbox} icon={<IconBalance />}>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-control bg-page px-3 py-2">
        <span className="text-sm text-ink-muted">{R.cashHeld}</span>
        <span dir="ltr" className={`text-lg font-bold tabular-nums ${held > 0 ? "text-warning" : ""}`}>
          {fmtNum(held)} <span className="text-xs font-normal">{m.common.currency}</span>
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
  const [rows, setRows] = useState<StoreRow[] | null>(null);
  const isRep = roles.includes("sales");

  useEffect(() => {
    const q = isRep ? `rep_id=${userID}` : `query=`;
    api<{ merchants: StoreRow[] }>(`/api/v1/admin/merchants?${q}&per_page=100`)
      .then((r) => setRows(r.merchants ?? []))
      .catch(() => setRows([]));
  }, [userID, isRep]);

  if (!rows) return <p className="py-8 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <FormSection title={isRep ? R.storesBrought : R.storesOwned} icon={<IconStore />}>
      {rows.length === 0 ? (
        <EmptyState icon={IconStore} title={R.storesEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((s) => (
            <li key={s.id}>
              <button
                onClick={() => router.push(`/dashboard/merchants?q=${encodeURIComponent(s.name)}`)}
                className="flex w-full items-center gap-3 py-2 text-start hover:bg-page/60"
              >
                <IconStore size={16} className="shrink-0 text-ink-muted" />
                <span className="min-w-0 flex-1 truncate font-medium">{s.name}</span>
                <Badge variant={s.status === "active" ? "success" : "danger"}>
                  {MERCHANT_STATUS[s.status] ?? s.status}
                </Badge>
                <span dir="ltr" className="shrink-0 text-sm tabular-nums text-ink-muted">
                  {fmtNum(s.commission_percent)}%
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </FormSection>
  );
}
