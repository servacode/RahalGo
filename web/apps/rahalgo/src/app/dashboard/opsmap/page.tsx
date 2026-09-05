"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خريطةُ العمليات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أين الناسُ والطلباتُ والتغطيةُ الآن** — في شاشةٍ واحدة.
 *
 * # ولماذا شاشةٌ وقد صارت الجداولُ كلُّها موجودة
 *
 * **الجدولُ يجيب «كم»، والخريطةُ تجيب «أين»** — **وسؤالُ «أين» لا
 * يُجاب بجدول.** «سائقٌ على الدوام في الطرف الآخر من المدينة وطلبٌ
 * ينتظر هنا» **حقيقةٌ في القاعدة منذ شهور ولا شاشةَ تقولها.**
 *
 * # واللوحةُ الجانبيّةُ لا نافذةٌ منبثقة (البند ٣٧)
 *
 * **المنبثقةُ فوق الخريطة تحجب ما جاء ينظر إليه** — **واللوحةُ تُزيح
 * ولا تحجب**، ولا تصنع تمريراً أفقيّاً.
 *
 * # ولا تُعدَّل الأشياءُ كلُّها من هنا (البند ١٠)
 *
 * **الخريطةُ تقول وتوصّل** — ومن أراد أن يعدّل طلباً فتح صفحتَه.
 */

import { useCallback, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { getMessages, defaultLocale, fmtDateTime, fmtMoney } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  Card,
  Button,
  EmptyState,
  LoadingState,
  Alert,
  Checkbox,
  Input,
  Select,
  useLiveData,
  themeColor,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((m) => m.OpsMapCanvas),
  { ssr: false },
);

const m = getMessages(defaultLocale);
const T = m.admin.opsMap;

// ══════════════════════════════════════════════════════════════════════
// **الأنواع**
// ══════════════════════════════════════════════════════════════════════

interface Meta {
  permissions: string[];
  location_ping_sec: number;
  assignable_sec: number;
}

interface Driver {
  id: string;
  name: string;
  lat: number;
  lng: number;
  status: string;
  on_shift: boolean;
  active_orders: number;
  current_order?: string;
  last_location_at: string;
  age_sec: number;
  freshness: "LIVE" | "FRESH" | "STALE" | "NO_LOCATION";
  cash_held?: number;
}

interface Merchant {
  id: string;
  name: string;
  lat: number;
  lng: number;
  status: string;
  open_now: boolean;
  city?: string;
  area?: string;
  active_orders: number;
  rep?: string;
}

interface OrderPin {
  id: string;
  number: number;
  status: string;
  kind: string;
  drop_lat: number;
  drop_lng: number;
  address?: string;
  merchant_id?: string;
  merchant?: string;
  pick_lat?: number;
  pick_lng?: number;
  driver_id?: string;
  driver?: string;
  driver_lat?: number;
  driver_lng?: number;
  created_at: string;
  accepted_at?: string;
  picked_up_at?: string;
  total?: number;
  payment_method?: string;
  delivery_fee?: number;
}

/** **ما هو المُحدَّد؟** — واللوحةُ الجانبيّةُ واحدةٌ لكلّ الطبقات. */
type Picked =
  | { kind: "driver"; v: Driver }
  | { kind: "merchant"; v: Merchant }
  | { kind: "order"; v: OrderPin };

/**
 * **ألوانُ الطزاجة — من الثيم لا من هنا.**
 *
 * **ولا لوحةَ ثانيةٌ تُولد** (وهي عينُ ما حُذف من `tokens.ts`): يُبدَّل
 * اللونُ في `theme.css` **فتتبعه الخريطةُ في التحديث نفسِه.**
 *
 * **والصفحةُ تُحمَّل `ssr: false`** — فالمتصفّحُ موجودٌ حين تُنادى.
 */
function freshColors(): Record<string, string> {
  return {
    LIVE: themeColor("success"),
    FRESH: themeColor("info"),
    STALE: themeColor("danger-solid"),
    NO_LOCATION: themeColor("disabled"),
  };
}

function fc(features: FeatureCollection["features"]): FeatureCollection {
  return { type: "FeatureCollection", features };
}

/** **معلَمُ نقطةٍ** — والخصائصُ نصوصٌ وأرقامٌ لا كائنات (MapLibre لا يمرّرها). */
function pt(lng: number, lat: number, props: Record<string, unknown>) {
  return {
    type: "Feature" as const,
    geometry: { type: "Point", coordinates: [lng, lat] },
    properties: props,
  };
}

export default function OpsMapPage() {
  // ── حالُ الواجهة (البند ٣٨) ──────────────────────────────────
  //
  // **ويُحفَظ ما يختاره المستخدم** — الطبقاتُ والمرشِّحات.
  // **ولا يُحفَظ ما يشيخ**: البياناتُ تُجلَب دائماً، **فلا تُعرض قديمةٌ
  // على أنّها حيّة.**
  const [visible, setVisible] = useState<Record<string, boolean>>(() => {
    if (typeof window === "undefined") return { drivers: true };
    try {
      const raw = window.localStorage.getItem("opsmap.layers");
      if (raw) return JSON.parse(raw) as Record<string, boolean>;
    } catch {
      /* تخزينٌ ممنوعٌ في نافذةٍ خاصّة — والافتراضُ يكفي */
    }
    return { drivers: true };
  });
  const toggle = useCallback((id: string) => {
    setVisible((prev) => {
      const next = { ...prev, [id]: !prev[id] };
      try {
        window.localStorage.setItem("opsmap.layers", JSON.stringify(next));
      } catch {
        /* لا يضرّ */
      }
      return next;
    });
  }, []);

  const [onShift, setOnShift] = useState<"" | "true" | "false">("");
  const [stale, setStale] = useState<"" | "true" | "false">("");
  const [hasActive, setHasActive] = useState<"" | "true" | "false">("");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Picked | null>(null);

  // ── ما يملكه من يقف أمامها ───────────────────────────────────
  const meta = useLiveData<Meta>(() => api<Meta>("/admin/ops-map/meta"), []);
  const can = useCallback(
    (p: string) => !!meta.data?.permissions.includes(p),
    [meta.data],
  );

  // ── السائقون ─────────────────────────────────────────────────
  //
  // # ولماذا مدّةٌ لا بثّ (البندان ٨ و٤٤)
  //
  // **كاتبُ الموضع لا يبثّ** — ولا يُمسّ (البند ٤٥). **والمدّةُ ليست
  // اعتباطاً**: هي نبضةُ السائق نفسُها من الإعدادات، **فلا تُسأل
  // القاعدةُ عمّا لم يتغيّر بعد.**
  //
  // **والطلباتُ تُحدَّث ببثّ `ops` القائم** — لا بمدّة.
  const driverQuery = useMemo(() => {
    const q = new URLSearchParams();
    if (onShift) q.set("on_shift", onShift);
    if (stale) q.set("stale", stale);
    if (hasActive) q.set("has_active", hasActive);
    if (search.trim()) q.set("q", search.trim());
    const s = q.toString();
    return s ? `?${s}` : "";
  }, [onShift, stale, hasActive, search]);

  const drivers = useLiveData<{ drivers: Driver[]; count: number }>(
    () =>
      can("VIEW_DRIVER_LOCATIONS") && visible.drivers
        ? api<{ drivers: Driver[]; count: number }>(`/admin/ops-map/drivers${driverQuery}`)
        : Promise.resolve({ drivers: [], count: 0 }),
    [],
    [driverQuery, visible.drivers, meta.data],
  );

  // ── المتاجر ──────────────────────────────────────────────────
  const merchants = useLiveData<{ merchants: Merchant[]; count: number }>(
    () =>
      can("VIEW_MERCHANT_LOCATIONS") && visible.merchants
        ? api<{ merchants: Merchant[]; count: number }>("/admin/ops-map/merchants")
        : Promise.resolve({ merchants: [], count: 0 }),
    // **والمتجرُ يتبدّل بكتابةِ اللوحة** — `catalog` إشارتُها القائمة.
    ["catalog"],
    [visible.merchants, meta.data],
  );

  // ── الطلباتُ النشطة ─────────────────────────────────────────
  //
  // **وهذه وحدَها تُحدَّث ببثٍّ حقيقيّ** (البند ٤٤): غرفةُ `ops` تبثّ
  // `{"type":"order"}` عند كلّ تبدّل — **ولا مدّةَ ولا استطلاع.**
  const orders = useLiveData<{ orders: OrderPin[]; count: number }>(
    () =>
      can("VIEW_ACTIVE_ORDERS") && visible.orders
        ? api<{ orders: OrderPin[]; count: number }>("/admin/ops-map/orders")
        : Promise.resolve({ orders: [], count: 0 }),
    ["order"],
    [visible.orders, meta.data],
  );

  const layers = useMemo<LayerSpec[]>(() => {
    const out: LayerSpec[] = [];
    const C = freshColors();
    const list = drivers.data?.drivers ?? [];
    out.push({
      id: "drivers",
      kind: "point",
      cluster: true,
      order: 50,
      visible: !!visible.drivers,
      color: [
        "match",
        ["get", "freshness"],
        "LIVE", C.LIVE,
        "FRESH", C.FRESH,
        "STALE", C.STALE,
        C.NO_LOCATION,
      ],
      // **ومن لا موضعَ له لا يُرسَم** (البند ٤) — ولا يُخترَع له دبّوس.
      data: fc(
        list
          .filter((d) => d.freshness !== "NO_LOCATION")
          .map((d) =>
            pt(d.lng, d.lat, {
              id: d.id,
              name: d.name,
              freshness: d.freshness,
            }),
          ),
      ),
    });

    // ── المتاجر ──────────────────────────────────────────────
    //
    // **والمغلقُ يُرى مغلقاً** — لونٌ باهتٌ لا اختفاء: «متجرٌ مغلقٌ
    // وطلبٌ ينتظره» حقيقةٌ يريد المكتبُ أن يراها.
    out.push({
      id: "merchants",
      kind: "point",
      cluster: true,
      order: 40,
      visible: !!visible.merchants,
      color: ["case", ["get", "open"], themeColor("accent"), themeColor("disabled")],
      data: fc(
        (merchants.data?.merchants ?? []).map((x) =>
          pt(x.lng, x.lat, { id: x.id, name: x.name, open: x.open_now }),
        ),
      ),
    });

    // ── الطلباتُ النشطة ──────────────────────────────────────
    //
    // **وتُميَّز بصريّاً بحالها** (البند ١٠): ما لا سائقَ له أحمرُ —
    // **وهو ما يحتاج يداً الآن.**
    const orderList = orders.data?.orders ?? [];
    out.push({
      id: "orders",
      kind: "point",
      cluster: true,
      order: 60,
      visible: !!visible.orders,
      color: [
        "match",
        ["get", "state"],
        "unassigned", themeColor("danger-solid"),
        "picked", themeColor("success"),
        "accepted", themeColor("warning"),
        themeColor("violet"),
      ],
      data: fc(
        orderList.map((o) =>
          pt(o.drop_lng, o.drop_lat, {
            id: o.id,
            number: o.number,
            state: !o.driver_id
              ? "unassigned"
              : o.picked_up_at
                ? "picked"
                : o.accepted_at
                  ? "accepted"
                  : "other",
          }),
        ),
      ),
    });

    // ── الربطُ الجغرافيُّ للطلب المُحدَّد (البند ١١) ─────────
    //
    // **وخطٌّ مستقيمٌ لا مسار** — **الخريطةُ ليست محرّكَ ملاحة**، وخطٌّ
    // يقول «من أين إلى أين ومن يحمله» يكفي المكتب.
    const link: FeatureCollection["features"] = [];
    if (selected?.kind === "order") {
      const o = selected.v;
      const line = (a: [number, number], b: [number, number]) => ({
        type: "Feature" as const,
        geometry: { type: "LineString", coordinates: [a, b] },
        properties: {},
      });
      if (o.pick_lng != null && o.pick_lat != null) {
        link.push(line([o.pick_lng, o.pick_lat], [o.drop_lng, o.drop_lat]));
        if (o.driver_lng != null && o.driver_lat != null) {
          link.push(line([o.driver_lng, o.driver_lat], [o.pick_lng, o.pick_lat]));
        }
      }
    }
    out.push({
      id: "order-link",
      kind: "line",
      order: 70,
      visible: link.length > 0,
      color: themeColor("primary"),
      data: fc(link),
    });

    return out;
  }, [drivers.data, merchants.data, orders.data, visible, selected]);

  const onFeature = useCallback(
    (layerID: string, props: Record<string, unknown>) => {
      if (layerID === "drivers") {
        const d = drivers.data?.drivers.find((x) => x.id === props.id);
        if (d) setSelected({ kind: "driver", v: d });
      }
      if (layerID === "merchants") {
        const x = merchants.data?.merchants.find((y) => y.id === props.id);
        if (x) setSelected({ kind: "merchant", v: x });
      }
      if (layerID === "orders") {
        const o = orders.data?.orders.find((y) => y.id === props.id);
        if (o) setSelected({ kind: "order", v: o });
      }
    },
    [drivers.data, merchants.data, orders.data],
  );

  // ── الحالاتُ الحدّيّة (البند ٣٩) ─────────────────────────────
  if (meta.loading) return <LoadingState />;
  if (meta.error) {
    return (
      <PageContainer>
        <Alert tone="error">{T.networkError}</Alert>
        <Button onClick={meta.reload}>{T.retry}</Button>
      </PageContainer>
    );
  }
  if (!can("VIEW_OPERATIONS_MAP")) {
    return (
      <PageContainer>
        <EmptyState title={T.noPermission} />
      </PageContainer>
    );
  }

  const noLoc = (drivers.data?.drivers ?? []).filter((d) => d.freshness === "NO_LOCATION");

  return (
    <PageContainer>
      <PageHeader title={T.title} subtitle={T.subtitle} />

      <div className="grid gap-4 lg:grid-cols-[18rem_1fr]">
        {/* ── الطبقاتُ والمرشِّحات ────────────────────────────── */}
        <div className="flex flex-col gap-4">
          <Card>
            <h3 className="mb-2 text-sm font-bold">{T.layers}</h3>
            <div className="flex flex-col gap-1">
              {can("VIEW_DRIVER_LOCATIONS") && (
                <LayerToggle
                  id="drivers"
                  label={T.layer.drivers}
                  on={!!visible.drivers}
                  count={drivers.data?.count}
                  onToggle={toggle}
                />
              )}
              {can("VIEW_MERCHANT_LOCATIONS") && (
                <LayerToggle
                  id="merchants"
                  label={T.layer.merchants}
                  on={!!visible.merchants}
                  count={merchants.data?.count}
                  onToggle={toggle}
                />
              )}
              {can("VIEW_ACTIVE_ORDERS") && (
                <LayerToggle
                  id="orders"
                  label={T.layer.orders}
                  on={!!visible.orders}
                  count={orders.data?.count}
                  onToggle={toggle}
                />
              )}
            </div>
          </Card>

          {can("VIEW_DRIVER_LOCATIONS") && visible.drivers && (
            <Card>
              <h3 className="mb-2 text-sm font-bold">{T.filters}</h3>
              <div className="flex flex-col gap-2 text-sm">
                <Input
                  placeholder={T.search}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
                <Tri label={T.filter.onShift} value={onShift} onChange={setOnShift} />
                <Tri label={T.filter.hasActive} value={hasActive} onChange={setHasActive} />
                <Tri label={T.filter.stale} value={stale} onChange={setStale} />
              </div>
              <p className="mt-3 text-xs leading-5 text-ink-muted">
                {T.freshness.explain}
              </p>
            </Card>
          )}

          {/* **ومن لا موضعَ له يُقال ولا يُرسَم** — وهو أهمُّ ما تقوله
              الخريطة: تطبيقٌ أوقفه النظامُ وورديّةٌ مفتوحة. */}
          {noLoc.length > 0 && (
            <Card>
              <h3 className="mb-1 text-sm font-bold">{T.freshness.NO_LOCATION}</h3>
              <p className="mb-2 text-xs text-ink-muted">{T.driver.noLocationHint}</p>
              <ul className="flex flex-col gap-1 text-sm">
                {noLoc.slice(0, 12).map((d) => (
                  <li key={d.id} className="flex items-center gap-2">
                    <span
                      className="inline-block h-2 w-2 rounded-full"
                      style={{ background: themeColor("disabled") }}
                    />
                    <span>{d.name}</span>
                    {d.on_shift && (
                      <span className="ms-auto text-xs text-ink-muted">
                        {T.driver.onShift}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </div>

        {/* ── الخريطةُ واللوحةُ الجانبيّة ─────────────────────── */}
        <div className="relative min-h-[28rem] lg:min-h-[calc(100vh-14rem)]">
          <OpsMapCanvas
            layers={layers}
            onFeatureClick={onFeature}
            unavailableLabel={T.unavailable}
          />
          {selected && (
            <aside
              className="absolute inset-y-0 end-0 z-10 w-full max-w-sm overflow-y-auto
                         border-s border-line-soft bg-surface p-4 shadow-xl"
            >
              <div className="mb-3 flex items-center gap-2">
                <h3 className="text-base font-bold">
                  {selected.kind === "driver" && selected.v.name}
                  {selected.kind === "merchant" && selected.v.name}
                  {selected.kind === "order" && `#${selected.v.number}`}
                </h3>
                <button
                  className="ms-auto text-sm text-ink-muted"
                  onClick={() => setSelected(null)}
                >
                  {T.close}
                </button>
              </div>

              {selected.kind === "driver" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.driver.status} v={selected.v.status} />
                    <Row k={T.driver.onShift} v={selected.v.on_shift ? T.yes : T.no} />
                    <Row k={T.driver.activeOrders} v={String(selected.v.active_orders)} />
                    <Row
                      k={T.driver.lastSeen}
                      v={selected.v.last_location_at ? fmtDateTime(selected.v.last_location_at) : "—"}
                    />
                    <Row k={T.live} v={T.freshness[selected.v.freshness]} />
                    {/* **والمالُ لمن يملك صلاحيّتَه وحدَه** (البند ٦). */}
                    {selected.v.cash_held !== undefined && (
                      <Row k={T.driver.cashHeld} v={fmtMoney(selected.v.cash_held)} />
                    )}
                  </dl>
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Link href={`/dashboard/users?id=${selected.v.id}`}>
                      <Button variant="secondary">{T.driver.openProfile}</Button>
                    </Link>
                    {selected.v.current_order && (
                      <Link href={`/dashboard/orders?id=${selected.v.current_order}`}>
                        <Button variant="secondary">{T.order.openPage}</Button>
                      </Link>
                    )}
                  </div>
                </>
              )}

              {selected.kind === "merchant" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.merchant.status} v={selected.v.status} />
                    <Row
                      k={T.merchant.open}
                      v={selected.v.open_now ? T.merchant.open : T.merchant.closed}
                    />
                    {selected.v.city && <Row k={T.merchant.city} v={selected.v.city} />}
                    <Row k={T.merchant.activeOrders} v={String(selected.v.active_orders)} />
                    {/* **والمندوبُ لمن يراقب نشاطَهم وحدَه** (البند ٩). */}
                    {selected.v.rep && <Row k={T.merchant.rep} v={selected.v.rep} />}
                  </dl>
                  <div className="mt-4">
                    <Link href={`/dashboard/sections?merchant=${selected.v.id}`}>
                      <Button variant="secondary">{T.merchant.openPage}</Button>
                    </Link>
                  </div>
                </>
              )}

              {selected.kind === "order" && (
                <>
                  <dl className="flex flex-col gap-2 text-sm">
                    <Row k={T.order.state} v={selected.v.status} />
                    <Row k={T.order.merchant} v={selected.v.merchant ?? "—"} />
                    <Row k={T.order.driver} v={selected.v.driver ?? T.order.noDriver} />
                    {selected.v.address && <Row k={T.order.area} v={selected.v.address} />}
                    <Row k={T.order.createdAt} v={fmtDateTime(selected.v.created_at)} />
                    {/* **والمالُ بصلاحيّته** (البند ١٠). */}
                    {selected.v.total !== undefined && (
                      <Row k={T.order.total} v={fmtMoney(selected.v.total)} />
                    )}
                    {selected.v.delivery_fee !== undefined && (
                      <Row k={T.order.delivery} v={fmtMoney(selected.v.delivery_fee)} />
                    )}
                    {selected.v.payment_method && (
                      <Row k={T.order.payment} v={selected.v.payment_method} />
                    )}
                  </dl>
                  {/* **ولا يُعدّل الطلبُ من الخريطة** (البند ١٠) —
                      تُوصّل إلى صفحته. */}
                  <div className="mt-4 flex flex-wrap gap-2">
                    <Link href={`/dashboard/orders?id=${selected.v.id}`}>
                      <Button variant="secondary">{T.order.openPage}</Button>
                    </Link>
                    {selected.v.driver_id && (
                      <Link href={`/dashboard/users?id=${selected.v.driver_id}`}>
                        <Button variant="secondary">{T.driver.openProfile}</Button>
                      </Link>
                    )}
                  </div>
                </>
              )}
            </aside>
          )}
        </div>
      </div>
    </PageContainer>
  );
}

/** **مفتاحُ طبقةٍ وعدّادُها** — والعددُ يقول «هل ثمّة شيءٌ أصلاً». */
function LayerToggle({
  id,
  label,
  on,
  count,
  onToggle,
}: {
  id: string;
  label: string;
  on: boolean;
  count?: number;
  onToggle: (id: string) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <Checkbox id={`layer-${id}`} label={label} checked={on} onChange={() => onToggle(id)} />
      <span className="ms-auto text-xs text-ink-muted">{count ?? 0}</span>
    </div>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <dt className="text-ink-muted">{k}</dt>
      <dd className="ms-auto font-medium">{v}</dd>
    </div>
  );
}

/** **مرشِّحٌ ثلاثيّ** — والفراغُ «الكلّ» لا «لا». */
function Tri({
  label,
  value,
  onChange,
}: {
  label: string;
  value: "" | "true" | "false";
  onChange: (v: "" | "true" | "false") => void;
}) {
  return (
    <Select
      label={label}
      value={value}
      onChange={(e) => onChange(e.target.value as "" | "true" | "false")}
    >
      <option value="">{T.all}</option>
      <option value="true">{T.yes}</option>
      <option value="false">{T.no}</option>
    </Select>
  );
}
