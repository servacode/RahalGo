"use client";

/** حسابي — مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AccountSettings, PageContainer, PageHeader, IconUser } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function AccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  return (
    <PageContainer>
      <PageHeader icon={IconUser} title={m.terms.account} />
      <AccountSettings
        api={api}
        mediaUrl={mediaUrl}
        phone={user?.phone}
        onDeleted={() => {
          logout();
          router.replace("/login");
        }}
      
        onLogout={() => {
          logout();
          router.replace("/login");
        }}
      />
    </PageContainer>
  );
}
