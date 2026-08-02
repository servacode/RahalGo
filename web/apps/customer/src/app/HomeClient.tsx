"use client";

/** الرئيسية: بانرات، فئات، متاجر (المفتوح أولاً). */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale , fmtTime } from "@rahalgo/i18n";
import { CategoryIcon, Badge, Input, IconSearch, IconStar, IconClose } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
}
interface Category {
  id: string;
  name: string;
  icon: string;
}
export interface HomeData {
  banners: Banner[];
  categories: Category[];
  merchants: Merchant[];
}
interface Merchant {
  id: string;
  name: string;
  description: string;
  category_id: string;
  category_icon: string;
  logo_thumb_url: string | null;
  open_now: boolean;
  /** موعدُ الفتح القادم — **وفارغٌ إن كان مفتوحاً**. */
  opens_at?: string | null;
}

interface SearchHit {
  id: string;
  name: string;
  category_icon: string;
  logo_thumb_url: string | null;
  emergency_closed: boolean;
  matched_items: string;
}

export default function HomeClient({ initial }: { initial: HomeData }) {
  const { banners, categories, merchants } = initial;
  const { user } = useAuth();
  const logged = isLoggedIn(user);
  const [cat, setCat] = useState("");
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<SearchHit[] | null>(null);
  const [favs, setFavs] = useState<Set<string>>(new Set());
  const error = "";

  // البحث بعد سكون الكتابة لا مع كل حرف: كلُّ حرفٍ طلبٌ للخادم، والكاتب لم
  // ينتهِ من كلمته بعد. ٣٠٠ مللي ثانية هي حدُّ ما يُحسّ به المستخدم تأخيراً.
  useEffect(() => {
    const term = q.trim();
    if (term.length < 2) {
      setHits(null);
      return;
    }
    const t = setTimeout(() => {
      fetch(`${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/v1/public/search?q=${encodeURIComponent(term)}`)
        .then((r) => r.json())
        .then((j) => setHits(j.data ?? []))
        .catch(() => setHits([]));
    }, 300);
    return () => clearTimeout(t);
  }, [q]);

  const loadFavs = useCallback(() => {
    if (!logged) return;
    api<{ id: string }[]>("/api/v1/my/favorites")
      .then((a) => setFavs(new Set(a.map((x) => x.id))))
      .catch(() => undefined);
  }, [logged]);

  useEffect(() => {
    loadFavs();
  }, [loadFavs]);

  async function toggleFav(id: string) {
    // تفاؤلياً: القلب ينقلب فوراً ثم يُصحَّح إن فشل — نقرةٌ تنتظر الشبكة تبدو معطّلة
    setFavs((s) => {
      const n = new Set(s);
      if (n.has(id)) n.delete(id);
      else n.add(id);
      return n;
    });
    try {
      await api(`/api/v1/my/favorites/${id}`, { method: "POST" });
    } catch {
      loadFavs();
    }
  }

  // المفضّلة أولاً ثم المفتوح — ترتيبٌ يخدم من يعرف ما يريد
  const visible = merchants
    .filter((mr) => !cat || mr.category_id === cat)
    .sort((a, b) => Number(favs.has(b.id)) - Number(favs.has(a.id)));

  return (
    <div>
      <h1 className="mb-4 text-2xl font-bold">{m.site.hero}</h1>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {banners.length > 0 && (
        <div className="mb-6 flex gap-3 overflow-x-auto pb-1">
          {banners.map((b) => (
            <div key={b.id} className="relative h-36 w-80 shrink-0 overflow-hidden rounded-card">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={mediaUrl(b.image_url) ?? ""}
                alt={b.title}
                className="h-full w-full object-cover"
              />
              <span className="absolute bottom-0 start-0 end-0 bg-gradient-to-t from-ink/70 to-transparent p-2 text-sm font-bold text-white">
                {b.title}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* البحث فوق كل شيء: من يعرف ما يريد لا يتصفّح */}
      <div className="relative mb-5">
        <Input
          id="site-search"
          icon={<IconSearch size={16} />}
          placeholder={m.site.search.placeholder}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {q && (
          <button
            type="button"
            onClick={() => setQ("")}
            aria-label={m.common.cancel}
            className="absolute inset-block-0 end-2 my-auto flex h-7 w-7 items-center justify-center rounded-badge text-ink-muted hover:bg-page"
          >
            <IconClose size={16} />
          </button>
        )}
      </div>

      {hits !== null && (
        <div className="mb-6">
          {hits.length === 0 ? (
            <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
              {m.site.search.empty}
            </p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {hits.map((h) => (
                <Link
                  key={h.id}
                  href={`/m/${h.id}`}
                  className="flex items-center gap-3 rounded-card border border-line bg-surface p-3 transition-shadow hover:shadow-md"
                >
                  {mediaUrl(h.logo_thumb_url) ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(h.logo_thumb_url) ?? ""}
                      alt=""
                      className="h-12 w-12 rounded-control object-cover"
                    />
                  ) : (
                    <span className="flex h-12 w-12 items-center justify-center rounded-control bg-primary-light">
                      <CategoryIcon name={h.category_icon} size={18} />
                    </span>
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-bold">{h.name}</p>
                    {/* سبب الظهور: يفهم الزبون لماذا ظهر هذا المتجر لكلمته */}
                    {h.matched_items && (
                      <p className="truncate text-xs text-ink-muted">
                        {m.site.search.matched} {h.matched_items}
                      </p>
                    )}
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="mb-5 flex flex-wrap gap-2">
        <button
          onClick={() => setCat("")}
          className={`rounded-badge px-3 py-1.5 text-sm ${
            cat === "" ? "bg-primary font-medium text-white" : "border border-line text-ink-muted"
          }`}
        >
          {m.site.allCategories}
        </button>
        {categories.map((c) => (
          <button
            key={c.id}
            onClick={() => setCat(c.id)}
            className={`rounded-badge px-3 py-1.5 text-sm ${
              cat === c.id
                ? "bg-primary font-medium text-white"
                : "border border-line text-ink-muted"
            }`}
          >
            <CategoryIcon name={c.icon} size={15} />
            {c.name}
          </button>
        ))}
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {visible.map((mr) => {
          const logo = mediaUrl(mr.logo_thumb_url);
          return (
            <div key={mr.id} className="relative">
              {logged && (
                // زرٌّ فوق الرابط لا داخله: زرٌّ داخل <Link> يبتلع نقرته الرابط
                <button
                  type="button"
                  onClick={() => toggleFav(mr.id)}
                  aria-label={favs.has(mr.id) ? m.site.favorites.remove : m.site.favorites.add}
                  title={favs.has(mr.id) ? m.site.favorites.remove : m.site.favorites.add}
                  className="absolute top-2 end-2 z-10 flex h-8 w-8 items-center justify-center rounded-badge bg-surface/90 shadow-sm"
                >
                  <IconStar
                    size={17}
                    className={favs.has(mr.id) ? "fill-accent text-accent" : "text-ink-muted"}
                  />
                </button>
              )}
            <Link
              href={`/m/${mr.id}`}
              className={`flex items-center gap-3 rounded-card border border-line bg-surface p-4 transition-shadow hover:shadow-md ${
                mr.open_now ? "" : "opacity-60"
              }`}
            >
              {logo ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={logo} alt="" className="h-14 w-14 rounded-control object-cover" />
              ) : (
                <span className="flex h-14 w-14 items-center justify-center rounded-control bg-primary-light text-xl">
                  <CategoryIcon name={mr.category_icon} size={18} />
                </span>
              )}
              <div className="min-w-0 flex-1">
                <p className="truncate font-bold">{mr.name}</p>
                {mr.description && (
                  <p className="truncate text-xs text-ink-muted">{mr.description}</p>
                )}
                {/* **قل متى يعود لا أنه مغلق** — «مغلق» طريقٌ مسدود،
                    و«يفتح ١١:٠٠» موعدٌ يُعاد إليه. */}
                <Badge variant={mr.open_now ? "success" : "warning"} className="mt-1">
                  {mr.open_now
                    ? m.site.open
                    : mr.opens_at
                      ? m.site.opensAt.replace("{t}", fmtTime(mr.opens_at))
                      : m.site.closed}
                </Badge>
              </div>
            </Link>
            </div>
          );
        })}
      </div>
    </div>
  );
}
