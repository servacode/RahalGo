"use client";

/** صفحة المتجر: القائمة كاملة، نافذة الصنف بخياراته (حدود min/max)، إضافة للسلة. */

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import {
  Alert,
  CategoryIcon,
  Badge,
  Button,
  Input,
  Modal,
  FavoriteButton,
  useFavorites,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import { useCart, type CartLine } from "@/lib/cart";

const m = getMessages(defaultLocale);

interface Option {
  id: string;
  name: string;
  price_delta: number;
  available: boolean;
}
interface Group {
  id: string;
  name: string;
  min_select: number;
  max_select: number;
  options: Option[];
}
interface Item {
  id: string;
  name: string;
  description: string;
  price: number;
  image_thumb_url: string | null;
  available: boolean;
  modifiers: Group[];
  /** مصدرُ الصنف خارجَ دوامه — **يقوله الوقتُ لا المتجر**. */
  source_closed?: boolean;
  source_opens_at?: string | null;
}
export interface Section {
  id: string;
  name: string;
  items: Item[];
}
export interface Merchant {
  id: string;
  name: string;
  description: string;
  category_icon: string;
  logo_thumb_url: string | null;
  open_now: boolean;
  /** موعدُ الفتح القادم — **وفارغٌ إن كان مفتوحاً**. */
  opens_at?: string | null;
}

export default function MerchantClient({ merchant, menu }: { merchant: Merchant; menu: Section[] }) {
  const [picking, setPicking] = useState<Item | null>(null);
  const router = useRouter();
  const { user } = useAuth();
  const signedIn = isLoggedIn(user);
  // **ولا يُنادى الخادمُ لزائرٍ لم يدخل** — يردّ «غيرَ مخوَّل» بلا فائدة.
  const { has, toggle } = useFavorites(api, signedIn);

  const logo = mediaUrl(merchant.logo_thumb_url);

  return (
    <div>
      <div className="mb-6 flex items-center gap-4">
        {logo ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={logo} alt="" loading="lazy" className="h-16 w-16 rounded-card object-cover" />
        ) : (
          <span className="flex h-16 w-16 items-center justify-center rounded-card bg-primary-light text-2xl">
            <CategoryIcon name={merchant.category_icon} size={18} />
          </span>
        )}
        <div>
          <h1 className="text-2xl font-bold">{merchant.name}</h1>
          {/* **قل متى يعود لا أنه مغلق.**

              «مغلق» **طريقٌ مسدود**: من قرأه أغلق الصفحة ولم يعد. و«يفتح
              ١١:٠٠» **موعدٌ يُعاد إليه** — والبياناتُ في `merchant_hours` منذ
              البداية، **ولم تكن تصعد إلى الردّ.**

              **والفرقُ بينهما هو الفرقُ بين زبونٍ ضاع وزبونٍ عاد.** */}
          <Badge variant={merchant.open_now ? "success" : "warning"}>
            {merchant.open_now
              ? m.site.open
              : merchant.opens_at
                ? m.site.opensAt.replace("{t}", fmtTime(merchant.opens_at))
                : m.site.closed}
          </Badge>
        </div>

      </div>

      <div className="space-y-6">
        {menu.map((sec) => (
          <section key={sec.id}>
            {/* **وعنوانُ القسم يلتصق أثناء التمرير.**

                قائمةُ مطعمٍ فيها ستّةُ أقسامٍ وأربعون صنفاً — **ومن نزل في
                «المشاوي» عشرين صنفاً لا يعرف أين هو**، فيصعد ليقرأ العنوانَ
                ثمّ ينزل من جديد.

                **والعنوانُ اللاصقُ يجيب بلا حركة**: يبقى في أعلى الشاشة ما
                دام قسمُه معروضاً، **ويُدفع بالذي بعده** حين ينتهي.

                **وخلفيّةٌ تحته لا شفافيّة**: بلاها تمرّ الصورُ من ورائه
                فيُقرأ نصّاً على طعام. */}
            <h2 className="sticky top-[4.5rem] z-20 mb-3 -mx-1 border-s-4 border-primary bg-surface px-1 py-1.5 ps-2 text-lg font-bold">
              {sec.name}
            </h2>
            <div className="grid gap-3 sm:grid-cols-2">
              {sec.items.map((item) => {
                const img = mediaUrl(item.image_thumb_url);
                return (
                  /* **البطاقةُ غلافٌ والزرُّ داخلَه.**

                     كانت البطاقةُ نفسُها `<button>`، **وزرٌّ داخل زرٍّ لا
                     يجوز**: المتصفّحُ يفكّه كما يشاء فتضيع إحدى الضغطتين.
                     فصار الغلافُ يحمل الحدَّ والخلفية، **والقلبُ أخاً للزرّ
                     لا ابناً له.** */
                  <div
                    key={item.id}
                    className="flex items-center gap-2 rounded-card border border-line bg-surface p-3 transition-shadow hover:elev-2"
                  >
                  <button
                    type="button"
                    disabled={!item.available || item.source_closed || !merchant.open_now}
                    onClick={() => setPicking(item)}
                    className="flex min-w-0 flex-1 items-center gap-3 text-start disabled:opacity-50"
                  >
                    {img ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      /* **والصورةُ تكبر — لأنّ الطعامَ يُشترى بالعين.**

                         كانت أربعةً وستّين بكسلاً في **الشاشة التي يُطلب
                         منها**، **والصنفُ نفسُه في صفحة القسم صورةٌ بنسبة
                         ٤:٣ بعرض البطاقة** (`ItemCard`).

                         **فالصحنُ الواحدُ شهيٌّ في شاشةٍ وطابعُ بريدٍ في
                         الأخرى** — وهي التي يُضغط فيها زرُّ الإضافة.

                         **ولم تُفتَّح خلفيّةُ الصفحة** كما كان في الخطّة:
                         جُرّبت فأسقطت التباين (النصُّ الخافت ٣٫٦٣ والحدُّ
                         ٢٫٥٥). **والعلاجُ في حجم الصورة لا في لون ما
                         حولَها.** */
                      <img
                        src={img}
                        alt=""
                        loading="lazy"
                        className="h-20 w-20 shrink-0 rounded-control object-cover sm:h-24 sm:w-24"
                      />
                    ) : (
                      <span className="flex h-20 w-20 shrink-0 items-center justify-center rounded-control bg-primary-light text-lg font-bold text-primary-strong sm:h-24 sm:w-24">
                        {item.name.charAt(0)}
                      </span>
                    )}
                    <div className="min-w-0 flex-1">
                      <p className="font-bold">{item.name}</p>
                      {item.description && (
                        <p className="line-clamp-1 text-xs text-ink-muted">{item.description}</p>
                      )}
                      <p className="mt-1 font-bold text-primary-dark">
                        {fmtNum(item.price)} {m.common.currency}
                      </p>
                    </div>
                    {/* **وسببُ الغياب يُقال.**

                        «غير متاح» تُخفي سببين: **نفد الصنفُ** (علَمٌ يرفعه
                        المتجرُ بيده) **ونام مصدرُه** (يقوله الوقتُ عنه).
                        والأوّلُ لا موعدَ لعودته، **والثاني موعدُه معروف** —
                        فمن قيل له «متاح من ١٠ صباحاً» عاد. */}
                    {!item.available ? (
                      <Badge variant="warning">{m.site.menu.unavailable}</Badge>
                    ) : item.source_closed ? (
                      <Badge variant="neutral">
                        {item.source_opens_at
                          ? m.site.menu.availableFrom.replace(
                              "{t}",
                              fmtTime(item.source_opens_at),
                            )
                          : m.site.menu.unavailable}
                      </Badge>
                    ) : null}
                  </button>

                  {/* **والقلبُ عند الصنف نفسِه** — حيث يقرّر الزبونُ أنّه
                      أعجبه.

                      **ويبقى عاملاً وإن كان الصنفُ موقوفاً**: من نفدت شاورماه
                      اليومَ يريد أن يحفظها لغدٍ — **ومنعُه يعني أن يبحث عنها
                      من جديد.** */}
                  <FavoriteButton
                    size="sm"
                    itemID={item.id}
                    on={has(item.id)}
                    onToggle={toggle}
                    onRequireLogin={
                      signedIn ? undefined : () => router.push(`/login?next=/m/${merchant.id}`)
                    }
                  />
                  </div>
                );
              })}
            </div>
          </section>
        ))}
      </div>

      {picking && (
        <ItemModal merchant={merchant} item={picking} onClose={() => setPicking(null)} />
      )}
    </div>
  );
}

function ItemModal({
  merchant,
  item,
  onClose,
}: {
  merchant: Merchant;
  item: Item;
  onClose: () => void;
}) {
  const { add } = useCart();
  const [qty, setQty] = useState(1);
  const [note, setNote] = useState("");
  const [chosen, setChosen] = useState<Record<string, string[]>>({});
  const [error, setError] = useState("");

  function toggle(g: Group, optID: string) {
    setChosen((c) => {
      const cur = c[g.id] ?? [];
      if (cur.includes(optID)) return { ...c, [g.id]: cur.filter((x) => x !== optID) };
      if (g.max_select === 1) return { ...c, [g.id]: [optID] };
      if (cur.length >= g.max_select) return c;
      return { ...c, [g.id]: [...cur, optID] };
    });
  }

  const allOpts = item.modifiers.flatMap((g) => g.options);
  const chosenIDs = Object.values(chosen).flat();
  const delta = chosenIDs.reduce(
    (sum, oid) => sum + (allOpts.find((o) => o.id === oid)?.price_delta ?? 0),
    0
  );
  const unit = item.price + delta;

  function buildLine(): CartLine | null {
    for (const g of item.modifiers) {
      const n = (chosen[g.id] ?? []).length;
      if (n < g.min_select) {
        setError(`${g.name}: ${m.site.menu.required}`);
        return null;
      }
    }
    return {
      menu_item_id: item.id,
      name: item.name,
      price: unit,
      qty,
      note,
      option_ids: chosenIDs,
      option_names: chosenIDs.map((oid) => allOpts.find((o) => o.id === oid)?.name ?? ""),
      options_delta: delta,
    };
  }

  function submit() {
    const line = buildLine();
    if (!line) return;
    // **والسلّةُ لا تعرف المصدر بعد اليوم** — سقفُ المصادر يُفرض في الخادم،
    // **وهو الموضعُ الذي لا يُلتفّ عليه.**
    add(line);
    onClose();
  }

  return (
    <Modal open onClose={onClose} title={item.name}>
      <div className="space-y-4">
        {item.modifiers.map((g) => (
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
                    className={`rounded-control border px-3 py-1.5 text-sm disabled:opacity-40 ${
                      on
                        ? "border-primary bg-primary-light font-medium text-primary-dark"
                        : "border-line"
                    }`}
                  >
                    {o.name}
                    {o.price_delta > 0 && (
                      <span className="text-xs text-ink-muted"> +{fmtNum(o.price_delta)}</span>
                    )}
                  </button>
                );
              })}
            </div>
          </div>
        ))}

        {/* **الحقلُ من العُدّة** — كان منسوخاً بأصنافه، **وحلقةُ التركيز
            ناقصةً فيه** (`focus:border-primary` بلا `ring`): يقف المؤشّر ولا
            يظهر أثرٌ يُذكر. */}
        <Input
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={m.site.menu.itemNote}
        />

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm text-ink-muted">{m.site.menu.qty}</span>
            <button
              onClick={() => setQty(Math.max(1, qty - 1))}
              className="h-8 w-8 rounded-control border border-line font-bold"
            >
              −
            </button>
            <span className="w-6 text-center font-bold">{fmtNum(qty)}</span>
            <button
              onClick={() => setQty(qty + 1)}
              className="h-8 w-8 rounded-control border border-line font-bold"
            >
              +
            </button>
          </div>
          <span className="font-bold text-primary-dark">
            {fmtNum(unit * qty)} {m.common.currency}
          </span>
        </div>

        {error && (
          <Alert>{error}</Alert>
        )}

        {/* **وسؤالُ «سلّتك من متجرٍ آخر» سقط بسقوط سببه.**

            كانت السلّةُ ترفض صنفاً من متجرٍ ثانٍ **وهي تعرف المتجرين**.
            واليومَ لا تعرفهما — **والخادمُ يعرف ويردّ إن تعدّدت المصادر**،
            وهو الموضعُ الذي لا يُلتفّ عليه. */}

        <Button onClick={submit} className="w-full py-2.5">
          {m.site.menu.addToCart} — {fmtNum(unit * qty)} {m.common.currency}
        </Button>
      </div>
    </Modal>
  );
}
