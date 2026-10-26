"use client";

/**
 * **كلُّ تقييمٍ بسطره** (المالك ٢٠٢٦-١٠-٢٥: «لازم أعرف كل زبون قيّم شو للسائق والمتجر») —
 * الزبونُ ونجومُه للطلب وللسائق، والمتجرُ وتقييمُ السائق له (السرعة والتعامل). الأحدثُ
 * أوّلاً، و«المنخفضُ وحدَه» بحبّة. من `GET /admin/ratings/list`.
 *
 * **تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦)** — الفلاترُ كلُّها في بطاقةٍ واحدة، والنجومُ مرسومة،
 * والتقييمُ الضعيفُ بإطارٍ أحمر.
 */

import { useCallback, useEffect, useState, type ComponentType } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtRef, fmtDateTime, errorText, storeLabel } from "@rahalgo/i18n";
import {
  Badge,
  Chips,
  LoadingState,
  Pagination,
  ReloadState,
  SearchSelect,
  Stars,
  IconStar,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const L = m.admin.ratingsPage.list;

interface Row {
  order_id: string;
  order_number: number;
  created_at: string;
  customer_id: string;
  customer_name: string;
  platform_stars: number | null;
  driver_id: string;
  driver_name: string;
  driver_stars: number | null;
  merchant_id: string;
  merchant_name: string;
  store_speed: number | null;
  store_conduct: number | null;
}

/** **لونُ الرقم** — المنخفضُ أحمر، والمتوسّطُ أصفر، والجيّدُ أخضر. */
export function starTone(n: number): string {
  return n < 3 ? "text-danger" : n < 4 ? "text-warning" : "text-success";
}

/** **نجومٌ مرسومةٌ ورقمُها بلونه** — والغائبُ شرطة. */
export function StarRow({ n }: { n: number | null }) {
  if (n == null) return <span className="text-ink-muted">—</span>;
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5">
      <Stars value={n} size="sm" />
      <span dir="ltr" className={`text-sm font-bold tabular-nums ${starTone(n)}`}>
        {Number.isInteger(n) ? n : n.toFixed(1)}
      </span>
    </span>
  );
}

/** **حالٌ فارغٌ** — دائرةٌ وعنوانٌ وجملةٌ تساعد (الشكلُ نفسُه في الخزينة). */
export function EmptyCircle({
  icon: Icon,
  title,
  hint,
}: {
  icon: ComponentType<{ size?: number }>;
  title: string;
  hint: string;
}) {
  return (
    <section className="surface flex flex-col items-center gap-3 p-8 text-center">
      <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
        <Icon size={28} />
      </span>
      <h3 className="heading-section">{title}</h3>
      <p className="max-w-md text-sm text-ink-muted">{hint}</p>
    </section>
  );
}

const isLow = (n: number | null) => n != null && n <= 2;

export function RatingsList({
  canProfile,
  low,
  onLow,
}: {
  canProfile: boolean;
  /** **«الضعيفةُ بس» من فوق** — بطاقةُ «نجمتان فأقل» تضغطها أيضاً. */
  low: boolean;
  onLow: (v: boolean) => void;
}) {
  const [page, setPage] = useState(1);
  // **فلترٌ بالنوع وبالطرف** (المالك ٢٠٢٦-١٠-٢٥: «لازم نقدر نحدّد المنصّة، المتجر، السائق، أيّ سائق»).
  const [kind, setKind] = useState<"" | "platform" | "driver" | "store">("");
  const [driverId, setDriverId] = useState("");
  const [merchantId, setMerchantId] = useState("");
  const [drivers, setDrivers] = useState<{ id: string; full_name: string; phone: string }[]>([]);
  const [stores, setStores] = useState<{ id: string; name: string; number: number }[]>([]);
  useEffect(() => {
    // **والقائمتان لمن يقرؤهما** — وتعذّرُهما يُخفي المرشِّحَ ولا يوقف الشاشة.
    api<{ drivers: { id: string; full_name: string; phone: string }[] } | { id: string; full_name: string; phone: string }[]>(
      "/api/v1/admin/drivers",
    )
      .then((r) => setDrivers(Array.isArray(r) ? r : r.drivers))
      // @empty-ok — **مرشِّحُ السائق يختفي لمن لا يقرأ السائقين**، والقائمةُ تعمل بلا فلتر.
      .catch(() => setDrivers([]));
    void (async () => {
      try {
        const all: { id: string; name: string; number: number }[] = [];
        for (let p = 1; p <= 20; p++) {
          const r = await api<{ merchants: { id: string; name: string; number: number }[]; total: number }>(
            `/api/v1/admin/merchants?per_page=100&page=${p}`,
          );
          all.push(...(r.merchants ?? []));
          if (all.length >= r.total || (r.merchants ?? []).length === 0) break;
        }
        setStores(all);
      } catch {
        // @empty-ok — **مرشِّحُ المتجر يختفي لمن لا يقرأ المتاجر**، والقائمةُ تعمل بلا فلتر.
        setStores([]);
      }
    })();
  }, []);
  const [data, setData] = useState<{ ratings: Row[]; total: number; per_page: number } | null>(null);
  const [error, setError] = useState("");

  // **وتغييرُ «الضعيفة بس» من فوق يرجع للصفحة الأولى** — كما لو ضُغطت الحبّة.
  useEffect(() => {
    setPage(1);
  }, [low]);

  const load = useCallback(async () => {
    try {
      const qs = new URLSearchParams({ page: String(page) });
      if (low) qs.set("low", "1");
      if (kind) qs.set("kind", kind);
      if (driverId) qs.set("driver_id", driverId);
      if (merchantId) qs.set("merchant_id", merchantId);
      setData(await api(`/api/v1/admin/ratings/list?${qs}`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [page, low, kind, driverId, merchantId]);

  useEffect(() => {
    void load();
  }, [load]);

  const person = (id: string, name: string) =>
    canProfile && id ? (
      <Link href={`/dashboard/users/${id}`} className="font-medium hover:underline">
        {name || "—"}
      </Link>
    ) : (
      <span className="font-medium">{name || "—"}</span>
    );

  return (
    <section className="space-y-3">
      <h2 className="heading-section">{L.title}</h2>

      {/* **الفلاترُ كلُّها في بطاقةٍ واحدة** — العرض، والنوع، ومين. */}
      <div className="surface relative z-30 space-y-3 p-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="w-20 shrink-0 text-xs text-ink-muted">{L.filterShow}</span>
          <Chips
            items={[
              { id: "all", label: L.all },
              { id: "low", label: L.lowOnly },
            ]}
            value={low ? "low" : "all"}
            onChange={(k) => {
              onLow(k === "low");
              setPage(1);
            }}
            wrap
          />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="w-20 shrink-0 text-xs text-ink-muted">{L.filterKind}</span>
          <Chips
            items={[
              { id: "", label: L.kindAll },
              { id: "platform", label: L.kindPlatform },
              { id: "driver", label: L.kindDriver },
              { id: "store", label: L.kindStore },
            ]}
            value={kind}
            onChange={(k) => {
              setKind(k as typeof kind);
              setPage(1);
            }}
            wrap
          />
        </div>
        {(drivers.length > 0 || stores.length > 0) && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="w-20 shrink-0 text-xs text-ink-muted">{L.filterWho}</span>
            <div className="grid min-w-0 flex-1 grid-cols-1 gap-2 sm:grid-cols-2">
              {drivers.length > 0 && (
                <SearchSelect
                  ariaLabel={L.anyDriver}
                  allLabel={L.anyDriver}
                  value={driverId}
                  options={drivers.map((d) => ({ value: d.id, label: d.full_name || d.phone }))}
                  onChange={(v) => {
                    setDriverId(v);
                    setPage(1);
                  }}
                />
              )}
              {stores.length > 0 && (
                <SearchSelect
                  ariaLabel={L.anyStore}
                  allLabel={L.anyStore}
                  value={merchantId}
                  options={stores.map((x) => ({ value: x.id, label: storeLabel(x.name, x.number) }))}
                  onChange={(v) => {
                    setMerchantId(v);
                    setPage(1);
                  }}
                />
              )}
            </div>
          </div>
        )}
      </div>

      {error && !data ? (
        <ReloadState label={error} onRetry={() => void load()} />
      ) : !data ? (
        <LoadingState variant="text" />
      ) : data.ratings.length === 0 ? (
        <EmptyCircle icon={IconStar} title={L.empty} hint={L.emptyHint} />
      ) : (
        // **بطاقاتٌ لا جدول** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — بطاقةٌ لكلّ طلبٍ قُيّم، والضعيفُ بإطارٍ أحمر.
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {data.ratings.map((r) => {
            const weak =
              isLow(r.platform_stars) || isLow(r.driver_stars) || isLow(r.store_speed) || isLow(r.store_conduct);
            return (
              <li
                key={r.order_id}
                className={`surface flex flex-col gap-3 p-4 text-sm ${weak ? "border-danger" : ""}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <span className="min-w-0">
                    <Link
                      href={`/dashboard/history?q=${r.order_number}`}
                      className="block font-bold text-primary hover:underline"
                      dir="ltr"
                    >
                      #{fmtRef(r.order_number)}
                    </Link>
                    <span className="block text-2xs text-ink-muted" dir="ltr">
                      {fmtDateTime(r.created_at)}
                    </span>
                  </span>
                  {weak && <Badge variant="danger">{L.lowCard}</Badge>}
                </div>

                <div className="rounded-control bg-field px-3 py-2">
                  <p className="text-2xs text-ink-muted">
                    {L.customer} · {L.orderStars}
                  </p>
                  <div className="mt-1 flex flex-wrap items-center justify-between gap-2">
                    <span className="min-w-0 truncate">{person(r.customer_id, r.customer_name)}</span>
                    <StarRow n={r.platform_stars} />
                  </div>
                </div>

                <div className="rounded-control bg-field px-3 py-2">
                  <p className="text-2xs text-ink-muted">{L.driverStars}</p>
                  <div className="mt-1 flex flex-wrap items-center justify-between gap-2">
                    <span className="min-w-0 truncate">
                      {r.driver_name ? person(r.driver_id, r.driver_name) : "—"}
                    </span>
                    <StarRow n={r.driver_stars} />
                  </div>
                </div>

                <div className="rounded-control bg-field px-3 py-2">
                  <p className="text-2xs text-ink-muted">{L.storeStars}</p>
                  <div className="mt-1 min-w-0 truncate">
                    {r.merchant_id ? (
                      <Link href={`/dashboard/merchants/${r.merchant_id}`} className="font-medium hover:underline">
                        {r.merchant_name}
                      </Link>
                    ) : (
                      "—"
                    )}
                  </div>
                  {r.store_speed != null && (
                    <div className="mt-1.5 space-y-1 text-xs">
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-ink-muted">{L.speed}</span>
                        <StarRow n={r.store_speed} />
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-ink-muted">{L.conduct}</span>
                        <StarRow n={r.store_conduct} />
                      </div>
                    </div>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
      {data && data.total > data.per_page && (
        <div className="flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}
      <p className="text-2xs text-ink-muted">{L.note}</p>
    </section>
  );
}
