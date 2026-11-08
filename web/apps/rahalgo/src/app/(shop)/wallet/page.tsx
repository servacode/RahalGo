"use client";

/**
 * **محفظتي — الرصيدُ والحركاتُ والكشف** (مرآةُ `WalletScreen` بلا سحب).
 *
 * **والزبونُ لا يسحب** — رصيدُه يُنفَق لا يُقبَض (قرارُ المالك ٢٠٢٦-٠٨-١٤).
 * **والشاشةُ شاشةُ العُدّة نفسُها** (`WalletPage`) — بطاقاتٌ لا جدول.
 */

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Alert, Button, Input, LoadingState, WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { abortPay, goPay, openPayTab, statusPath, useShamConfig, type ShamConfig, type ShamStart } from "@/lib/customer/shamcash";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const m = getMessages(defaultLocale);
const W = m.shop.wallet;
const SC = m.shop.shamcash;

/** **شحنُ المحفظة من شام كاش** — يظهر فقط حين يكون مفعّلاً. نافذةُ الدفع تُفتح داخل الضغطة. */
function ShamTopup({ cfg }: { cfg: ShamConfig }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const value = Math.round(Number(amount));
  const valid = Number.isFinite(value) && value >= cfg.min && value <= cfg.max;

  const submit = async () => {
    if (!valid || busy) return;
    const w = openPayTab();
    setBusy(true);
    setError("");
    try {
      const r = await api<ShamStart>("/api/v1/my/shamcash/topups", {
        method: "POST",
        body: JSON.stringify({ amount: value, return: "web" }),
      });
      if (goPay(w, r.payment_url)) router.push(statusPath(r.bill.id));
    } catch (err) {
      abortPay(w);
      setError(errorText(err));
      setBusy(false);
    }
  };

  if (!open) {
    return (
      <Button variant="secondary" className="w-full" onClick={() => setOpen(true)}>
        {SC.topupBtn}
      </Button>
    );
  }
  return (
    <div className="surface flex flex-col gap-3 p-4">
      <h2 className="heading-card">{SC.topupTitle}</h2>
      <Input
        id="sham-amount"
        label={SC.amountLabel}
        type="number"
        inputMode="numeric"
        dir="ltr"
        min={cfg.min}
        max={cfg.max}
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
      />
      <p className="text-xs text-ink-muted">
        {SC.rangeHint.replace("{min}", fmtMoney(cfg.min)).replace("{max}", fmtMoney(cfg.max))}
      </p>
      {error && <Alert>{error}</Alert>}
      <div className="flex gap-2">
        <Button className="flex-1" onClick={() => void submit()} disabled={!valid || busy}>
          {SC.submit}
        </Button>
        <Button variant="secondary" onClick={() => setOpen(false)} disabled={busy}>
          {SC.cancel}
        </Button>
      </div>
    </div>
  );
}

export default function WalletRoute() {
  const { user, loading } = useAuth();
  const sham = useShamConfig(!!user);
  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-4">
      <ShopPageHeader title={W.title} />
      {sham && <ShamTopup cfg={sham} />}
      <WalletPage
        api={api}
        path="/api/v1/my/wallet"
        balanceLabel={W.balance}
        holderName={user.full_name}
        holderPhone={user.phone}
        cardsOnly
      />
    </div>
  );
}
