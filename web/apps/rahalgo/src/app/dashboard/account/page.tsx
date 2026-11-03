"use client";

/** حسابي: بطاقة الأدوار + إعدادات الحساب المشتركة + آخر الدخولات. */

import { useRouter } from "next/navigation";
import { useState } from "react";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import {
  AccountSettings,
  AdminAsync,
  AdminCard,
  AdminPage,
  Row,
  RowList,
  useLiveData,
  IconUser,
  IconStatus,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import RoleBadge from "@/components/admin/RoleBadge";
import PinSection from "@/components/admin/PinSection";

const m = getMessages(defaultLocale);
const A = m.admin.myAccount;

interface Login {
  action: string;
  ip: string;
  created_at: string;
}
/* **أسماءُ الأفعال من مَعْجمٍ واحد.**
   كان هنا انتقاءُ ثلاثةٍ من خريطةٍ ثانيةٍ صغيرة — **ومن انتقى ثلاثةً قَبِل
   أن يُطبع الرابعُ خامّاً.** */
const ACTIONS: Record<string, string> = m.admin.audit.actions;

interface LoginPage {
  logins: Login[];
  total: number;
  per_page: number;
}

export default function MyAccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  /** **صفحةُ سجلّ الدخول** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **وهذه الشاشةُ تسأل «هل كان هذا أنت؟»** — **ومن سُرق حسابُه يبحث عن
      دخولٍ غريبٍ قد يكون قبل عشرين محاولة.** */
  const [page, setPage] = useState(1);

  // **وفشلُ السجلّ يُقال فشلاً** — كان يُقرأ «لا دخولات»، وهذا مضلِّلٌ في شاشة أمان.
  const q = useLiveData<LoginPage>(() => api(`/api/v1/auth/my-logins?page=${page}`), [], [page]);
  const logins = q.data?.logins ?? [];
  const count = q.data?.total ?? 0;
  const perPage = q.data?.per_page || 20;

  return (
    <AdminPage
      icon={IconUser}
      title={m.terms.account}
      pagination={q.data ? { page, total: count, perPage, onChange: setPage } : undefined}
    >
      <AdminCard>
        <p className="font-bold">{user?.full_name || "—"}</p>
        <p className="text-sm text-ink-muted" dir="ltr">
          {user?.phone}
        </p>
        <div className="mt-2 flex flex-wrap gap-1">
          {user?.roles.map((r) => (
            <RoleBadge key={r} role={r} />
          ))}
        </div>
      </AdminCard>

      <AccountSettings
        api={api}
        mediaUrl={mediaUrl}
        phone={user?.phone}
        onDeleted={() => {
          logout();
          router.replace("/adminrahalgo");
        }}
        onLogout={() => {
          logout();
          router.replace("/adminrahalgo");
        }}
      />

      {/* **والرمزُ السرّيُّ فوق سجلّ الدخول** — كلاهما أمنُ الحساب،
          **والفعلُ يسبق التاريخ.** */}
      <PinSection />

      <section className="space-y-2">
        <h2 className="admin-h2">{A.recentLogins}</h2>
        <AdminAsync
          loading={q.loading}
          error={q.error}
          onRetry={q.reload}
          hasData={!!q.data}
          empty={!!q.data && logins.length === 0}
          emptyTitle={A.loginsEmpty}
          emptyIcon={IconStatus}
        >
          <RowList>
            {logins.map((l, i) => (
              <Row
                key={i}
                title={
                  <span className={l.action === "auth.password_failed" ? "text-danger" : ""}>
                    {ACTIONS[l.action] ?? l.action}
                  </span>
                }
                meta={<span dir="ltr">{l.ip}</span>}
                trailing={<span dir="ltr">{fmtDateTime(l.created_at)}</span>}
              />
            ))}
          </RowList>
        </AdminAsync>
      </section>
    </AdminPage>
  );
}
