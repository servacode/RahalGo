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
 * # والقسمُ صورةٌ واسم — لا أكثر
 *
 * كان معه **شبكةُ أيقوناتٍ تُختار منها وحقلُ هامشٍ خاصّ**. فحقلان لهويّةٍ
 * واحدة: **من اختار أيقونةً ظنّ أنّه أنهى وجهَ القسم** ولم يرفع صورة.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «ألغِ الأيقوناتِ والهامش، ولازم تعرض الصورةَ بمكان
 * الأيقونة كصورة لا أيقونة».)
 *
 * **وهامشُ القسم لم يُحذف من القاعدة** — العمودُ باقٍ وما ضُبط سابقاً يعمل.
 * **ولا يُضبط من هذه الشاشة**: التسعيرُ يُدار من الإعدادات، **ورقمٌ يُغيَّر في
 * موضعين يُنسى أحدُهما.**
 */

import { useState } from "react";
import { useRouter } from "next/navigation";
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
  useLiveData,
  IconStore,
  IconStatus,
  IconCamera,
} from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import MenuReviewQueue from "@/components/admin/MenuReviewQueue";
import ImageUpload from "@/components/admin/ImageUpload";

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

export default function SectionsPage() {
  const { data, reload } = useLiveData<{ sections: Section[] }>(
    () => api("/api/v1/admin/sections"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Section | null | "new">(null);
  const router = useRouter();

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
              className={`flex flex-col overflow-hidden surface transition-shadow hover:elev-2 ${
                sec.active ? "" : "opacity-60"
              }`}
            >
              {/* **الصورةُ أوّلاً — وهي هويّةُ القسم لا زينتُه.**

                  **ولا أيقونةَ بديلاً**: قرارُ المالك (٢٠٢٦-٠٨-٠٤) «رح نرفع
                  صورةً معبّرةً عن القسم، ما بدّي أيقوناتٍ عادية». **ورمزٌ
                  رماديٌّ يملأ الفراغَ يجعل القسمَ يبدو تامّاً وهو ناقص** —
                  فيُنسى أنّ صورتَه لم تُرفع.

                  **فيُقال صراحةً «أضف صورة»** — نقصٌ يُرى يُعالَج. */}
              <div className="relative flex aspect-[4/3] items-center justify-center bg-field">
                {sec.image_url || sec.image_thumb_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={mediaUrl(sec.image_url ?? sec.image_thumb_url) ?? ""}
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
                {/* **الاسمُ يميناً والعددُ يساراً** — لا تحته.

                    سطرٌ تحت الاسم يُقرأ امتداداً له، **والعددُ رقمٌ يُمسح
                    بالعين في عمودٍ واحدٍ حين يقابل الاسمَ.** (قرارُ المالك
                    ٢٠٢٦-٠٨-٠٤: «عددُ الأصناف خلّيها محاذاةً لليسار أفضل».) */}
                <div className="flex min-w-0 flex-1 items-baseline justify-between gap-2">
                  <p className="truncate font-bold">{sec.name}</p>
                  <p className="shrink-0 text-xs text-ink-muted">
                    {S.items.replace("{n}", fmtNum(sec.items))}
                  </p>
                </div>

                <div className="flex flex-wrap gap-1.5 [&_button]:flex-1 [&_button]:!px-2 [&_button]:text-xs">
                  {/* **صفحةٌ لا نافذة.** (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «عرضُ القسم
                      يجب أن يفتح صفحةً منفصلة وليس نافذةً منبثقة».) */}
                  <Button
                    variant="secondary"
                    onClick={() => router.push(`/dashboard/sections/${sec.id}`)}
                  >
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
  const [sort, setSort] = useState(String(section?.sort_order ?? 0));
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
        sort_order: Number(sort) || 0,
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
        {/* **الصورةُ أوّلاً — بترتيب البطاقة نفسِه.**

            قرارُ المالك (٢٠٢٦-٠٨-٠٤): «أوّلُ شيءٍ الصورة، بعدها اسمُ القسم،
            بعدها الترتيب». **ونافذةُ التحرير تُقرأ كما تُقرأ البطاقة** — ومن
            رأى الصورةَ فوقها في السوق يبحث عنها فوقها هنا. */}
        <ImageUpload
          kind="banner"
          label={S.image}
          initialUrl={section?.image_url}
          onChange={setImageID}
        />
        <Input label={S.name} value={name} onChange={(e) => setName(e.target.value)} />
        <Input
          label={S.sort}
          type="number"
          value={sort}
          onChange={(e) => setSort(e.target.value)}
        />
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

