"use client";

/**
 * **حالُ الدفع بشام كاش** — `/wallet/shamcash/<id>`.
 *
 * **وهو نفسُه عنوانُ العودة من شام كاش** (`redirectUrl`): يُفتح طازجاً بعد الدخول، فيقرأ الفاتورةَ
 * من الخادم ويسأل كلَّ ٣ ثوانٍ ما دامت بانتظار الدفع. **والطلبُ يُنشئه الخادمُ بعد تأكّد الدفع لا قبله**،
 * فالسلّةُ لا تُمسح إلّا هنا حين يظهر رقمُ الطلب.
 */

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Alert, ButtonLink, LoadingState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCart } from "@/lib/customer/cart";
import { useCustomer } from "@/lib/customer/context";
import { readJSON, removeKey } from "@/lib/customer/storage";
import { SHAM_CART_KEY, type ShamBill } from "@/lib/customer/shamcash";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const m = getMessages(defaultLocale);
const S = m.shop.shamcash;
const POLL_MS = 3000;

export default function ShamcashStatusPage() {
  const { user, loading } = useAuth();
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const cart = useCart();
  const { reloadWallet } = useCustomer();
  const [bill, setBill] = useState<ShamBill | null>(null);
  const [error, setError] = useState("");

  const pending = !bill || bill.status === "pending";
  const uid = user?.id;

  useEffect(() => {
    if (!uid || !id || !pending) return;
    let alive = true;
    let timer = 0;
    const tick = async () => {
      try {
        const r = await api<{ bill: ShamBill }>(`/api/v1/my/shamcash/bills/${id}`);
        if (!alive) return;
        setBill(r.bill);
        setError("");
        if (r.bill.status !== "pending") return;
      } catch (err) {
        if (!alive) return;
        setError(errorText(err));
      }
      timer = window.setTimeout(() => void tick(), POLL_MS);
    };
    void tick();
    return () => {
      alive = false;
      window.clearTimeout(timer);
    };
  }, [uid, id, pending]);

  // **السلّةُ تُمسح حين أُنشئ الطلب فعلاً** — وللفاتورة التي خرجت من السلّة وحدَها.
  const orderMade = !!(bill && bill.purpose === "order" && bill.status !== "pending" && bill.order_id);
  useEffect(() => {
    if (!bill) return;
    if (bill.status === "paid" || bill.status === "partly_refunded" || bill.status === "refunded") reloadWallet();
    if (orderMade && readJSON<string>(SHAM_CART_KEY, "") === bill.id) {
      removeKey(SHAM_CART_KEY);
      cart.clear();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bill?.id, bill?.status, orderMade]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  const body = (() => {
    if (!bill) {
      return error ? <Alert>{error || S.loadFail}</Alert> : <LoadingState variant="list" rows={2} />;
    }
    const head = (title: string, hint?: string) => (
      <div className="flex flex-col gap-1">
        <h2 className="heading-card">{title}</h2>
        {hint && <p className="text-sm text-ink-muted">{hint}</p>}
        <p className="text-xs text-ink-muted">
          {S.billNo.replace("{n}", bill.number)} · {fmtMoney(bill.amount)}
        </p>
      </div>
    );
    if (bill.status === "pending") {
      return (
        <>
          {head(S.waiting, S.waitingHint)}
          {bill.payment_url && (
            <a href={bill.payment_url} target="_blank" rel="noopener noreferrer" className="text-sm font-bold text-primary underline">
              {S.reopen}
            </a>
          )}
        </>
      );
    }
    if (bill.status === "expired" || bill.status === "failed") {
      return (
        <>
          {head(bill.status === "expired" ? S.expired : S.failed, S.endedHint)}
          <div className="flex flex-wrap gap-2">
            <ButtonLink Link={Link} href={bill.purpose === "order" ? "/cart" : "/wallet"}>
              {S.retry}
            </ButtonLink>
          </div>
        </>
      );
    }
    // مدفوعة (أو مرجّعة بعد الدفع)
    const refunded = bill.status !== "paid" ? <p className="text-sm text-warning">{S.refunded}</p> : null;
    if (bill.purpose === "topup") {
      return (
        <>
          {head(S.paidTopup, S.paidTopupHint.replace("{amount}", fmtMoney(bill.amount)))}
          {refunded}
          <div>
            <ButtonLink Link={Link} href="/wallet">
              {S.toWallet}
            </ButtonLink>
          </div>
        </>
      );
    }
    if (bill.order_error) {
      return (
        <>
          {head(S.orderErrorTitle, S.orderErrorHint)}
          <Alert tone="warning">{bill.order_error}</Alert>
          {refunded}
          <div>
            <ButtonLink Link={Link} href="/cart">
              {S.toCart}
            </ButtonLink>
          </div>
        </>
      );
    }
    return (
      <>
        {head(S.paidOrder, bill.order_number != null ? S.orderNo.replace("{n}", String(bill.order_number)) : undefined)}
        {refunded}
        <div>
          <ButtonLink Link={Link} href={bill.order_id ? `/orders/${bill.order_id}` : "/orders"}>
            {S.toOrder}
          </ButtonLink>
        </div>
      </>
    );
  })();

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4">
      <ShopPageHeader title={S.statusTitle} />
      <div className="surface flex flex-col gap-3 p-4">{body}</div>
    </div>
  );
}
