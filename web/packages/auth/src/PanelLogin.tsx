"use client";

/**
 * بابُ اللوحة — نموذج دخولٍ في كل تطبيق، لا تحويلٌ إلى بوابةٍ واحدة.
 *
 * ## لماذا تغيّر هذا
 *
 * كانت اللوحات الأربع تُحوّل إلى `:3003/login`. وبدا ذلك توحيداً نظيفاً، حتى
 * جُرّب فتحُ الأربع معاً فانكشف عطبان يُبطلانه:
 *
 *  1. **البوابة لا تعرض النموذج لمن هو داخلٌ أصلاً** — تُوجّهه فوراً إلى لوحته.
 *     فبعد دخول الحساب الأول لا سبيل إلى إدخال الثاني.
 *  2. **والخروجُ لتفريغها يقتل ما بُني للتوّ**: تسليم SSO يُشارك الجلسة نفسها،
 *     فالخروج من البوابة يُبطل اللوحة التي سلّمَت إليها (R-16، وهو سلوكٌ مقصود).
 *
 * فصار فتحُ لوحتين بحسابين مستحيلاً — وهو ما يفعله كلُّ من يطوّر هذه المنصة
 * أو يجرّبها.
 *
 * ## وليست هذه حيلةَ تطوير تُعكس لاحقاً
 *
 * بابٌ لكل لوحة هو ما تفعله المنصات الحقيقية: لوحةُ الإدارة لها بابها وبوابةُ
 * المتجر لها بابها. **والغريبُ هو العكس** — أن يمرّ صاحبُ مطعمٍ بواجهة تسوّقٍ
 * ليصل إلى شاشة مطبخه.
 *
 * والبوابة الموحّدة تبقى في تطبيق الزبون لسببها الحقيقي: **من لا يعرف أيُّ
 * تطبيقٍ له** يدخل من بابٍ واحد فيُوجَّه بدوره. فالاثنان معاً لا أحدهما.
 *
 * ## ولمن دخل بحسابٍ لا يخصّ هذه اللوحة
 *
 * لا يُعاد إلى `/login` فيدور — يُقال له صراحةً إن حسابه ليس حساب هذه اللوحة،
 * ويُعطى زرّ خروجٍ ليدخل بغيره. **والدوران الصامت أسوأ من الرفض الصريح**: من
 * دار لا يعرف أخطأ الحساب أم انكسر النظام.
 */

import { useEffect, useRef, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "@rahalgo/ui";
import { LoginCard } from "./LoginCard";
import { useAuth } from "./provider";
import type { AuthUser } from "./client";

const m = getMessages(defaultLocale);

export function PanelLogin({
  title,
  subtitle,
  /** هل يخصّ هذا الحساب هذه اللوحة؟ يُمرَّر حارسُ التطبيق نفسه — لا تُكرَّر القاعدة */
  allows,
  /** نصّ الرفض من معجم هذا التطبيق (`admin.notAllowed` وأخواته) */
  notAllowed,
  /** وجهةُ من دخل بحسابٍ صحيح */
  home,
  /** التحويل — يُحقن لأن الحزمة لا تعرف موجّه Next */
  replace,
  methods = "both",
  footer,
}: {
  title: string;
  subtitle?: string;
  allows: (user: AuthUser | null) => boolean;
  notAllowed: string;
  home: string;
  replace: (href: string) => void;
  methods?: "both" | "password" | "otp";
  footer?: ReactNode;
}) {
  const { user, loading, setUser, logout } = useAuth();
  const sent = useRef(false);

  // من دخل بحسابٍ يخصّ هذه اللوحة يمضي إليها. **ولا يُحوَّل من لا يخصّها** —
  // يبقى هنا ليقرأ السبب.
  useEffect(() => {
    if (loading || !user || sent.current) return;
    if (!allows(user)) return;
    sent.current = true;
    replace(home);
  }, [user, loading, allows, home, replace]);

  if (loading) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  if (user && !allows(user)) {
    return (
      <main className="flex flex-1 items-center justify-center p-6">
        <div className="w-full max-w-sm rounded-card border border-line bg-surface p-6 text-center">
          <p className="font-bold">{notAllowed}</p>
          <p className="mt-1 text-sm text-ink-muted">
            {user.full_name || user.phone}
          </p>
          <Button
            className="mt-4 w-full"
            variant="danger"
            onClick={() => {
              sent.current = false;
              logout();
            }}
          >
            {m.auth.logout}
          </Button>
        </div>
      </main>
    );
  }

  if (user) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  return (
    <LoginCard
      title={title}
      subtitle={subtitle}
      methods={methods}
      footer={footer}
      onSuccess={(u) => {
        setUser(u);
        // **ولا تحويل هنا إن كان الحساب لا يخصّ اللوحة**: إعادة الرسم تُظهر
        // رسالة الرفض. فمن دخل بحساب زبونٍ في بوابة المتجر يعرف لماذا رُدّ.
        if (allows(u)) replace(home);
      }}
    />
  );
}
