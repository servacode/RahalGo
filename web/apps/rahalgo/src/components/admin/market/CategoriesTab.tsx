"use client";

/**
 * **تبويبُ تصنيفات المتاجر** — مطاعم · بقالة · صيدليات… (قرارُ المالك ٢٠٢٦-١٠-٠٦: «تكون
 * بلوحة التحكم مع السوق… مو بالإعدادات لأنها تابعة للسوق»).
 *
 * **ومنها يختار المندوبُ ولوحةُ المتاجر تصنيفَ المتجر.** وكانت مساراتُها في المحرّك
 * (`/admin/categories`) **بلا شاشةٍ تصلها** — فلا يُضاف تصنيفٌ ولا يُعدَّل.
 *
 * **ولا حذف**: متجرٌ قائمٌ يحمل تصنيفه — **فالتصنيفُ يُوقَف** فيختفي من الاختيار.
 *
 * **والترتيبُ بالأسهم كالأقسام** (تدقيقُ السوق ٢٠٢٦-١٠-٢٧) — كان رقماً يُكتب في النافذة ويُطبع على
 * البطاقة «الترتيب ٣». **والأزرارُ تتعطّل وقت الطلب**، وخطأُ الجلب يُقال لا «ما في تصنيفات».
 */

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
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
  IconList,
  IconChevronDown,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { LoadError, StateBox } from "./MarketParts";

const m = getMessages(defaultLocale);
const C = m.admin.market.categories;
const U = m.admin.market.ui;

interface Category {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
}

export default function CategoriesTab() {
  const toast = useToast();
  const { data, error, reload } = useLiveData<Category[]>(
    () => api<Category[]>("/api/v1/admin/categories"),
    ["catalog"],
  );
  const [editing, setEditing] = useState<Category | null | "new">(null);
  const [removing, setRemoving] = useState<Category | null>(null);
  /** **ما ينتظر ردّ المحرّك** — معرّفُ التصنيف، أو «order» للترتيب، أو «remove» للحذف. */
  const [busy, setBusy] = useState<string | null>(null);

  // **والحذفُ لتصنيفٍ لا متجرَ عليه** — والمحرّكُ يردّ غيرَه بعدد متاجره.
  async function remove(c: Category) {
    setBusy("remove");
    try {
      await api(`/api/v1/admin/categories/${c.id}`, { method: "DELETE" });
      setRemoving(null);
      reload();
    } catch (e) {
      setRemoving(null);
      toast.push(errorText(e), "error");
    } finally {
      setBusy(null);
    }
  }

  async function toggle(c: Category, on: boolean) {
    if (busy) return;
    setBusy(c.id);
    try {
      await api(`/api/v1/admin/categories/${c.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: on }),
      });
      reload();
    } catch (e) {
      toast.push(errorText(e), "error");
    } finally {
      setBusy(null);
    }
  }

  if (error && !data) return <LoadError onRetry={reload} />;
  if (!data) return <LoadingState />;
  const cats = [...data].sort((a, b) => a.sort_order - b.sort_order);

  /**
   * **سهمٌ لفوق أو لتحت** — يُعاد ترقيمُ الكلّ ١، ٢، ٣… ويُرسَل ما تغيّر رقمُه وحدَه
   * (`PATCH /categories/{id}` بـ`sort_order` — بلا بابٍ جديد).
   */
  async function nudge(from: number, to: number) {
    if (busy || to < 0 || to >= cats.length) return;
    const next = [...cats];
    const [moved] = next.splice(from, 1);
    if (moved) next.splice(to, 0, moved);
    setBusy("order");
    try {
      for (let i = 0; i < next.length; i++) {
        const c = next[i];
        if (c && c.sort_order !== i + 1) {
          await api(`/api/v1/admin/categories/${c.id}`, {
            method: "PATCH",
            body: JSON.stringify({ sort_order: i + 1 }),
          });
        }
      }
      toast.push(U.orderSaved);
    } catch (e) {
      toast.push(errorText(e), "error");
    } finally {
      setBusy(null);
      reload();
    }
  }

  return (
    <div className="space-y-3">
      <section className="surface flex flex-wrap items-center justify-between gap-2 p-3">
        <p className="min-w-0 flex-1 text-sm text-ink-muted">{C.hint}</p>
        <Button onClick={() => setEditing("new")} className="inline-flex shrink-0 items-center gap-1.5">
          <IconAdd size={16} />
          {C.add}
        </Button>
      </section>
      {cats.length === 0 ? (
        <StateBox
          icon={IconList}
          tone="accent"
          title={C.empty}
          hint={U.categoriesEmptyHint}
          action={<Button onClick={() => setEditing("new")}>{C.add}</Button>}
        />
      ) : (
        // **بطاقاتٌ والخياراتُ أيقونات** (قرارُ المالك ٢٠٢٦-١٠-٢٥).
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {cats.map((c, idx) => (
            <div key={c.id} className={`surface flex flex-col items-center gap-2 p-4 text-center ${c.active ? "" : "opacity-70"}`}>
              <span className="flex h-14 w-14 items-center justify-center rounded-full bg-field text-accent">
                <CategoryIcon name={c.icon} size={26} />
              </span>
              <span className="line-clamp-2 font-bold">{c.name}</span>
              <Badge variant={c.active ? "success" : "warning"}>{c.active ? C.active : C.stopped}</Badge>
              <div className="mt-auto flex items-center justify-center gap-1 pt-1">
                <IconBtn
                  title={c.active ? C.stop : C.activate}
                  className={c.active ? "!text-success" : "!text-warning"}
                  disabled={busy !== null}
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
              <div className="flex w-full items-center gap-1">
                <Button
                  variant="secondary"
                  disabled={idx === 0 || busy !== null}
                  title={U.moveUp}
                  aria-label={U.moveUp}
                  onClick={() => void nudge(idx, idx - 1)}
                  className="!min-h-0 flex-1 !py-1"
                >
                  <IconChevronDown size={16} className="rotate-180" />
                </Button>
                <Button
                  variant="secondary"
                  disabled={idx === cats.length - 1 || busy !== null}
                  title={U.moveDown}
                  aria-label={U.moveDown}
                  onClick={() => void nudge(idx, idx + 1)}
                  className="!min-h-0 flex-1 !py-1"
                >
                  <IconChevronDown size={16} />
                </Button>
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
          busy={busy === "remove"}
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
  // **والترتيبُ لا يُكتب هنا** — يُبدَّل بالأسهم على البطاقة، والجديدُ يُلحق آخرَها.
  const order = category?.sort_order ?? nextOrder;
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
      const body = JSON.stringify({ name: name.trim(), sort_order: order });
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
  disabled = false,
  onClick,
  children,
}: {
  title: string;
  className?: string;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      variant="ghost"
      title={title}
      aria-label={title}
      disabled={disabled}
      onClick={onClick}
      className={`!h-9 !w-9 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full text-ink-muted hover:text-ink ${className}`}
    >
      {children}
    </Button>
  );
}
