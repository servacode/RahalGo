"use client";

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Select,
  Badge,
  Modal,
  FormSection,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconOrder,
  IconAdd,
  IconSearch,
  IconUser,
  IconStore,
  IconWallet,
  IconDriver,
  IconStatus,
  IconLocation,
  IconPhone,
  IconDelete,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";

const PickMap = dynamic(() => import("@/components/map/PickMap"), { ssr: false });

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

// ---------- الأنواع ----------

interface OrderRow {
  id: string;
  number: number;
  customer_phone: string;
  customer_name: string;
  merchant_name: string;
  driver_phone: string | null;
  status: string;
  payment_method: "cash" | "wallet" | "mixed";
  subtotal: number;
  delivery_fee: number;
  discount: number;
  total: number;
  wallet_paid: number;
  cash_due: number;
  promo_code: string | null;
  address_text: string;
  notes: string;
  cancel_reason: string;
  created_at: string;
  items?: {
    id: string;
    name: string;
    unit_price: number;
    qty: number;
    note: string;
    options: { group: string; name: string; price_delta: number }[];
  }[];
  events?: { from_status: string; to_status: string; note: string; created_at: string }[];
}

interface OrderPage {
  orders: OrderRow[];
  total: number;
  page: number;
  per_page: number;
}

const STATUS_LABELS: Record<string, string> = m.orders.status;
const ACTION_LABELS: Record<string, string> = m.admin.ordersPage.actions;
const PAYMENT_LABELS: Record<string, string> = m.orders.payment;

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

// أزرار الانتقال المتاحة للعمليات/الأدمن حسب الحالة (مرآة لخارطة الخادم)
const OPS_NEXT: Record<string, string[]> = {
  pending: ["accepted", "rejected", "cancelled"],
  accepted: ["preparing", "cancelled"],
  preparing: ["cancelled"], // + إسناد سائق
  dispatching: ["cancelled"], // + إسناد سائق
  assigned: ["at_pickup", "cancelled"],
  at_pickup: ["picked_up", "cancelled"],
  picked_up: ["on_the_way"],
  on_the_way: ["at_dropoff"],
  at_dropoff: ["delivered", "failed"],
  delivered: ["refunded"],
};

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

// ---------- الشاشة الرئيسية ----------

export default function OrdersPage() {
  const [data, setData] = useState<OrderPage | null>(null);
  const [status, setStatus] = useState("");
  const [query, setQuery] = useState("");
  const [openOnly, setOpenOnly] = useState(true);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [detailID, setDetailID] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [view, setView] = useViewMode("orders");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        status,
        query,
        open: openOnly ? "1" : "",
        page: String(page),
        per_page: "12",
      });
      setData(await api<OrderPage>(`/api/v1/admin/orders?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [status, query, openOnly, page]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  // تحديث دوري كل 10 ثوانٍ (حتى يصل WebSocket)
  useEffect(() => {
    const t = setInterval(load, 10000);
    return () => clearInterval(t);
  }, [load]);

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<OrderRow>[] = [
    {
      id: "number",
      header: m.admin.ordersPage.number,
      icon: <IconOrder />,
      primary: true,
      cell: (o) => <span className="font-bold">#{o.number}</span>,
    },
    {
      id: "customer",
      header: m.admin.ordersPage.customer,
      icon: <IconUser />,
      primary: true,
      cell: (o) => (
        <span>
          {o.customer_name || "—"}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {o.customer_phone}
          </span>
        </span>
      ),
    },
    {
      id: "merchant",
      header: m.admin.ordersPage.merchant,
      icon: <IconStore />,
      cell: (o) => o.merchant_name,
    },
    {
      id: "total",
      header: m.admin.ordersPage.total,
      icon: <IconWallet />,
      cell: (o) => (
        <span>
          <span className="font-bold text-primary-dark">{fmt.format(o.total)}</span>{" "}
          <span className="text-xs text-ink-muted">{PAYMENT_LABELS[o.payment_method]}</span>
        </span>
      ),
    },
    {
      id: "driver",
      header: m.admin.ordersPage.driver,
      icon: <IconDriver />,
      cell: (o) =>
        o.driver_phone ? (
          <span dir="ltr">{o.driver_phone}</span>
        ) : (
          <span className="text-ink-muted">{m.admin.ordersPage.noDriver}</span>
        ),
    },
    {
      id: "status",
      header: m.admin.ordersPage.statusCol,
      icon: <IconStatus />,
      cell: (o) => (
        <Badge variant={STATUS_VARIANT[o.status] ?? "neutral"}>{STATUS_LABELS[o.status]}</Badge>
      ),
    },
    {
      id: "time",
      header: m.admin.ordersPage.time,
      cell: (o) =>
        new Date(o.created_at).toLocaleTimeString("ar-SY", { hour: "2-digit", minute: "2-digit" }),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconOrder className="text-primary" />
          {m.admin.ordersPage.title}
        </h1>
        <Button onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
          <IconAdd size={16} />
          <IconPhone size={15} />
          {m.admin.ordersPage.newOrder}
        </Button>
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.ordersPage.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.ordersPage.allStatuses}</option>
            {Object.entries(STATUS_LABELS).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </Select>
        </div>
        <label className="flex cursor-pointer items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={openOnly}
            onChange={(e) => {
              setOpenOnly(e.target.checked);
              setPage(1);
            }}
            className="h-4 w-4 accent-primary"
          />
          {m.admin.ordersPage.openOnly}
        </label>
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
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <DataView
        items={data?.orders ?? []}
        getKey={(o) => o.id}
        columns={columns}
        view={view}
        empty={m.admin.ordersPage.empty}
        actions={(o) => (
          <Button variant="secondary" onClick={() => setDetailID(o.id)}>
            {m.admin.ordersPage.details}
          </Button>
        )}
      />

      {data && (
        <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", String(data.total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="secondary" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              {m.admin.users.prev}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant="secondary"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              {m.admin.users.next}
            </Button>
          </div>
        </div>
      )}

      {detailID && (
        <OrderDetailModal orderID={detailID} onClose={() => setDetailID(null)} onChanged={load} />
      )}
      {createOpen && (
        <CreateOrderModal
          onClose={() => setCreateOpen(false)}
          onCreated={(id) => {
            setCreateOpen(false);
            void load();
            setDetailID(id);
          }}
        />
      )}
    </div>
  );
}

// ---------- تفاصيل الطلب ----------

function OrderDetailModal({
  orderID,
  onClose,
  onChanged,
}: {
  orderID: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const [order, setOrder] = useState<OrderRow | null>(null);
  const [drivers, setDrivers] = useState<AuthUser[]>([]);
  const [driverID, setDriverID] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      setOrder(await api<OrderRow>(`/api/v1/admin/orders/${orderID}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [orderID]);

  useEffect(() => {
    void load();
    api<{ users: AuthUser[] }>("/api/v1/admin/users?role=driver&per_page=100")
      .then((p) => setDrivers(p.users.filter((u) => u.status === "active")))
      .catch(() => undefined);
  }, [load]);

  async function transition(to: string) {
    let note = "";
    if (to === "cancelled" || to === "rejected") {
      note = prompt(m.admin.ordersPage.cancelReasonPrompt) ?? "";
      if (!note) return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/orders/${orderID}/transition`, {
        method: "POST",
        body: JSON.stringify({ to, note }),
      });
      await load();
      onChanged();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  async function assign() {
    if (!driverID) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/orders/${orderID}/assign`, {
        method: "POST",
        body: JSON.stringify({ driver_id: driverID }),
      });
      await load();
      onChanged();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  if (!order) {
    return (
      <Modal open onClose={onClose} title={m.admin.ordersPage.orderDetails}>
        <p className="p-4 text-center text-ink-muted">{m.common.loading}</p>
      </Modal>
    );
  }

  const next = OPS_NEXT[order.status] ?? [];
  const canAssign = order.status === "preparing" || order.status === "dispatching";

  return (
    <Modal open onClose={onClose} size="xl" title={`${m.admin.ordersPage.orderDetails} #${order.number}`}>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Badge variant={STATUS_VARIANT[order.status] ?? "neutral"}>
          {STATUS_LABELS[order.status]}
        </Badge>
        <Badge variant="neutral">{PAYMENT_LABELS[order.payment_method]}</Badge>
        {order.promo_code && <Badge variant="warning">{order.promo_code}</Badge>}
        {order.cancel_reason && (
          <span className="text-xs text-danger">({order.cancel_reason})</span>
        )}
      </div>

      <div className="grid gap-5 md:grid-cols-2">
        <div className="space-y-5">
          <FormSection title={m.admin.ordersPage.customer} icon={<IconUser />}>
            <p className="text-sm">
              {order.customer_name || "—"} —{" "}
              <span dir="ltr" className="font-medium">
                {order.customer_phone}
              </span>
            </p>
            <p className="mt-1 flex items-start gap-1.5 text-sm text-ink-muted">
              <IconLocation size={15} className="mt-0.5 shrink-0" />
              {order.address_text}
            </p>
            {order.notes && <p className="mt-1 text-sm text-ink-muted">📝 {order.notes}</p>}
          </FormSection>

          <FormSection title={m.admin.ordersPage.itemsSection} icon={<IconOrder />}>
            <ul className="space-y-2 text-sm">
              {order.items?.map((it) => (
                <li key={it.id} className="rounded-control border border-line p-2.5">
                  <div className="flex justify-between font-medium">
                    <span>
                      {it.name} ×{it.qty}
                    </span>
                    <span>{fmt.format(it.unit_price * it.qty)}</span>
                  </div>
                  {it.options.length > 0 && (
                    <p className="mt-0.5 text-xs text-ink-muted">
                      {it.options.map((op) => `${op.group}: ${op.name}`).join(" · ")}
                    </p>
                  )}
                  {it.note && <p className="mt-0.5 text-xs text-accent-dark">✎ {it.note}</p>}
                </li>
              ))}
            </ul>
          </FormSection>

          <FormSection title={m.admin.ordersPage.financials} icon={<IconWallet />}>
            <dl className="space-y-1 text-sm">
              <div className="flex justify-between">
                <dt className="text-ink-muted">{m.admin.ordersPage.subtotal}</dt>
                <dd>{fmt.format(order.subtotal)}</dd>
              </div>
              {order.discount > 0 && (
                <div className="flex justify-between text-success">
                  <dt>{m.admin.ordersPage.discount}</dt>
                  <dd>-{fmt.format(order.discount)}</dd>
                </div>
              )}
              <div className="flex justify-between">
                <dt className="text-ink-muted">{m.admin.ordersPage.deliveryFee}</dt>
                <dd>{fmt.format(order.delivery_fee)}</dd>
              </div>
              <div className="flex justify-between border-t border-line pt-1 font-bold">
                <dt>{m.admin.ordersPage.total}</dt>
                <dd>
                  {fmt.format(order.total)} {m.common.currency}
                </dd>
              </div>
              {order.wallet_paid > 0 && (
                <div className="flex justify-between text-primary-dark">
                  <dt>{m.admin.ordersPage.walletPaid}</dt>
                  <dd>{fmt.format(order.wallet_paid)}</dd>
                </div>
              )}
              <div className="flex justify-between">
                <dt className="text-ink-muted">{m.admin.ordersPage.cashDue}</dt>
                <dd className="font-medium">{fmt.format(order.cash_due)}</dd>
              </div>
            </dl>
          </FormSection>
        </div>

        <div className="space-y-5">
          <FormSection title={m.admin.ordersPage.statusCol} icon={<IconStatus />}>
            {canAssign && (
              <div className="mb-3 flex items-end gap-2 rounded-control bg-page p-3">
                <div className="flex-1">
                  <Select
                    id="assign-driver"
                    label={m.admin.ordersPage.chooseDriver}
                    value={driverID}
                    onChange={(e) => setDriverID(e.target.value)}
                  >
                    <option value="">—</option>
                    {drivers.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.full_name || d.phone}
                      </option>
                    ))}
                  </Select>
                </div>
                <Button onClick={assign} disabled={busy || !driverID}>
                  {m.admin.ordersPage.assignDriver}
                </Button>
              </div>
            )}
            <div className="flex flex-wrap gap-2">
              {next.map((to) => (
                <Button
                  key={to}
                  variant={
                    to === "cancelled" || to === "rejected" || to === "failed"
                      ? "danger"
                      : to === "delivered"
                        ? "primary"
                        : "secondary"
                  }
                  disabled={busy}
                  onClick={() => transition(to)}
                >
                  {ACTION_LABELS[to]}
                </Button>
              ))}
            </div>
            {error && (
              <p className="mt-3 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">
                {error}
              </p>
            )}
          </FormSection>

          <FormSection title={m.admin.ordersPage.timeline} icon={<IconStatus />}>
            <ol className="space-y-1.5 text-sm">
              {order.events?.map((e, i) => (
                <li key={i} className="flex items-center gap-2">
                  <span className="h-2 w-2 shrink-0 rounded-badge bg-primary" />
                  <span className="font-medium">{STATUS_LABELS[e.to_status] ?? e.to_status}</span>
                  <span className="text-xs text-ink-muted">
                    {new Date(e.created_at).toLocaleTimeString("ar-SY", {
                      hour: "2-digit",
                      minute: "2-digit",
                      second: "2-digit",
                    })}
                  </span>
                  {e.note && <span className="text-xs text-ink-muted">— {e.note}</span>}
                </li>
              ))}
            </ol>
          </FormSection>
        </div>
      </div>
    </Modal>
  );
}

// ---------- الطلب الهاتفي ----------

interface MenuOption {
  id: string;
  name: string;
  price_delta: number;
  available: boolean;
}
interface MenuGroup {
  id: string;
  name: string;
  min_select: number;
  max_select: number;
  options: MenuOption[];
}
interface MenuItemT {
  id: string;
  name: string;
  price: number;
  available: boolean;
  modifiers: MenuGroup[];
}
interface MenuSection {
  id: string;
  name: string;
  items: MenuItemT[];
}
interface MerchantT {
  id: string;
  name: string;
  category_icon: string;
  status: string;
}

interface CartLine {
  item: MenuItemT;
  qty: number;
  note: string;
  optionIDs: string[];
}

function CreateOrderModal({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (orderID: string) => void;
}) {
  const c = m.admin.ordersPage.create;
  const [phone, setPhone] = useState("");
  const [merchants, setMerchants] = useState<MerchantT[]>([]);
  const [merchantID, setMerchantID] = useState("");
  const [menu, setMenu] = useState<MenuSection[]>([]);
  const [cart, setCart] = useState<CartLine[]>([]);
  const [picking, setPicking] = useState<MenuItemT | null>(null);
  const [address, setAddress] = useState("");
  const [lat, setLat] = useState<number | null>(null);
  const [lng, setLng] = useState<number | null>(null);
  const [payment, setPayment] = useState("cash");
  const [promo, setPromo] = useState("");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api<{ merchants: MerchantT[] }>("/api/v1/admin/merchants?status=active&per_page=100")
      .then((p) => setMerchants(p.merchants))
      .catch((err) => setError(errText(err)));
  }, []);

  useEffect(() => {
    setMenu([]);
    setCart([]);
    if (!merchantID) return;
    api<MenuSection[]>(`/api/v1/admin/merchants/${merchantID}/menu`)
      .then(setMenu)
      .catch((err) => setError(errText(err)));
  }, [merchantID]);

  const subtotal = cart.reduce((sum, l) => {
    const deltas = l.item.modifiers
      .flatMap((g) => g.options)
      .filter((o) => l.optionIDs.includes(o.id))
      .reduce((s, o) => s + o.price_delta, 0);
    return sum + (l.item.price + deltas) * l.qty;
  }, 0);

  async function submit() {
    setBusy(true);
    setError("");
    try {
      const o = await api<OrderRow>("/api/v1/admin/orders", {
        method: "POST",
        body: JSON.stringify({
          customer_phone: phone,
          merchant_id: merchantID,
          items: cart.map((l) => ({
            menu_item_id: l.item.id,
            qty: l.qty,
            note: l.note,
            option_ids: l.optionIDs,
          })),
          address_text: address,
          lat,
          lng,
          payment_method: payment,
          promo_code: promo,
          notes,
        }),
      });
      onCreated(o.id);
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  const canSubmit =
    phone && merchantID && cart.length > 0 && address && lat != null && !busy;

  return (
    <Modal open onClose={onClose} size="xl" title={c.title}>
      <div className="grid gap-5 md:grid-cols-2">
        <div className="space-y-4">
          <Input
            id="o-phone"
            label={c.customerPhone}
            icon={<IconPhone />}
            dir="ltr"
            required
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            className="text-end"
            placeholder="09xxxxxxxx"
          />
          <p className="-mt-3 text-xs text-ink-muted">{c.customerHint}</p>
          <Select
            id="o-merchant"
            label={c.merchant}
            value={merchantID}
            onChange={(e) => setMerchantID(e.target.value)}
          >
            <option value="">{c.chooseMerchant}</option>
            {merchants.map((mr) => (
              <option key={mr.id} value={mr.id}>
                {mr.category_icon} {mr.name}
              </option>
            ))}
          </Select>

          {/* قائمة المتجر */}
          {menu.length > 0 && (
            <div className="max-h-64 space-y-3 overflow-y-auto rounded-control border border-line p-3">
              {menu.map((sec) => (
                <div key={sec.id}>
                  <p className="mb-1 text-xs font-bold text-ink-muted">{sec.name}</p>
                  <ul className="space-y-1">
                    {sec.items
                      .filter((it) => it.available)
                      .map((it) => (
                        <li key={it.id} className="flex items-center justify-between text-sm">
                          <span>
                            {it.name}{" "}
                            <span className="text-xs text-ink-muted">{fmt.format(it.price)}</span>
                          </span>
                          <Button
                            variant="ghost"
                            onClick={() =>
                              it.modifiers.length > 0
                                ? setPicking(it)
                                : setCart((cs) => [...cs, { item: it, qty: 1, note: "", optionIDs: [] }])
                            }
                            className="flex items-center gap-1"
                          >
                            <IconAdd size={14} />
                            {c.add}
                          </Button>
                        </li>
                      ))}
                  </ul>
                </div>
              ))}
            </div>
          )}

          {/* السلة */}
          <div>
            <p className="mb-1 text-sm font-bold">
              {c.cart} {cart.length > 0 && `(${fmt.format(subtotal)} ${m.common.currency})`}
            </p>
            {cart.length === 0 ? (
              <p className="rounded-control bg-page p-3 text-center text-xs text-ink-muted">
                {c.cartEmpty}
              </p>
            ) : (
              <ul className="space-y-1.5">
                {cart.map((l, i) => (
                  <li
                    key={i}
                    className="flex items-center justify-between rounded-control border border-line px-2.5 py-1.5 text-sm"
                  >
                    <span>
                      {l.item.name} ×{l.qty}
                      {l.optionIDs.length > 0 && (
                        <span className="text-xs text-ink-muted"> (+{l.optionIDs.length})</span>
                      )}
                    </span>
                    <span className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        onClick={() =>
                          setCart((cs) => cs.map((x, j) => (j === i ? { ...x, qty: x.qty + 1 } : x)))
                        }
                      >
                        +
                      </Button>
                      <span className="w-5 text-center">{l.qty}</span>
                      <Button
                        variant="ghost"
                        onClick={() =>
                          setCart((cs) =>
                            l.qty > 1
                              ? cs.map((x, j) => (j === i ? { ...x, qty: x.qty - 1 } : x))
                              : cs.filter((_, j) => j !== i),
                          )
                        }
                      >
                        −
                      </Button>
                      <Button variant="ghost" onClick={() => setCart((cs) => cs.filter((_, j) => j !== i))}>
                        <IconDelete size={14} className="text-danger" />
                      </Button>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div className="space-y-4">
          <Input
            id="o-address"
            label={c.address}
            icon={<IconLocation />}
            required
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder={m.admin.merchants.addressPlaceholder}
          />
          <div>
            <p className="mb-1 text-sm font-medium">{c.pinHint}</p>
            <div className="overflow-hidden rounded-control border border-line">
              <PickMap
                lat={lat}
                lng={lng}
                onPick={(la, ln) => {
                  setLat(la);
                  setLng(ln);
                }}
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Select
              id="o-payment"
              label={c.payment}
              value={payment}
              onChange={(e) => setPayment(e.target.value)}
            >
              <option value="cash">{PAYMENT_LABELS.cash}</option>
              <option value="wallet">{PAYMENT_LABELS.wallet}</option>
              <option value="mixed">{PAYMENT_LABELS.mixed}</option>
            </Select>
            <Input
              id="o-promo"
              label={c.promo}
              dir="ltr"
              value={promo}
              onChange={(e) => setPromo(e.target.value.toUpperCase())}
              className="font-mono uppercase"
            />
          </div>
          <Input
            id="o-notes"
            label={c.orderNotes}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
          {error && (
            <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
          )}
          <div className="flex justify-end gap-2 border-t border-line pt-4">
            <Button variant="secondary" onClick={onClose}>
              {m.common.cancel}
            </Button>
            <Button onClick={submit} disabled={!canSubmit}>
              {c.submit}
            </Button>
          </div>
        </div>
      </div>

      {picking && (
        <OptionsPicker
          item={picking}
          onClose={() => setPicking(null)}
          onAdd={(optionIDs) => {
            setCart((cs) => [...cs, { item: picking, qty: 1, note: "", optionIDs }]);
            setPicking(null);
          }}
        />
      )}
    </Modal>
  );
}

/** اختيار مُعدِّلات الصنف مع فرض أدنى/أقصى اختيار لكل مجموعة */
function OptionsPicker({
  item,
  onClose,
  onAdd,
}: {
  item: MenuItemT;
  onClose: () => void;
  onAdd: (optionIDs: string[]) => void;
}) {
  const [selected, setSelected] = useState<string[]>([]);

  function toggle(g: MenuGroup, optID: string) {
    setSelected((sel) => {
      const inGroup = g.options.map((o) => o.id);
      const chosen = sel.filter((id) => inGroup.includes(id));
      if (sel.includes(optID)) return sel.filter((id) => id !== optID);
      if (g.max_select === 1) return [...sel.filter((id) => !inGroup.includes(id)), optID];
      if (chosen.length >= g.max_select) return sel;
      return [...sel, optID];
    });
  }

  const valid = item.modifiers.every((g) => {
    const chosen = selected.filter((id) => g.options.some((o) => o.id === id)).length;
    return chosen >= g.min_select && chosen <= g.max_select;
  });

  return (
    <Modal open onClose={onClose} title={item.name}>
      <div className="space-y-4">
        {item.modifiers.map((g) => (
          <div key={g.id}>
            <p className="mb-1.5 flex items-center gap-2 text-sm font-medium">
              {g.name}
              <Badge variant={g.min_select > 0 ? "warning" : "neutral"}>
                {g.min_select > 0 ? m.admin.menu.required : m.admin.menu.optional}
              </Badge>
            </p>
            <div className="flex flex-wrap gap-2">
              {g.options
                .filter((o) => o.available)
                .map((o) => (
                  <button
                    key={o.id}
                    type="button"
                    onClick={() => toggle(g, o.id)}
                    className={`rounded-badge border px-3 py-1.5 text-sm transition-colors ${
                      selected.includes(o.id)
                        ? "border-primary bg-primary-light text-primary-dark"
                        : "border-line text-ink-muted hover:border-primary/50"
                    }`}
                  >
                    {o.name}
                    {o.price_delta !== 0 && (
                      <span className="text-xs"> ({o.price_delta > 0 ? "+" : ""}{fmt.format(o.price_delta)})</span>
                    )}
                  </button>
                ))}
            </div>
          </div>
        ))}
        <div className="flex justify-end gap-2 border-t border-line pt-3">
          <Button variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button disabled={!valid} onClick={() => onAdd(selected)}>
            {m.admin.ordersPage.create.add}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
