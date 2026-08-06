"use client";

/** حسابي — يستخدم مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import Link from "next/link";
import {
  AccountSettings,
  MyAddresses,
  ReputationComplaints,
  PageContainer,
  PageHeader,
  LoadingState,
  Card,
  IconUser,
  IconWallet,
  IconOrder,
  IconHeart,
  IconPromos,
  IconSupport,
  IconLink,
  IconNext,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);
const R = m.customer.myReputation;

export default function AccountPage() {
  const { user, loading, logout } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isLoggedIn(user)) router.replace("/login?next=/account");
  }, [user, loading, router]);

  if (loading || !isLoggedIn(user)) {
    return <LoadingState />;
  }

  return (
    <PageContainer>
      <PageHeader icon={IconUser} title={m.terms.account} />

      {/* ══════════════════════════════════════════════════════════════════
          **بوّابةُ ما لا يتّسع له شريطٌ على الجوّال**
          ══════════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «شوف ع جوال كيف تطلع الصفحة… الشكلُ كلُّه
          مو مفهوم — ولازم يُطبَّق على كلّ الصفحات».)

          # ما كان

          **الشريطُ العلويُّ يحمل ثمانيةَ رموزٍ على شاشةٍ بثلاثمئةٍ وستّين** —
          بلا تسمياتٍ، ينزلق أفقيّاً، **وتسجيلُ الخروج يقع خارجَ النظر.**
          **واثنان منها يُكرّران الشريطَ السفليّ** (التسوّق وحسابي).

          # وما صار

          **الشريطُ السفليُّ للتنقّل الأوّل** — بتسمياتٍ تحت كلّ أيقونة.
          **والعلويُّ لما يتبدّل**: الجرسُ والرصيد. **والباقي هنا.**

          # ولا صفحةَ تُترك بلا باب

          **إخفاءُ أيقونةٍ ليس حذفَ صفحة**: الشكاوى والعروضُ والدعوةُ لم يكن
          إليها طريقٌ ثانٍ — **فمن أخفاها من الشريط قتلها على الجوّال.**
          فصارت هنا، **ويبقى الشريطُ يحملها على الشاشات المتّسعة.** */}
      <Card title={m.terms.account} icon={IconUser}>
        <ul className="grid gap-2 sm:grid-cols-2">
          {[
            { href: "/wallet", label: m.terms.wallet, icon: IconWallet },
            { href: "/orders", label: m.terms.orders, icon: IconOrder },
            { href: "/favorites", label: m.customer.favorites.title, icon: IconHeart },
            { href: "/offers", label: m.customer.offers.title, icon: IconPromos },
            { href: "/complaints", label: m.terms.complaints, icon: IconSupport },
            { href: "/invite", label: m.customer.invite.title, icon: IconLink },
            { href: "/help", label: m.site.legal.helpTitle, icon: IconSupport },
          ].map((it) => (
            <li key={it.href}>
              {/* **وسطرٌ كاملٌ يُضغط لا أيقونةٌ صغيرة** — الإبهامُ على الجوّال
                  يخطئ ما دون أربعةٍ وأربعين، **وسطرٌ بارتفاع خمسين لا يُخطأ.** */}
              <Link
                href={it.href}
                className="flex items-center gap-3 rounded-control border border-line bg-field px-3 py-3 transition-colors hover:border-primary-edge"
              >
                <it.icon size={18} className="shrink-0 text-primary" />
                <span className="min-w-0 flex-1 truncate text-sm font-medium">{it.label}</span>
                {/* **والسهمُ يقول «يُفتح»** — سطرٌ بلا علامةٍ يُقرأ خبراً. */}
                <IconNext size={16} className="shrink-0 text-ink-muted" />
              </Link>
            </li>
          ))}
        </ul>
      </Card>
      <AccountSettings
        api={api}
        mediaUrl={mediaUrl}
        phone={user?.phone}
        onDeleted={() => {
          logout();
          router.replace("/");
        }}
      
        onLogout={() => {
          logout();
          router.replace("/login");
        }}
      />

      {/* **دفترُ العناوين — من الحزمة المشتركة.** كان مكتوباً هنا
          بلاقطه وخريطته، **ونسخُه إلى لوحةٍ أخرى نسختان تفترقان يوماً.** */}
      <MyAddresses api={api} />

      {/* **وما يُكتب عنه — يراه صاحبُه.**

          السائقون يرفعون بلاغاتٍ على الزبائن (عنوانٌ وهميّ، سوءُ تعامل،
          امتناعٌ عن الاستلام)، **وكانت تُجمع في ملفّه ولا يبلغه منها شيء.**

          **ومن لا يعلم لا يصحّح**: يُحظر يوماً ويقول «بلا سبب»، **ومن أُنذر
          مرّةً يصحّح.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)

          **ولا نجومَ له**: لا أحدَ يقيّم الزبون، **و«٠٫٠ من ٥» في شاشته
          تُقرأ حكماً عليه** وهي لا تقيس شيئاً. */}
      <ReputationComplaints
        api={api}
        labels={{
          complaintsTitle: R.title,
          complaintsHint: R.hint,
          complaintsEmpty: R.empty,
        }}
      />
    </PageContainer>
  );
}
