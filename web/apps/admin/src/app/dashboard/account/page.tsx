"use client";

/** حسابي: بطاقة الأدوار + إعدادات الحساب المشتركة + آخر الدخولات. */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import { AccountSettings, FormSection, PageContainer, PageHeader, Pagination, IconUser, IconStatus } from "@rahalgo/ui";
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
/* **أسماءُ الأفعال من مَعْجمٍ واحد.**
   كان هنا انتقاءُ ثلاثةٍ من خريطةٍ ثانيةٍ صغيرة — **ومن انتقى ثلاثةً قَبِل
   أن يُطبع الرابعُ خامّاً.** */
const ACTIONS: Record<string, string> = m.admin.audit.actions;

export default function MyAccountPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [logins, setLogins] = useState<Login[]>([]);
  /** **صفحةُ سجلّ الدخول** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **وهذه الشاشةُ تسأل «هل كان هذا أنت؟»** — **ومن سُرق حسابُه يبحث عن
      دخولٍ غريبٍ قد يكون قبل عشرين محاولة.** */
  const [page, setPage] = useState(1);
  const [count, setCount] = useState(0);
  const [perPage, setPerPage] = useState(20);

  useEffect(() => {
    api<{ logins: Login[]; total: number; per_page: number }>(
      `/api/v1/auth/my-logins?page=${page}`,
    )
      .then((r) => {
        setLogins(r?.logins ?? []);
        setCount(r?.total ?? 0);
        setPerPage(r?.per_page || 20);
      })
      // @empty-ok **وسجلٌّ لا يُجلب لا يُسقط الصفحة** — بقيّةُ الحساب تُقرأ.
      .catch(() => setLogins([]));
  }, [page]);

  return (
    <PageContainer>
      <PageHeader icon={IconUser} title={m.terms.account} subtitle={A.subtitle} />

      <div className="flex items-center gap-3 surface p-4">
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
                    {ACTIONS[l.action] ?? l.action}
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
          {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
          {count > perPage && (
            <div className="mt-3 flex justify-center">
              <Pagination page={page} total={count} perPage={perPage} onChange={setPage} />
            </div>
          )}
        </FormSection>
      </div>
    </PageContainer>
  );
}
