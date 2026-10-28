"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/account — «حسابي» داخل لوحة المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `AccountScreen(worker = true)` في تطبيق المتجر.)
 *
 * **إعداداتُ الحساب المشتركةُ نفسُها** (`AccountSettings`): الاسمُ والصورة ·
 * الواتساب · تغييرُ الرقم · **كلمةُ المرور** · حذفُ الحساب.
 *
 * **وفوقها مفتاحُ «صوت الطلبات»** — يُحفظ لهذا الجهاز وحدَه، وافتراضُه
 * «شغّال». **وتحتها «تسوّق كزبون»** لمن له الدوران، **و«خروج».**
 */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AccountSettings, Button, ButtonLink, IconBag, IconLogout, Switch } from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import { hasRole, useAuth } from "@/lib/auth";
import { useStoreOrders } from "@/lib/store/orders";
import WebPushCard from "@/components/WebPushCard";

const m = getMessages(defaultLocale);
const A = m.store.account;

export default function StoreAccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const { soundEnabled, setSoundEnabled } = useStoreOrders();
  // **شارةُ الدور** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — «صاحب المتجر» أو «موظف»، من ردّ المحرّك:
  // بابُ الموظّفين يردّ الموظّفَ `staff_not_allowed` (كتطبيق المتجر `TeamAccess`).
  const [role, setRole] = useState<"" | "owner" | "staff">("");
  useEffect(() => {
    api("/api/v1/merchant/staff")
      .then(() => setRole("owner"))
      .catch((e) => {
        if (e instanceof ApiError && e.body.code === "staff_not_allowed") setRole("staff");
      });
  }, []);
  const out = () => {
    logout();
    router.replace("/login");
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="heading-page">{A.title}</h2>
        {role && (
          <span
            className={`rounded-full px-3 py-1 text-xs font-bold ${
              role === "owner" ? "bg-accent-tint text-accent" : "bg-field text-ink"
            }`}
          >
            {role === "owner" ? A.roleOwner : A.roleStaff}
          </span>
        )}
      </div>

      <section className="surface rounded-card p-4" aria-label={A.sound} data-testid="store-sound">
        <Switch
          id="store-sound"
          checked={soundEnabled}
          onChange={setSoundEnabled}
          label={A.sound}
          hint={soundEnabled ? A.soundOn : A.soundOff}
        />
      </section>

      <WebPushCard app="merchant" />

      <AccountSettings api={api} mediaUrl={mediaUrl} phone={user?.phone} onDeleted={out} whatsapp={false} />

      {user && hasRole(user, "customer") && (
        <ButtonLink href="/shop" variant="secondary" className="flex items-center justify-center gap-2 py-3">
          <IconBag size={16} />
          {A.shop}
        </ButtonLink>
      )}

      <Button variant="danger" className="flex items-center justify-center gap-2 py-3" onClick={out} data-testid="account-logout">
        <IconLogout size={16} />
        {A.logout}
      </Button>
    </div>
  );
}
