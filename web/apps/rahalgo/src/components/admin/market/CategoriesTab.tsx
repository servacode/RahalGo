"use client";

/**
 * **تبويبُ تصنيفات المتاجر** — مطاعم · بقالة · صيدليات… (قرارُ المالك ٢٠٢٦-١٠-٠٦: «تكون
 * بلوحة التحكم مع السوق… مو بالإعدادات لأنها تابعة للسوق»).
 *
 * **ومنها يختار المندوبُ ولوحةُ المتاجر تصنيفَ المتجر.** وكانت مساراتُها في المحرّك
 * (`/admin/categories`) **بلا شاشةٍ تصلها** — فلا يُضاف تصنيفٌ ولا يُعدَّل.
 *
 * **ولا حذف**: متجرٌ قائمٌ يحمل تصنيفه — **فالتصنيفُ يُوقَف** فيختفي من الاختيار.
 */

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  EmptyState,
  LoadingState,
  useLiveData,
  useToast,
  FormActions,
  Confirm,
  CategoryIcon,
  IconEdit,
  IconDelete,
  IconCheck,
  IconBlock,
  IconAdd,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.admin.market.categories;

interface Category {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
}

export default function CategoriesTab() {
  const toast = useToast();
  const { data, loading, reload } = useLiveData<Category[]>(
    () => api<Category[]>("/api/v1/admin/categories"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Category | null | "new">(null);
  const [removing, setRemoving] = useState<Category | null>(null);

  // **والحذفُ لتصنيفٍ لا متجرَ عليه** — والمحرّكُ يردّ غيرَه بعدد متاجره.
  async function remove(c: Category) {
    try {
      await api(`/api/v1/admin/categories/${c.id}`, { method: "DELETE" });
      setRemoving(null);
      reload();
    } catch (e) {
      setRemoving(null);
      toast.push(errorText(e), "error");
    }
  }

  async function toggle(c: Category, on: boolean) {
    try {
      await api(`/api/v1/admin/categories/${c.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: on }),
      });
      reload();
    } catch (e) {
      toast.push(errorText(e), "error");
    }
  }

  if (loading && !data) return <LoadingState />;
  const cats = [...(data ?? [])].sort((a, b) => a.sort_order - b.sort_order);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm text-ink-muted">{C.hint}</p>
        <Button onClick={() => setEditing("new")} className="inline-flex items-center gap-1.5">
          <IconAdd size={16} />
          {C.add}
        </Button>
      </div>
      {cats.length === 0 ? (
        <EmptyState title={C.empty} />
      ) : (
        // **بطاقاتٌ والخياراتُ أيقونات** (قرارُ المالك ٢٠٢٦-١٠-٢٥).
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {cats.map((c) => (
            <div key={c.id} className={`surface flex flex-col items-center gap-2 p-4 text-center ${c.active ? "" : "opacity-70"}`}>
              <span className="flex h-14 w-14 items-center justify-center rounded-full bg-field text-accent">
                <CategoryIcon name={c.icon} size={26} />
              </span>
              <span className="line-clamp-2 font-bold">{c.name}</span>
              <span className="text-2xs text-ink-muted">{C.order.replace("{n}", String(c.sort_order))}</span>
              {!c.active && <Badge variant="warning">{C.stopped}</Badge>}
              <div className="mt-auto flex items-center justify-center gap-1 pt-1">
                <IconBtn
                  title={c.active ? C.stop : C.activate}
                  className={c.active ? "!text-success" : "!text-warning"}
                  onClick={() => void toggle(c, !c.active)}
                >
                  {c.active ? <IconCheck size={18} /> : <IconBlock size={18} />}
                </IconBtn>
                <IconBtn title={C.edit} onClick={() => setEditing(c)}>
                  <IconEdit size={17} />
                </IconBtn>
                <IconBtn title={C.remove} className="!text-danger" onClick={() => setRemoving(c)}>
                  <IconDelete size={17} />
                </IconBtn>
              </div>
            </div>
          ))}
        </div>
      )}
      {removing && (
        <Confirm
          open
          title={C.removeTitle}
          body={C.removeBody.replace("{name}", removing.name)}
          confirmLabel={C.remove}
          onConfirm={() => void remove(removing)}
          onCancel={() => setRemoving(null)}
        />
      )}
      {editing && (
        <CategoryModal
          category={editing === "new" ? null : editing}
          nextOrder={(cats.at(-1)?.sort_order ?? 0) + 1}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            reload();
          }}
        />
      )}
    </div>
  );
}

function CategoryModal({
  category,
  nextOrder,
  onClose,
  onSaved,
}: {
  category: Category | null;
  nextOrder: number;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(category?.name ?? "");
  const [order, setOrder] = useState(String(category?.sort_order ?? nextOrder));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    if (!name.trim()) {
      setError(C.nameRequired);
      return;
    }
    setBusy(true);
    setError("");
    try {
      const body = JSON.stringify({ name: name.trim(), sort_order: Number(order) || 0 });
      if (category) {
        await api(`/api/v1/admin/categories/${category.id}`, { method: "PATCH", body });
      } else {
        await api("/api/v1/admin/categories", { method: "POST", body });
      }
      onSaved();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open title={category ? C.edit : C.add} onClose={onClose}>
      <div className="space-y-3">
        <Input label={C.name} value={name} onChange={(e) => setName(e.target.value)} />
        <Input
          label={C.sort}
          type="number"
          inputMode="numeric"
          value={order}
          onChange={(e) => setOrder(e.target.value)}
        />
        {error && <p className="text-sm text-danger">{error}</p>}
        <FormActions onSave={submit} onCancel={onClose} busy={busy} />
      </div>
    </Modal>
  );
}

/** **زرُّ أيقونةٍ دائريّ** — الأيقونةُ في وسط ظلّها. */
function IconBtn({
  title,
  className = "",
  onClick,
  children,
}: {
  title: string;
  className?: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      variant="ghost"
      title={title}
      aria-label={title}
      onClick={onClick}
      className={`!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full text-ink-muted hover:text-ink ${className}`}
    >
      {children}
    </Button>
  );
}
