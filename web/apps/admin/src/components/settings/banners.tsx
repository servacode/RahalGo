"use client";

/**
 * **لافتاتُ صفحة التسوّق — في صفحتها لا في «العروض».**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «وزّع كلّاً حسب مكانه — صور وإعدادات التسوّق
 *  بالتسوّق» ثمّ «نضيف السلايدر بالإعدادات لصفحة التسوّق».)
 *
 * # ولماذا انتقلت
 *
 * **كانت تبويباً في «العروض والخصومات»** بجانب أكواد الخصم — **وهي ليست
 * خصماً**: صورةٌ في صفحة، وضبطُها ضبطُ صفحة.
 *
 * **ومن أراد أن يبدّل لافتةَ التسوّق كان يفتح «العروض»** — ولا شيءَ يقوده
 * إلى هناك. **والضبطُ يُطلب حيث يُرى أثرُه** — ومهلةُ التبديل تحته الآن.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  EmptyState,
  Input,
  Modal,
  Badge,
  IconAdd,
  IconEdit,
  IconDelete,
  IconPromos,
} from "@rahalgo/ui";
import ImageUpload from "@/components/ImageUpload";
import { api, mediaUrl, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  image_thumb_url: string | null;
  target: string;
  sort_order: number;
  active: boolean;
}

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    const known = (m.errors as Record<string, string>)[key];
    if (known) return known;
  }
  return m.errors.internal;
}

export default function BannersPanel({ isAdmin = true }: { isAdmin?: boolean }) {
  const [banners, setBanners] = useState<Banner[]>([]);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<Banner | null | "new">(null);

  const load = useCallback(async () => {
    try {
      setBanners(await api<Banner[]>("/api/v1/admin/banners"));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggleActive(b: Banner) {
    try {
      await api(`/api/v1/admin/banners/${b.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !b.active }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function remove(b: Banner) {
    try {
      await api(`/api/v1/admin/banners/${b.id}`, { method: "DELETE" });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <div>
      {isAdmin && (
        <Button onClick={() => setEditing("new")} className="mb-4 flex items-center gap-1.5">
          <IconAdd size={16} />
          {m.admin.promos.bannerCreate}
        </Button>
      )}
      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}
      {banners.length === 0 && (
        <div className="surface p-10 text-center text-ink-muted">
          {m.admin.promos.bannersEmpty}
        </div>
      )}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {banners.map((b) => (
          <div key={b.id} className="overflow-hidden surface">
            <div className="flex h-32 items-center justify-center bg-primary-tint">
              {b.image_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={mediaUrl(b.image_url) ?? ""}
                  alt={b.title}
                  className="h-full w-full object-cover"
                />
              ) : (
                <IconPromos size={40} className="text-primary-edge" />
              )}
            </div>
            <div className="p-4">
              <div className="flex items-center justify-between">
                <span className="font-bold">{b.title}</span>
                <Badge variant={b.active ? "success" : "danger"}>
                  {b.active ? m.admin.merchants.active : m.admin.merchants.inactive}
                </Badge>
              </div>
              {isAdmin && (
                <div className="mt-3 flex justify-end gap-2 border-t border-line-soft pt-3">
                  <Button variant="ghost" onClick={() => setEditing(b)}>
                    <IconEdit size={15} />
                  </Button>
                  <Button variant={b.active ? "danger" : "secondary"} onClick={() => toggleActive(b)}>
                    {b.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
                  </Button>
                  <Button variant="ghost" onClick={() => remove(b)}>
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
      {editing && (
        <BannerModal
          banner={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
    </div>
  );
}

function BannerModal({
  banner,
  onClose,
  onSaved,
}: {
  banner: Banner | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  // null = لم تُلمس (لا تُرسل)، "" = إزالة، معرف = صورة جديدة
  const [imageID, setImageID] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const body = {
      /* **والعنوانُ يُرسل فارغاً** — العمودُ باقٍ في القاعدة لصفوفٍ قديمة،
         **وحذفُ عمودٍ فيه بياناتٌ قرارٌ آخر.** */
      title: "",
      target: "",
      ...(imageID !== null ? { image_media_id: imageID } : {}),
    };
    try {
      if (banner) {
        await api(`/api/v1/admin/banners/${banner.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
      } else {
        await api("/api/v1/admin/banners", { method: "POST", body: JSON.stringify(body) });
      }
      onSaved();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={m.admin.promos.bannerCreate}>
      <form onSubmit={submit} className="space-y-4">
        {/* **ولا عنوانَ ولا وجهة.**

            (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لا يوجد داعٍ لعنوان البانر ولا للزرّ
             أيضاً».)

            **واللافتةُ صورةٌ تُعرض** — تصميمٌ فيه كلامُه ودعوتُه. **وحقلٌ
            يُملأ ولا يظهر أثرُه يُضبط ثمّ يُنتظر.** */}
        <ImageUpload
          kind="banner"
          label={m.admin.promos.bannerImage}
          initialUrl={banner?.image_thumb_url}
          onChange={setImageID}
        />
        {/* **والمقاسُ يُقال قبل الرفع لا بعده.**

            الإطارُ عريضٌ ويقصّ ما زاد عن نسبته، **فصورةٌ طويلةٌ يضيع أعلاها
            وأسفلُها** — ومن رفعها لا يعرف لماذا خرجت ناقصة. */}
        <p className="-mt-2 text-xs text-ink-muted">{m.admin.promos.bannerImageHint}</p>
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
