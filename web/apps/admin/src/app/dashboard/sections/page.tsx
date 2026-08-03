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

import { useState } from "react";
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
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import MenuReviewQueue from "@/components/MenuReviewQueue";

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
}

export default function SectionsPage() {
  const { data, reload } = useLiveData<{ sections: Section[] }>(
    () => api("/api/v1/admin/sections"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Section | null | "new">(null);

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

      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={S.empty} />
      ) : (
        <div className="space-y-2">
          {list.map((sec) => (
            <div
              key={sec.id}
              className={`flex flex-wrap items-center gap-3 rounded-card border border-line bg-surface p-4 ${
                sec.active ? "" : "opacity-60"
              }`}
            >
              <span className="flex h-11 w-11 items-center justify-center rounded-control bg-primary-light">
                <CategoryIcon name={sec.icon} size={18} />
              </span>
              <div className="min-w-0 flex-1">
                <p className="font-bold">{sec.name}</p>
                <p className="text-xs text-ink-muted">
                  {S.items.replace("{n}", fmtNum(sec.items))}
                  {/* **والهامشُ يُقال حين يُخصّ** — وسكوتُه يعني الوراثة. */}
                  {sec.margin_override !== null && (
                    <> · {S.margin.replace("{n}", fmtNum(sec.margin_override))}</>
                  )}
                </p>
              </div>
              <Badge variant={sec.active ? "success" : "neutral"}>
                {sec.active ? S.active : S.off}
              </Badge>
              <Button variant="ghost" onClick={() => toggle(sec)}>
                {sec.active ? S.turnOff : S.turnOn}
              </Button>
              <Button variant="secondary" onClick={() => setEditing(sec)}>
                {m.common.edit}
              </Button>
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
  const [icon, setIcon] = useState(section?.icon ?? "food");
  const [sort, setSort] = useState(String(section?.sort_order ?? 0));
  // **فارغٌ يعني «اتبع العام»** — لا صفراً. والحقلُ نصٌّ كي يُفرَّق الفراغُ
  // من الصفر، **ورقمٌ لا يملك أن يكون فارغاً يجعلهما شيئاً واحداً.**
  const [margin, setMargin] = useState(
    section?.margin_override === null || section?.margin_override === undefined
      ? ""
      : String(section.margin_override),
  );
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
