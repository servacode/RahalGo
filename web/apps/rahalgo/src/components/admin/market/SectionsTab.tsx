"use client";

/**
 * **تبويبُ الأقسام** — بطاقةٌ لكلّ قسمٍ بصورته، **تُرتَّب بالسحب**.
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الأقسام بترتيبٍ بالسحب هو ترتيبُها عند الزبون ·
 * ويُضاف زرُّ حذف القسم».)
 *
 * **والترتيبُ يُرسَل كلُّه بعد كلّ إفلات** (`PUT /sections/order`) — لا خطوةً
 * واحدة، فلا يبقى قسمان على رقمٍ واحد.
 *
 * # والقسمُ صورةٌ واسم — لا أكثر
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «ألغِ الأيقوناتِ والهامش، ولازم تعرض الصورةَ بمكان
 * الأيقونة كصورة لا أيقونة».) **وهامشُ القسم لا يُضبط من هنا** — التسعيرُ
 * يُدار من الإعدادات.
 */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  Select,
  EmptyState,
  LoadingState,
  useLiveData,
  useToast,
  IconStatus,
  IconCamera,
  FormActions,
  Confirm,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import ImageUpload from "@/components/admin/ImageUpload";

/** **صورُ السوق من بابها** — بقدرة السوق لا المحتوى (قرارُ المالك ٢٠٢٦-١٠-٠٥). */
const MARKET_MEDIA = "/api/v1/admin/market/media";

const m = getMessages(defaultLocale);
const S = m.admin.sections;

export interface Section {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
  margin_override: number | null;
  items: number;
  /** **صورةُ القسم** — وجهُه في السوق. */
  image_url: string | null;
  image_thumb_url: string | null;
  image_media_id: string | null;
}

export default function SectionsTab() {
  const { data, reload } = useLiveData<{ sections: Section[] }>(
    () => api("/api/v1/admin/sections"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Section | null | "new">(null);
  const [deleting, setDeleting] = useState<Section | null>(null);
  /** **الترتيبُ المحلّيّ أثناء السحب** — يُعرض فوراً ثمّ يُحفظ. */
  const [order, setOrder] = useState<Section[] | null>(null);
  const [dragID, setDragID] = useState<string | null>(null);
  const router = useRouter();
  const { push, Toaster } = useToast();

  useEffect(() => {
    if (data) setOrder(data.sections ?? []);
  }, [data]);

  if (!data || !order) return <LoadingState />;
  const list = order;

  async function toggle(sec: Section) {
    try {
      await api(`/api/v1/admin/sections/${sec.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !sec.active }),
      });
      reload();
    } catch (err) {
      push(errorText(err), "error");
    }
  }

  /** **يُسقط القسمَ المسحوبَ مكانَ الهدف** — ويحفظ الترتيبَ كلَّه. */
  async function drop(targetID: string) {
    if (!dragID || dragID === targetID) return setDragID(null);
    const from = list.findIndex((s) => s.id === dragID);
    const to = list.findIndex((s) => s.id === targetID);
    if (from < 0 || to < 0) return setDragID(null);
    const next = [...list];
    const [moved] = next.splice(from, 1);
    if (moved) next.splice(to, 0, moved);
    setOrder(next);
    setDragID(null);
    try {
      await api("/api/v1/admin/sections/order", {
        method: "PUT",
        body: JSON.stringify({ ids: next.map((s) => s.id) }),
      });
      push(S.orderSaved);
      reload();
    } catch (err) {
      push(errorText(err), "error");
      reload();
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-ink-muted">{S.dragHint}</p>
        <Button onClick={() => setEditing("new")}>{S.add}</Button>
      </div>

      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={S.empty} />
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {list.map((sec) => (
            <div
              key={sec.id}
              draggable
              onDragStart={() => setDragID(sec.id)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => void drop(sec.id)}
              onDragEnd={() => setDragID(null)}
              className={`flex cursor-grab flex-col overflow-hidden surface transition-shadow hover:elev-2 ${
                sec.active ? "" : "opacity-60"
              } ${dragID === sec.id ? "ring-2 ring-primary" : ""}`}
            >
              {/* **الصورةُ أوّلاً — وهي هويّةُ القسم لا زينتُه.** ولا أيقونةَ
                  بديلاً: يُقال صراحةً «أضف صورة» — نقصٌ يُرى يُعالَج. */}
              <div className="relative flex aspect-[4/3] items-center justify-center bg-field">
                {sec.image_url || sec.image_thumb_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
                    alt={sec.name}
                    draggable={false}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <button
                    onClick={() => setEditing(sec)}
                    className="flex flex-col items-center gap-1 text-xs text-ink-muted hover:text-ink"
                  >
                    <IconCamera size={20} />
                    {S.addImage}
                  </button>
                )}
                <span className="absolute end-1.5 top-1.5">
                  <Badge variant={sec.active ? "success" : "neutral"}>
                    {sec.active ? S.available : S.unavailable}
                  </Badge>
                </span>
              </div>

              <div className="flex flex-1 flex-col gap-2 p-3">
                <div className="flex min-w-0 flex-1 items-baseline justify-between gap-2">
                  <p className="truncate font-bold">{sec.name}</p>
                  <p className="shrink-0 text-xs text-ink-muted">
                    {S.items.replace("{n}", fmtNum(sec.items))}
                  </p>
                </div>

                <div className="flex flex-wrap gap-1.5 [&_button]:flex-1 [&_button]:!px-2 [&_button]:text-xs">
                  <Button
                    variant="secondary"
                    onClick={() => router.push(`/dashboard/sections/${sec.id}`)}
                  >
                    {S.view}
                  </Button>
                  <Button variant="secondary" onClick={() => setEditing(sec)}>
                    {m.common.edit}
                  </Button>
                  <Button
                    variant={sec.active ? "ghost" : "primary"}
                    onClick={() => void toggle(sec)}
                  >
                    {sec.active ? S.makeUnavailable : S.makeAvailable}
                  </Button>
                  <Button variant="ghost" onClick={() => setDeleting(sec)}>
                    {S.delete}
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <SectionModal
          section={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
      {deleting && (
        <DeleteSectionModal
          section={deleting}
          others={list.filter((s) => s.id !== deleting.id)}
          onClose={() => setDeleting(null)}
          onDone={() => {
            setDeleting(null);
            reload();
          }}
        />
      )}
      <Toaster />
    </div>
  );
}

/**
 * **حذفُ القسم** — الفارغُ يُحذف مباشرة، **والعامرُ تُنقل أصنافُه أوّلاً إلى
 * قسمٍ يختاره الموظّف.** ولا يُحذف صنفٌ معه أبداً.
 */
function DeleteSectionModal({
  section,
  others,
  onClose,
  onDone,
}: {
  section: Section;
  others: Section[];
  onClose: () => void;
  onDone: () => void;
}) {
  const [target, setTarget] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const title = S.deleteTitle.replace("{s}", section.name);

  async function run() {
    if (section.items > 0 && !target) return setError(S.pickTarget);
    setBusy(true);
    setError("");
    try {
      const q = section.items > 0 ? `?move_to=${encodeURIComponent(target)}` : "";
      await api(`/api/v1/admin/sections/${section.id}${q}`, { method: "DELETE" });
      onDone();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  if (section.items === 0) {
    return (
      <Confirm
        open
        title={title}
        body={error || S.deleteEmpty}
        confirmLabel={S.deleteConfirm}
        onConfirm={() => void run()}
        onCancel={onClose}
        busy={busy}
      />
    );
  }

  return (
    <Modal open title={title} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-sm">{S.deleteHasItems.replace("{n}", fmtNum(section.items))}</p>
        <Select label={S.moveTo} value={target} onChange={(e) => setTarget(e.target.value)}>
          <option value="">{S.pickTarget}</option>
          {others.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
        {error && <p className="text-sm text-danger">{error}</p>}
        <FormActions
          onSave={() => void run()}
          onCancel={onClose}
          busy={busy}
          saveLabel={S.deleteConfirm}
          tone="danger"
        />
      </div>
    </Modal>
  );
}

function SectionModal({
  section,
  onClose,
  onSaved,
}: {
  section: Section | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(section?.name ?? "");
  /**
   * **صورةُ القسم** — و`null` تعني «بلا تغيير»، و`""` تعني «ارفعها».
   */
  const [imageID, setImageID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    if (!name.trim()) return setError(S.nameRequired);
    setBusy(true);
    setError("");
    try {
      // **والترتيبُ لا يُكتب هنا** — يُسحب في الشبكة، والجديدُ يُلحق آخرَها.
      const body = JSON.stringify({ name: name.trim(), image_media_id: imageID });
      if (section) {
        await api(`/api/v1/admin/sections/${section.id}`, { method: "PATCH", body });
      } else {
        await api("/api/v1/admin/sections", { method: "POST", body });
      }
      onSaved();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open title={section ? S.editTitle : S.add} onClose={onClose}>
      <div className="space-y-3">
        <ImageUpload
          path={MARKET_MEDIA}
          kind="banner"
          label={S.image}
          initialUrl={section?.image_url}
          onChange={setImageID}
        />
        <Input label={S.name} value={name} onChange={(e) => setName(e.target.value)} />
        {error && <p className="text-sm text-danger">{error}</p>}
        <FormActions onSave={submit} onCancel={onClose} busy={busy} />
      </div>
    </Modal>
  );
}
