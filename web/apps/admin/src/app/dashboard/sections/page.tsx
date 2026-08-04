"use client";

/**
 * **أقسامُ المنصة** — ما نبيعه، لا من نشتري منه.
 *
 * # والقسمُ غيرُ التصنيف
 *
 *	التصنيف  ←  مطاعم · بقالة · صيدليات   ← **من نشتري منه**
 *	القسم    ←  شاورما · بيتزا · خضار     ← **ما نبيعه**
 *
 * **والزبونُ يرى الثاني وحدَه.**
 *
 * # وهامشُ القسم هنا لا في تصنيف المتجر
 *
 * الشاورما تُسعَّر كشاورما **سواءٌ جاءت من مطعمٍ أو مشاوٍ أو كافتيريا**.
 * وتصنيفُ المتجر يصف بائعَه لا سلعتَه، **وهامشٌ يتبع البائعَ يجعل الصنفَ
 * الواحد بسعرين.**
 *
 * **والفراغُ غيرُ الصفر**: الصفرُ يعني «لا هامشَ على هذا القسم»، والفراغُ
 * «اتبع الهامشَ العام». **ومن خلط بينهما جعل قسماً كاملاً يُباع بسعر شرائه.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  CategoryIcon,
  CategoryIconPicker,
  useLiveData,
  IconStore,
  IconStatus,
  IconCamera,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import MenuReviewQueue from "@/components/MenuReviewQueue";
import ImageUpload from "@/components/ImageUpload";

const m = getMessages(defaultLocale);
const S = m.admin.sections;

interface Section {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
  margin_override: number | null;
  items: number;
  /** **صورةُ القسم** — وجهُه في السوق، والأيقونةُ بديلُها حين تغيب. */
  image_url: string | null;
  image_thumb_url: string | null;
  image_media_id: string | null;
}

/** صنفٌ في قسم — **وحالُه يقول لماذا يظهر أو لا يظهر.** */
interface SectionItem {
  id: string;
  name: string;
  merchant_price: number;
  available: boolean;
  approved: boolean;
  merchant_name: string;
  merchant_status: string;
}

export default function SectionsPage() {
  const { data, reload } = useLiveData<{ sections: Section[] }>(
    () => api("/api/v1/admin/sections"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Section | null | "new">(null);
  /** القسمُ المفتوحُ لعرض أصنافه — **كما هي لا كما يراها الزبون.** */
  const [viewing, setViewing] = useState<Section | null>(null);

  if (!data) return <LoadingState />;
  const list = data.sections ?? [];

  async function toggle(sec: Section) {
    await api(`/api/v1/admin/sections/${sec.id}`, {
      method: "PATCH",
      body: JSON.stringify({ active: !sec.active }),
    });
    reload();
  }

  return (
    <PageContainer>
      <PageHeader
        icon={IconStore}
        title={S.title}
        subtitle={S.hint}
        actions={<Button onClick={() => setEditing("new")}>{S.add}</Button>}
      />

      {/* **طابورُ مراجعة القائمة — حيث يُقرَّر ما يُعرض في السوق.**

          يظهر حين يكون فيه عمل، **ولا يظهر حين يكون المفتاحُ مُطفأً**: قسمٌ
          فارغٌ دائماً يُتعلَّم تجاهلُه، ثمّ يُرفع المفتاحُ يوماً فلا يُنظر إليه. */}
      <MenuReviewQueue />

      {/* **بطاقةٌ لكلّ قسمٍ بصورته — والسوقُ يُتصفَّح بالصور لا بالرموز.**

          قرارُ المالك (٢٠٢٦-٠٨-٠٤): «لازم يكون كرت لكلّ قسم فيه اسمُ القسم
          وصورةُ القسم · وأزرار: عرضُ القسم · تعديل · زرٌّ ذكيّ متاح/غير متاح
          بدل إيقافٍ وتشغيل · والأقسامُ كلُّ ٥ أقسامٍ بسطر».

          **والزرُّ الذكيُّ يقول الحالَ لا الفعل**: «متاح» و«غير متاح» يقرأهما
          من ينظر، **و«تشغيل/إيقاف» يسأل: أهذا حالُه أم ما سيصير إليه؟** */}
      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={S.empty} />
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {list.map((sec) => (
            <div
              key={sec.id}
              className={`flex flex-col overflow-hidden rounded-card border border-line bg-surface transition-shadow hover:shadow-md ${
                sec.active ? "" : "opacity-60"
              }`}
            >
              {/* **الصورةُ أوّلاً — وهي هويّةُ القسم لا زينتُه.**

                  **ولا أيقونةَ بديلاً**: قرارُ المالك (٢٠٢٦-٠٨-٠٤) «رح نرفع
                  صورةً معبّرةً عن القسم، ما بدّي أيقوناتٍ عادية». **ورمزٌ
                  رماديٌّ يملأ الفراغَ يجعل القسمَ يبدو تامّاً وهو ناقص** —
                  فيُنسى أنّ صورتَه لم تُرفع.

                  **فيُقال صراحةً «أضف صورة»** — نقصٌ يُرى يُعالَج. */}
              <div className="relative flex h-24 items-center justify-center bg-page">
                {sec.image_thumb_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={sec.image_thumb_url}
                    alt={sec.name}
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
                <div className="min-w-0 flex-1">
                  <p className="truncate font-bold">{sec.name}</p>
                  <p className="text-xs text-ink-muted">
                    {S.items.replace("{n}", fmtNum(sec.items))}
                    {/* **والهامشُ يُقال حين يُخصّ** — وسكوتُه يعني الوراثة. */}
                    {sec.margin_override !== null && (
                      <> · {S.margin.replace("{n}", fmtNum(sec.margin_override))}</>
                    )}
                  </p>
                </div>

                <div className="flex flex-wrap gap-1.5 [&_button]:flex-1 [&_button]:!px-2 [&_button]:text-xs">
                  <Button variant="secondary" onClick={() => setViewing(sec)}>
                    {S.view}
                  </Button>
                  <Button variant="secondary" onClick={() => setEditing(sec)}>
                    {m.common.edit}
                  </Button>
                  {/* **زرٌّ واحدٌ يقلب الحال** — لا زرّان أحدُهما معطّلٌ دائماً. */}
                  <Button
                    variant={sec.active ? "ghost" : "primary"}
                    onClick={() => toggle(sec)}
                  >
                    {sec.active ? S.makeUnavailable : S.makeAvailable}
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
      {viewing && <SectionItemsModal section={viewing} onClose={() => setViewing(null)} />}
    </PageContainer>
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
  const [icon, setIcon] = useState(section?.icon ?? "food");
  const [sort, setSort] = useState(String(section?.sort_order ?? 0));
  // **فارغٌ يعني «اتبع العام»** — لا صفراً. والحقلُ نصٌّ كي يُفرَّق الفراغُ
  // من الصفر، **ورقمٌ لا يملك أن يكون فارغاً يجعلهما شيئاً واحداً.**
  const [margin, setMargin] = useState(
    section?.margin_override === null || section?.margin_override === undefined
      ? ""
      : String(section.margin_override),
  );
  /**
   * **صورةُ القسم** — و`null` تعني «بلا تغيير»، و`""` تعني «ارفعها».
   *
   * **والفرقُ بينهما لازم**: من فتح النافذة ليصحّح اسماً لا يريد أن يفقد
   * صورتَه، **ومن ضغط «إزالة» يريد ذلك صراحةً.**
   */
  const [imageID, setImageID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    if (!name.trim()) return setError(S.nameRequired);
    setBusy(true);
    setError("");
    try {
      const body = JSON.stringify({
        name: name.trim(),
        icon,
        sort_order: Number(sort) || 0,
        // **وسالبُ الواحدِ يمحو التجاوز** — `null` وحدَه يُقرأ «بلا تغيير».
        margin_override: margin.trim() === "" ? -1 : Number(margin),
        image_media_id: imageID,
      });
      if (section) {
        await api(`/api/v1/admin/sections/${section.id}`, { method: "PATCH", body });
      } else {
        await api("/api/v1/admin/sections", { method: "POST", body });
      }
      onSaved();
    } catch (err) {
      const key = err instanceof ApiError ? (err.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal open title={section ? S.editTitle : S.add} onClose={onClose}>
      <div className="space-y-3">
        <Input label={S.name} value={name} onChange={(e) => setName(e.target.value)} />
        {/* **الصورةُ وجهُ القسم، والأيقونةُ بديلُها.**

            **ولا تُغني إحداهما عن الأخرى**: الصورةُ في السوق حيث المساحة،
            **والأيقونةُ في الشريط والقوائم المختصرة** حيث لا تتّسع صورة. */}
        <ImageUpload
          kind="banner"
          label={S.image}
          initialUrl={section?.image_url}
          onChange={setImageID}
        />
        <CategoryIconPicker value={icon} onChange={setIcon} />
        <div className="grid grid-cols-2 gap-3">
          <Input
            label={S.sort}
            type="number"
            value={sort}
            onChange={(e) => setSort(e.target.value)}
          />
          <div>
            <Input
              label={S.marginField}
              type="number"
              value={margin}
              placeholder={S.marginInherit}
              onChange={(e) => setMargin(e.target.value)}
            />
            <p className="mt-1 text-2xs text-ink-muted">{S.marginHint}</p>
          </div>
        </div>
        {error && <p className="text-sm text-danger">{error}</p>}
        <div className="flex gap-2">
          <Button disabled={busy} onClick={submit}>
            {m.common.save}
          </Button>
          <Button variant="ghost" onClick={onClose}>
            {m.common.cancel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

/**
 * **أصنافُ القسم — كما هي لا كما يراها الزبون.**
 *
 * نقطةُ التصفّح العامّة تُرشِّح: متجرٌ فعّالٌ وقسمٌ فعّالٌ وصنفٌ مُقَرّ. **وهي
 * الصواب للزبون وخطأٌ للإدارة**: من يفتح قسماً ليقرّر إطفاءَه يريد ما فيه
 * كلَّه — **بما لا يظهر ولماذا لا يظهر.**
 *
 * **وقسمٌ يبدو فارغاً وفيه عشرةُ أصنافٍ من متجرٍ مُطفَأ يُحذف بلا علمٍ بما فيه.**
 */
function SectionItemsModal({
  section,
  onClose,
}: {
  section: Section;
  onClose: () => void;
}) {
  const [rows, setRows] = useState<SectionItem[] | null>(null);

  useEffect(() => {
    api<{ items: SectionItem[] }>(`/api/v1/admin/sections/${section.id}/items`)
      .then((r) => setRows(r.items ?? []))
      .catch(() => setRows([]));
  }, [section.id]);

  return (
    <Modal open onClose={onClose} title={`${S.view}: ${section.name}`}>
      {!rows ? (
        <p className="py-6 text-center text-ink-muted">{m.common.loading}</p>
      ) : rows.length === 0 ? (
        <EmptyState icon={IconStore} title={S.noItems} />
      ) : (
        <ul className="max-h-[26rem] divide-y divide-line overflow-y-auto">
          {rows.map((it) => (
            <li key={it.id} className="flex items-center gap-3 py-2 text-sm">
              <span className="min-w-0 flex-1">
                <span className="block font-medium">{it.name}</span>
                <span className="block text-xs text-ink-muted">{it.merchant_name}</span>
              </span>
              <span dir="ltr" className="shrink-0 tabular-nums">
                {fmtNum(it.merchant_price)}
              </span>
              {/* **ولماذا لا يظهر يُقال** — لا يُترك للتخمين. */}
              {!it.approved ? (
                <Badge variant="warning">{S.itemPending}</Badge>
              ) : !it.available ? (
                <Badge variant="warning">{S.itemOut}</Badge>
              ) : it.merchant_status !== "active" ? (
                <Badge variant="danger">{S.itemStoreOff}</Badge>
              ) : (
                <Badge variant="success">{S.itemLive}</Badge>
              )}
            </li>
          ))}
        </ul>
      )}
    </Modal>
  );
}
