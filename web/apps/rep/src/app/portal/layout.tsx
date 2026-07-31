"use client";

/** هيكل بوابة المندوب العائم — سايدبار + توب بار (موحّد مع لوحة الإدارة). */

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  IconOverview,
  IconLink,
  IconStore,
  IconWallet,
  IconOrder,
  IconUser,
  IconStar,
  IconSupport,
  IconLogout,
  IconHamburger,
  IconClose,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isRep } from "@/lib/auth";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

const NAV = [
  { href: "/portal", label: m.rep.nav.overview, icon: IconOverview },
  { href: "/portal/link", label: m.rep.nav.link, icon: IconLink },
  { href: "/portal/leads", label: m.rep.nav.leads, icon: IconOrder },
  { href: "/portal/merchants", label: m.rep.nav.merchants, icon: IconStore },
  { href: "/portal/wallet", label: m.rep.nav.wallet, icon: IconWallet },
  { href: "/portal/reviews", label: m.rep.nav.reviews, icon: IconStar },
  { href: "/portal/complaints", label: m.rep.nav.complaints, icon: IconSupport },
  { href: "/portal/account", label: m.rep.nav.account, icon: IconUser },
];

interface MeSummary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
}
interface Reputation {
  rating: { avg: number; count: number; trend: "up" | "down" | "flat" };
}

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const [summary, setSummary] = useState<MeSummary | null>(null);
  const [rep, setRep] = useState<Reputation | null>(null);

  useEffect(() => {
    if (!loading && !isRep(user)) router.replace("/login");
  }, [user, loading, router]);

  useEffect(() => {
    if (isRep(user)) {
      api<MeSummary>("/api/v1/me/summary").then(setSummary).catch(() => undefined);
      api<Reputation>("/api/v1/me/reputation").then(setRep).catch(() => undefined);
    }
  }, [user, pathname]);

  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  // تسوّق كزبون: نطلب رمز تسليم ثم نفتح تطبيق الزبون مسجّلاً بنفس الحساب.
  async function shopAsCustomer() {
    try {
      const { code } = await api<{ code: string }>("/api/v1/auth/handoff", { method: "POST" });
      const site = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";
      window.location.href = `${site}/sso?code=${encodeURIComponent(code)}`;
    } catch {
      /* تجاهل — يبقى المستخدم في لوحته */
    }
  }

  if (loading || !isRep(user)) {
    return (
      <main className="flex min-h-screen items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  const activeLabel =
    NAV.find((i) => (i.href === "/portal" ? pathname === i.href : pathname.startsWith(i.href)))
      ?.label ?? m.rep.loginTitle;

  const sidebar = (
    <>
      <div className="flex items-center justify-between border-b border-line p-4">
        <div className="flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            ر
          </div>
          <span className="font-bold">{m.rep.loginTitle}</span>
        </div>
        <button
          onClick={() => setMenuOpen(false)}
          className="text-ink-muted hover:text-ink lg:hidden"
          aria-label={m.common.cancel}
        >
          <IconClose size={20} />
        </button>
      </div>
      {/* تسوّق كزبون — بالأعلى ليكون واضحاً */}
      <div className="p-3 pb-0">
        <button
          onClick={shopAsCustomer}
          className="flex w-full items-center gap-2.5 rounded-control bg-accent/10 px-3 py-2 text-sm font-medium text-accent-dark transition-colors hover:bg-accent/20"
        >
          <IconStore size={17} />
          {m.rep.shopAsCustomer}
        </button>
      </div>
      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {NAV.map((item) => {
          const active =
            item.href === "/portal" ? pathname === item.href : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-control px-3 py-2 text-sm transition-colors ${
                active
                  ? "bg-primary-light font-medium text-primary-dark"
                  : "text-ink-muted hover:bg-page hover:text-ink"
              }`}
            >
              <item.icon size={17} strokeWidth={active ? 2.2 : 1.8} />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </>
  );

  return (
    <div className="flex min-h-screen bg-page">
      <aside className="sticky top-3 m-3 me-0 hidden h-[calc(100vh-1.5rem)] w-60 shrink-0 flex-col overflow-hidden rounded-card border border-line bg-surface shadow-sm lg:flex">
        {sidebar}
      </aside>

      {menuOpen && (
        <div className="fixed inset-0 z-40 bg-ink/40 lg:hidden" onClick={() => setMenuOpen(false)} />
      )}
      <aside
        className={`fixed inset-y-0 start-0 z-50 flex w-64 flex-col bg-surface shadow-xl transition-transform duration-200 lg:hidden ${
          menuOpen ? "translate-x-0" : "translate-x-full rtl:translate-x-full ltr:-translate-x-full"
        }`}
      >
        {sidebar}
      </aside>

      <div className="flex min-w-0 flex-1 flex-col p-3">
        <header className="mb-3 flex items-center gap-3 rounded-card border border-line bg-surface px-4 py-2.5 shadow-sm">
          <button
            onClick={() => setMenuOpen(true)}
            className="text-ink-muted hover:text-ink lg:hidden"
            aria-label={m.rep.loginTitle}
          >
            <IconHamburger size={22} />
          </button>
          <h2 className="text-sm font-bold text-ink">{activeLabel}</h2>
          <div className="ms-auto flex items-center gap-2.5">
            {/* تقييمي — المتوسط واتجاهه */}
            {rep && rep.rating.count > 0 && (
              <Link
                href="/portal/reviews"
                className="flex items-center gap-1 rounded-control bg-amber-50 px-2.5 py-1.5 text-sm font-bold text-amber-600 hover:bg-amber-100"
                title={m.rep.reputation.myRating}
              >
                <span dir="ltr">{rep.rating.avg.toFixed(1)}</span>
                <span className="text-amber-400">★</span>
                {rep.rating.trend === "up" && <span className="text-success">▲</span>}
                {rep.rating.trend === "down" && <span className="text-danger">▼</span>}
              </Link>
            )}
            {/* رصيد المحفظة */}
            <Link
              href="/portal/wallet"
              className="flex items-center gap-1.5 rounded-control bg-primary-light px-2.5 py-1.5 text-sm font-bold text-primary-dark hover:bg-primary-light/70"
              title={m.rep.stats.balance}
            >
              <IconWallet size={15} />
              <span dir="ltr">{fmt.format(summary?.balance ?? 0)}</span>
              <span className="hidden text-xs font-normal sm:inline">{m.common.currency}</span>
            </Link>
            {/* الصورة الشخصية → إعدادات الحساب */}
            <Link
              href="/portal/account"
              title={m.rep.nav.account}
              className="flex h-8 w-8 items-center justify-center overflow-hidden rounded-full border border-line bg-primary-light text-sm font-bold text-primary-dark"
            >
              {summary?.avatar_thumb_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={mediaUrl(summary.avatar_thumb_url) ?? ""} alt="" className="h-full w-full object-cover" />
              ) : (
                (summary?.full_name || user?.phone || "؟").slice(0, 1)
              )}
            </Link>
            <button
              onClick={() => {
                logout();
                router.replace("/login");
              }}
              className="flex items-center gap-1.5 rounded-control px-2 py-1.5 text-sm text-danger hover:bg-danger/10"
            >
              <IconLogout size={16} />
              <span className="hidden sm:inline">{m.auth.logout}</span>
            </button>
          </div>
        </header>

        <main className="min-w-0 flex-1 rounded-card border border-line bg-surface p-4 shadow-sm">
          {children}
        </main>
      </div>
    </div>
  );
}
