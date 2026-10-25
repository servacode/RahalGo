"use client";

/**
 * **تبويبُ جودة البيانات** — أصنافٌ بلا صورة · متاجرُ بلا دوام · البياناتُ
 * التجريبيّةُ وحذفُها. (قرارُ المالك ٢٠٢٦-١٠-٠٤، وشكلُه ٢٠٢٦-١٠-٢٦: «تصميم احترافي».)
 *
 * **ولا حذفَ آليّاً أبداً**: يختار الموظّفُ ما يُحذف، **ثمّ تأكيدٌ يسرد كلَّ
 * ما سيُحذف بالاسم**، والمحرّكُ يعيد فحصَ كلِّ معرّفٍ قبل الحذف.
 * **ومتجرٌ عليه طلباتٌ لا يُحذف من هنا** — الطلبُ أثرُ مالٍ في الدفتر.
 */

import { useState, type ComponentType, type ReactNode } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Button,
  Checkbox,
  Confirm,
  LoadingState,
  useLiveData,
  useToast,
  IconCamera,
  IconHourglass,
  IconWarning,
  IconCheck,
  IconStore,
  IconOrder,
  IconDelete,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const Q = m.admin.market.quality;

type IconT = ComponentType<{ size?: number; className?: string }>;

interface QItem {
  id: string;
  name: string;
  merchant_id: string;
  merchant_name: string;
  section_name: string;
  created_at: string;
}

interface QStore {
  id: string;
  name: string;
  status: string;
  items: number;
  orders: number;
}

interface Quality {
  items_without_image: QItem[];
  items_without_image_count: number;
  stores_without_hours: QStore[];
  test_stores: QStore[];
  test_items: QItem[];
  test_items_count: number;
  limit: number;
}

const openBtn =
  "inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-ink-muted hover:bg-row-hover hover:text-ink";
const row = "flex items-center gap-3 rounded-card border border-line-soft p-2.5";

export default function QualityTab() {
  const { data, reload } = useLiveData<Quality>(
    () => api("/api/v1/admin/market/quality"),
    ["menu", "catalog"],
  );
  const [stores, setStores] = useState<Set<string>>(new Set());
  const [items, setItems] = useState<Set<string>>(new Set());
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const { push, Toaster } = useToast();

  if (!data) return <LoadingState />;

  const flip = (set: Set<string>, id: string) => {
    const n = new Set(set);
    if (n.has(id)) n.delete(id);
    else n.add(id);
    return n;
  };
  const pickedStores = data.test_stores.filter((s) => stores.has(s.id));
  const pickedItems = data.test_items.filter((i) => items.has(i.id));
  const picked = pickedStores.length + pickedItems.length;

  async function remove() {
    setBusy(true);
    try {
      const res = await api<{ items_deleted: number; stores_deleted: number }>(
        "/api/v1/admin/market/test-data/delete",
        {
          method: "POST",
          body: JSON.stringify({
            store_ids: pickedStores.map((s) => s.id),
            item_ids: pickedItems.map((i) => i.id),
          }),
        },
      );
      push(
        Q.deleted
          .replace("{i}", fmtNum(res.items_deleted))
          .replace("{s}", fmtNum(res.stores_deleted)),
      );
      setStores(new Set());
      setItems(new Set());
      setConfirming(false);
      reload();
    } catch (err) {
      push(errorText(err), "error");
    } finally {
      setBusy(false);
    }
  }

  const showing = (n: number, all: number) =>
    all > n ? Q.showing.replace("{n}", fmtNum(n)).replace("{all}", fmtNum(all)) : "";

  const testCount = data.test_stores.length + data.test_items_count;
  const tiles: { label: string; n: number; Icon: IconT; href: string }[] = [
    { label: Q.noImageTitle, n: data.items_without_image_count, Icon: IconCamera, href: "#q-img" },
    { label: Q.noHoursTitle, n: data.stores_without_hours.length, Icon: IconHourglass, href: "#q-hours" },
    { label: Q.testTitle, n: testCount, Icon: IconWarning, href: "#q-test" },
  ];
  const imgMore = showing(data.items_without_image.length, data.items_without_image_count);
  const testMore = showing(data.test_items.length, data.test_items_count);

  return (
    <div className="space-y-4">
      {/* **ملخّصٌ بثلاث بطاقات** — الأخضرُ نظيف، والبرتقاليُّ فيه شغل، والضغطُ ينزل لقسمه. */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {tiles.map(({ label, n, Icon, href }) => (
          <a key={href} href={href} className="surface flex items-center gap-3 p-4 transition-shadow hover:elev-2">
            <span
              className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-on-solid ${
                n > 0 ? "bg-warning-solid" : "bg-success-solid"
              }`}
            >
              {n > 0 ? <Icon size={20} /> : <IconCheck size={20} />}
            </span>
            <span className="min-w-0">
              <span className="figure block leading-none">{fmtNum(n)}</span>
              <span className="block truncate text-sm text-ink-muted">{label}</span>
            </span>
          </a>
        ))}
      </div>

      {/* ── أصنافٌ بلا صورة ── */}
      <Block id="q-img" Icon={IconCamera} title={Q.noImageTitle} n={data.items_without_image_count} hint={Q.noImageHint}>
        {data.items_without_image.length === 0 ? (
          <Clean />
        ) : (
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-3">
            {data.items_without_image.map((it) => (
              <div key={it.id} className={row}>
                <span className="inline-flex h-12 w-12 shrink-0 items-center justify-center rounded-control bg-field text-ink-muted">
                  <IconCamera size={18} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-bold">{it.name}</p>
                  <p className="truncate text-xs text-ink-muted">
                    {it.merchant_name} · {it.section_name}
                  </p>
                  <p className="text-2xs text-ink-muted">{fmtDateTime(it.created_at)}</p>
                </div>
                <Link
                  href={`/dashboard/merchants/${it.merchant_id}`}
                  title={Q.openStore}
                  aria-label={Q.openStore}
                  className={openBtn}
                >
                  <IconStore size={17} />
                </Link>
              </div>
            ))}
          </div>
        )}
        {imgMore && <p className="text-xs text-ink-muted">{imgMore}</p>}
      </Block>

      {/* ── متاجرُ بلا دوام ── */}
      <Block
        id="q-hours"
        Icon={IconHourglass}
        title={Q.noHoursTitle}
        n={data.stores_without_hours.length}
        hint={Q.noHoursHint}
      >
        {data.stores_without_hours.length === 0 ? (
          <Clean />
        ) : (
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-3">
            {data.stores_without_hours.map((s) => (
              <div key={s.id} className={row}>
                <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-field text-ink-muted">
                  <IconStore size={17} />
                </span>
                <span className="min-w-0 flex-1 truncate text-sm font-bold">{s.name}</span>
                <Link href={`/dashboard/merchants/${s.id}`} title={Q.openStore} aria-label={Q.openStore} className={openBtn}>
                  <IconHourglass size={17} />
                </Link>
              </div>
            ))}
          </div>
        )}
      </Block>

      {/* ── البياناتُ التجريبيّة ── */}
      <Block
        id="q-test"
        Icon={IconWarning}
        title={Q.testTitle}
        n={testCount}
        hint={Q.testHint}
        action={
          <Button
            variant="danger"
            disabled={picked === 0}
            onClick={() => setConfirming(true)}
            className="inline-flex items-center gap-1.5"
          >
            <IconDelete size={15} />
            {Q.deleteSelected}
            {picked > 0 && ` (${fmtNum(picked)})`}
          </Button>
        }
      >
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div>
            <p className="mb-2 flex items-center gap-1.5 text-sm font-bold">
              <IconStore size={15} className="text-ink-muted" />
              {Q.testStores}
            </p>
            {data.test_stores.length === 0 ? (
              <Clean />
            ) : (
              <div className="space-y-2">
                {data.test_stores.map((s) => (
                  <div key={s.id} className={`${row} flex-wrap`}>
                    <Checkbox
                      id={`tq-s-${s.id}`}
                      label={<b>{s.name}</b>}
                      disabled={s.orders > 0}
                      checked={stores.has(s.id)}
                      onChange={() => setStores(flip(stores, s.id))}
                    />
                    <span className="text-xs text-ink-muted">
                      {Q.storeLine.replace("{n}", fmtNum(s.items)).replace("{o}", fmtNum(s.orders))}
                    </span>
                    {s.orders > 0 && (
                      <span className="ms-auto rounded-full bg-warning-solid px-2 py-0.5 text-2xs font-bold text-on-solid">
                        {Q.hasOrders}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
          <div>
            <p className="mb-2 flex items-center gap-1.5 text-sm font-bold">
              <IconOrder size={15} className="text-ink-muted" />
              {Q.testItems}
            </p>
            {data.test_items.length === 0 ? (
              <Clean />
            ) : (
              <div className="space-y-2">
                {data.test_items.map((it) => (
                  <div key={it.id} className={`${row} flex-wrap`}>
                    <Checkbox
                      id={`tq-i-${it.id}`}
                      label={<b>{it.name}</b>}
                      checked={items.has(it.id)}
                      onChange={() => setItems(flip(items, it.id))}
                    />
                    <span className="text-xs text-ink-muted">{it.merchant_name}</span>
                  </div>
                ))}
              </div>
            )}
            {testMore && <p className="mt-2 text-xs text-ink-muted">{testMore}</p>}
          </div>
        </div>
      </Block>

      {/* **التأكيدُ يسرد ما سيُحذف بالاسم** — وعمومُ الكلام يجعل الضغطَ عادة. */}
      <Confirm
        open={confirming}
        title={Q.confirmTitle}
        body={
          <div className="space-y-2 text-sm">
            <p>{Q.confirmBody}</p>
            {pickedStores.length > 0 && (
              <div>
                <p className="font-bold">
                  {Q.confirmStores.replace("{n}", fmtNum(pickedStores.length))}
                </p>
                <ul className="list-disc ps-5">
                  {pickedStores.map((s) => (
                    <li key={s.id}>
                      {s.name} — {Q.storeLine.replace("{n}", fmtNum(s.items)).replace("{o}", fmtNum(s.orders))}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {pickedItems.length > 0 && (
              <div>
                <p className="font-bold">
                  {Q.confirmItems.replace("{n}", fmtNum(pickedItems.length))}
                </p>
                <ul className="list-disc ps-5">
                  {pickedItems.map((i) => (
                    <li key={i.id}>
                      {i.name} — {i.merchant_name}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        }
        confirmLabel={Q.confirm}
        onConfirm={() => void remove()}
        onCancel={() => setConfirming(false)}
        busy={busy}
      />
      <Toaster />
    </div>
  );
}

/** **قسمٌ من الجودة** — رأسٌ بأيقونةٍ ملوّنةٍ وعدّ، وشرحٌ قصير، ثمّ المحتوى. */
function Block({
  id,
  Icon,
  title,
  n,
  hint,
  action,
  children,
}: {
  id: string;
  Icon: IconT;
  title: string;
  n: number;
  hint: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section id={id} className="surface scroll-mt-4 space-y-3 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <span
            className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-on-solid ${
              n > 0 ? "bg-warning-solid" : "bg-success-solid"
            }`}
          >
            <Icon size={18} />
          </span>
          <div className="min-w-0">
            <h3 className="heading-card flex items-center gap-2">
              {title}
              <span className="rounded-full bg-field px-2 py-0.5 text-xs tabular-nums text-ink-muted">{fmtNum(n)}</span>
            </h3>
            <p className="text-xs text-ink-muted">{hint}</p>
          </div>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

function Clean() {
  return (
    <p className="flex items-center gap-1.5 text-sm font-bold text-success">
      <IconCheck size={16} />
      {Q.clean}
    </p>
  );
}
