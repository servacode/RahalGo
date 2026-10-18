"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/stores/[id] — صفحةُ متجر** (قرارُ المالك ٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **رأسٌ** (الشعارُ والاسمُ ومفتوحٌ/«مغلق — يفتح الساعة …» ومدّةُ التحضير)
 * **ثمّ شريطُ أقسامٍ لاصق** يقفز إلى عناوينها، **ثمّ أصنافُ المتجر كلُّها
 * تحت عناوين أقسام المنصّة** — بالبطاقة والنافذة والسلّة نفسِها.
 *
 * **والمغلقُ للتصفّح وحدَه**: أصنافُه تصل `source_closed` فلا «+» عليها،
 * **والنافذةُ تقول السببَ مكانَ زرّ الإضافة.**
 *
 * **و`/store` لوحةُ صاحب المتجر** — وهذه صفحةُ الزبون، فاسمُها `/stores`.
 */

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Badge, EmptyState, LoadingState, ReloadState, IconPrev, IconStore } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useBrowseQuery } from "@/lib/customer/context";
import type { Item, StorePage } from "@/lib/customer/types";
import { backAt, fill } from "@/lib/customer/texts";
import { ItemGrid, ItemSheet } from "@/components/customer/items";
import { StoreLogo } from "@/components/customer/stores";

const m = getMessages(defaultLocale);
const ST = m.shop.stores;

/** **حالُ المتجر بلفظ الزبون** — «مفتوح هلّق» أو «مغلق — يفتح الساعة …». */
function statusText(open: boolean, opensAt: string | null): string {
  if (open) return ST.open;
  const t = backAt(opensAt);
  return t ? fill(ST.closedOpensAt, { time: t }) : ST.closed;
}

export default function StorePageView() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const scope = useBrowseQuery("?");
  const [page, setPage] = useState<StorePage | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);
  const [active, setActive] = useState("");
  const heads = useRef<Record<string, HTMLElement | null>>({});

  const load = useCallback(() => {
    if (!id) return;
    setError("");
    setPage(null);
    api<StorePage>(`/api/v1/public/stores/${id}${scope}`)
      .then((p) => {
        setPage(p);
        setActive(p.sections[0]?.id ?? "");
      })
      .catch((err) => setError(errorText(err)));
  }, [id, scope]);

  useEffect(load, [load]);

  /* **والشريطُ يتبع التمرير** — العنوانُ الظاهرُ أعلى الشاشة هو المختار. */
  useEffect(() => {
    if (!page || page.sections.length === 0) return;
    const io = new IntersectionObserver(
      (entries) => {
        const seen = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        const first = seen[0]?.target.getAttribute("data-section");
        if (first) setActive(first);
      },
      { rootMargin: "-40% 0px -55% 0px" },
    );
    for (const s of page.sections) {
      const el = heads.current[s.id];
      if (el) io.observe(el);
    }
    return () => io.disconnect();
  }, [page]);

  const jump = (sid: string) => {
    setActive(sid);
    heads.current[sid]?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  return (
    <div className="flex flex-col gap-4">
      <Link href="/shop" className="flex w-fit items-center gap-1 text-sm text-accent-text">
        <IconPrev size={16} />
        {ST.back}
      </Link>

      {error ? (
        <ReloadState label={error} onRetry={load} />
      ) : !page ? (
        <LoadingState variant="list" rows={4} />
      ) : (
        <>
          {/* ── الرأس ── */}
          <header className="flex items-center gap-3">
            <StoreLogo name={page.store.name} logo={page.store.logo_thumb_url ?? page.store.logo_url} className="h-20 w-20" />
            <div className="min-w-0 flex-1 space-y-1">
              <h1 className="heading-page truncate">{page.store.name}</h1>
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant={page.store.open_now ? "success" : "danger"}>
                  {statusText(page.store.open_now, page.store.opens_at)}
                </Badge>
                {page.store.prep_minutes > 0 && (
                  <span className="text-xs text-ink-muted">
                    {fill(ST.prep, { n: fmtNum(page.store.prep_minutes) })}
                  </span>
                )}
              </div>
            </div>
          </header>

          {!page.store.open_now && <Alert tone="warning">{ST.browseOnly}</Alert>}

          {page.sections.length === 0 ? (
            <EmptyState icon={IconStore} title={ST.empty} action={<span className="text-xs text-ink-muted">{ST.emptyHint}</span>} />
          ) : (
            <>
              {/* ── شريطُ الأقسام اللاصق — تحت الشريط العلويّ مباشرةً ── */}
              <nav className="surface-lit sticky top-18 z-30 -mx-3 px-3 py-2 sm:-mx-4 sm:px-4">
                <ul className="flex gap-2 overflow-x-auto">
                  {page.sections.map((s) => {
                    const on = s.id === active;
                    return (
                      <li key={s.id} className="shrink-0">
                        <button
                          type="button"
                          onClick={() => jump(s.id)}
                          aria-current={on ? "true" : undefined}
                          className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                            on
                              ? "border-accent bg-accent-tint font-bold text-accent-text"
                              : "border-line text-ink-muted hover:border-accent-edge"
                          }`}
                        >
                          {s.name}
                        </button>
                      </li>
                    );
                  })}
                </ul>
              </nav>

              {page.sections.map((s) => (
                <section
                  key={s.id}
                  data-section={s.id}
                  ref={(el) => {
                    heads.current[s.id] = el;
                  }}
                  className="flex scroll-mt-32 flex-col gap-3"
                >
                  <h2 className="heading-section">{s.name}</h2>
                  <ItemGrid items={s.items} onOpen={setOpen} storeLink={false} />
                </section>
              ))}
            </>
          )}
        </>
      )}

      <ItemSheet item={open} onClose={() => setOpen(null)} storeLink={false} />
    </div>
  );
}
