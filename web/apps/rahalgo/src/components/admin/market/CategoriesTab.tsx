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

import { useState } from "react";
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
  Switch,
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
        <Button onClick={() => setEditing("new")}>{C.add}</Button>
      </div>
      {cats.length === 0 ? (
        <EmptyState title={C.empty} />
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {cats.map((c) => (
            <div
              key={c.id}
              className="flex items-center justify-between gap-3 surface p-4"
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-bold">{c.name}</span>
                  {!c.active && <Badge variant="warning">{C.stopped}</Badge>}
                </div>
                <span className="text-xs text-ink-muted">
                  {C.order.replace("{n}", String(c.sort_order))}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <Switch checked={c.active} onChange={(v: boolean) => void toggle(c, v)} label={C.active} />
                <Button variant="secondary" onClick={() => setEditing(c)}>
                  {C.edit}
                </Button>
              </div>
            </div>
          ))}
        </div>
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
