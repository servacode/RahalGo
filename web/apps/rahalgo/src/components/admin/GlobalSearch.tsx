"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **البحثُ العامُّ أعلى اللوحة** — تصميمُ اللوحة ٢٠٢٦-١٠-٢٥
 * ══════════════════════════════════════════════════════════════════════
 *
 * **رقمُ طلبٍ أو هاتفٌ أو اسم — من أيّ صفحة** (`GET /admin/search`). كان الموظّفُ
 * يتذكّر أيَّ شاشةٍ تحمل ما يبحث عنه ثمّ يبحث فيها؛ **والآن خانةٌ واحدة** تفتح
 * الطلبَ في درجه، والحسابَ في ملفّه، والمتجرَ في صفحته. و`Ctrl K` يضع المؤشّرَ فيها.
 *
 * **وكلُّ قسمٍ يصل بقدرته من الخادم** — ما لم يُرسَل لا يُرسم.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import { Badge, IconOrder, IconSearch, IconStore, IconUser } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const S = m.admin.search;

interface OrderHit {
  id: string;
  number: number;
  status: string;
  kind: string;
  merchant_name: string;
  customer_name: string;
  open: boolean;
  created_at: string;
}
interface UserHit {
  id: string;
  full_name: string;
  phone: string;
  roles: string[];
  status: string;
}
interface MerchantHit {
  id: string;
  name: string;
  number: number;
  status: string;
}
interface Hits {
  orders?: OrderHit[];
  users?: UserHit[];
  merchants?: MerchantHit[];
}

type Row = { key: string; href: string; node: React.ReactNode };

export function GlobalSearch() {
  const router = useRouter();
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<Hits | null>(null);
  const [failed, setFailed] = useState(false);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const box = useRef<HTMLDivElement>(null);

  // **`Ctrl K` يضع المؤشّرَ في الخانة** — من أيّ صفحة.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        input.current?.focus();
        input.current?.select();
        setOpen(true);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  // **ونقرةٌ خارجها تُغلقها.**
  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, []);

  const term = q.trim();
  useEffect(() => {
    if (term.length < 2) {
      setHits(null);
      setFailed(false);
      return;
    }
    let alive = true;
    const t = setTimeout(() => {
      api<Hits>(`/api/v1/admin/search?q=${encodeURIComponent(term)}`)
        .then((r) => {
          if (!alive) return;
          setHits(r);
          setFailed(false);
          setActive(0);
        })
        .catch(() => alive && setFailed(true));
    }, 220);
    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [term]);

  const rows = useMemo(() => {
    const groups: { title: string; Icon: typeof IconOrder; rows: Row[] }[] = [];
    if (hits?.orders?.length) {
      groups.push({
        title: S.orders,
        Icon: IconOrder,
        rows: hits.orders.map((o) => ({
          key: `o-${o.id}`,
          href: o.open ? `/dashboard/orders?id=${o.id}` : `/dashboard/history?order=${o.id}`,
          node: (
            <>
              <span dir="ltr" className="rounded-control bg-accent px-2 py-0.5 text-xs font-bold tabular-nums text-on-bright">
                #{o.number}
              </span>
              <span className="min-w-0 flex-1 truncate">
                {S.orderLine
                  .replace("{store}", o.kind === "custom" ? S.customOrder : o.merchant_name || "—")
                  .replace("{customer}", o.customer_name || "—")}
              </span>
              <Badge variant={o.open ? "accent" : "neutral"}>{o.open ? S.openOrder : S.closedOrder}</Badge>
              <span dir="ltr" className="hidden shrink-0 text-2xs text-ink-muted sm:inline">
                {fmtDateTime(o.created_at)}
              </span>
            </>
          ),
        })),
      });
    }
    if (hits?.merchants?.length) {
      groups.push({
        title: S.merchants,
        Icon: IconStore,
        rows: hits.merchants.map((x) => ({
          key: `m-${x.id}`,
          href: `/dashboard/merchants/${x.id}`,
          node: (
            <>
              <span className="min-w-0 flex-1 truncate font-medium">{x.name}</span>
              <span className="shrink-0 text-xs text-ink-muted">
                {S.storeNumber.replace("{n}", fmtNum(x.number))}
              </span>
            </>
          ),
        })),
      });
    }
    if (hits?.users?.length) {
      groups.push({
        title: S.users,
        Icon: IconUser,
        rows: hits.users.map((u) => ({
          key: `u-${u.id}`,
          href: `/dashboard/users/${u.id}`,
          node: (
            <>
              <span className="min-w-0 flex-1 truncate font-medium">{u.full_name || "—"}</span>
              <span className="hidden shrink-0 text-xs text-ink-muted sm:inline">
                {u.roles.map((r) => roleLabelByCode(r)).join(m.common.listSeparator)}
              </span>
              <span dir="ltr" className="shrink-0 text-xs text-ink-muted">
                {u.phone}
              </span>
            </>
          ),
        })),
      });
    }
    return groups;
  }, [hits]);
  const flat = rows.flatMap((g) => g.rows);

  function go(href: string) {
    setOpen(false);
    setQ("");
    setHits(null);
    router.push(href);
  }

  const showPanel = open && term !== "";
  return (
    <div ref={box} className="relative w-44 sm:w-72 lg:w-[30rem]">
      <label className="flex h-10 items-center gap-2 rounded-control border border-line bg-field px-3 focus-within:border-primary">
        <IconSearch size={17} className="shrink-0 text-ink-muted" />
        <span className="sr-only">{S.label}</span>
        <input
          ref={input}
          type="search"
          value={q}
          placeholder={S.placeholder}
          onChange={(e) => {
            setQ(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              setOpen(false);
              input.current?.blur();
            } else if (e.key === "ArrowDown") {
              e.preventDefault();
              setActive((a) => Math.min(a + 1, Math.max(0, flat.length - 1)));
            } else if (e.key === "ArrowUp") {
              e.preventDefault();
              setActive((a) => Math.max(a - 1, 0));
            } else if (e.key === "Enter" && flat[active]) {
              e.preventDefault();
              go(flat[active].href);
            }
          }}
          className="min-w-0 flex-1 bg-transparent text-sm text-ink outline-none placeholder:text-ink-muted"
        />
        <kbd dir="ltr" className="hidden shrink-0 rounded-control bg-ink-faint px-1.5 py-0.5 text-2xs text-ink-muted md:inline">
          {S.shortcut}
        </kbd>
      </label>
      {showPanel && (
        <div className="absolute start-0 top-full z-50 mt-2 max-h-[70vh] w-[min(36rem,calc(100vw-1.5rem))] overflow-y-auto rounded-card border border-line surface-sheet elev-4">
          {term.length < 2 ? (
            <p className="px-4 py-3 text-sm text-ink-muted">{S.tooShort}</p>
          ) : failed ? (
            <p className="px-4 py-3 text-sm text-warning">{S.failed}</p>
          ) : hits && flat.length === 0 ? (
            <p className="px-4 py-3 text-sm text-ink-muted">{S.none.replace("{q}", term)}</p>
          ) : (
            rows.map((g) => (
              <div key={g.title} className="border-b border-line-soft py-1 last:border-b-0">
                <p className="flex items-center gap-1.5 px-4 pb-1 pt-2 text-2xs font-bold text-ink-muted">
                  <g.Icon size={13} />
                  {g.title}
                </p>
                {g.rows.map((r) => {
                  const idx = flat.indexOf(r);
                  return (
                    <button
                      key={r.key}
                      type="button"
                      onMouseEnter={() => setActive(idx)}
                      onClick={() => go(r.href)}
                      className={`flex w-full items-center gap-2 px-4 py-2 text-start text-sm transition-colors ${
                        idx === active ? "bg-primary-tint" : "hover:bg-row-hover"
                      }`}
                    >
                      {r.node}
                    </button>
                  );
                })}
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
}
