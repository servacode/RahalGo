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

import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AccountSettings, Badge, Button, ButtonLink, IconBag, IconLogout, Switch } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useTeamRole } from "@/lib/store/team";
import { hasRole, useAuth } from "@/lib/auth";
import { useStoreOrders } from "@/lib/store/orders";
import WebPushCard from "@/components/WebPushCard";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const A = m.store.account;

export default function StoreAccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const { soundEnabled, setSoundEnabled } = useStoreOrders();
  // **شارةُ الدور** (قرارُ المالك ٢٠٢٦-١٠-٢٩) — «صاحب المتجر» أو «موظف»، من ردّ المحرّك
  // (`useTeamRole`، مرآةُ `TeamAccess`). **ولا رأسَ باسمٍ ورقمٍ ومحفظة** — كـ«حسابي» في التطبيق.
  const role = useTeamRole();
  const out = () => {
    logout();
    router.replace("/login");
  };

  return (
    <div className="console-page" data-testid="store-account">
      <PageHeader
        title={A.title}
        actions={
          role && (
            <Badge variant={role === "owner" ? "primary" : "neutral"} className="px-3 py-1 text-sm font-bold">
              <span data-testid="account-role">{role === "owner" ? A.roleOwner : A.roleStaff}</span>
            </Badge>
          )
        }
      />

      {/* ── عمودان على الحاسوب: التنبيهات والجلسة | بيانات الحساب. جوّالاً متتابعان ── */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 lg:items-start lg:gap-5">
        <div className="flex min-w-0 flex-col gap-4">
          <section className="panel p-4" aria-label={A.sound} data-testid="store-sound">
            <Switch
              id="store-sound"
              checked={soundEnabled}
              onChange={setSoundEnabled}
              label={A.sound}
              hint={soundEnabled ? A.soundOn : A.soundOff}
            />
          </section>

          <WebPushCard app="merchant" />

          {user && hasRole(user, "customer") && (
            <ButtonLink Link={Link} href="/shop" variant="secondary" className="flex min-h-11 items-center justify-center gap-2 py-3">
              <IconBag size={16} />
              {A.shop}
            </ButtonLink>
          )}

          <Button variant="danger" className="flex min-h-11 items-center justify-center gap-2 py-3" onClick={out} data-testid="account-logout">
            <IconLogout size={16} />
            {A.logout}
          </Button>
        </div>

        <div className="min-w-0">
          <AccountSettings api={api} mediaUrl={mediaUrl} phone={user?.phone} onDeleted={out} whatsapp={false} />
        </div>
      </div>
    </div>
  );
}
