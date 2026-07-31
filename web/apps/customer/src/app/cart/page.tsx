"use client";

/** السلة والدفع: عنوان بدبوس، معاينة رسوم المنطقة، دفع نقدي/محفظة/مختلط، كود خصم. */

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  IconEdit,
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

interface Zone {
  name: string;
  delivery_fee: number;
  min_order: number;
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
  const [zone, setZone] = useState<Zone | null>(null);
  const [zoneErr, setZoneErr] = useState("");
  const [payment, setPayment] = useState("cash");
  const [promo, setPromo] = useState("");
  const [notes, setNotes] = useState("");
  const [balance, setBalance] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const logged = isLoggedIn(user);

  useEffect(() => {
    if (!logged) return;
    api<{ balance: number }>("/api/v1/my/wallet")
      .then((s) => setBalance(s.balance))
      .catch(() => undefined);
  }, [logged]);

  const quote = useCallback(async () => {
    try {
      setZone(await api<Zone>(`/api/v1/public/zone?lat=${lat}&lng=${lng}`));
      setZoneErr("");
    } catch (err) {
      setZone(null);
      setZoneErr(errText(err));
    }
  }, [lat, lng]);

  useEffect(() => {
    const t = setTimeout(quote, 400);
    return () => clearTimeout(t);
  }, [quote]);

  if (!cart || cart.lines.length === 0) {
    return (
      <div className="py-16 text-center text-ink-muted">
        <p className="mb-4">{m.site.cart.empty}</p>
        <Link href="/" className="font-medium text-primary hover:underline">
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
          merchant_id: cart.merchant_id,
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
      router.push(`/orders/${o.id}?placed=1`);
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <section>
        <h1 className="mb-4 text-xl font-bold">
          {m.site.cart.title}{" "}
          <span className="text-sm font-normal text-ink-muted">
            {m.site.cart.from} {cart.merchant_name}
          </span>
        </h1>
        <ul className="space-y-2">
          {cart.lines.map((l, i) => (
            <li
              key={i}
              className="flex items-center gap-3 rounded-card border border-line bg-surface p-3"
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

        <dl className="mt-4 space-y-1 rounded-card border border-line bg-surface p-4 text-sm">
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
          <div className="rounded-card border border-line bg-surface p-6 text-center">
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
              onChange={(e) => setAddress(e.target.value)}
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
                  {m.site.cart.zoneFee
                    .replace("{name}", zone.name)
                    .replace("{fee}", `${fmtNum(zone.delivery_fee)} ${m.common.currency}`)}
                  {zone.min_order > 0 &&
                    ` — ${m.site.cart.minOrder.replace("{v}", `${fmtNum(zone.min_order)} ${m.common.currency}`)}`}
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
              <option value="mixed">{m.orders.payment.mixed}</option>
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
            <Input
              id="notes"
              label={m.site.cart.notes}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />

            {error && (
              <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
            )}
            <Button
              onClick={placeOrder}
              disabled={busy || !address || !zone}
              className="w-full py-3 text-base"
            >
              {busy ? m.site.cart.placing : m.site.cart.placeOrder}
            </Button>
          </div>
        )}
      </section>
    </div>
  );
}
