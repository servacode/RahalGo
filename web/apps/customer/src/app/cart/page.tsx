"use client";

/** السلة والدفع: عنوان بدبوس، معاينة رسوم المنطقة، دفع نقدي/محفظة/مختلط، كود خصم. */

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  IconEdit,
  IconCheck,
  IconLocation,
  IconWhatsApp,
  Button,
  Input,
  Select,
  AddressBook,
  type SavedAddress,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import { useCart } from "@/lib/cart";

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), { ssr: false });

const m = getMessages(defaultLocale);
const A = m.site.addresses;

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

// مركز الرقة الافتراضي
const RAQQA = { lat: 35.9528, lng: 39.0079 };

/**
 * تسعيرةُ السلّة — **من الخادم لا من حسبةٍ محلّية.**
 *
 * **ولا تُسمّى `Zone` بعد اليوم**: لم تعد رسمَ منطقةٍ بل حسبةً كاملة تعرف
 * المصادرَ والمسافةَ بينها. **واسمٌ يبقى على معناه القديم يُقرأ خطأً.**
 */
interface Quote {
  subtotal: number;
  delivery_fee: number;
  base_fee: number;
  /** ما زِيد لأجل المصادر الإضافية — **وصفرٌ حين لا يكلّف**. */
  sources_fee: number;
  total: number;
  sources: number;
  max_sources: number;
  too_many_sources: boolean;
  far_apart: boolean;
}

export default function CartPage() {
  const { user, loading } = useAuth();
  const { cart, setQty, clear } = useCart();
  const router = useRouter();

  const [address, setAddress] = useState("");
  const [saved, setSaved] = useState<SavedAddress[]>([]);
  const [pickedID, setPickedID] = useState("");
  const [lat, setLat] = useState(RAQQA.lat);
  const [lng, setLng] = useState(RAQQA.lng);

  useEffect(() => {
    api<{ whatsapp_verified: boolean }>("/api/v1/me/summary")
      .then((s) => setWaVerified(s.whatsapp_verified))
      .catch(() => setWaVerified(null));
  }, []);

  // العنوان الافتراضي يُملأ تلقائياً — من له عنوانٌ واحد لا يُسأل عنه
  useEffect(() => {
    api<SavedAddress[]>("/api/v1/my/addresses")
      .then((a) => {
        setSaved(a);
        const def = a.find((x) => x.is_default) ?? a[0];
        if (def) {
          setPickedID(def.id);
          setAddress(def.address_text);
          setLat(def.lat);
          setLng(def.lng);
        }
      })
      .catch(() => undefined);
  }, []);
  const [zone, setZone] = useState<Quote | null>(null);
  const [zoneErr, setZoneErr] = useState("");
  const [payment, setPayment] = useState("cash");
  const [promo, setPromo] = useState("");
  const [notes, setNotes] = useState("");
  const [balance, setBalance] = useState<number | null>(null);

  /**
   * توثيقُ واتساب — شرطُ الطلب.
   *
   * الرقمُ الوهميّ يعني سائقاً يقف أمام بابٍ لا أحد فيه، وطلباً نقدياً لا
   * يُقبض، ومتجراً حضّر بضاعةً لا تُستلَم. **والخسارة تقع على ثلاثة أطراف لا
   * على من كتب الرقم.**
   *
   * و`null` تعني «لم نعرف بعد» لا «غير موثَّق»: لو بدأناها `false` لظهرت
   * اللافتة لحظةً لكل زبونٍ موثَّق — **ووميضُ تحذيرٍ كاذب يُفقد الثقة بكل تحذير**.
   */
  const [waVerified, setWaVerified] = useState<boolean | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  /**
   * الطلبُ نجح بعنوانٍ ليس من المحفوظة — نسأل صاحبَه أيحفظه.
   *
   * **والعنوان عندنا دبّوسٌ لا نصّ**، ودقّتُه هي ما يُبلغ السائقَ البابَ. فمن
   * كتب عنوانه وحرّك دبّوسه بدقّة ثم ضاع عملُه، يُعيده في كل طلب.
   *
   * **والسؤال بعد النجاح لا قبله**: ما يقف بين الزبون وزرّ التأكيد يُسرَّع
   * عليه ويُغلَق بلا قراءة — ولا شأن له بالطلب أصلاً.
   */
  const [placed, setPlaced] = useState<{ id: string } | null>(null);
  const [saveLabel, setSaveLabel] = useState("");
  const [saveErr, setSaveErr] = useState("");

  const logged = isLoggedIn(user);

  useEffect(() => {
    if (!logged) return;
    api<{ balance: number }>("/api/v1/my/wallet")
      .then((s) => setBalance(s.balance))
      .catch(() => undefined);
  }, [logged]);

  /**
   * **التسعيرةُ من الخادم — رقمٌ يتغيّر أمام العين.**
   *
   * رسمُ التوصيل لم يعد رقمَ المنطقة وحدَه: **يتغيّر بعدد المطابخ وبُعدها.**
   * وكانت السلّةُ تقرأ رسمَ المنطقة — **فيرى الزبونُ رقماً ويُحاسَب بآخر.**
   *
   * **ورقمٌ يتغيّر أمام العين يُقبل، ورقمٌ يظهر عند الدفع يُراجَع**: من أضاف
   * صنفاً فارتفع الرسمُ أمامه يفهم أن السببَ إضافتُه، **ومن رآه عند الدفع
   * يظنّ أنه خُدع.**
   *
   * **والسلّةُ لا تحسب بنفسها**: لا تعرف المصادر — أخفيناها عنها عمداً — ولا
   * المسافةَ بينها. **والحسبةُ حيث المعرفة.**
   */
  const quote = useCallback(async () => {
    try {
      const q = await api<Quote>("/api/v1/public/quote", {
        method: "POST",
        body: JSON.stringify({
          lat,
          lng,
          items: (cart?.lines ?? []).map((l) => ({
            menu_item_id: l.menu_item_id,
            qty: l.qty,
            option_ids: l.option_ids,
          })),
        }),
      });
      setZone(q);
      setZoneErr("");
    } catch (err) {
      setZone(null);
      setZoneErr(errText(err));
    }
  }, [lat, lng, cart]);

  useEffect(() => {
    const t = setTimeout(quote, 400);
    return () => clearTimeout(t);
  }, [quote]);

  /**
   * سؤالُ حفظ العنوان — **قبل حارس «السلّة فارغة»**.
   *
   * السلّة تُفرَّغ لحظةَ نجاح الطلب (وهذا صحيح: طلبٌ وُلد لا يُطلب مرّتين)،
   * فلو جاء السؤال بعد الحارس لَما ظهر أبداً — تُفرَّغ السلّة فيرتدّ الحارس
   * قبل أن يُقرأ السؤال.
   */
  if (placed) {
    // **إلى قائمة الطلبات لا إلى صفحةِ طلبٍ منفردة.**
    //
    // صفحةُ التفاصيل حُذفت: **البطاقةُ صارت تحمل كلَّ ما كان فيها** — الأصنافَ
    // وأسعارَها والمسارَ والوقتَ المتوقَّع والفاتورةَ والشكوى. **وصفحتان تقولان
    // الشيءَ نفسَه تفترقان يوماً**، وإحداهما تبقى بلا صيانة.
    const goOn = () => router.push("/orders");
    return (
      <div className="mx-auto max-w-md py-10">
        <div className="surface p-6">
          <p className="mb-1 flex items-center gap-2 font-bold">
            <IconCheck size={18} strokeWidth={3} className="text-success" />
            {m.site.orders.placed}
          </p>
          <p className="mb-5 text-sm text-ink-muted">{m.site.orders.placedHint}</p>

          <div className="rounded-card border border-dashed border-accent bg-accent-tint p-4">
            <p className="flex items-center gap-2 font-medium">
              <IconLocation size={17} className="text-accent-dark" />
              {A.saveThisTitle}
            </p>
            <p className="mt-1 truncate text-xs text-ink-muted">{address}</p>
            <p className="mt-1 text-xs text-ink-muted">{A.saveThisHint}</p>

            <div className="mt-3">
              <Input
                id="save-label"
                label={A.label}
                value={saveLabel}
                onChange={(e) => {
                  setSaveLabel(e.target.value);
                  setSaveErr("");
                }}
                placeholder={A.labelPlaceholder}
              />
            </div>
            {saveErr && <p className="mt-1 text-xs text-danger">{saveErr}</p>}

            <div className="mt-3 flex gap-2">
              <Button
                className="flex-1"
                disabled={!saveLabel.trim()}
                onClick={async () => {
                  try {
                    await api("/api/v1/my/addresses", {
                      method: "POST",
                      body: JSON.stringify({
                        label: saveLabel.trim(),
                        address_text: address,
                        lat,
                        lng,
                      }),
                    });
                    goOn();
                  } catch (err) {
                    // **ولا يُحبس الزبون عن طلبه إن فشل الحفظ**: الطلب تمّ،
                    // والعنوان زيادةٌ عليه. يُقال له السبب ويُترك له المضيّ.
                    setSaveErr(errText(err));
                  }
                }}
              >
                {m.common.save}
              </Button>
              <Button variant="secondary" className="flex-1" onClick={goOn}>
                {A.saveThisSkip}
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!cart || cart.lines.length === 0) {
    return (
      /* **والفراغُ داخلَ لوحٍ لا على الخلفيّة.**

         (جردُ ٢٠٢٦-٠٨-٠٦: «الرئيسية» عند **١٫٢٧** — نصٌّ يقف على أفتح بقعةٍ
          في التدرّج.)

         **وكلُّ نصٍّ في المنصة داخلَ سطح** — وهذه كانت آخرَ صفحةٍ تخالف.
         **ونصُّ الرابط بالنبرة لا بالأساسيّ**: الأساسيُّ ٣٫٠٩ على السطح
         والنبرةُ ٤٫٥٨، **وهي المُعدّةُ للنصّ العاري.** */
      <div className="surface px-4 py-16 text-center text-ink-muted">
        <p className="mb-4">{m.site.cart.empty}</p>
        <Link href="/" className="font-medium text-accent-text hover:underline">
          {m.site.nav.home}
        </Link>
      </div>
    );
  }

  const subtotal = cart.lines.reduce((s, l) => s + l.price * l.qty, 0);

  async function placeOrder() {
    if (!cart) return;
    setBusy(true);
    setError("");
    try {
      const o = await api<{ id: string; number: number }>("/api/v1/orders", {
        method: "POST",
        body: JSON.stringify({
          // **ولا `merchant_id`** — الخادمُ يستنتجه من الأصناف، وهو من
          // يعرف المصادر لا نحن.
          items: cart.lines.map((l) => ({
            menu_item_id: l.menu_item_id,
            qty: l.qty,
            note: l.note,
            option_ids: l.option_ids,
          })),
          address_text: address,
          lat,
          lng,
          payment_method: payment,
          promo_code: promo.trim(),
          notes,
        }),
      });
      clear();
      // أهو من عناوينه المحفوظة؟ المقارنة بالنصّ **وبالدبّوس معاً**: من عدّل
      // موضع دبّوسه على العنوان نفسه فقد صنع عنواناً آخر فعلاً.
      const known = saved.some(
        (a) =>
          a.address_text.trim() === address.trim() &&
          Math.abs(a.lat - lat) < 1e-5 &&
          Math.abs(a.lng - lng) < 1e-5,
      );
      if (known) {
        router.push("/orders");
        return;
      }
      setPlaced({ id: o.id });
      setBusy(false);
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  /**
   * **شريطُ الحسم — المبلغُ والزرُّ في مكانٍ واحدٍ لا يغيب.**
   *
   * # المسألة
   *
   * الصفحةُ خمسُمئةٍ وخمسةٌ وستّون سطراً: أصنافٌ ثمّ ملخّصٌ ثمّ دخولٌ ثمّ
   * عناوينُ محفوظةٌ ثمّ خريطةٌ ثمّ طريقةُ دفعٍ ثمّ كودُ خصمٍ ثمّ توثيقُ
   * واتساب — **ثمّ الزرّ.**
   *
   * **وعلى الجوّال أربعُ شاشاتٍ بين المبلغ والزرّ**: من قرأ «٤٧٬٠٠٠» في
   * الأعلى ونزل يملأ عنوانَه **نسي الرقم**، ومن وصل الزرَّ لا يراه.
   *
   * **والمبلغُ آخرُ ما يُراجَع قبل الالتزام** — ورقمٌ لا يُرى لحظةَ الضغط
   * يُراجَع بعد الطلب لا قبله.
   *
   * # ولماذا فوق الشريط السفليّ لا مكانَه
   *
   * **التنقّلُ يبقى**: من اكتشف أنّه نسي صنفاً يعود، **وحبسُه في السلّة
   * يجعله يُلغي بدل أن يُكمل.**
   */
  const canPlace = !busy && !!address && !!zone && waVerified !== false;
  const stickyTotal = zone ? subtotal + zone.delivery_fee : subtotal;

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section>
        <h1 className="mb-1 text-xl font-bold">
          {m.site.cart.title}{" "}
          <span className="text-sm font-normal text-ink-muted">
            {m.site.cart.fromPlatform}
          </span>
        </h1>
        {/* حُذف سطرُ «طلبٌ واحد مهما تعدّدت أصنافه»: صار الكرتُ الجامع يقوله
            بلا كلام — **ما يُرى لا يُشرح**. (كان لازماً حين كانت الأصناف أسطراً
            متفرّقة على الصفحة — R-88.) */}
        <div className="mt-3 rounded-card border border-line bg-page p-3">
        <ul className="space-y-2">
          {cart.lines.map((l, i) => (
            <li
              key={i}
              className="flex items-center gap-3 surface p-3"
            >
              <div className="min-w-0 flex-1">
                <p className="font-medium">{l.name}</p>
                {l.option_names.length > 0 && (
                  <p className="text-xs text-ink-muted">{l.option_names.join(m.common.listSeparator)}</p>
                )}
                {l.note && <p className="text-xs text-accent-dark"><IconEdit size={11} className="inline align-[-1px]" /> {l.note}</p>}
              </div>
              <span className="text-sm font-bold text-primary-dark">
                {fmtNum(l.price * l.qty)}
              </span>
              <div className="flex items-center gap-1.5">
                <button
                  onClick={() => setQty(i, l.qty - 1)}
                  className="h-7 w-7 rounded-control border border-line text-sm font-bold"
                >
                  −
                </button>
                <span className="w-5 text-center text-sm font-bold">{fmtNum(l.qty)}</span>
                <button
                  onClick={() => setQty(i, l.qty + 1)}
                  className="h-7 w-7 rounded-control border border-line text-sm font-bold"
                >
                  +
                </button>
              </div>
            </li>
          ))}
        </ul>

        {/* **ملاحظات المطعم مع الأصناف لا في عمود الدفع.**
            هي تخصّ ما يُطبَخ لا ما يُدفَع — ومكانُها بجانب ما تصفه. وكانت
            «ملاحظات عامّة» في آخر عمود الدفع، فتُقرأ ملاحظةً على الطلب كلِّه
            (العنوان؟ الوقت؟) لا على الطعام. */}
        <div className="mt-3 border-t border-line pt-3">
          <Input
            id="notes"
            label={m.site.cart.notes}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder={m.site.cart.notesHint}
          />
        </div>
        </div>

        <dl className="mt-4 space-y-1 surface p-4 text-sm">
          <div className="flex justify-between">
            <dt className="text-ink-muted">{m.site.cart.subtotal}</dt>
            <dd className="font-medium">
              {fmtNum(subtotal)} {m.common.currency}
            </dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-ink-muted">{m.site.cart.delivery}</dt>
            <dd className="font-medium">
              {zone ? `${fmtNum(zone.delivery_fee)} ${m.common.currency}` : "—"}
            </dd>
          </div>
          {/* **ورقمان يقولان سببَ الزيادة.**

              رقمٌ واحدٌ يرتفع بلا تفسير يُقرأ زيادةً بلا سبب، **ورقمان يقولان
              «هذا للمنطقة وهذا لأنك اخترت من مكانين بعيدين» يُقرآن حساباً.**

              **ولا يُعرض حين يكون صفراً**: سطرٌ بصفرٍ يلفت إلى ما لا يعني. */}
          {zone && zone.sources_fee > 0 && (
            <div className="flex justify-between text-xs">
              <dt className="text-ink-muted">{m.site.cart.extraSource}</dt>
              <dd className="text-ink-muted">
                {fmtNum(zone.sources_fee)} {m.common.currency}
              </dd>
            </div>
          )}
          {/* **وتجاوزُ السقف يُقال في السلّة لا عند الدفع.** */}
          {zone?.too_many_sources && (
            <Alert tone="warning">
              {m.site.cart.tooManySources.replace("{n}", fmtNum(zone.max_sources))}
            </Alert>
          )}
          <div className="flex justify-between border-t border-line pt-1 text-base">
            <dt className="font-bold">{m.site.cart.total}</dt>
            <dd className="font-bold text-primary-dark">
              {zone ? fmtNum(subtotal + zone.delivery_fee) : fmtNum(subtotal)}{" "}
              {m.common.currency}
            </dd>
          </div>
        </dl>
      </section>

      <section>
        <h2 className="mb-4 text-xl font-bold">{m.site.cart.checkout}</h2>

        {!logged && !loading ? (
          <div className="surface p-6 text-center">
            <p className="mb-3 text-ink-muted">{m.site.cart.loginFirst}</p>
            <Link href="/login?next=/cart">
              <Button>{m.site.nav.login}</Button>
            </Link>
          </div>
        ) : (
          <div className="space-y-4">
            {/* العناوين المحفوظة أولاً: من حفظ عنوانه لا يُطلب منه رسم دبّوسه ثانيةً.
                والاختيار يملأ الحقلين والدبّوس معاً. */}
            {saved.length > 0 && (
              <div>
                <p className="mb-2 text-xs font-medium">{m.site.addresses.title}</p>
                <AddressBook
                  api={api}
                  selectedID={pickedID}
                  onPick={(a) => {
                    setPickedID(a.id);
                    setAddress(a.address_text);
                    setLat(a.lat);
                    setLng(a.lng);
                  }}
                />
              </div>
            )}
            <Input
              id="address"
              label={m.site.cart.address}
              required
              value={address}
              // تعديلُ النصّ يُلغي علامة العنوان المحفوظ: العلامةُ على عنوانٍ
              // لم يعد هو المستعمَل **تقول للزبون غيرَ الواقع**.
              onChange={(e) => {
                setAddress(e.target.value);
                setPickedID("");
              }}
              placeholder={m.site.cart.addressPlaceholder}
            />
            <div>
              <p className="mb-1.5 text-xs text-ink-muted">{m.site.cart.pinHint}</p>
              <div className="overflow-hidden rounded-control border border-line">
                <PickMap
                  lat={lat}
                  lng={lng}
                  onPick={(la, ln) => {
                    setLat(la);
                    setLng(ln);
                  }}
                />
              </div>
              {zone && (
                <p className="mt-1.5 text-xs text-success">
                  {/* **واسمُ المنطقة لم يعد يُعرض.**

                      الرسمُ لم يعد رسمَ المنطقة وحدَه: **فيه أساسُها وفيه ما
                      زِيد للمصادر.** ونسبتُه إلى المنطقة وحدَها **تجعل الزبونَ
                      يقارنه برسم جاره فيجده أعلى ولا يعرف لماذا.** */}
                  {m.site.cart.feeLine.replace(
                    "{fee}",
                    `${fmtNum(zone.delivery_fee)} ${m.common.currency}`,
                  )}
                  {/* لا حدّ أدنى في هذه المنصة — رسم التوصيل كاملٌ من الزبون
                      مهما كانت قيمة طلبه، فلا شأن للمنصة بها. */}
                </p>
              )}
              {zoneErr && <p className="mt-1.5 text-xs text-danger">{zoneErr}</p>}
            </div>

            <Select
              id="payment"
              label={m.site.cart.payment}
              value={payment}
              onChange={(e) => setPayment(e.target.value)}
            >
              <option value="cash">{m.orders.payment.cash}</option>
              <option value="wallet">{m.orders.payment.wallet}</option>
              {/* لا «مختلط»: أُلغي من المحرّك — مصدرا دفعٍ ومسارا تسويةٍ
                  ومسارا استرجاع لطلبٍ واحد. **وخيارٌ يُعرض ويرفضه الخادم أسوأ
                  من خيارٍ غائب**: من اختاره ظنّ أن النظام انكسر. */}
            </Select>
            {balance != null && payment !== "cash" && (
              <p className="text-xs text-ink-muted">
                {m.site.cart.walletBalance.replace(
                  "{v}",
                  `${fmtNum(balance)} ${m.common.currency}`
                )}
              </p>
            )}

            <div>
              <Input
                id="promo"
                label={m.site.cart.promo}
                dir="ltr"
                value={promo}
                onChange={(e) => setPromo(e.target.value.toUpperCase())}
                className="text-center font-mono uppercase"
              />
              <p className="mt-1 text-xs text-ink-muted">{m.site.cart.promoHint}</p>
            </div>
            {error && (
              <Alert>{error}</Alert>
            )}

            {/* التوثيق يُقال **قبل** الملء لا عند الرفض: من ملأ سلّته ثم رُدّ
                يشعر أنه خُدع، ومن عرف أوّلاً يوثّق ويمضي. */}
            {waVerified === false && (
              <div className="rounded-card border border-warning-edge bg-warning-tint p-4">
                <p className="flex items-center gap-2 font-medium text-warning">
                  <IconWhatsApp size={17} />
                  {m.site.cart.waTitle}
                </p>
                <p className="mt-1 text-xs leading-relaxed text-ink-muted">
                  {m.site.cart.waHint}
                </p>
                <Link
                  href="/account"
                  className="mt-3 inline-flex items-center gap-1.5 rounded-control bg-warning px-4 py-2 text-sm font-medium text-on-solid"
                >
                  <IconWhatsApp size={15} />
                  {m.site.cart.waAction}
                </Link>
              </div>
            )}

            <Button
              onClick={placeOrder}
              disabled={!canPlace}
              className="w-full py-3 text-base"
            >
              {busy ? m.site.cart.placing : m.site.cart.placeOrder}
            </Button>
          </div>
        )}
      </section>

      {/* **وشريطُ الحسم على الجوّال وحدَه** — على الواسع الملخّصُ والزرُّ
          في العمود الثاني، **وكلاهما مرئيٌّ بلا تمرير.** */}
      {logged && cart.lines.length > 0 && (
        <div
          className="elev-4 fixed inset-x-0 bottom-[3.75rem] z-30 flex items-center gap-3 border-t border-line bg-raised px-4 py-3 md:hidden"
          style={{ marginBottom: "env(safe-area-inset-bottom)" }}
        >
          <div className="min-w-0">
            <p className="text-2xs text-ink-muted">{m.site.cart.total}</p>
            <p dir="ltr" className="font-bold tabular-nums text-primary-strong">
              {fmtNum(stickyTotal)} {m.common.currency}
            </p>
          </div>
          {/* **والزرُّ يُعطَّل بالسبب نفسِه** — ولو اختلفا لَظهر زرٌّ يُضغط
              ولا يقع شيء. */}
          <Button onClick={placeOrder} disabled={!canPlace} className="ms-auto shrink-0 px-6 py-3">
            {busy ? m.site.cart.placing : m.site.cart.placeOrder}
          </Button>
        </div>
      )}
      {/* **وفراغٌ بارتفاع الشريطين** — وبلاه يختفي زرُّ الصفحة خلفهما. */}
      {logged && cart.lines.length > 0 && <div aria-hidden className="h-20 md:hidden" />}
    </div>
  );
}
