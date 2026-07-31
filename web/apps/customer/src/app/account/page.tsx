"use client";

/** حسابي — يستخدم مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AccountSettings, IconUser } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function AccountPage() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isLoggedIn(user)) router.replace("/login?next=/account");
  }, [user, loading, router]);

  if (loading || !isLoggedIn(user)) {
    return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;
  }

  return (
    <div className="mx-auto max-w-lg">
      <h1 className="mb-5 flex items-center gap-2 text-xl font-bold">
        <IconUser className="text-primary" />
        {m.terms.account}
      </h1>
      <AccountSettings api={api} mediaUrl={mediaUrl} phone={user?.phone} />
    </div>
  );
}
