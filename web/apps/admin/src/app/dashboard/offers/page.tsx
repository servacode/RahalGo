"use client";

/**
 * **العروض والخصومات** — ما تُنزله المنصةُ بنفسها.
 *
 * # والفرقُ عن أكواد الخصم
 *
 * **الكودُ يُكتب والعرضُ يُرى.** ومن لم يسمع بالكود لا يستفيد منه **ولا يعلم
 * أنّه فاته.** والعرضُ في شاشة الزبون: يفتح الأيقونةَ فيرى.
 *
 * # ولا حذف — رفعٌ وإنزال
 *
 * **عرضٌ حُذف لا يُقرأ في تقريرٍ لاحق**، ومن سأل «كم خسرنا على عروض رمضان؟»
 * لم يجد ما يقرؤه.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
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
  kind: "banner" | "discount";
  title: string;
  body: string;
  href: string;
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

export default function OffersPage() {
  const [rows, setRows] = useState<Offer[] | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<"banner" | "discount">("discount");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [href, setHref] = useState("");
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
    if (kind !== "discount" || itemQuery.trim().length < 2) return;
    const t = setTimeout(() => {
      api<{ items: Item[] }>(
        `/api/v1/public/search/items?q=${encodeURIComponent(itemQuery.trim())}`,
      )
        .then((r) => setItems(r.items ?? []))
        .catch(() => setItems([]));
    }, 250);
    return () => clearTimeout(t);
  }, [itemQuery, kind]);

  async function submit() {
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/offers", {
        method: "POST",
        body: JSON.stringify({
          kind,
          title: title.trim(),
          body: body.trim(),
          href: href.trim(),
          menu_item_id: kind === "discount" ? itemID : null,
          discount_percent: kind === "discount" ? Number(percent) : null,
          borne_by: kind === "discount" ? borneBy : null,
          ends_at: endsAt ? new Date(endsAt).toISOString() : null,
        }),
      });
      setOpen(false);
      setTitle("");
      setBody("");
      setHref("");
      setItemID("");
      setItemQuery("");
      setPercent("");
      load();
    } catch (e) {
      setError(
        e instanceof ApiError
          ? ((m.errors as Record<string, string>)[
              (e.body.message_key ?? "").split(".").pop() ?? ""
            ] ?? m.errors.internal)
          : m.errors.internal,
      );
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
    } catch {
      setError(m.errors.internal);
    }
  }

  return (
    <PageContainer>
      <PageHeader
        icon={IconPromos}
        title={P.title}
        subtitle={P.subtitle}
        actions={<Button onClick={() => setOpen(true)}>{P.create}</Button>}
      />

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
                    <Badge variant={o.kind === "discount" ? "primary" : "neutral"}>
                      {o.kind === "discount" ? P.kindDiscount : P.kindBanner}
                    </Badge>
                    {/* **وسارٍ ينامُ عن «مفعّل»**: عرضٌ مفعَّلٌ انتهى تاريخُه
                        ليس سارياً، **وشارةٌ تقول «مفعّل» تُقرأ «يعمل».** */}
                    {o.live ? (
                      <Badge variant="success">{P.live}</Badge>
                    ) : (
                      <Badge variant="neutral">{P.notLive}</Badge>
                    )}
                  </p>
                  {o.body && <p className="mt-1 text-sm text-ink-muted">{o.body}</p>}
                  {o.kind === "discount" && (
                    <p className="mt-1.5 flex items-center gap-1.5 text-sm">
                      <IconStore size={14} className="text-ink-muted" />
                      {o.item_name} — {o.merchant_name}
                    </p>
                  )}
                </div>
                {o.kind === "discount" && (
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
                )}
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
          <div className="flex gap-1">
            {(["discount", "banner"] as const).map((k) => (
              <button
                key={k}
                type="button"
                onClick={() => setKind(k)}
                className={`rounded-control border px-3 py-1.5 text-sm transition-colors ${
                  kind === k
                    ? "border-accent bg-accent/10 font-medium"
                    : "border-line text-ink-muted hover:border-accent/60"
                }`}
              >
                {k === "discount" ? P.kindDiscount : P.kindBanner}
              </button>
            ))}
          </div>

          <Input id="offer-title" label={P.fTitle} value={title} onChange={(e) => setTitle(e.target.value)} />
          <Input id="offer-body" label={P.fBody} value={body} onChange={(e) => setBody(e.target.value)} />

          {kind === "banner" ? (
            <Input id="offer-href" label={P.fHref} value={href} onChange={(e) => setHref(e.target.value)} />
          ) : (
            <>
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
              {/* **ومن يتحمّل حقلٌ لا قاعدة** — قاعدةٌ واحدةٌ تُجبر على
                  واحدٍ من أمرين: إمّا لا نعرض ما يتحمّله المتجر، **أو نخصم
                  من جيبه بلا إذنه.** */}
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
            </>
          )}

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
              disabled={
                busy ||
                !title.trim() ||
                (kind === "discount" && (!itemID || !(Number(percent) >= 1)))
              }
              onClick={() => void submit()}
            >
              {P.publish}
            </Button>
          </div>
        </div>
      </Modal>
    </PageContainer>
  );
}
