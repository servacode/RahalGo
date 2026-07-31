"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  IconPrev,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  IconAdd,
  IconEdit,
  IconDelete,
  IconStore,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import ImageUpload, { MediaThumb } from "@/components/ImageUpload";

const m = getMessages(defaultLocale);

interface ModifierOption {
  name: string;
  price_delta: number;
}
interface ModifierGroup {
  name: string;
  min_select: number;
  max_select: number;
  options: ModifierOption[];
}
interface MenuItem {
  id: string;
  section_id: string;
  name: string;
  description: string;
  price: number;
  image_url: string | null;
  image_thumb_url: string | null;
  available: boolean;
  modifiers: ModifierGroup[];
}
interface MenuSection {
  id: string;
  name: string;
  items: MenuItem[];
}
interface Merchant {
  id: string;
  name: string;
  category_icon: string;
}
interface MerchantPage {
  merchants: Merchant[];
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

export default function MenuPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [merchantName, setMerchantName] = useState("");
  const [sections, setSections] = useState<MenuSection[]>([]);
  const [error, setError] = useState("");
  const [sectionName, setSectionName] = useState("");
  const [editing, setEditing] = useState<{ item: MenuItem | null; sectionId: string } | null>(null);

  const load = useCallback(async () => {
    try {
      setSections(await api<MenuSection[]>(`/api/v1/admin/merchants/${id}/menu`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [id]);

  useEffect(() => {
    void load();
    // اسم المتجر للعنوان
    api<MerchantPage>(`/api/v1/admin/merchants?per_page=100`)
      .then((p) => {
        const mr = p.merchants.find((x) => x.id === id);
        if (mr) setMerchantName(`${mr.category_icon} ${mr.name}`);
      })
      .catch(() => undefined);
  }, [id, load]);

  async function addSection(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api(`/api/v1/admin/merchants/${id}/menu/sections`, {
        method: "POST",
        body: JSON.stringify({ name: sectionName }),
      });
      setSectionName("");
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function deleteSection(sectionID: string) {
    try {
      await api(`/api/v1/admin/menu/sections/${sectionID}`, { method: "DELETE" });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function toggleAvailable(item: MenuItem) {
    try {
      await api(`/api/v1/admin/menu/items/${item.id}`, {
        method: "PATCH",
        body: JSON.stringify({ available: !item.available }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function deleteItem(item: MenuItem) {
    if (!confirm(m.admin.menu.confirmDeleteItem)) return;
    try {
      await api(`/api/v1/admin/menu/items/${item.id}`, { method: "DELETE" });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <button
            onClick={() => router.push("/dashboard/merchants")}
            className="mb-1 text-sm text-ink-muted hover:text-primary"
          >
            <IconPrev size={15} /> {m.admin.merchants.title}
          </button>
          <h1 className="flex items-center gap-2 text-2xl font-bold">
            <IconStore className="text-primary" />
            {m.admin.menu.title}: {merchantName}
          </h1>
        </div>
        <form onSubmit={addSection} className="flex items-center gap-2">
          <Input
            id="sec-name"
            placeholder={m.admin.menu.sectionName}
            required
            value={sectionName}
            onChange={(e) => setSectionName(e.target.value)}
          />
          <Button type="submit" className="flex shrink-0 items-center gap-1.5">
            <IconAdd size={16} />
            {m.admin.menu.addSection}
          </Button>
        </form>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {sections.length === 0 && (
        <div className="rounded-card border border-line bg-surface p-10 text-center text-ink-muted">
          {m.admin.menu.empty}
        </div>
      )}

      <div className="space-y-6">
        {sections.map((sec) => (
          <section key={sec.id} className="rounded-card border border-line bg-surface">
            <div className="flex items-center justify-between border-b border-line px-4 py-3">
              <h2 className="font-bold">{sec.name}</h2>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  onClick={() => setEditing({ item: null, sectionId: sec.id })}
                  className="flex items-center gap-1.5"
                >
                  <IconAdd size={15} />
                  {m.admin.menu.addItem}
                </Button>
                {sec.items.length === 0 && (
                  <Button variant="ghost" onClick={() => deleteSection(sec.id)}>
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                )}
              </div>
            </div>

            {sec.items.length === 0 ? (
              <p className="p-6 text-center text-sm text-ink-muted">{m.admin.menu.noItems}</p>
            ) : (
              <ul className="divide-y divide-line">
                {sec.items.map((item) => (
                  <li key={item.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                    <MediaThumb
                      url={item.image_thumb_url}
                      alt={item.name}
                      fallback={item.name}
                      size={48}
                    />
                    <div className="min-w-48 flex-1">
                      <p className={`font-medium ${item.available ? "" : "text-ink-muted line-through"}`}>
                        {item.name}
                      </p>
                      {item.description && (
                        <p className="text-sm text-ink-muted">{item.description}</p>
                      )}
                      {item.modifiers.length > 0 && (
                        <div className="mt-1 flex flex-wrap gap-1">
                          {item.modifiers.map((g, i) => (
                            <Badge key={i} variant="neutral">
                              {g.name} ({g.options.length})
                            </Badge>
                          ))}
                        </div>
                      )}
                    </div>
                    <span className="font-bold text-primary-dark">
                      {fmtNum(item.price)} {m.common.currency}
                    </span>
                    <Badge variant={item.available ? "success" : "warning"}>
                      {item.available ? m.admin.menu.available : m.admin.menu.unavailable}
                    </Badge>
                    <div className="flex gap-1.5">
                      <Button variant="secondary" onClick={() => toggleAvailable(item)}>
                        {item.available ? m.admin.menu.markUnavailable : m.admin.menu.markAvailable}
                      </Button>
                      <Button
                        variant="ghost"
                        onClick={() => setEditing({ item, sectionId: sec.id })}
                        className="flex items-center gap-1"
                      >
                        <IconEdit size={15} />
                      </Button>
                      <Button variant="ghost" onClick={() => deleteItem(item)}>
                        <IconDelete size={15} className="text-danger" />
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        ))}
      </div>

      {editing && (
        <ItemModal
          merchantId={id}
          sections={sections}
          editing={editing}
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

function ItemModal({
  merchantId,
  sections,
  editing,
  onClose,
  onSaved,
}: {
  merchantId: string;
  sections: MenuSection[];
  editing: { item: MenuItem | null; sectionId: string };
  onClose: () => void;
  onSaved: () => void;
}) {
  const item = editing.item;
  const [name, setName] = useState(item?.name ?? "");
  const [description, setDescription] = useState(item?.description ?? "");
  const [price, setPrice] = useState(item ? String(item.price) : "");
  const [sectionId, setSectionId] = useState(editing.sectionId);
  const [groups, setGroups] = useState<ModifierGroup[]>(
    item?.modifiers.map((g) => ({ ...g, options: [...g.options] })) ?? [],
  );
  // null = لم تُلمس (لا تُرسل)، "" = إزالة، معرف = صورة جديدة
  const [imageID, setImageID] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  function updateGroup(i: number, patch: Partial<ModifierGroup>) {
    setGroups((gs) => gs.map((g, gi) => (gi === i ? { ...g, ...patch } : g)));
  }
  function updateOption(gi: number, oi: number, patch: Partial<ModifierOption>) {
    setGroups((gs) =>
      gs.map((g, i) =>
        i === gi
          ? { ...g, options: g.options.map((o, j) => (j === oi ? { ...o, ...patch } : o)) }
          : g,
      ),
    );
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const body = {
      section_id: sectionId,
      name,
      description,
      price: Number(price) || 0,
      modifiers: groups,
      ...(imageID !== null ? { image_media_id: imageID } : {}),
    };
    try {
      if (item) {
        await api(`/api/v1/admin/menu/items/${item.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
      } else {
        await api(`/api/v1/admin/merchants/${merchantId}/menu/items`, {
          method: "POST",
          body: JSON.stringify(body),
        });
      }
      onSaved();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={item ? m.admin.menu.editItem : m.admin.menu.addItem}>
      <form onSubmit={submit} className="max-h-[70vh] space-y-4 overflow-y-auto p-0.5">
        <Input
          id="i-name"
          label={m.admin.menu.itemName}
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <div className="grid grid-cols-2 gap-3">
          <Input
            id="i-price"
            label={`${m.admin.menu.price} (${m.common.currency})`}
            type="number"
            min="0"
            required
            value={price}
            onChange={(e) => setPrice(e.target.value)}
          />
          <Select
            id="i-section"
            label={m.admin.merchants.category}
            value={sectionId}
            onChange={(e) => setSectionId(e.target.value)}
          >
            {sections.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </Select>
        </div>
        <Input
          id="i-desc"
          label={m.admin.menu.itemDescription}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <ImageUpload
          kind="menu_item"
          label={m.admin.menu.itemImage}
          initialUrl={item?.image_thumb_url}
          onChange={setImageID}
        />

        <div>
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium">{m.admin.menu.modifiers}</span>
            <Button
              type="button"
              variant="secondary"
              onClick={() =>
                setGroups((gs) => [...gs, { name: "", min_select: 0, max_select: 1, options: [] }])
              }
              className="flex items-center gap-1"
            >
              <IconAdd size={14} />
              {m.admin.menu.addGroup}
            </Button>
          </div>

          <div className="space-y-3">
            {groups.map((g, gi) => (
              <div key={gi} className="rounded-control border border-line bg-page/50 p-3">
                <div className="mb-2 flex items-end gap-2">
                  <div className="flex-1">
                    <Input
                      id={`g-name-${gi}`}
                      label={m.admin.menu.groupName}
                      required
                      value={g.name}
                      onChange={(e) => updateGroup(gi, { name: e.target.value })}
                    />
                  </div>
                  <div className="w-20">
                    <Input
                      id={`g-min-${gi}`}
                      label={m.admin.menu.minSelect}
                      type="number"
                      min="0"
                      value={g.min_select}
                      onChange={(e) => updateGroup(gi, { min_select: Number(e.target.value) })}
                    />
                  </div>
                  <div className="w-20">
                    <Input
                      id={`g-max-${gi}`}
                      label={m.admin.menu.maxSelect}
                      type="number"
                      min="1"
                      value={g.max_select}
                      onChange={(e) => updateGroup(gi, { max_select: Number(e.target.value) })}
                    />
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => setGroups((gs) => gs.filter((_, i) => i !== gi))}
                  >
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                </div>
                <Badge variant={g.min_select > 0 ? "warning" : "neutral"}>
                  {g.min_select > 0 ? m.admin.menu.required : m.admin.menu.optional}
                </Badge>

                <div className="mt-2 space-y-2">
                  {g.options.map((o, oi) => (
                    <div key={oi} className="flex items-center gap-2">
                      <Input
                        id={`o-name-${gi}-${oi}`}
                        placeholder={m.admin.menu.optionName}
                        required
                        value={o.name}
                        onChange={(e) => updateOption(gi, oi, { name: e.target.value })}
                      />
                      <div className="w-32">
                        <Input
                          id={`o-delta-${gi}-${oi}`}
                          placeholder={m.admin.menu.priceDelta}
                          type="number"
                          value={o.price_delta}
                          onChange={(e) =>
                            updateOption(gi, oi, { price_delta: Number(e.target.value) })
                          }
                        />
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        onClick={() =>
                          updateGroup(gi, { options: g.options.filter((_, j) => j !== oi) })
                        }
                      >
                        <IconDelete size={14} className="text-danger" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() =>
                      updateGroup(gi, { options: [...g.options, { name: "", price_delta: 0 }] })
                    }
                    className="flex items-center gap-1"
                  >
                    <IconAdd size={13} />
                    {m.admin.menu.addOption}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
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
