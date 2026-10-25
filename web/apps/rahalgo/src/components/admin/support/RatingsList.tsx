"use client";

/**
 * **كلُّ تقييمٍ بسطره** (المالك ٢٠٢٦-١٠-٢٥: «لازم أعرف كل زبون قيّم شو للسائق والمتجر») —
 * الزبونُ ونجومُه للطلب وللسائق، والمتجرُ وتقييمُ السائق له (السرعة والتعامل). الأحدثُ
 * أوّلاً، و«المنخفضُ وحدَه» بحبّة. من `GET /admin/ratings/list`.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtRef, fmtDateTime, errorText } from "@rahalgo/i18n";
import { Chips, EmptyState, LoadingState, Pagination, ReloadState, IconStar } from "@rahalgo/ui";
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
  const [data, setData] = useState<{ ratings: Row[]; total: number; per_page: number } | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await api(`/api/v1/admin/ratings/list?page=${page}${low ? "&low=1" : ""}`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [page, low]);

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
      {error && !data ? (
        <ReloadState label={error} onRetry={() => void load()} />
      ) : !data ? (
        <LoadingState variant="text" />
      ) : data.ratings.length === 0 ? (
        <EmptyState icon={IconStar} title={L.empty} />
      ) : (
        <div className="overflow-x-auto rounded-card border border-line-soft">
          <table className="w-full min-w-[46rem] text-sm">
            <thead>
              <tr className="border-b border-line-soft text-xs text-ink-muted">
                <th className="p-3 text-start font-medium">{L.order}</th>
                <th className="p-3 text-start font-medium">{L.customer}</th>
                <th className="p-3 text-start font-medium">{L.orderStars}</th>
                <th className="p-3 text-start font-medium">{L.driver}</th>
                <th className="p-3 text-start font-medium">{L.driverStars}</th>
                <th className="p-3 text-start font-medium">{L.store}</th>
                <th className="p-3 text-start font-medium">{L.storeStars}</th>
              </tr>
            </thead>
            <tbody>
              {data.ratings.map((r) => (
                <tr key={r.order_id} className="border-b border-line-soft last:border-b-0">
                  <td className="p-3">
                    <Link href={`/dashboard/history?q=${r.order_number}`} className="font-bold hover:underline" dir="ltr">
                      #{fmtRef(r.order_number)}
                    </Link>
                    <span className="block text-2xs text-ink-muted" dir="ltr">
                      {fmtDateTime(r.created_at)}
                    </span>
                  </td>
                  <td className="p-3">{person(r.customer_id, r.customer_name)}</td>
                  <td className="p-3">
                    <Stars n={r.platform_stars} />
                  </td>
                  <td className="p-3">{r.driver_name ? person(r.driver_id, r.driver_name) : "—"}</td>
                  <td className="p-3">
                    <Stars n={r.driver_stars} />
                  </td>
                  <td className="p-3">
                    {r.merchant_id ? (
                      <Link href={`/dashboard/merchants/${r.merchant_id}`} className="hover:underline">
                        {r.merchant_name}
                      </Link>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="p-3">
                    {r.store_speed == null ? (
                      <span className="text-ink-muted">—</span>
                    ) : (
                      <span className="flex flex-col gap-0.5 text-xs">
                        <span>
                          {L.speed} <Stars n={r.store_speed} />
                        </span>
                        <span>
                          {L.conduct} <Stars n={r.store_conduct} />
                        </span>
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {data && data.total > data.per_page && (
        <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
      )}
      <p className="text-2xs text-ink-muted">{L.note}</p>
    </section>
  );
}
