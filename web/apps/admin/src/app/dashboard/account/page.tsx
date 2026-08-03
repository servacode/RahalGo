"use client";

/** حسابي: بطاقة الأدوار + إعدادات الحساب المشتركة + آخر الدخولات. */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import { AccountSettings, FormSection, PageContainer, PageHeader, IconUser, IconStatus } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import RoleBadge from "@/components/RoleBadge";

const m = getMessages(defaultLocale);
const A = m.admin.myAccount;

interface Login {
  action: string;
  ip: string;
  created_at: string;
}
const LOGIN_LABELS: Record<string, string> = {
  "auth.otp_login": m.admin.users.auditActions["auth.otp_login"],
  "auth.password_login": m.admin.users.auditActions["auth.password_login"],
  "auth.password_failed": m.admin.users.auditActions["auth.password_failed"],
};

export default function MyAccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [logins, setLogins] = useState<Login[]>([]);

  useEffect(() => {
    api<Login[]>("/api/v1/auth/my-logins")
      .then(setLogins)
      .catch(() => undefined);
  }, []);

  return (
    <PageContainer>
      <PageHeader icon={IconUser} title={m.terms.account} subtitle={A.subtitle} />

      <div className="flex items-center gap-3 rounded-card border border-line bg-surface p-4">
        <div className="min-w-0 flex-1">
          <p className="font-bold">{user?.full_name || "—"}</p>
          <p className="text-xs text-ink-muted" dir="ltr">
            {user?.phone}
          </p>
          <div className="mt-1 flex flex-wrap gap-1">
            {user?.roles.map((r) => (
              <RoleBadge key={r} role={r} />
            ))}
          </div>
        </div>
      </div>

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

      <div>
        <FormSection title={A.recentLogins} icon={<IconStatus />}>
          <p className="mb-2 text-xs text-ink-muted">{A.loginsHint}</p>
          {logins.length === 0 ? (
            <p className="py-4 text-center text-sm text-ink-muted">{A.loginsEmpty}</p>
          ) : (
            <ul className="space-y-1.5">
              {logins.map((l, i) => (
                <li
                  key={i}
                  className="flex items-center justify-between gap-2 rounded-control border border-line px-3 py-2 text-sm"
                >
                  <span className={`font-medium ${l.action === "auth.password_failed" ? "text-danger" : ""}`}>
                    {LOGIN_LABELS[l.action] ?? l.action}
                  </span>
                  <span className="flex items-center gap-3 text-xs text-ink-muted" dir="ltr">
                    <span>{l.ip}</span>
                    <span>
                      {fmtDateTime(l.created_at)}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </FormSection>
      </div>
    </PageContainer>
  );
}
