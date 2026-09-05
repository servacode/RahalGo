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
  const [selected, setSelected] = useState<Driver | null>(null);

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
    return out;
  }, [drivers.data, visible]);

  const onFeature = useCallback(
    (layerID: string, props: Record<string, unknown>) => {
      if (layerID === "drivers") {
        const d = drivers.data?.drivers.find((x) => x.id === props.id);
        if (d) setSelected(d);
      }
    },
    [drivers.data],
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
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="layer-drivers"
                    label={T.layer.drivers}
                    checked={!!visible.drivers}
                    onChange={() => toggle("drivers")}
                  />
                  <span className="ms-auto text-xs text-[var(--muted)]">
                    {drivers.data?.count ?? 0}
                  </span>
                </div>
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
              <p className="mt-3 text-xs leading-5 text-[var(--muted)]">
                {T.freshness.explain}
              </p>
            </Card>
          )}

          {/* **ومن لا موضعَ له يُقال ولا يُرسَم** — وهو أهمُّ ما تقوله
              الخريطة: تطبيقٌ أوقفه النظامُ وورديّةٌ مفتوحة. */}
          {noLoc.length > 0 && (
            <Card>
              <h3 className="mb-1 text-sm font-bold">{T.freshness.NO_LOCATION}</h3>
              <p className="mb-2 text-xs text-[var(--muted)]">{T.driver.noLocationHint}</p>
              <ul className="flex flex-col gap-1 text-sm">
                {noLoc.slice(0, 12).map((d) => (
                  <li key={d.id} className="flex items-center gap-2">
                    <span
                      className="inline-block h-2 w-2 rounded-full"
                      style={{ background: themeColor("disabled") }}
                    />
                    <span>{d.name}</span>
                    {d.on_shift && (
                      <span className="ms-auto text-xs text-[var(--muted)]">
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
                         border-s border-[var(--border)] bg-[var(--surface)] p-4 shadow-xl"
            >
              <div className="mb-3 flex items-center gap-2">
                <h3 className="text-base font-bold">{selected.name}</h3>
                <button
                  className="ms-auto text-sm text-[var(--muted)]"
                  onClick={() => setSelected(null)}
                >
                  {T.close}
                </button>
              </div>
              <dl className="flex flex-col gap-2 text-sm">
                <Row k={T.driver.status} v={selected.status} />
                <Row
                  k={T.driver.onShift}
                  v={selected.on_shift ? T.yes : T.no}
                />
                <Row k={T.driver.activeOrders} v={String(selected.active_orders)} />
                <Row
                  k={T.driver.lastSeen}
                  v={
                    selected.last_location_at
                      ? fmtDateTime(selected.last_location_at)
                      : "—"
                  }
                />
                <Row k={T.live} v={T.freshness[selected.freshness]} />
                {/* **والمالُ لمن يملك صلاحيّتَه وحدَه** (البند ٦). */}
                {selected.cash_held !== undefined && (
                  <Row k={T.driver.cashHeld} v={fmtMoney(selected.cash_held)} />
                )}
              </dl>
              <div className="mt-4 flex flex-wrap gap-2">
                <Link href={`/dashboard/users?id=${selected.id}`}>
                  <Button variant="secondary">{T.driver.openProfile}</Button>
                </Link>
                {selected.current_order && (
                  <Link href={`/dashboard/orders?id=${selected.current_order}`}>
                    <Button variant="secondary">{T.order.openPage}</Button>
                  </Link>
                )}
              </div>
            </aside>
          )}
        </div>
      </div>
    </PageContainer>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <dt className="text-[var(--muted)]">{k}</dt>
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
