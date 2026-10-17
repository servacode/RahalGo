"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/wallet — المحفظةُ وطريقةُ استلام المستحقّات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `WalletScreen(payouts = true)` وبطاقةِ «كيف تحب تستلم مستحقاتك؟»
 *  في `StoreScreen.kt`.)
 *
 * **والطريقةُ تُبدَّل بسؤال** — «ينطبق على الطلبات الجديدة فقط» — كما في
 * التطبيق. **وما لك نقداً من الطلبات السابقة** يُعرض تحتها ما دامت نقديّة.
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Confirm, WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useStore } from "@/lib/store/context";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const S = m.store;

function SettlementCard() {
  const { store, patch } = useStore();
  const toast = useShopToast();
  const [ask, setAsk] = useState<"" | "cash" | "wallet">("");
  const [busy, setBusy] = useState(false);
  if (!store) return null;
  const confirmed = store.settlement_confirmed;

  async function save(method: "cash" | "wallet") {
    if (!store) return;
    setBusy(true);
    try {
      const r = await api<{ settlement_method: string }>(
        `/api/v1/merchant/stores/${store.id}/settlement-method`,
        { method: "PATCH", body: JSON.stringify({ method }) },
      );
      patch({ settlement_method: r.settlement_method, settlement_confirmed: true });
      toast(S.settlement.saved, "success");
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
      setAsk("");
    }
  }

  const options: { key: "cash" | "wallet"; label: string }[] = [
    { key: "cash", label: S.settlement.cash },
    { key: "wallet", label: S.settlement.wallet },
  ];

  return (
    <section className="surface flex flex-col gap-3 rounded-card p-4" aria-label={S.settlement.title}>
      <h3 className="heading-card">{S.settlement.q}</h3>
      <div className="grid grid-cols-2 gap-2">
        {options.map((o) => {
          const on = confirmed && store.settlement_method === o.key;
          return (
            <button
              key={o.key}
              type="button"
              aria-pressed={on}
              disabled={busy || store.status === "suspended"}
              onClick={() => {
                if (!confirmed || store.settlement_method !== o.key) setAsk(o.key);
              }}
              className={`rounded-control border px-3 py-3 text-sm font-bold transition-colors disabled:opacity-60 ${
                on ? "border-2 border-accent bg-accent-tint" : "border-line bg-field hover:bg-row-hover"
              }`}
            >
              {o.label}
            </button>
          );
        })}
      </div>
      {store.settlement_method === "cash" && store.unpaid_cash_due > 0 && (
        <p className="flex items-center justify-between text-sm">
          <span>{S.settlement.unpaid}</span>
          <span className="font-bold text-accent-text">{fmtMoney(store.unpaid_cash_due)}</span>
        </p>
      )}
      <Confirm
        open={!!ask}
        title={S.settlement.confirmTitle}
        body={S.settlement.confirmBody}
        confirmLabel={S.settlement.confirmYes}
        tone="primary"
        busy={busy}
        onConfirm={() => ask && void save(ask)}
        onCancel={() => setAsk("")}
      />
    </section>
  );
}

export default function StoreWallet() {
  const { user } = useAuth();
  return (
    <div className="flex flex-col gap-4">
      <h2 className="heading-page">{S.wallet.title}</h2>
      <SettlementCard />
      <WalletPage
        api={api}
        path="/api/v1/my/wallet"
        balanceLabel={S.wallet.balance}
        holderName={user?.full_name}
        holderPhone={user?.phone}
        payouts
        cardsOnly
      />
    </div>
  );
}
