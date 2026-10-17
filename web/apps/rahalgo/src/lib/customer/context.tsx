"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ الزبون المشتركة — المنصّةُ وعنوانُ التوصيل والمحفظة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ثلاثةٌ تقرؤها شاشاتٌ كثيرة** — الشريطُ والسلّةُ والطلبات. **ومن جلبها
 * كلُّ شاشةٍ بنفسها جلبها ثلاثَ مرّاتٍ وعرض ثلاثةَ أرقام.**
 *
 * - **المنصّة** (`/public/platform`): أيُستقبَل الطلبُ الآن؟ وأيُطلب رمزٌ
 *   عند التسجيل؟ — **وتُعاد قراءتُها كلَّ دقيقتين** كما يفعل `Serving.kt`.
 * - **العنوانُ المختار**: يُحفظ في المتصفّح، **والافتراضيُّ من المحرّك إن
 *   لم يُختر شيء.**
 * - **المحفظة**: الرصيدُ المتاح — ومنه يُعرف أتكفي للطلب.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Address, CustomerPlatform, Wallet } from "./types";
import { readJSON, writeJSON } from "./storage";

const PICK_KEY = "rahalgo_delivery_address";
const FRESH_MS = 2 * 60 * 1000;

interface CustomerState {
  platform: CustomerPlatform | null;
  refreshPlatform: () => void;
  /** **عناوينُه** — و`null` قبل أوّل قراءة. */
  addresses: Address[] | null;
  addressesError: unknown;
  reloadAddresses: () => Promise<Address[]>;
  address: Address | null;
  pickAddress: (id: string) => void;
  wallet: Wallet | null;
  reloadWallet: () => void;
  /** **الرصيدُ المتاح للدفع** — المتاحُ إن أُرسل، وإلّا الرصيد. */
  available: number;
}

const Ctx = createContext<CustomerState | null>(null);

function normalizePlatform(raw: Record<string, unknown>): CustomerPlatform {
  const ord = (raw.ordering ?? {}) as Record<string, unknown>;
  const launch = (raw.launch ?? {}) as Record<string, unknown>;
  const social = (raw.social ?? {}) as Record<string, unknown>;
  const cd = (raw.custom_delivery ?? {}) as Record<string, unknown>;
  return {
    ordering: {
      ordering_available: ord.ordering_available !== false,
      reason: typeof ord.reason === "string" ? ord.reason : "",
      message: typeof ord.message === "string" ? ord.message : "",
      next_available_at: typeof ord.next_available_at === "string" ? ord.next_available_at : "",
    },
    launch: {
      customer_signup: launch.customer_signup !== false,
      customer_browse: launch.customer_browse !== false,
      customer_orders: launch.customer_orders !== false,
      customer_custom_orders: launch.customer_custom_orders !== false,
      notice: typeof launch.notice === "string" ? launch.notice : "",
    },
    signup_verify: raw.signup_verify === true,
    password_min_length:
      typeof raw.password_min_length === "number" && raw.password_min_length > 0
        ? raw.password_min_length
        : 8,
    social: { whatsapp: typeof social.whatsapp === "string" ? social.whatsapp : "" },
    custom_delivery: {
      source: typeof cd.source === "string" ? cd.source : "driver_defined",
      fee: typeof cd.fee === "number" ? cd.fee : 0,
      fee_amanah: typeof cd.fee_amanah === "number" ? cd.fee_amanah : 0,
    },
    otp_login: raw.otp_login === true,
  };
}

export function CustomerProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const [platform, setPlatform] = useState<CustomerPlatform | null>(null);
  const [addresses, setAddresses] = useState<Address[] | null>(null);
  const [addressesError, setAddressesError] = useState<unknown>(null);
  const [picked, setPicked] = useState<string>("");
  const [wallet, setWallet] = useState<Wallet | null>(null);

  const refreshPlatform = useCallback(() => {
    api<Record<string, unknown>>("/api/v1/public/platform")
      .then((raw) => setPlatform(normalizePlatform(raw)))
      .catch(() => undefined); // **ومنصّةٌ لم تُقرأ لا تُغلق شيئاً** — والمحرّكُ يحرس الطلب.
  }, []);

  useEffect(() => {
    refreshPlatform();
    const t = setInterval(refreshPlatform, FRESH_MS);
    return () => clearInterval(t);
  }, [refreshPlatform]);

  useEffect(() => setPicked(readJSON<string>(PICK_KEY, "")), []);

  const reloadAddresses = useCallback(async () => {
    try {
      const list = await api<Address[]>("/api/v1/my/addresses");
      const safe = Array.isArray(list) ? list : [];
      setAddresses(safe);
      setAddressesError(null);
      return safe;
    } catch (err) {
      setAddressesError(err);
      setAddresses((cur) => cur ?? []);
      return [];
    }
  }, []);

  const reloadWallet = useCallback(() => {
    api<Wallet>("/api/v1/my/wallet")
      .then(setWallet)
      .catch(() => setWallet(null));
  }, []);

  useEffect(() => {
    if (!user) {
      setAddresses(null);
      setWallet(null);
      return;
    }
    void reloadAddresses();
    reloadWallet();
  }, [user, reloadAddresses, reloadWallet]);

  const pickAddress = useCallback((id: string) => {
    setPicked(id);
    writeJSON(PICK_KEY, id);
  }, []);

  const address = useMemo(() => {
    if (!addresses || addresses.length === 0) return null;
    return (
      addresses.find((a) => a.id === picked) ?? addresses.find((a) => a.is_default) ?? addresses[0] ?? null
    );
  }, [addresses, picked]);

  const available = wallet ? (wallet.available ?? wallet.balance) : 0;

  const value = useMemo<CustomerState>(
    () => ({
      platform,
      refreshPlatform,
      addresses,
      addressesError,
      reloadAddresses,
      address,
      pickAddress,
      wallet,
      reloadWallet,
      available,
    }),
    [platform, refreshPlatform, addresses, addressesError, reloadAddresses, address, pickAddress, wallet, reloadWallet, available],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useCustomer(): CustomerState {
  const c = useContext(Ctx);
  if (!c) throw new Error("useCustomer must be used inside CustomerProvider");
  return c;
}

/** **موضعُ التصفّح** — عنوانُ التوصيل إن وُجد، **فيُرشَّح السوقُ بمدينته.** */
export function useBrowseQuery(sep: "?" | "&"): string {
  const { address } = useCustomer();
  if (!address) return "";
  return `${sep}lat=${address.lat}&lng=${address.lng}`;
}
