"use client";

/**
 * صفحةُ الصنف — **الشراءُ من هنا، بلا مرورٍ بمتجر.**
 *
 * # ولماذا صفحةٌ لا نافذة
 *
 * النافذةُ تُغلق فيضيع ما فيها، **والصفحةُ لها رابطٌ يُرسَل ويُحفظ ويُشارَك**.
 * ومن أراد أن يبعث لصديقه «جرّب هذه» بعث رابطاً — **وهو أرخصُ دعايةٍ وأصدقُها.**
 *
 * # والسلّةُ لا تعرف المصدر
 *
 * تُخزَّن الأصنافُ بمعرّفاتها وحدَها، **والخادمُ يستنتج المصدر عند الطلب**.
 * ولو حُفظ معرّفُ المتجر في السلّة **لَقُرئ من ذاكرة المتصفّح** — والإخفاءُ
 * الذي حرسناه في الشبكة يسقط في مكانٍ آخر.
 */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import { Badge, Button, Input } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";
import { useCart, type CartLine } from "@/lib/cart";
import type { BrowseItem } from "@/components/ItemCard";

const m = getMessages(defaultLocale);

interface Option {
  id: string;
  name: string;
  price_delta: number;
  available: boolean;
}
export interface Group {
  id: string;
  name: string;
  min_select: number;
  max_select: number;
  options: Option[];
}

export default function ItemClient({ item, modifiers }: { item: BrowseItem; modifiers: Group[] }) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);
  const [note, setNote] = useState("");
  const [chosen, setChosen] = useState<Record<string, string[]>>({});
  const [error, setError] = useState("");
  const [added, setAdded] = useState(false);

  function toggle(g: Group, optID: string) {
    setChosen((c) => {
      const cur = c[g.id] ?? [];
      if (cur.includes(optID)) return { ...c, [g.id]: cur.filter((x) => x !== optID) };
      if (g.max_select === 1) return { ...c, [g.id]: [optID] };
      if (cur.length >= g.max_select) return c;
      return { ...c, [g.id]: [...cur, optID] };
    });
  }

  const allOpts = modifiers.flatMap((g) => g.options);
  const chosenIDs = Object.values(chosen).flat();
  const delta = chosenIDs.reduce(
    (sum, oid) => sum + (allOpts.find((o) => o.id === oid)?.price_delta ?? 0),
    0,
  );
  const unit = item.price + delta;
  // **وخصمٌ بلا سعرٍ سابقٍ لا يُعرض** — الرقمُ وحدَه لا يقول إنّه أرخص.
  const discounted = !!item.price_before && item.price_before > item.price;
  const off = !item.available || item.source_closed;

  function submit() {
    for (const g of modifiers) {
      if ((chosen[g.id] ?? []).length < g.min_select) {
        setError(`${g.name}: ${m.site.menu.required}`);
        return;
      }
    }
    const line: CartLine = {
      menu_item_id: item.id,
      name: item.name,
      price: unit,
      qty,
      note,
      option_ids: chosenIDs,
      option_names: chosenIDs.map((oid) => allOpts.find((o) => o.id === oid)?.name ?? ""),
      options_delta: delta,
    };
    // **والسلّةُ تقبله بلا سؤالٍ عن مصدره** — سقفُ المصادر يُفرض في الخادم،
    // **وهو الموضعُ الذي لا يُلتفّ عليه.**
    add(line);
    setAdded(true);
  }

  const img = mediaUrl(item.image_thumb_url);

  return (
    <div className="mx-auto max-w-2xl">
      <Link
        href={`/s/${item.section_id}`}
        className="mb-3 inline-block text-sm text-ink-muted hover:text-primary-dark"
      >
        ← {item.section_name}
      </Link>

      {img && (
        // eslint-disable-next-line @next/next/no-img-element
        /* **والصورةُ بنسبةٍ لا بارتفاعٍ مكتوب.**

           كان `h-56` — **مئتان وأربعةٌ وعشرون بكسلاً مهما كان عرضُ النافذة**:
           على الجوّال تقارب المربّع وعلى الحاسب تصير شريطاً مقصوصاً. **ونسبةُ
           ٤:٣ هي نسبةُ البطاقة التي ضُغطت لفتحها** — فلا تُقصّ الصورةُ قصّاً
           آخرَ بين الشاشتين. */
        <img
          src={img}
          alt=""
          loading="lazy"
          className="mb-4 aspect-[4/3] w-full rounded-card object-cover elev-2 sm:aspect-[16/9]"
        />
      )}

      <h1 className="text-2xl font-bold">{item.name}</h1>
      {item.description && <p className="mt-1 text-ink-muted">{item.description}</p>}
      {/* **والخصمُ يُرى هنا كما يُرى في البطاقة.**

          **كانت النافذةُ تعرض السعرَ كاملاً والبطاقةُ تحته تعرض المخصوم** —
          رقمان متناقضان في شاشةٍ واحدة، **وزرُّ «أضف للسلّة» يحمل الأكبر.**
          (شهده المالك ٢٠٢٦-٠٨-٠٥.)

          **والمشطوبُ يُحسب من `price_before` لا يُقدَّر**: الخيارات تُضاف
          إلى الاثنين بالمقدار نفسِه، **فالفرقُ بينهما يبقى هو الخصم.** */}
      <p className="mt-2 flex items-baseline gap-2 text-xl font-bold text-primary-dark">
        <span>
          {fmtNum(unit)} {m.common.currency}
        </span>
        {discounted && (
          <>
            <span className="text-sm font-normal text-ink-muted line-through">
              {fmtNum(item.price_before! + delta)}
            </span>
            {item.discount_percent ? (
              <Badge variant="danger">−{item.discount_percent}%</Badge>
            ) : null}
          </>
        )}
      </p>

      {/* **قل متى يعود لا أنه غير متاح.** */}
      {off && (
        <Badge variant={item.available ? "neutral" : "warning"} className="mt-2">
          {!item.available
            ? m.site.menu.unavailable
            : item.source_opens_at
              ? m.site.menu.availableFrom.replace("{t}", fmtTime(item.source_opens_at))
              : m.site.menu.unavailable}
        </Badge>
      )}

      <div className="mt-5 space-y-4">
        {modifiers.map((g) => (
          <div key={g.id}>
            <p className="mb-1.5 text-sm font-bold">
              {g.name}{" "}
              <span className="text-xs font-normal text-ink-muted">
                {g.min_select > 0
                  ? `(${m.site.menu.required})`
                  : `(${m.site.menu.chooseUpTo.replace("{n}", String(g.max_select))})`}
              </span>
            </p>
            <div className="flex flex-wrap gap-2">
              {g.options.map((o) => {
                const on = (chosen[g.id] ?? []).includes(o.id);
                return (
                  <button
                    key={o.id}
                    type="button"
                    disabled={!o.available}
                    onClick={() => toggle(g, o.id)}
                    /* **والمختارُ يُقرأ بلمحةٍ لا بحدٍّ وحدَه.**

                       كان الفرقُ حدّاً تركوازيّاً وحشوةً باهتة — **وفي صفٍّ
                       من ستّة خياراتٍ لا تُميَّز إلّا بالتدقيق.** فصار النصُّ
                       نفسُه تركوازاً: **اللونُ يسبق الشكلَ في العين.**

                       **ويرتفع بكسلاً**: ما اختير أقربُ إلى الناظر. */
                    className={`rounded-badge border px-3 py-1.5 text-sm transition-[background-color,border-color,color,transform] duration-[--duration-fast] ease-[--ease-out] disabled:opacity-40 ${
                      on
                        ? "-translate-y-px border-primary bg-primary-light font-medium text-primary-strong"
                        : "border-line hover:border-primary/40 hover:bg-page"
                    }`}
                  >
                    {o.name}
                    {o.price_delta !== 0 && (
                      <span className="ms-1 text-xs text-ink-muted">
                        +{fmtNum(o.price_delta)}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          </div>
        ))}

        <Input
          id="item-note"
          label={m.site.menu.note}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />

        <div className="flex items-center gap-3">
          {/* **والعدّادُ زرّان لا محرفان.**

              كان `px-2` على محرفٍ نصّيّ — **مساحةُ لمسٍ نحو عشرين بكسلاً**،
              وهي نصفُ لبّ الإصبع. **ومن أخطأ «+» فزاد اثنين بدل واحدٍ يشتري
              ما لا يريد.**

              **والحقلُ يغور** كسائر الحقول، فيُقرأ شيئاً يُعدَّل لا نصّاً
              يُقرأ. */}
          <div className="flex items-center gap-1 rounded-control border border-line bg-page p-1">
            <button
              type="button"
              aria-label={m.common.decrease}
              onClick={() => setQty((n) => Math.max(1, n - 1))}
              className="flex h-9 w-9 items-center justify-center rounded-control text-lg text-ink-muted transition-colors hover:bg-surface hover:text-ink"
            >
              −
            </button>
            <span className="min-w-8 text-center font-bold tabular-nums">{fmtNum(qty)}</span>
            <button
              type="button"
              aria-label={m.common.increase}
              onClick={() => setQty((n) => Math.min(50, n + 1))}
              className="flex h-9 w-9 items-center justify-center rounded-control text-lg text-ink-muted transition-colors hover:bg-surface hover:text-ink"
            >
              +
            </button>
          </div>
          <Button disabled={off} onClick={submit}>
            {m.site.menu.add} — {fmtNum(unit * qty)} {m.common.currency}
          </Button>
        </div>

        {error && <p className="text-sm text-danger">{error}</p>}
        {added && (
          <p className="text-sm text-success">
            {m.site.menu.added}{" "}
            <Link href="/cart" className="font-bold underline">
              {m.terms.cart}
            </Link>
          </p>
        )}
      </div>
    </div>
  );
}
