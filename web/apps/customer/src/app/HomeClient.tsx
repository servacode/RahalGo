"use client";

/** الرئيسية: بانرات، فئات، متاجر (المفتوح أولاً). */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";

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
}

export default function HomeClient({ initial }: { initial: HomeData }) {
  const { banners, categories, merchants } = initial;
  const [cat, setCat] = useState("");
  const error = "";

  const visible = merchants.filter((mr) => !cat || mr.category_id === cat);

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
            {c.icon} {c.name}
          </button>
        ))}
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {visible.map((mr) => {
          const logo = mediaUrl(mr.logo_thumb_url);
          return (
            <Link
              key={mr.id}
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
                  {mr.category_icon}
                </span>
              )}
              <div className="min-w-0 flex-1">
                <p className="truncate font-bold">{mr.name}</p>
                {mr.description && (
                  <p className="truncate text-xs text-ink-muted">{mr.description}</p>
                )}
                <Badge variant={mr.open_now ? "success" : "danger"} className="mt-1">
                  {mr.open_now ? m.site.open : m.site.closed}
                </Badge>
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
