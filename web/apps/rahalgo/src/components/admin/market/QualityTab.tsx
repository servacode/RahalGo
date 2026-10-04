"use client";

/**
 * **تبويبُ جودة البيانات** — أصنافٌ بلا صورة · متاجرُ بلا دوام · البياناتُ
 * التجريبيّةُ وحذفُها. (قرارُ المالك ٢٠٢٦-١٠-٠٤.)
 *
 * **ولا حذفَ آليّاً أبداً**: يختار الموظّفُ ما يُحذف، **ثمّ تأكيدٌ يسرد كلَّ
 * ما سيُحذف بالاسم**، والمحرّكُ يعيد فحصَ كلِّ معرّفٍ قبل الحذف.
 * **ومتجرٌ عليه طلباتٌ لا يُحذف من هنا** — الطلبُ أثرُ مالٍ في الدفتر.
 */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  Confirm,
  LoadingState,
  useLiveData,
  useToast,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const Q = m.admin.market.quality;

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

  return (
    <div className="space-y-4">
      {/* ── أصنافٌ بلا صورة ── */}
      <Card title={`${Q.noImageTitle} (${fmtNum(data.items_without_image_count)})`}>
        <p className="mb-2 text-xs text-ink-muted">{Q.noImageHint}</p>
        {data.items_without_image.length === 0 ? (
          <p className="text-sm text-ink-muted">{Q.clean}</p>
        ) : (
          <ul className="divide-y divide-line-soft text-sm">
            {data.items_without_image.map((it) => (
              <li key={it.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                <span className="min-w-0 truncate">
                  <b>{it.name}</b>
                  <span className="text-ink-muted">
                    {" · "}
                    {it.merchant_name}
                    {" · "}
                    {it.section_name}
                    {" · "}
                    {fmtDateTime(it.created_at)}
                  </span>
                </span>
                <Link
                  className="text-xs text-primary hover:underline"
                  href={`/dashboard/merchants/${it.merchant_id}`}
                >
                  {Q.openStore}
                </Link>
              </li>
            ))}
          </ul>
        )}
        <p className="mt-2 text-xs text-ink-muted">
          {showing(data.items_without_image.length, data.items_without_image_count)}
        </p>
      </Card>

      {/* ── متاجرُ بلا دوام ── */}
      <Card title={`${Q.noHoursTitle} (${fmtNum(data.stores_without_hours.length)})`}>
        <p className="mb-2 text-xs text-ink-muted">{Q.noHoursHint}</p>
        {data.stores_without_hours.length === 0 ? (
          <p className="text-sm text-ink-muted">{Q.clean}</p>
        ) : (
          <ul className="divide-y divide-line-soft text-sm">
            {data.stores_without_hours.map((s) => (
              <li key={s.id} className="flex items-center justify-between gap-2 py-2">
                <span className="truncate font-bold">{s.name}</span>
                <Link
                  className="text-xs text-primary hover:underline"
                  href={`/dashboard/merchants/${s.id}`}
                >
                  {Q.openStore}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Card>

      {/* ── البياناتُ التجريبيّة ── */}
      <Card title={Q.testTitle}>
        <p className="mb-3 text-xs text-ink-muted">{Q.testHint}</p>

        <p className="mb-1 text-sm font-bold">{Q.testStores}</p>
        {data.test_stores.length === 0 ? (
          <p className="mb-3 text-sm text-ink-muted">{Q.clean}</p>
        ) : (
          <ul className="mb-3 divide-y divide-line-soft text-sm">
            {data.test_stores.map((s) => (
              <li key={s.id} className="flex flex-wrap items-center gap-2 py-2">
                <Checkbox
                  id={`tq-s-${s.id}`}
                  label={s.name}
                  disabled={s.orders > 0}
                  checked={stores.has(s.id)}
                  onChange={() => setStores(flip(stores, s.id))}
                />
                <span className="text-xs text-ink-muted">
                  {Q.storeLine.replace("{n}", fmtNum(s.items)).replace("{o}", fmtNum(s.orders))}
                </span>
                {s.orders > 0 && <Badge variant="warning">{Q.hasOrders}</Badge>}
              </li>
            ))}
          </ul>
        )}

        <p className="mb-1 text-sm font-bold">{Q.testItems}</p>
        {data.test_items.length === 0 ? (
          <p className="text-sm text-ink-muted">{Q.clean}</p>
        ) : (
          <ul className="divide-y divide-line-soft text-sm">
            {data.test_items.map((it) => (
              <li key={it.id} className="flex flex-wrap items-center gap-2 py-2">
                <Checkbox
                  id={`tq-i-${it.id}`}
                  label={it.name}
                  checked={items.has(it.id)}
                  onChange={() => setItems(flip(items, it.id))}
                />
                <span className="text-xs text-ink-muted">{it.merchant_name}</span>
              </li>
            ))}
          </ul>
        )}
        <p className="mt-2 text-xs text-ink-muted">
          {showing(data.test_items.length, data.test_items_count)}
        </p>

        <div className="mt-3 flex justify-end">
          <Button
            variant="danger"
            disabled={pickedStores.length + pickedItems.length === 0}
            onClick={() => setConfirming(true)}
          >
            {Q.deleteSelected}
          </Button>
        </div>
      </Card>

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
