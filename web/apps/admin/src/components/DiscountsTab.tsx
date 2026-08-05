"use client";

/**
 * **الخصوماتُ على الأصناف** — تبويبٌ في صفحة العروض.
 *
 * # ولماذا صفحةٌ واحدة
 *
 * كانت اللافتاتُ في مكانين: جدولُ `banners` وشاشتُه في «العروض»، **ونوعُ
 * `banner` في العروض الذي أُضيف بالأمس.** **وشيءٌ واحدٌ في مكانين يفترق** —
 * تُضاف لافتةٌ في أحدهما فلا تظهر في عرض الآخر، **ويُسأل «لماذا لا تظهر
 * لافتتي؟» ولا جوابَ يُقنع.**
 *
 * (شهده المالك ٢٠٢٦-٠٨-٠٥: «هي بنفس الهدف — ادمج جميع الميزات بصفحة واحدة».)
 *
 * **فطُويت لافتتي وبقي ما لا يفعله غيرُها**: خصمٌ على صنفٍ بعينه **يمرّ في
 * لقطة البند فيغيّر ما يُقيَّد في الدفتر.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Button,
  Badge,
  Modal,
  Input,
  EmptyState,
  LoadingState,
  IconPromos,
  IconStore,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const P = m.admin.offersPage;

interface Offer {
  id: string;
  title: string;
  body: string;
  item_name: string;
  merchant_name: string;
  price_before: number;
  price_after: number;
  discount_percent: number | null;
  borne_by: "platform" | "merchant" | null;
  ends_at: string | null;
  active: boolean;
  live: boolean;
}

interface Item {
  id: string;
  name: string;
  merchant_name: string;
}

export default function DiscountsTab() {
  const [rows, setRows] = useState<Offer[] | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [itemQuery, setItemQuery] = useState("");
  const [items, setItems] = useState<Item[]>([]);
  const [itemID, setItemID] = useState("");
  const [percent, setPercent] = useState("");
  const [borneBy, setBorneBy] = useState<"platform" | "merchant">("platform");
  const [endsAt, setEndsAt] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api<{ offers: Offer[] }>("/api/v1/admin/offers")
      .then((r) => setRows(r.offers ?? []))
      .catch(() => {
        setRows([]);
        setError(m.errors.internal);
      });
  }, []);

  useEffect(load, [load]);

  /** **والصنفُ يُبحث لا يُكتب معرّفُه** — لا أحدَ يحفظ `uuid`. */
  useEffect(() => {
    if (itemQuery.trim().length < 2 || itemID) return;
    const t = setTimeout(() => {
      api<{ items: Item[] }>(
        `/api/v1/public/search/items?q=${encodeURIComponent(itemQuery.trim())}`,
      )
        .then((r) => setItems(r.items ?? []))
        .catch(() => setItems([]));
    }, 250);
    return () => clearTimeout(t);
  }, [itemQuery, itemID]);

  function errOf(e: unknown): string {
    return e instanceof ApiError
      ? ((m.errors as Record<string, string>)[
          (e.body.message_key ?? "").split(".").pop() ?? ""
        ] ?? m.errors.internal)
      : m.errors.internal;
  }

  async function submit() {
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/offers", {
        method: "POST",
        body: JSON.stringify({
          title: title.trim(),
          body: body.trim(),
          menu_item_id: itemID,
          discount_percent: Number(percent),
          borne_by: borneBy,
          ends_at: endsAt ? new Date(endsAt).toISOString() : null,
        }),
      });
      setOpen(false);
      setTitle("");
      setBody("");
      setItemID("");
      setItemQuery("");
      setPercent("");
      load();
    } catch (e) {
      setError(errOf(e));
    } finally {
      setBusy(false);
    }
  }

  async function toggle(o: Offer) {
    try {
      await api(`/api/v1/admin/offers/${o.id}/active`, {
        method: "POST",
        body: JSON.stringify({ active: !o.active }),
      });
      load();
    } catch (e) {
      setError(errOf(e));
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-ink-muted">{P.subtitle}</p>
        <Button onClick={() => setOpen(true)}>{P.create}</Button>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {rows === null ? (
        <LoadingState />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconPromos} title={P.empty} />
      ) : (
        <div className="grid gap-3 xl:grid-cols-2">
          {rows.map((o) => (
            <div key={o.id} className="rounded-card border border-line bg-surface p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="flex items-center gap-2 font-bold">
                    {o.title}
                    {/* **وسارٍ ينامُ عن «مفعّل»**: عرضٌ مفعَّلٌ انتهى تاريخُه
                        ليس سارياً، **وشارةٌ تقول «مفعّل» تُقرأ «يعمل».** */}
                    {o.live ? (
                      <Badge variant="success">{P.live}</Badge>
                    ) : (
                      <Badge variant="neutral">{P.notLive}</Badge>
                    )}
                  </p>
                  {o.body && <p className="mt-1 text-sm text-ink-muted">{o.body}</p>}
                  <p className="mt-1.5 flex items-center gap-1.5 text-sm">
                    <IconStore size={14} className="text-ink-muted" />
                    {o.item_name} — {o.merchant_name}
                  </p>
                </div>
                <div className="shrink-0 text-end">
                  <p className="text-xs text-ink-muted line-through" dir="ltr">
                    {fmtNum(o.price_before)}
                  </p>
                  <p className="font-bold text-success" dir="ltr">
                    {fmtNum(o.price_after)} {m.common.currency}
                  </p>
                  <p className="text-xs text-ink-muted">
                    −{o.discount_percent}% ·{" "}
                    {o.borne_by === "platform" ? P.byPlatform : P.byMerchant}
                  </p>
                </div>
              </div>
              <div className="mt-3 flex items-center justify-between gap-2">
                <span className="text-xs text-ink-muted">
                  {o.ends_at ? `${P.until} ${fmtDate(o.ends_at)}` : P.noEnd}
                </span>
                <Button variant="secondary" onClick={() => void toggle(o)}>
                  {o.active ? P.deactivate : P.activate}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Modal open={open} onClose={() => setOpen(false)} title={P.create}>
        <div className="space-y-3">
          <Input
            id="offer-title"
            label={P.fTitle}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
          />
          <Input
            id="offer-body"
            label={P.fBody}
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />

          <Input
            id="offer-item"
            label={P.fItem}
            value={itemQuery}
            onChange={(e) => {
              setItemQuery(e.target.value);
              setItemID("");
            }}
          />
          {items.length > 0 && !itemID && (
            <ul className="max-h-40 space-y-1 overflow-y-auto">
              {items.map((it) => (
                <li key={it.id}>
                  <button
                    type="button"
                    onClick={() => {
                      setItemID(it.id);
                      setItemQuery(`${it.name} — ${it.merchant_name}`);
                    }}
                    className="w-full rounded-control border border-line px-3 py-2 text-start text-sm hover:border-accent"
                  >
                    {it.name} — {it.merchant_name}
                  </button>
                </li>
              ))}
            </ul>
          )}

          <Input
            id="offer-percent"
            label={P.fPercent}
            type="number"
            value={percent}
            onChange={(e) => setPercent(e.target.value)}
          />

          {/* **ومن يتحمّل حقلٌ لا قاعدة** — قاعدةٌ واحدةٌ تُجبر على واحدٍ من
              أمرين: إمّا لا نعرض ما يتحمّله المتجر، **أو نخصم من جيبه بلا
              إذنه.** */}
          <p className="text-sm font-medium">{P.fBorneBy}</p>
          <div className="flex gap-1">
            {(["platform", "merchant"] as const).map((b) => (
              <button
                key={b}
                type="button"
                onClick={() => setBorneBy(b)}
                className={`rounded-control border px-3 py-1.5 text-sm transition-colors ${
                  borneBy === b
                    ? "border-accent bg-accent/10 font-medium"
                    : "border-line text-ink-muted hover:border-accent/60"
                }`}
              >
                {b === "platform" ? P.byPlatform : P.byMerchant}
              </button>
            ))}
          </div>

          <Input
            id="offer-ends"
            label={P.fEndsAt}
            type="date"
            value={endsAt}
            onChange={(e) => setEndsAt(e.target.value)}
          />

          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setOpen(false)}>
              {m.common.cancel}
            </Button>
            <Button
              disabled={busy || !title.trim() || !itemID || !(Number(percent) >= 1)}
              onClick={() => void submit()}
            >
              {P.publish}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
