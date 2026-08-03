"use client";

/** حسابي — يستخدم مكوّن إعدادات الحساب المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import dynamicImport from "next/dynamic";
import {
  AccountSettings,
  AddressBook,
  PageContainer,
  PageHeader,
  FormSection,
  LoadingState,
  IconUser,
  IconLocation,
  Input,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

// الخريطة من الحزمة المشتركة — مدخل فرعي كي لا تُجرّ مكتبتها لكل صفحة
const PickMap = dynamicImport(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
  loading: () => <div className="h-64 w-full animate-pulse rounded-card bg-page" />,
});

/**
 * لاقط العنوان: خريطةٌ وحقلُ نصّ يتبادلان الخدمة.
 *
 * من يعرف عنوانه لا يجيد الخريطة، ومن يعرف مكانه لا يجيد وصفه. فتحريك الدبّوس
 * يملأ النصّ عبر الترميز العكسي، والنصّ يبقى قابلاً للتحرير لأن العنوان الرسمي
 * لا يصف بيتاً في الرقة: «خلف الجامع، الطابق الثاني» أدلّ من أي إحداثية.
 */
function AddressPicker({
  value,
  onChange,
}: {
  value: { lat: number; lng: number; address: string } | null;
  onChange: (v: { lat: number; lng: number; address: string }) => void;
}) {
  const lat = value?.lat ?? 35.9528;
  const lng = value?.lng ?? 39.0079;
  return (
    <div className="space-y-2">
      <PickMap
        lat={value ? lat : null}
        lng={value ? lng : null}
        onPick={(la, ln) => {
          onChange({ lat: la, lng: ln, address: value?.address ?? "" });
          api<{ address: string }>(`/api/v1/geo/reverse?lat=${la}&lng=${ln}`)
            .then((r) => r.address && onChange({ lat: la, lng: ln, address: r.address }))
            .catch(() => undefined);
        }}
      />
      <Input
        id="addr-text"
        label={m.site.addresses.addressText}
        required
        value={value?.address ?? ""}
        onChange={(e) => onChange({ lat, lng, address: e.target.value })}
      />
    </div>
  );
}

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

      {/* دفتر العناوين — يُكتب مرّة ويُستعمل في كل طلب */}
      <FormSection title={m.site.addresses.title} icon={<IconLocation />}>
        <p className="mb-3 text-xs text-ink-muted">{m.site.addresses.hint}</p>
        <AddressBook
          api={api}
          picker={(value, onChange) => (
            <AddressPicker value={value} onChange={onChange} />
          )}
        />
      </FormSection>
    </PageContainer>
  );
}
