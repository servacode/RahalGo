"use client";

/** حسابي — مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AccountSettings, IconUser } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function AccountPage() {
  const { user } = useAuth();
  return (
    <div className="mx-auto max-w-lg">
      <h1 className="mb-5 flex items-center gap-2 text-lg font-bold">
        <IconUser className="text-primary" />
        {m.rep.account.title}
      </h1>
      <AccountSettings api={api} mediaUrl={mediaUrl} phone={user?.phone} />
    </div>
  );
}
