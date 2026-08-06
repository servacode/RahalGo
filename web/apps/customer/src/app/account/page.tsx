"use client";

/** حسابي — يستخدم مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import Link from "next/link";
import {
  AccountSettings,
  MyAddresses,
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

      {/* **ولا شبكةَ روابطَ هنا.**

          (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «احذفها، مكرّرة بلا فائدة — المساعدةُ
           موجودةٌ أصلاً بالفوتر، وباقي الخيارات موجودةٌ بالقائمة المنسدلة
           والتوب بار».)

          **كانت سبعةَ روابطَ بُنيت يومَ كان الشريطُ يُخفي أيقوناتِه على
          الجوّال** — بابٌ ثانٍ لِما فقد بابَه. **ثمّ صارت القائمةُ المنسدلة
          تحملها في كلّ المقاسات** (العروضُ والمفضّلةُ والدعوةُ والشكاوى)،
          والشريطُ يحمل الطلباتِ والمحفظة، والذيلُ يحمل المساعدة.

          **فبقيت تكرّر ما صار له بابٌ دائم** — ومن رأى الرابطَ نفسَه في
          ثلاثة مواضعَ لا يعرف أيَّها الأصل. */}
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

      {/* **ولا قسمَ بلاغاتٍ هنا** — انتقل إلى «الشكاوى والبلاغات»، حيث
          يقع ما رفعه الزبونُ كذلك. (قرارُ المالك ٢٠٢٦-٠٨-٠٧.) */}
    </PageContainer>
  );
}
