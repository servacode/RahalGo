"use client";

/** حسابي — يستخدم مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AccountSettings, PageContainer, PageHeader, LoadingState, IconUser } from "@rahalgo/ui";
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
      <AccountSettings
        api={api}
        mediaUrl={mediaUrl}
        phone={user?.phone}
        onDeleted={() => {
          logout();
          router.replace("/");
        }}
      />
    </PageContainer>
  );
}
