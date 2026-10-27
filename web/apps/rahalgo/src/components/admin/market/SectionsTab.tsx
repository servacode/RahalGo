"use client";

/**
 * **تبويبُ الأقسام** — بطاقةٌ لكلّ قسمٍ بصورته، **تُرتَّب بالسحب أو بالأسهم**.
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الأقسام بترتيبٍ بالسحب هو ترتيبُها عند الزبون ·
 * ويُضاف زرُّ حذف القسم».) **والأسهمُ للّمس** (تدقيقُ السوق ٢٠٢٦-١٠-٢٧): السحبُ بأحداث HTML5
 * لا يعمل على الجوّال، فكان الموبايلُ بلا أيّ طريقةٍ للترتيب.
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

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Modal,
  Select,
  LoadingState,
  useLiveData,
  useToast,
  IconCamera,
  FormActions,
  Confirm,
  IconEdit,
  IconDelete,
  IconCheck,
  IconBlock,
  IconAdd,
  IconGrid,
  IconChevronDown,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import ImageUpload, { useImagePicker } from "@/components/admin/ImageUpload";
import StateBadge from "./StateBadge";
import { LoadError, StateBox } from "./MarketParts";

/** **صورُ السوق من بابها** — بقدرة السوق لا المحتوى (قرارُ المالك ٢٠٢٦-١٠-٠٥). */
const MARKET_MEDIA = "/api/v1/admin/market/media";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const U = m.admin.market.ui;

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
  const { data, error, reload } = useLiveData<{ sections: Section[] }>(
    () => api("/api/v1/admin/sections"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Section | null | "new">(null);
  const [deleting, setDeleting] = useState<Section | null>(null);
  /** **الترتيبُ المحلّيّ أثناء السحب** — يُعرض فوراً ثمّ يُحفظ. */
  const [order, setOrder] = useState<Section[] | null>(null);
  const [dragID, setDragID] = useState<string | null>(null);
  /** **القسمُ الذي يتبدّل توفّرُه الآن** — والزرُّ معطّلٌ حتّى يردّ المحرّك، فلا ترجعه كبستان ورا بعض. */
  const [toggling, setToggling] = useState<string | null>(null);
  const [ordering, setOrdering] = useState(false);
  const router = useRouter();
  const { push, Toaster } = useToast();
  // **الضغطُ على صورة القسم يفتح ملفّات الجهاز فوراً** (قرارُ المالك ٢٠٢٦-١٠-٢٦).
  const imageTarget = useRef<Section | null>(null);
  const picker = useImagePicker({
    kind: "banner",
    path: MARKET_MEDIA,
    onUploaded: async (mediaID) => {
      const sec = imageTarget.current;
      if (!sec) return;
      await api(`/api/v1/admin/sections/${sec.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name: sec.name, image_media_id: mediaID }),
      });
      reload();
    },
    onError: (msg) => push(msg, "error"),
  });

  useEffect(() => {
    if (data) setOrder(data.sections ?? []);
  }, [data]);

  if (error && !data) return <LoadError onRetry={reload} />;
  if (!data || !order) return <LoadingState />;
  const list = order;

  async function toggle(sec: Section) {
    if (toggling) return;
    setToggling(sec.id);
    try {
      await api(`/api/v1/admin/sections/${sec.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !sec.active }),
      });
      reload();
    } catch (err) {
      push(errorText(err), "error");
    } finally {
      setToggling(null);
    }
  }

  /** **يُسقط القسمَ المسحوبَ مكانَ الهدف** — ويحفظ الترتيبَ كلَّه. */
  async function drop(targetID: string) {
    const from = list.findIndex((s) => s.id === dragID);
    const to = list.findIndex((s) => s.id === targetID);
    setDragID(null);
    await moveTo(from, to);
  }

  /** **ينقل القسمَ من موضعٍ لموضع** — للسحب وللأسهم معاً — ثمّ يحفظ الترتيبَ كلَّه. */
  async function moveTo(from: number, to: number) {
    if (ordering || from < 0 || to < 0 || from >= list.length || to >= list.length || from === to) return;
    const next = [...list];
    const [moved] = next.splice(from, 1);
    if (moved) next.splice(to, 0, moved);
    setOrder(next);
    setOrdering(true);
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
    } finally {
      setOrdering(false);
    }
  }

  const roundBtn = "!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full";

  return (
    <div className="space-y-3">
      <section className="surface flex flex-wrap items-center justify-between gap-2 p-3">
        <p className="min-w-0 flex-1 text-sm text-ink-muted">{S.dragHint}</p>
        <Button onClick={() => setEditing("new")} className="inline-flex shrink-0 items-center gap-1.5">
          <IconAdd size={16} />
          {S.add}
        </Button>
      </section>

      {list.length === 0 ? (
        <StateBox
          icon={IconGrid}
          tone="accent"
          title={S.empty}
          hint={U.sectionsEmptyHint}
          action={<Button onClick={() => setEditing("new")}>{S.add}</Button>}
        />
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {list.map((sec, idx) => (
            <div
              key={sec.id}
              draggable
              onDragStart={() => setDragID(sec.id)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => void drop(sec.id)}
              onDragEnd={() => setDragID(null)}
              // **البطاقةُ كلُّها تفتح القسم** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — بلا زرّ «عرض القسم».
              onClick={() => router.push(`/dashboard/sections/${sec.id}`)}
              className={`flex cursor-pointer flex-col overflow-hidden surface transition-shadow hover:elev-2 ${
                sec.active ? "" : "opacity-60"
              } ${dragID === sec.id ? "ring-2 ring-primary" : ""}`}
            >
              {/* **الصورةُ أوّلاً — وهي هويّةُ القسم لا زينتُه.** ولا أيقونةَ
                  بديلاً: يُقال صراحةً «أضف صورة» — نقصٌ يُرى يُعالَج. */}
              <div className="relative flex aspect-[4/3] items-center justify-center bg-field">
                <button
                  type="button"
                  title={S.image}
                  aria-label={S.image}
                  disabled={picker.busy}
                  onClick={(e) => {
                    e.stopPropagation();
                    imageTarget.current = sec;
                    picker.open();
                  }}
                  className="group absolute inset-0 flex items-center justify-center"
                >
                  {sec.image_url || sec.image_thumb_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
                      alt={sec.name}
                      draggable={false}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <span className="flex flex-col items-center gap-1 text-xs text-ink-muted">
                      <IconCamera size={20} />
                      {S.addImage}
                    </span>
                  )}
                  <span className="absolute bottom-2 start-2 inline-flex h-8 w-8 items-center justify-center rounded-full bg-surface text-ink elev-1 opacity-80 group-hover:opacity-100">
                    <IconCamera size={16} />
                  </span>
                </button>
                <span className="absolute end-1.5 top-1.5">
                  <StateBadge label={sec.active ? S.available : S.unavailable} variant={sec.active ? "success" : "warning"} />
                </span>
              </div>

              <div className="flex flex-1 flex-col gap-2 p-3">
                <div className="flex min-w-0 flex-1 items-baseline justify-between gap-2">
                  <p className="truncate font-bold">{sec.name}</p>
                  <p className="shrink-0 text-xs text-ink-muted">{S.items.replace("{n}", fmtNum(sec.items))}</p>
                </div>

                {/* **زرٌّ ذكيٌّ للتوفّر** يقول الحالَ بلونه ويقلبها بضغطة، **والتعديلُ والحذفُ أيقونات**. */}
                <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={sec.active}
                    aria-busy={toggling === sec.id}
                    title={sec.active ? U.makeSectionOff : U.makeSectionOn}
                    disabled={toggling === sec.id}
                    onClick={() => void toggle(sec)}
                    className={`inline-flex flex-1 items-center justify-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-bold text-on-solid transition-colors disabled:opacity-60 ${
                      sec.active ? "bg-success-solid" : "bg-warning-solid"
                    }`}
                  >
                    {sec.active ? <IconCheck size={14} /> : <IconBlock size={14} />}
                    {toggling === sec.id ? U.saving : sec.active ? S.available : S.unavailable}
                  </button>
                  <Button
                    variant="ghost"
                    title={m.common.edit}
                    aria-label={m.common.edit}
                    onClick={() => setEditing(sec)}
                    className={`${roundBtn} text-ink-muted hover:text-ink`}
                  >
                    <IconEdit size={17} />
                  </Button>
                  <Button
                    variant="ghost"
                    title={S.delete}
                    aria-label={S.delete}
                    onClick={() => setDeleting(sec)}
                    className={`${roundBtn} !text-danger`}
                  >
                    <IconDelete size={17} />
                  </Button>
                </div>
                {/* **أسهمُ الترتيب للّمس** — والسحبُ يبقى للفأرة. */}
                <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                  <Button
                    variant="secondary"
                    disabled={idx === 0 || ordering}
                    title={U.moveUp}
                    aria-label={U.moveUp}
                    onClick={() => void moveTo(idx, idx - 1)}
                    className="!min-h-0 flex-1 !py-1"
                  >
                    <IconChevronDown size={16} className="rotate-180" />
                  </Button>
                  <Button
                    variant="secondary"
                    disabled={idx === list.length - 1 || ordering}
                    title={U.moveDown}
                    aria-label={U.moveDown}
                    onClick={() => void moveTo(idx, idx + 1)}
                    className="!min-h-0 flex-1 !py-1"
                  >
                    <IconChevronDown size={16} />
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
      {picker.input}
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
