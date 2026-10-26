"use client";

/**
 * **كلُّ تقييمٍ بسطره** (المالك ٢٠٢٦-١٠-٢٥: «لازم أعرف كل زبون قيّم شو للسائق والمتجر») —
 * الزبونُ ونجومُه للطلب وللسائق، والمتجرُ وتقييمُ السائق له (السرعة والتعامل). الأحدثُ
 * أوّلاً، و«المنخفضُ وحدَه» بحبّة. من `GET /admin/ratings/list`.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtRef, fmtDateTime, errorText, storeLabel } from "@rahalgo/i18n";
import { Chips, EmptyState, LoadingState, Pagination, ReloadState, SearchSelect, IconStar } from "@rahalgo/ui";
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

/** **نجومٌ بلونها** — المنخفضُ أحمر، والمتوسّطُ أصفر، والجيّدُ أخضر؛ والغائبُ شرطة. */
function Stars({ n }: { n: number | null }) {
  if (n == null) return <span className="text-ink-muted">—</span>;
  const tone = n <= 2 ? "text-danger" : n === 3 ? "text-warning" : "text-success";
  return (
    <span className={`inline-flex items-center gap-0.5 font-bold ${tone}`}>
      {n}
      <IconStar size={13} className="fill-current" />
    </span>
  );
}

export function RatingsList({ canProfile }: { canProfile: boolean }) {
  const [low, setLow] = useState(false);
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
      <Link href={`/dashboard/users/${id}`} className="hover:underline">
        {name || "—"}
      </Link>
    ) : (
      <span>{name || "—"}</span>
    );

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="font-bold">{L.title}</h2>
        <Chips
          items={[
            { id: "all", label: L.all },
            { id: "low", label: L.lowOnly },
          ]}
          value={low ? "low" : "all"}
          onChange={(k) => {
            setLow(k === "low");
            setPage(1);
          }}
        />
      </div>
      <div className="relative z-30 flex flex-wrap items-center gap-2">
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
        />
        {drivers.length > 0 && (
          <div className="w-full sm:w-56">
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
          </div>
        )}
        {stores.length > 0 && (
          <div className="w-full sm:w-56">
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
          </div>
        )}
      </div>
      {error && !data ? (
        <ReloadState label={error} onRetry={() => void load()} />
      ) : !data ? (
        <LoadingState variant="text" />
      ) : data.ratings.length === 0 ? (
        <EmptyState icon={IconStar} title={L.empty} />
      ) : (
        // **بطاقاتٌ لا جدول** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — بطاقةٌ لكلّ طلبٍ قُيّم.
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {data.ratings.map((r) => (
            <div key={r.order_id} className="flex flex-col gap-3 rounded-card border border-line-soft surface-lit p-4 text-sm">
              <div className="flex items-start justify-between gap-2">
                <Link href={`/dashboard/history?q=${r.order_number}`} className="font-bold hover:underline" dir="ltr">
                  #{fmtRef(r.order_number)}
                </Link>
                <span className="text-2xs text-ink-muted" dir="ltr">
                  {fmtDateTime(r.created_at)}
                </span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="min-w-0 truncate">
                  <span className="text-xs text-ink-muted">{L.customer}: </span>
                  {person(r.customer_id, r.customer_name)}
                </span>
                <span className="flex shrink-0 items-center gap-1 text-xs text-ink-muted">
                  {L.orderStars} <Stars n={r.platform_stars} />
                </span>
              </div>
              <div className="flex items-center justify-between gap-2 border-t border-line-soft pt-2">
                <span className="min-w-0 truncate">
                  <span className="text-xs text-ink-muted">{L.driver}: </span>
                  {r.driver_name ? person(r.driver_id, r.driver_name) : "—"}
                </span>
                <Stars n={r.driver_stars} />
              </div>
              <div className="flex items-center justify-between gap-2 border-t border-line-soft pt-2">
                <span className="min-w-0 truncate">
                  <span className="text-xs text-ink-muted">{L.store}: </span>
                  {r.merchant_id ? (
                    <Link href={`/dashboard/merchants/${r.merchant_id}`} className="hover:underline">
                      {r.merchant_name}
                    </Link>
                  ) : (
                    "—"
                  )}
                </span>
                {r.store_speed == null ? (
                  <span className="text-ink-muted">—</span>
                ) : (
                  <span className="flex shrink-0 items-center gap-2 text-xs">
                    <span>
                      {L.speed} <Stars n={r.store_speed} />
                    </span>
                    <span>
                      {L.conduct} <Stars n={r.store_conduct} />
                    </span>
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
      {data && data.total > data.per_page && (
        <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
      )}
      <p className="text-2xs text-ink-muted">{L.note}</p>
    </section>
  );
}
