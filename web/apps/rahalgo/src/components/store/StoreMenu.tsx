"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ اللوحة الجانبيّة — بنودُ درج تطبيق المتجر نفسُها وبترتيبها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `MERCHANT_ITEMS + PlatformPages.items` في `Menu.kt`، وزرِّ الخروج
 *  أسفلَ الدرج.)
 *
 * **«ما يخصني»**: العروض · التقارير · سجل الطلبات · سجل التوصيلات ·
 * الموظفون · أرقام الطلبات · الشكاوى · الإنذارات — **والمحفظةُ معها** (في التطبيق زرٌّ في
 * الشريط العلويّ، وهنا الشريطُ أضيق).
 *
 * **«المنصة»**: شرحُ التطبيق بالفيديو (رابطٌ خارجيّ من الإعدادات) ·
 * التعليمات · من نحن · تواصل معنا · الشروط · الخصوصية — **صفحاتُ الموقع
 * العامّة نفسُها**، فلا نسخةَ ثانيةً من نصٍّ قانونيّ.
 */

import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Drawer,
  IconApp,
  IconChat,
  IconLogout,
  IconMoto,
  IconPhone,
  IconPromos,
  IconReceipt,
  IconSupport,
  IconTrendUp,
  IconUsers,
  IconWallet,
  IconWarning,
} from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useTutorialUrl } from "@/lib/store/platform";
import { useTeamRole } from "@/lib/store/team";

const m = getMessages(defaultLocale);
const N = m.store.nav;

type Icon = typeof IconWarning;

const MINE: readonly (readonly [string, string, Icon])[] = [
  // **ترتيبُ مراجعة التصميم ٢٠٢٦-١٠-٢٩**: ما يجلب البيعَ أوّلاً، والإنذاراتُ آخراً.
  ["/store/offers", N.offers, IconPromos],
  ["/store/reports", N.reports, IconTrendUp],
  ["/store/history", N.history, IconReceipt],
  ["/store/deliveries", N.deliveries, IconMoto],
  // **والموظّفُ لا يرى بابَ الموظّفين.**
  ["/store/team", N.staff, IconUsers],
  ["/store/order-phones", N.orderPhones, IconPhone],
  ["/store/complaints", N.complaints, IconChat],
  ["/store/alerts", N.alerts, IconWarning],
  ["/store/wallet", N.wallet, IconWallet],
];

const PLATFORM: readonly (readonly [string, string])[] = [
  ["/help", N.help],
  ["/about", N.about],
  ["/contact", N.contact],
  ["/terms", N.terms],
  ["/privacy", N.privacy],
];

const ROW = "flex items-center gap-3 rounded-control px-3 py-3 text-sm hover:bg-row-hover";

export default function StoreMenu({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { logout } = useAuth();
  const router = useRouter();
  const tutorial = useTutorialUrl();
  const role = useTeamRole();
  return (
    <Drawer open={open} onClose={onClose} title={N.more}>
      <nav className="flex flex-col gap-4" aria-label={N.more} data-testid="store-menu">
        <section>
          <p className="px-3 pb-1 text-xs font-bold text-ink-muted">{N.mine}</p>
          <ul className="flex flex-col">
            {MINE.filter(([href]) => !(role === "staff" && href === "/store/team")).map(([href, label, Ico]) => (
              <li key={href}>
                <Link href={href} onClick={onClose} className={ROW}>
                  <Ico size={18} className="shrink-0 text-primary" />
                  {label}
                </Link>
              </li>
            ))}
          </ul>
        </section>
        <section>
          <p className="px-3 pb-1 text-xs font-bold text-ink-muted">{N.platform}</p>
          <ul className="flex flex-col">
            <li>
              <a href={tutorial} target="_blank" rel="noopener noreferrer" onClick={onClose} className={ROW}>
                <IconApp size={18} className="shrink-0 text-primary" />
                {N.tutorial}
              </a>
            </li>
            {PLATFORM.map(([href, label]) => (
              <li key={href}>
                <Link href={href} onClick={onClose} className={ROW}>
                  <IconSupport size={18} className="shrink-0 text-ink-muted" />
                  {label}
                </Link>
              </li>
            ))}
          </ul>
        </section>
        <button
          type="button"
          data-testid="store-logout"
          onClick={() => {
            onClose();
            logout();
            router.replace("/login");
          }}
          className={`${ROW} text-danger`}
        >
          <IconLogout size={18} className="shrink-0" />
          {N.logout}
        </button>
      </nav>
    </Drawer>
  );
}
