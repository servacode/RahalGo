"use client";

/** حسابي — يستخدم مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  AccountSettings,
  MyAddresses,
  ReputationComplaints,
  PageContainer,
  PageHeader,
  LoadingState,
  IconUser,
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
