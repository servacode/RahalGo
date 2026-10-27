"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/menu — «أصنافي»: متوفر / غير متوفر وتعديلُ السعر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `MenuScreen.kt` و`MenuViewModel.kt` في تطبيق المتجر.)
 *
 * **الأصنافُ تحت أقسام السوق** (`platform_section_name`) — كما يجمعها
 * التطبيق، **والأكثرُ أصنافاً أوّلاً.**
 *
 * **والمفتاحُ يتبدّل فوراً** ثمّ يُسأل المحرّك — **فإن ردّه عاد كما كان**
 * وقيل السببُ كما قاله الخادم.
 *
 * # والسعرُ بعملة الصنف
 *
 * **صنفٌ بالدولار يُعدَّل بالدولار** (`price_currency = USD`) **والليرةُ
 * تُحسب منه في المحرّك** بسعر صرف المتجر مدوَّرةً لأقرب خمسة — **وتُعرض هنا
 * قبل الحفظ بالقاعدة نفسِها** (`usdToSyp` في `ItemDraft.kt`). **وصنفٌ
 * بالليرة يُعدَّل بالليرة.** ومن أرسل ليرةً لصنفٍ بالدولار محا دولارَه —
 * **فلا يُرسَل إلّا ما يخصّ عملتَه.**
 */

import Link from "next/link";
import { useRouter } from "next/navigation";
import DrinksCard from "@/components/store/DrinksCard";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getMessages,
  defaultLocale,
  errorText,
  fmtMoney,
  fmtNum,
} from "@rahalgo/i18n";
import {
  Alert,
  Button,
  ButtonLink,
  EmptyState,
  IconAdd,
  IconCopy,
  IconEdit,
  FormActions,
  IconList,
  IconSearch,
  Input,
  LoadingState,
  MediaThumb,
  Modal,
  ReloadState,
  Select,
  Switch,
  useLiveData,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MenuItem, MenuSection, SectionRef } from "@/lib/store/types";
import { dollars, fill } from "@/lib/customer/texts";
import { parseUsd, usdInput, usdToSyp } from "@/lib/store/usd";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const M = m.store.menu;

function priceLine(it: MenuItem): string {
  return it.price_currency === "USD" && it.merchant_price_usd != null
    ? `${dollars(it.merchant_price_usd)} · ${fmtMoney(it.price)}`
    : fmtMoney(it.price);
}

function PriceDialog({
  item,
  rate,
  onClose,
  onSaved,
}: {
  item: MenuItem;
  rate: number | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const usd = item.price_currency === "USD";
  const [text, setText] = useState(
    usd
      ? item.merchant_price_usd != null
        ? String(item.merchant_price_usd)
        : ""
      : String(item.price || ""),
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const toast = useShopToast();

  const usdValue = usd ? parseUsd(text) : null;
  const sypValue = usd ? null : Number(text.replace(/\D/g, "")) || 0;

  async function save() {
    if (
      (usd && (usdValue == null || usdValue <= 0)) ||
      (!usd && (!sypValue || sypValue <= 0))
    ) {
      setError(M.badPrice);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}`, {
        method: "PATCH",
        body: JSON.stringify(
          usd
            ? { price_currency: "USD", price_usd: usdValue }
            : { price_currency: "SYP", price: sypValue },
        ),
      });
      toast(M.saved, "success");
      onSaved();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${M.editPrice} · ${item.name}`}>
      <div className="flex flex-col gap-3">
        <Input
          label={usd ? M.priceUsd : M.priceSyp}
          inputMode={usd ? "decimal" : "numeric"}
          dir="ltr"
          value={text}
          onChange={(e) =>
            setText(
              usd
                ? usdInput(e.target.value)
                : e.target.value.replace(/\D/g, "").slice(0, 9),
            )
          }
          autoFocus
        />
        {usd && rate != null && (
          <p className="text-xs text-ink-muted">
            {fill(M.rate, { rate: fmtNum(rate) })}
            {usdValue != null && usdValue > 0 && (
              <>
                {" — "}
                {fill(M.usdPreview, {
                  syp: fmtMoney(usdToSyp(usdValue, rate)),
                })}
              </>
            )}
          </p>
        )}
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void save()}
          onCancel={onClose}
          busy={busy}
        />
      </div>
    </Modal>
  );
}

/** **أقسامُ السوق التي فيها أصناف** — لمرشِّح «تغيير الأسعار». */
function sectionsOf(data: MenuSection[]): { id: string; name: string }[] {
  const seen = new Map<string, string>();
  for (const it of data.flatMap((s) => s.items)) {
    if (it.platform_section_id && !seen.has(it.platform_section_id)) {
      seen.set(it.platform_section_id, it.platform_section_name);
    }
  }
  return [...seen.entries()].map(([id, name]) => ({ id, name }));
}

interface BulkPreview {
  count: number;
  samples: { name: string; old_price: number; new_price: number }[];
  applied: boolean;
}

/**
 * **تغييرُ الأسعار بنسبة** — `POST /stores/{id}/menu/prices`. **بلا `confirm` معاينةٌ لا
 * تغيّر شيئاً**: العددُ وأمثلةٌ، ثمّ «طبّق» يرسلها نفسَها بـ`confirm: true`.
 */
function BulkPriceDialog({
  storeId,
  sections,
  onClose,
  onApplied,
}: {
  storeId: string;
  sections: { id: string; name: string }[];
  onClose: () => void;
  onApplied: () => void;
}) {
  const B = M.bulk;
  const toast = useShopToast();
  const [text, setText] = useState("");
  const [section, setSection] = useState("");
  const [preview, setPreview] = useState<BulkPreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const percent = Number(text);
  const valid =
    text.trim() !== "" &&
    Number.isFinite(percent) &&
    percent !== 0 &&
    percent >= -50 &&
    percent <= 100;

  async function send(confirm: boolean) {
    if (!valid) {
      setError(B.badPercent);
      return;
    }
    setBusy(true);
    setError("");
    try {
      const r = await api<BulkPreview>(
        `/api/v1/merchant/stores/${storeId}/menu/prices`,
        {
          method: "POST",
          body: JSON.stringify({
            percent,
            ...(section ? { platform_section_id: section } : {}),
            confirm,
          }),
        },
      );
      if (confirm) {
        toast(fill(B.applied, { n: fmtNum(r.count) }), "success");
        onApplied();
      } else {
        setPreview(r);
      }
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const samples = preview?.samples ?? [];
  const rest = preview ? preview.count - samples.length : 0;

  return (
    <Modal open onClose={onClose} title={B.title}>
      <div className="flex flex-col gap-3" data-testid="bulk-price">
        {preview == null ? (
          <>
            <p className="text-xs text-ink-muted">{B.hint}</p>
            <Input
              label={B.percent}
              inputMode="decimal"
              dir="ltr"
              value={text}
              onChange={(e) => {
                setText(e.target.value.replace(/[^\d.-]/g, "").slice(0, 6));
                setError("");
              }}
              autoFocus
            />
            {sections.length > 0 && (
              <Select
                label={B.section}
                value={section}
                onChange={(e) => setSection(e.target.value)}
              >
                <option value="">{B.allSections}</option>
                {sections.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </Select>
            )}
            {error && <Alert>{error}</Alert>}
            <FormActions
              onSave={() => void send(false)}
              onCancel={onClose}
              busy={busy}
              saveLabel={B.preview}
            />
          </>
        ) : (
          <>
            <p className="font-bold">
              {preview.count > 0
                ? fill(B.count, { n: fmtNum(preview.count) })
                : B.none}
            </p>
            {samples.length > 0 && (
              <ul className="surface divide-y divide-line rounded-card px-3">
                {samples.map((s, i) => (
                  <li
                    key={`${s.name}-${i}`}
                    className="flex items-center justify-between gap-2 py-2 text-sm"
                  >
                    <span className="min-w-0 truncate">{s.name}</span>
                    <span className="shrink-0 tabular-nums">
                      <span className="text-ink-muted line-through">
                        {fmtMoney(s.old_price)}
                      </span>
                      {" ← "}
                      <span className="font-bold">{fmtMoney(s.new_price)}</span>
                    </span>
                  </li>
                ))}
              </ul>
            )}
            {rest > 0 && (
              <p className="text-xs text-ink-muted">
                {fill(B.more, { n: fmtNum(rest) })}
              </p>
            )}
            {error && <Alert>{error}</Alert>}
            {preview.count > 0 ? (
              <FormActions
                onSave={() => void send(true)}
                onCancel={() => setPreview(null)}
                busy={busy}
                saveLabel={B.apply}
                cancelLabel={B.back}
              />
            ) : (
              <Button variant="secondary" onClick={() => setPreview(null)}>
                {B.back}
              </Button>
            )}
          </>
        )}
      </div>
    </Modal>
  );
}

function ItemRow({
  item,
  disabled,
  onToggle,
  onPrice,
  onDuplicate,
}: {
  item: MenuItem;
  disabled: boolean;
  onToggle: () => void;
  onPrice: () => void;
  onDuplicate: () => void;
}) {
  return (
    <li
      className="flex items-center gap-3 py-2"
      data-testid="store-menu-item"
      data-item-id={item.id}
    >
      <MediaThumb
        url={item.image_thumb_url ?? item.image_url}
        alt={item.name}
        fallback={item.name}
        size={52}
        mediaUrl={mediaUrl}
      />
      <div className="min-w-0 flex-1">
        <Link
          href={`/store/menu/${item.id}`}
          aria-label={`${M.edit} · ${item.name}`}
          data-testid="store-menu-edit"
          className={`flex min-w-0 items-center gap-1 font-medium hover:underline ${item.available ? "text-ink" : "text-ink-muted"}`}
        >
          <span className="truncate">{item.name}</span>
          <IconEdit size={13} className="shrink-0 text-ink-muted" />
        </Link>
        <button
          type="button"
          disabled={disabled}
          onClick={onPrice}
          className="text-xs text-accent-text underline-offset-2 hover:underline disabled:no-underline"
        >
          {priceLine(item)}
        </button>
        {item.approved === false && (
          <p className="text-xs text-warning">
            {M.pendingReview}
            {item.review_note ? ` — ${item.review_note}` : ""}
          </p>
        )}
      </div>
      {/* **«نسخ»** — صنفٌ بالصورة والقسم والخيارات، ويُفتح للتعديل (٢٠٢٦-١٠-٢٧). */}
      <button
        type="button"
        disabled={disabled}
        onClick={onDuplicate}
        aria-label={`${M.duplicate} · ${item.name}`}
        title={M.duplicate}
        data-testid="store-menu-duplicate"
        className="shrink-0 rounded-control p-1.5 text-ink-muted hover:bg-row-hover hover:text-ink disabled:opacity-50"
      >
        <IconCopy size={16} />
      </button>
      <div className="shrink-0">
        <Switch
          checked={item.available}
          disabled={disabled}
          onChange={onToggle}
          label={item.available ? M.available : M.unavailable}
        />
      </div>
    </li>
  );
}

export default function MenuPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const toast = useShopToast();
  const load = useCallback(
    () => api<MenuSection[]>(`/api/v1/merchant/stores/${id}/menu`),
    [id],
  );
  const { data, loading, error, reload, setData } = useLiveData(
    load,
    ["menu"],
    [id],
  );
  const [q, setQ] = useState("");
  const [editing, setEditing] = useState<MenuItem | null>(null);
  const [bulkOpen, setBulkOpen] = useState(false);
  const [copying, setCopying] = useState(false);
  const router = useRouter();
  const suspended = store?.status === "suspended";

  // **أقسامُ المتجر بترتيبها كما يراها الزبون** (مرآةُ `MenuViewModel`، تدقيقُ ٢٠٢٦-١٠-٢٧).
  // **وتعذّرُها لا يُخفي شيئاً** — تبقى المجموعاتُ بعدد أصنافها كما كانت.
  const [order, setOrder] = useState<string[]>([]);
  useEffect(() => {
    if (!id) return;
    api<{ sections: SectionRef[] }>(`/api/v1/merchant/stores/${id}/sections`)
      .then((r) => setOrder((r.sections ?? []).map((x) => x.name)))
      .catch(() => undefined);
  }, [id]);

  const groups = useMemo(() => {
    const all = (data ?? []).flatMap((s) => s.items);
    const needle = q.trim();
    const hit = needle ? all.filter((i) => i.name.includes(needle)) : all;
    const by = new Map<string, MenuItem[]>();
    for (const it of hit) {
      const k = it.platform_section_name.trim() || M.unsorted;
      by.set(k, [...(by.get(k) ?? []), it]);
    }
    return [...by.entries()]
      .map(([name, items]) => ({
        name,
        items: items.sort((a, b) => a.name.localeCompare(b.name, "ar")),
      }))
      // **بترتيب أقسام المتجر** — وما ليس في أقسامه آخراً بعدد أصنافه.
      .sort((a, b) => {
        const ia = order.indexOf(a.name);
        const ib = order.indexOf(b.name);
        const ra = ia < 0 ? Number.MAX_SAFE_INTEGER : ia;
        const rb = ib < 0 ? Number.MAX_SAFE_INTEGER : ib;
        return ra - rb || b.items.length - a.items.length;
      });
  }, [data, q, order]);

  async function toggle(item: MenuItem) {
    const next = !item.available;
    const flip = (v: boolean) =>
      setData((d) =>
        d
          ? d.map((s) => ({
              ...s,
              items: s.items.map((i) =>
                i.id === item.id ? { ...i, available: v } : i,
              ),
            }))
          : d,
      );
    flip(next);
    try {
      await api(`/api/v1/merchant/menu/items/${item.id}/availability`, {
        method: "PATCH",
        body: JSON.stringify({ available: next }),
      });
    } catch (e) {
      flip(!next);
      toast(errorText(e), "error");
    }
  }

  /** **نسخُ الصنف** — ثمّ يُفتح الجديدُ في المحرّر. */
  async function duplicate(item: MenuItem) {
    if (copying) return;
    setCopying(true);
    try {
      const r = await api<{ id: string }>(
        `/api/v1/merchant/menu/items/${item.id}/duplicate`,
        { method: "POST" },
      );
      toast(M.duplicated, "success");
      router.push(`/store/menu/${r.id}`);
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setCopying(false);
    }
  }

  if (loading && !data) return <LoadingState variant="list" rows={5} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;

  const empty = (data ?? []).every((s) => s.items.length === 0);

  return (
    <div className="flex flex-col gap-3">
      <header className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <h2 className="heading-page">{M.title}</h2>
          <p className="text-sm text-ink-muted">{M.hint}</p>
        </div>
        {!suspended && (
          <ButtonLink
            href="/store/menu/new"
            className="flex shrink-0 items-center gap-1"
          >
            <IconAdd size={15} />
            {M.add}
          </ButtonLink>
        )}
      </header>
      {/* **مزامنةُ الأمين** — للمولات (٢٠٢٦-١٠-١٩). */}
      {!suspended && (
        <ButtonLink href="/store/price-sync" variant="secondary" className="w-fit">
          {m.store.priceSync.open}
        </ButtonLink>
      )}
      {/* **تغييرُ الأسعار بنسبة** — معاينةٌ ثمّ تأكيد (٢٠٢٦-١٠-٢٧). */}
      {!suspended && !empty && (
        <Button
          variant="secondary"
          className="w-fit"
          data-testid="store-menu-bulk-price"
          onClick={() => setBulkOpen(true)}
        >
          {M.bulkPrice}
        </Button>
      )}
      {/* **قائمةُ المشروبات — تُكتب مرّةً وتنضاف لكلّ الأصناف** (٢٠٢٦-١٠-١٩). */}
      {!suspended && id && (
        <DrinksCard
          storeId={id}
          itemCount={(data ?? []).reduce((n, s) => n + s.items.length, 0)}
        />
      )}
      {empty ? (
        <EmptyState icon={IconList} title={M.empty} />
      ) : (
        <>
          <Input
            type="search"
            aria-label={M.search}
            placeholder={M.search}
            icon={<IconSearch size={16} />}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          {groups.length === 0 && (
            <p className="py-6 text-center text-sm text-ink-muted">
              {M.noMatch}
            </p>
          )}
          {groups.map((g) => (
            <section key={g.name} className="surface rounded-card p-3">
              <h3 className="heading-card flex items-center justify-between gap-2">
                <span className="min-w-0 truncate">{g.name}</span>
                <span className="flex shrink-0 items-center gap-2 text-xs font-normal text-ink-muted">
                  {fill(M.count, { n: fmtNum(g.items.length) })}
                  {!suspended && g.items[0]?.platform_section_id && (
                    <Link
                      href={`/store/menu/new?section=${g.items[0].platform_section_id}`}
                      aria-label={`${M.add} · ${g.name}`}
                      className="rounded-control p-1 text-accent-text hover:bg-row-hover"
                    >
                      <IconAdd size={16} />
                    </Link>
                  )}
                </span>
              </h3>
              <ul className="divide-y divide-line">
                {g.items.map((it) => (
                  <ItemRow
                    key={it.id}
                    item={it}
                    disabled={suspended}
                    onToggle={() => void toggle(it)}
                    onPrice={() => setEditing(it)}
                    onDuplicate={() => void duplicate(it)}
                  />
                ))}
              </ul>
            </section>
          ))}
        </>
      )}
      {bulkOpen && id && (
        <BulkPriceDialog
          storeId={id}
          sections={sectionsOf(data ?? [])}
          onClose={() => setBulkOpen(false)}
          onApplied={() => {
            setBulkOpen(false);
            void reload();
          }}
        />
      )}
      {editing && (
        <PriceDialog
          item={editing}
          rate={store?.exchange_rate ?? null}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void reload();
          }}
        />
      )}
    </div>
  );
}
