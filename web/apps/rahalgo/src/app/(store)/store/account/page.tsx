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
import {
  AccountSettings,
  Badge,
  Button,
  ButtonLink,
  IGroup,
  IRow,
  IconWarning,
  IconBag,
  IconLogout,
  IconPhone,
  Toggle,
  useConsoleTheme,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useTeamRole } from "@/lib/store/team";
import { hasRole, useAuth } from "@/lib/auth";
import { useStoreOrders } from "@/lib/store/orders";
import WebPushCard from "@/components/WebPushCard";
import { PageHeader } from "@/components/store/SetupParts";
import { useStoreNav } from "@/components/store/navModel";

const m = getMessages(defaultLocale);
const A = m.store.account;
const SH = m.store.shell;
const N = m.store.nav;

/** **ألوانُ مربّعات آيفون لكلّ مجموعة** — لتفريق التشغيل عن المال عن الحساب بنظرة. */
const TONES = { ops: "info", money: "good", account: "violet" } as const;

export default function StoreAccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const { soundEnabled, setSoundEnabled } = useStoreOrders();
  const { theme, toggle } = useConsoleTheme();
  const nav = useStoreNav("/store/account");
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

      {/* ── **جوّالاً «حسابي» مركزُ الأبواب** — ما ليس في شريط التبويب (المحفظة · التقارير · السجلّ · العروض · التوصيلات…)
          كقائمة إعدادات آيفون. وعلى الحاسوب السايدبارُ يغني. ── */}
      <div className="flex flex-col gap-1 md:max-lg:block md:max-lg:columns-2 md:max-lg:gap-4 md:max-lg:[&>*]:mb-4 md:max-lg:[&>*]:break-inside-avoid lg:hidden" data-testid="store-account-hub">
        {nav.groups.map((g) => {
          const items = g.items.filter((it) => !["/store", "/store/orders", "/store/menu", "/store/account"].includes(it.href));
          const extra = g.key === "account" ? [{ href: "/store/order-phones", label: N.orderPhones, icon: IconPhone }, { href: "/store/alerts", label: N.alerts, icon: IconWarning }] : [];
          const rows = [...items, ...extra];
          if (rows.length === 0) return null;
          return (
            <IGroup key={g.key} title={g.title}>
              {rows.map((it) => (
                <IRow key={it.href} icon={it.icon} tone={TONES[g.key as keyof typeof TONES]} title={it.label} href={it.href} Link={Link} />
              ))}
            </IGroup>
          );
        })}
      </div>

      {/* ── الحاسوب: صفُّ تفضيلات (صوت · داكن · إشعارات) ثمّ بطاقاتُ الحساب بعرض الصفحة كاملاً، وتحتها الخروج. ── */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3 md:items-stretch md:gap-5">
        <section className="panel p-4" aria-label={A.sound} data-testid="store-sound">
          <Toggle
            id="store-sound"
            checked={soundEnabled}
            onChange={setSoundEnabled}
            label={A.sound}
            hint={soundEnabled ? A.soundOn : A.soundOff}
          />
        </section>

        <section className="panel p-4" aria-label={SH.darkMode}>
          <Toggle id="store-dark" checked={theme === "dark"} onChange={() => toggle()} label={SH.darkMode} hint={SH.darkModeHint} />
        </section>

        <div className="min-w-0 [&>*]:h-full">
          <WebPushCard app="merchant" />
        </div>
      </div>

      <AccountSettings api={api} mediaUrl={mediaUrl} phone={user?.phone} onDeleted={out} whatsapp={false} />

      <div className="flex flex-col gap-3 sm:flex-row sm:justify-end">
        {user && hasRole(user, "customer") && (
          <ButtonLink Link={Link} href="/shop" variant="secondary" className="flex min-h-11 items-center justify-center gap-2 px-5 py-3">
            <IconBag size={16} />
            {A.shop}
          </ButtonLink>
        )}
        <Button variant="danger" className="flex min-h-11 items-center justify-center gap-2 px-5 py-3" onClick={out} data-testid="account-logout">
          <IconLogout size={16} />
          {A.logout}
        </Button>
      </div>
    </div>
  );
}
