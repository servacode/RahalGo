"use client";

/**
 * محرّر الأصناف — **نسخة واحدة مركزية** للإدارة وبوابة المتجر.
 *
 * كان المحرّر الكامل في لوحة الإدارة (521 سطراً)، وبوابة المتجر تملك **الإتاحة
 * وحدها**. فبناء محرّرٍ ثانٍ للمتجر كان سيكرّر المنطق كلّه — وقد رأينا مرّتين في
 * هذا المشروع ما ينتج عن تكرار التركيب: نسختان تنحرفان بلا أن يقصد أحد
 * (R-35 المحفظة، R-44 الشريط العلوي).
 *
 * فما يختلف بين البوابتين **مسارات لا نسخ**: الإدارة تعمل تحت `/admin` والمتجر
 * تحت `/merchant`، والردّ واحد (`catalog.GetMenu`) والحارس في الخادم مختلف.
 *
 * ونسمّيها **الأصناف** لا «القائمة» ولا «المينو»: «القائمة» ملتبسة (list/menu)،
 * و«المينو» تخصّ المطاعم وحدها — ورحّال فيها بقالة وصيدليات وخضار. والوحدة
 * تُسمّى «صنف» في كل المشروع، فجمعُها هو اللفظ المتّسق.
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { Button, Input, Select, Badge, Modal } from "./components";
import { EmptyState } from "./layout";
import { IconAdd, IconEdit, IconDelete, IconStore } from "./icons";

const m = getMessages(defaultLocale);
const L = m.shared.menuEditor;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

export interface ModifierOption {
  name: string;
  price_delta: number;
}
export interface ModifierGroup {
  name: string;
  min_select: number;
  max_select: number;
  options: ModifierOption[];
}
export interface MenuItem {
  id: string;
  section_id: string;
  name: string;
  description: string;
  /** **سعرُ البيع** — ما يدفعه الزبون. تحسبه المنصةُ ولا يُكتب هنا. */
  price: number;
  /** **سعرُ الشراء** — ما يضعه المتجر وهو ما يقبضه. */
  merchant_price: number;
  image_url: string | null;
  image_thumb_url: string | null;
  available: boolean;
  modifiers: ModifierGroup[];
}
export interface MenuSection {
  id: string;
  name: string;
  items: MenuItem[];
}

/** مسارات النقاط — تختلف بالبوابة والعملية واحدة. */
export interface MenuPaths {
  /** قراءة القائمة: `/api/v1/admin/merchants/{id}/menu` أو `/api/v1/merchant/stores/{id}/menu` */
  menu: (merchantID: string) => string;
  sections: (merchantID: string) => string;
  section: (sectionID: string) => string;
  items: (merchantID: string) => string;
  item: (itemID: string) => string;
}

function errText(err: unknown): string {
  const key =
    typeof err === "object" && err && "body" in err
      ? ((err as { body?: { message_key?: string } }).body?.message_key ?? "")
      : "";
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

export function MenuManager({
  api,
  paths,
  merchantID,
  title,
  imageUpload,
  thumb,
}: {
  api: ApiFn;
  paths: MenuPaths;
  merchantID: string;
  title?: ReactNode;
  /** رافع الصور — يبقى محقوناً لأنه يعتمد على عميل الرفع الخاص بكل تطبيق */
  imageUpload?: (initialUrl: string | null | undefined, onChange: (id: string | null) => void) => ReactNode;
  thumb?: (url: string | null, alt: string) => ReactNode;
}) {
  const [sections, setSections] = useState<MenuSection[]>([]);
  const [error, setError] = useState("");
  const [sectionName, setSectionName] = useState("");
  const [editing, setEditing] = useState<{ item: MenuItem | null; sectionId: string } | null>(null);

  const load = useCallback(async () => {
    try {
      setSections(await api<MenuSection[]>(paths.menu(merchantID)));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [api, paths, merchantID]);

  useEffect(() => {
    void load();
  }, [load]);

  async function addSection(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api(paths.sections(merchantID), {
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
      await api(paths.section(sectionID), { method: "DELETE" });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function toggleAvailable(item: MenuItem) {
    try {
      await api(paths.item(item.id), {
        method: "PATCH",
        body: JSON.stringify({ available: !item.available }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function deleteItem(item: MenuItem) {
    if (!confirm(L.confirmDeleteItem)) return;
    try {
      await api(paths.item(item.id), { method: "DELETE" });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        {title}
        {/* إضافة قسم في الرأس: أول ما يحتاجه متجرٌ فارغ، وآخر ما يحتاجه متجرٌ ممتلئ */}
        <form onSubmit={addSection} className="flex items-end gap-2">
          <Input
            id="sec-name"
            label={L.sectionName}
            required
            value={sectionName}
            onChange={(e) => setSectionName(e.target.value)}
          />
          <Button type="submit" className="flex shrink-0 items-center gap-1.5">
            <IconAdd size={16} />
            {L.addSection}
          </Button>
        </form>
      </div>

      {error && (
        <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {sections.length === 0 ? (
        <EmptyState icon={IconStore} title={L.empty} />
      ) : (
        <div className="space-y-5">
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
                    {L.addItem}
                  </Button>
                  {/* الحذف لقسمٍ فارغ وحده: حذفُ قسمٍ فيه أصناف يمحوها معه بلا قصد */}
                  {sec.items.length === 0 && (
                    <Button variant="ghost" onClick={() => deleteSection(sec.id)}>
                      <IconDelete size={15} className="text-danger" />
                    </Button>
                  )}
                </div>
              </div>

              {sec.items.length === 0 ? (
                <p className="p-6 text-center text-sm text-ink-muted">{L.noItems}</p>
              ) : (
                <ul className="divide-y divide-line">
                  {sec.items.map((item) => (
                    <li key={item.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                      {thumb?.(item.image_thumb_url, item.name)}
                      <div className="min-w-48 flex-1">
                        <p
                          className={`font-medium ${
                            item.available ? "" : "text-ink-muted line-through"
                          }`}
                        >
                          {item.name}
                        </p>
                        {item.description && (
                          <p className="text-sm text-ink-muted">{item.description}</p>
                        )}
                        {item.modifiers.length > 0 && (
                          <div className="mt-1 flex flex-wrap gap-1">
                            {item.modifiers.map((g, i) => (
                              <Badge key={i} variant="neutral">
                                {g.name} ({fmtNum(g.options.length)})
                              </Badge>
                            ))}
                          </div>
                        )}
                      </div>
                      <span className="font-bold text-primary-dark" dir="ltr">
                        {/* **السعران معاً لمن يراهما.**

                            المتجرُ يرى سعرَه وحدَه (`merchant_price === price`
                            حين لا هامش، والخادمُ يُسقط سعرَ الشراء عن الزبون).
                            **والأدمن يرى الاثنين** — وهو من يضع الهامش،
                            **ومن يضع رقماً لا يرى أثرَه يضعه أعمى.** */}
                        {fmtNum(item.merchant_price || item.price)} {m.common.currency}
                        {item.merchant_price > 0 && item.price > item.merchant_price && (
                          <span className="ms-1.5 text-2xs text-success">
                            ← {fmtNum(item.price)}
                          </span>
                        )}
                      </span>
                      <Badge variant={item.available ? "success" : "warning"}>
                        {item.available ? L.available : L.unavailable}
                      </Badge>
                      <div className="flex gap-1.5">
                        <Button variant="secondary" onClick={() => toggleAvailable(item)}>
                          {item.available ? L.markUnavailable : L.markAvailable}
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => setEditing({ item, sectionId: sec.id })}
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
      )}

      {editing && (
        <ItemModal
          api={api}
          paths={paths}
          merchantID={merchantID}
          sections={sections}
          editing={editing}
          imageUpload={imageUpload}
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
  api,
  paths,
  merchantID,
  sections,
  editing,
  imageUpload,
  onClose,
  onSaved,
}: {
  api: ApiFn;
  paths: MenuPaths;
  merchantID: string;
  sections: MenuSection[];
  editing: { item: MenuItem | null; sectionId: string };
  imageUpload?: (
    initialUrl: string | null | undefined,
    onChange: (id: string | null) => void,
  ) => ReactNode;
  onClose: () => void;
  onSaved: () => void;
}) {
  const item = editing.item;
  const [name, setName] = useState(item?.name ?? "");
  const [description, setDescription] = useState(item?.description ?? "");
  // **المُحرَّرُ سعرُ الشراء لا سعرُ البيع.**
  //
  // **ولو حُرِّر سعرُ البيع لَضاع الهامشُ في أوّل تعديل**: يفتح المتجرُ الصنفَ
  // فيرى ١٢٬٠٠٠ (وسعرُه ١٠٬٠٠٠)، **فيحفظ فيصير سعرُ شرائه اثني عشر** ويُضاف
  // عليه الهامشُ من جديد — **ورقمٌ يرتفع بكلّ فتحةٍ للنافذة.**
  const [price, setPrice] = useState(item ? String(item.merchant_price || item.price) : "");
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
        await api(paths.item(item.id), { method: "PATCH", body: JSON.stringify(body) });
      } else {
        await api(paths.items(merchantID), { method: "POST", body: JSON.stringify(body) });
      }
      onSaved();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={item ? L.editItem : L.addItem}>
      <form onSubmit={submit} className="max-h-[70vh] space-y-4 overflow-y-auto p-0.5">
        <Input
          id="i-name"
          label={L.itemName}
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <div className="grid grid-cols-2 gap-3">
          <div>
          {/* **اللفظُ يقول أيَّ سعرٍ هو.**

              «السعر» وحدَها تحتمل الاثنين، **وصاحبُ المتجر يقرؤها سعرَ البيع**
              فيضع فيها ما يريد أن يدفعه الزبون — **فيُضاف عليه هامشُنا فيصير
              الصنفُ أغلى ممّا قصد**، ويشكو من رقمٍ لم يضعه. */}
          <Input
            id="i-price"
            label={`${L.price} (${m.common.currency})`}
            type="number"
            min="0"
            required
            value={price}
            onChange={(e) => setPrice(e.target.value)}
          />
          <p className="mt-1 text-2xs text-ink-muted">{L.priceHint}</p>
          </div>
          <Select
            id="i-section"
            label={L.section}
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
          label={L.itemDescription}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        {imageUpload?.(item?.image_thumb_url, setImageID)}

        <div>
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium">{L.modifiers}</span>
            <Button
              type="button"
              variant="secondary"
              onClick={() =>
                setGroups((gs) => [...gs, { name: "", min_select: 0, max_select: 1, options: [] }])
              }
              className="flex items-center gap-1"
            >
              <IconAdd size={14} />
              {L.addGroup}
            </Button>
          </div>

          <div className="space-y-3">
            {groups.map((g, gi) => (
              <div key={gi} className="rounded-control border border-line bg-page/50 p-3">
                <div className="mb-2 flex items-end gap-2">
                  <div className="flex-1">
                    <Input
                      id={`g-name-${gi}`}
                      label={L.groupName}
                      required
                      value={g.name}
                      onChange={(e) => updateGroup(gi, { name: e.target.value })}
                    />
                  </div>
                  <div className="w-20">
                    <Input
                      id={`g-min-${gi}`}
                      label={L.minSelect}
                      type="number"
                      min="0"
                      value={g.min_select}
                      onChange={(e) => updateGroup(gi, { min_select: Number(e.target.value) })}
                    />
                  </div>
                  <div className="w-20">
                    <Input
                      id={`g-max-${gi}`}
                      label={L.maxSelect}
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
                {/* «إلزامية» تُشتق من الحدّ الأدنى لا تُكتب: رقمٌ واحد لا حقلان يتناقضان */}
                <Badge variant={g.min_select > 0 ? "warning" : "neutral"}>
                  {g.min_select > 0 ? L.required : L.optional}
                </Badge>

                <div className="mt-2 space-y-2">
                  {g.options.map((o, oi) => (
                    <div key={oi} className="flex items-center gap-2">
                      <Input
                        id={`o-name-${gi}-${oi}`}
                        placeholder={L.optionName}
                        required
                        value={o.name}
                        onChange={(e) => updateOption(gi, oi, { name: e.target.value })}
                      />
                      <div className="w-32">
                        <Input
                          id={`o-delta-${gi}-${oi}`}
                          placeholder={L.priceDelta}
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
                    {L.addOption}
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
