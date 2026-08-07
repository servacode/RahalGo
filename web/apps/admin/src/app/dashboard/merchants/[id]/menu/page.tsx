"use client";

/** أصناف متجر — المحرّر المركزي نفسه بمسارات الإدارة. */

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  MenuManager,
  PageContainer,
  Button,
  IconPrev,
  IconStore,
  type MenuPaths,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import ImageUpload, { MediaThumb } from "@/components/ImageUpload";

const m = getMessages(defaultLocale);

/** مسارات الإدارة — الحارس دورٌ لا ملكية. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/admin/merchants/${id}/menu`,
  sections: (id) => `/api/v1/admin/merchants/${id}/menu/sections`,
  section: (id) => `/api/v1/admin/menu/sections/${id}`,
  items: (id) => `/api/v1/admin/merchants/${id}/menu/items`,
  item: (id) => `/api/v1/admin/menu/items/${id}`,
  platformSections: () => `/api/v1/admin/sections`,
};

interface Merchant {
  id: string;
  name: string;
}

export default function AdminMenuPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [name, setName] = useState("");

  useEffect(() => {
    api<{ merchants: Merchant[] }>("/api/v1/admin/merchants?per_page=100")
      .then((p) => setName(p.merchants.find((x) => x.id === id)?.name ?? ""))
      .catch(() => undefined);
  }, [id]);

  return (
    <PageContainer>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <IconStore className="text-primary" />
          {m.terms.menu}
          {name && <span className="font-normal text-ink-muted">— {name}</span>}
        </h1>
        <Button
          variant="secondary"
          /* **والرجوعُ إلى بابٍ موجود** — كانت الصفحةُ تردّ 404،
             والمتاجرُ تُدار في صفحة الحسابات بتبويبٍ لها. (٢٠٢٦-٠٨-٠٧.) */
          onClick={() => router.push("/dashboard/users")}
          className="flex items-center gap-1.5"
        >
          <IconPrev size={15} />
          {m.admin.merchants.title}
        </Button>
      </div>

      <MenuManager
        api={api}
        paths={PATHS}
        merchantID={id}
        imageUpload={(initialUrl, onChange) => (
          <ImageUpload
            kind="menu_item"
            label={m.shared.menuEditor.itemImage}
            initialUrl={initialUrl}
            onChange={onChange}
          />
        )}
        thumb={(url, alt) => <MediaThumb url={url} alt={alt} fallback={alt} size={48} />}
      />
    </PageContainer>
  );
}
