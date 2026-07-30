"use client";

/** التوفر اليومي: صلاحية المتجر الوحيدة على القائمة — إيقاف/إعادة صنف فوراً. */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, Button, IconStore } from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

interface MenuItem {
  id: string;
  name: string;
  description: string;
  price: number;
  image_thumb_url: string | null;
  available: boolean;
}

interface MenuSection {
  id: string;
  name: string;
  items: MenuItem[];
}

function errText(err: unknown): string {
  return err instanceof ApiError ? m.errors.internal : m.errors.internal;
}

export default function MerchantMenuPage() {
  const { store } = useStore();
  const [sections, setSections] = useState<MenuSection[]>([]);
  const [error, setError] = useState("");
  const [busyItem, setBusyItem] = useState("");

  const load = useCallback(async () => {
    if (!store) return;
    try {
      setSections(await api<MenuSection[]>(`/api/v1/merchant/stores/${store.id}/menu`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [store]);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggle(item: MenuItem) {
    setBusyItem(item.id);
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}/availability`, {
        method: "PATCH",
        body: JSON.stringify({ available: !item.available }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusyItem("");
    }
  }

  return (
    <div>
      <h1 className="flex items-center gap-2 text-xl font-bold">
        <IconStore className="text-primary" />
        {m.merchant.menu.title}
      </h1>
      <p className="mb-5 mt-1 text-sm text-ink-muted">{m.merchant.menu.hint}</p>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <div className="space-y-5">
        {sections.map((sec) => (
          <section key={sec.id} className="rounded-card border border-line bg-surface">
            <h2 className="border-b border-line px-4 py-3 font-bold">{sec.name}</h2>
            <ul className="divide-y divide-line">
              {sec.items.map((item) => {
                const img = mediaUrl(item.image_thumb_url);
                return (
                  <li key={item.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                    {img ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={img}
                        alt=""
                        className="h-12 w-12 shrink-0 rounded-control object-cover"
                      />
                    ) : (
                      <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-control bg-primary-light font-bold text-primary-dark">
                        {item.name.charAt(0)}
                      </span>
                    )}
                    <div className="min-w-40 flex-1">
                      <p
                        className={`font-medium ${item.available ? "" : "text-ink-muted line-through"}`}
                      >
                        {item.name}
                      </p>
                      {item.description && (
                        <p className="text-xs text-ink-muted">{item.description}</p>
                      )}
                    </div>
                    <span className="font-bold text-primary-dark">
                      {fmt.format(item.price)} {m.common.currency}
                    </span>
                    <Badge variant={item.available ? "success" : "warning"}>
                      {item.available ? m.merchant.menu.available : m.merchant.menu.unavailable}
                    </Badge>
                    <Button
                      variant={item.available ? "danger" : "primary"}
                      disabled={busyItem === item.id}
                      onClick={() => toggle(item)}
                    >
                      {item.available
                        ? m.merchant.menu.markUnavailable
                        : m.merchant.menu.markAvailable}
                    </Button>
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
      </div>
    </div>
  );
}
