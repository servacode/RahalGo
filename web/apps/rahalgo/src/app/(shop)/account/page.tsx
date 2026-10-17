"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حسابي — اسمُه ورقمُه ومحفظتُه وعناوينُه، والخروج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أساسيّاتُ المرحلة الأولى** (قرارُ المالك ٢٠٢٦-١٠-١٧). **والزائرُ يرى
 * بابَ الدخول وروابطَ المنصّة** — كما في وضع الزائر بأندرويد.
 */

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  FormSection,
  Input,
  LoadingState,
  Money,
  IconLocation,
  IconLogout,
  IconSmartphone,
  IconUser,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCustomer } from "@/lib/customer/context";
import { AddressSheet } from "@/components/customer/AddressSheet";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const A = m.shop.account;

function Links() {
  const items = [
    ["/download/customer", A.androidApp],
    ["/help", A.help],
    ["/contact", A.contact],
    ["/terms", A.terms],
    ["/privacy", A.privacy],
  ] as const;
  return (
    <nav className="surface flex flex-col divide-y divide-line-soft">
      {items.map(([href, label]) => (
        <Link key={href} href={href} className="flex items-center gap-2 px-4 py-3 text-sm hover:bg-row-hover">
          {href === "/download/customer" && <IconSmartphone size={16} className="text-accent-text" />}
          {label}
        </Link>
      ))}
    </nav>
  );
}

export default function AccountPage() {
  const { user, loading, logout, setUser } = useAuth();
  const { wallet, addresses, address, reloadWallet } = useCustomer();
  const router = useRouter();
  const toast = useShopToast();
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [addr, setAddr] = useState(false);

  useEffect(() => setName(user?.full_name ?? ""), [user?.full_name]);
  useEffect(() => {
    if (user) reloadWallet();
  }, [user, reloadWallet]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user)
    return (
      <div className="flex flex-col gap-4">
        <NeedLogin />
        <Links />
      </div>
    );

  const saveName = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/me/name", { method: "PATCH", body: JSON.stringify({ full_name: name.trim() }) });
      setUser({ ...user, full_name: name.trim() });
      toast(A.nameSaved);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <h1 className="heading-page">{A.title}</h1>

      <FormSection title={A.wallet} icon={<IconWallet />}>
        {wallet ? (
          <div className="flex flex-col gap-1">
            <Money value={wallet.available ?? wallet.balance} className="figure text-accent-text" />
            <p className="text-xs text-ink-muted">{A.walletAvailable}</p>
            {(wallet.reserved ?? 0) > 0 && (
              <p className="text-xs text-ink-muted">
                {A.walletReserved}: <Money value={wallet.reserved ?? 0} small />
              </p>
            )}
          </div>
        ) : (
          <LoadingState variant="inline" />
        )}
      </FormSection>

      <FormSection title={A.profile} icon={<IconUser />}>
        <form onSubmit={saveName} className="flex flex-col gap-3">
          <Input id="acc-name" label={A.name} value={name} onChange={(e) => setName(e.target.value)} required />
          <Input id="acc-phone" label={A.phone} value={user.phone} dir="ltr" readOnly className="text-end" />
          {error && <Alert>{error}</Alert>}
          <Button type="submit" disabled={busy || !name.trim() || name.trim() === user.full_name} className="w-fit">
            {A.saveName}
          </Button>
        </form>
      </FormSection>

      <FormSection title={A.addresses} icon={<IconLocation />}>
        <button type="button" onClick={() => setAddr(true)} className="flex w-full flex-col gap-1 text-start">
          {addresses === null ? (
            <LoadingState variant="inline" />
          ) : (
            <>
              <span className="text-sm">{address ? address.address_text : m.shop.address.none}</span>
              <span className="text-xs text-accent-text">{m.shop.address.change}</span>
            </>
          )}
        </button>
      </FormSection>

      <Links />

      <Button
        variant="danger"
        className="flex items-center justify-center gap-2 py-3"
        onClick={() => {
          logout();
          router.replace("/shop");
        }}
      >
        <IconLogout size={16} />
        {A.logout}
      </Button>
      <AddressSheet open={addr} onClose={() => setAddr(false)} />
    </div>
  );
}
