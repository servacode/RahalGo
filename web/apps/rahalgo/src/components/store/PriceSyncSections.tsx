"use client";

/**
 * **توزيع الأقسام — أصناف الأمين على أقسام السوق** (قرار المالك ٢٠٢٦-١١-٠٧)
 *
 * «كل صنف يروح على قسمو». الأصناف الجديدة بتروح لقسمها من اسمها وقت الاستيراد،
 * واللي ما انعرف قسمه بيضل بقسم الانتظار. هون:
 *  - «وزّع الأصناف الباقية تلقائياً» — الفعل الأساسي (`POST …/price-sync/auto-sort`).
 *  - مراجعة الباقي بالإيد: صفحة صفحة مع بحث، قسم لكل صنف، وتحديد ونقل دفعة وحدة
 *    (`GET …/pending` و`POST …/move`) — والنقل بالإيد ما بترجّعه المزامنة.
 *  - مجموعات الأمين إذا الملف فيه عمود مجموعة (`GET/PUT …/groups`).
 *  - «حطّ الباقي بقسم منوّعات» (`POST …/move-rest`) بعد تأكيد جوّا الصفحة — كل اللي ضل
 *    بالانتظار بيصير يدويّ بقسم «منوّعات».
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Button, Card, Checkbox, Input, Select } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { fill } from "@/lib/customer/texts";
import type { SectionRef } from "@/lib/store/types";

const m = getMessages(defaultLocale);
const P = m.store.priceSync;
const PAGE = 50;

interface Group {
  group_key: string;
  group_name: string;
  items: number;
  waiting: number;
  platform_section_id: string | null;
  suggested_section_id: string | null;
  suggested_section_name: string | null;
}
interface PendingItem {
  id: string;
  name: string;
  external_code: string | null;
  external_group: string | null;
  price: number;
  suggested_section_id: string | null;
  suggested_section_name: string | null;
}
interface PendingPage {
  items: PendingItem[];
  total: number;
  next: string;
}
interface SortResult {
  by_group: number;
  by_name: number;
  waiting: number;
}

export function PriceSyncSections({ storeId, refreshKey }: { storeId: string; refreshKey: number }) {
  const base = `/api/v1/merchant/stores/${storeId}/price-sync`;
  const [sections, setSections] = useState<SectionRef[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [choice, setChoice] = useState<Record<string, string>>({});
  const [pending, setPending] = useState(0);
  const [items, setItems] = useState<PendingItem[]>([]);
  const [total, setTotal] = useState(0);
  const [next, setNext] = useState("");
  const [q, setQ] = useState("");
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [rowSec, setRowSec] = useState<Record<string, string>>({});
  const [bulkSec, setBulkSec] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [askRest, setAskRest] = useState(false);

  const loadGroups = useCallback(() => {
    api<{ groups: Group[]; pending: number }>(`${base}/groups`)
      .then((r) => {
        setGroups(r.groups);
        setPending(r.pending);
        const c: Record<string, string> = {};
        for (const g of r.groups) {
          c[g.group_key] = g.platform_section_id ?? g.suggested_section_id ?? "";
        }
        setChoice(c);
      })
      .catch((e) => setError(errorText(e)));
  }, [base]);

  const loadItems = useCallback(
    (after: string, query: string) => {
      const p = new URLSearchParams({ limit: String(PAGE) });
      if (after) p.set("after", after);
      if (query.trim()) p.set("q", query.trim());
      api<PendingPage>(`${base}/pending?${p.toString()}`)
        .then((r) => {
          setItems((old) => (after ? [...old, ...r.items] : r.items));
          setTotal(r.total);
          setNext(r.next);
          setRowSec((old) => {
            const c = after ? { ...old } : {};
            for (const it of r.items) c[it.id] = it.suggested_section_id ?? "";
            return c;
          });
          if (!after) setPicked(new Set());
        })
        .catch((e) => setError(errorText(e)));
    },
    [base],
  );

  useEffect(() => {
    api<{ sections: SectionRef[] }>(`/api/v1/merchant/platform-sections`)
      .then((r) => setSections(r.sections))
      .catch((e) => setError(errorText(e)));
  }, []);
  useEffect(() => {
    loadGroups();
  }, [loadGroups, refreshKey]);
  /** **البحث بعد ما يوقف عن الكتابة** — مو طلب لكل حرف. */
  useEffect(() => {
    const t = setTimeout(() => loadItems("", q), q ? 300 : 0);
    return () => clearTimeout(t);
  }, [loadItems, refreshKey, q]);

  /** **بعد كل تغيير** — العدد والقائمة من جديد. */
  function reload() {
    loadGroups();
    loadItems("", q);
  }

  async function act(fn: () => Promise<string>) {
    setBusy(true);
    setError("");
    setNote("");
    try {
      setNote(await fn());
      reload();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const autoSort = () =>
    act(async () => {
      const r = await api<SortResult>(`${base}/auto-sort`, { method: "POST" });
      return fill(P.sortDone, {
        byName: fmtNum(r.by_name),
        byGroup: fmtNum(r.by_group),
        waiting: fmtNum(r.waiting),
      });
    });

  const moveRest = () =>
    act(async () => {
      setAskRest(false);
      const r = await api<{ moved: number }>(`${base}/move-rest`, { method: "POST" });
      return fill(P.moveRestDone, { n: fmtNum(r.moved) });
    });

  const saveGroups = () =>
    act(async () => {
      const body = groups.map((g) => ({
        group_key: g.group_key,
        platform_section_id: choice[g.group_key] ?? "",
      }));
      const r = await api<{ moved: number }>(`${base}/groups`, {
        method: "PUT",
        body: JSON.stringify(body),
      });
      return fill(P.groupSaved, { n: fmtNum(r.moved) });
    });

  const move = (ids: string[], section: string) =>
    act(async () => {
      const r = await api<{ moved: number }>(`${base}/move`, {
        method: "POST",
        body: JSON.stringify({ item_ids: ids, platform_section_id: section }),
      });
      return fill(P.moved, { n: fmtNum(r.moved) });
    });

  /** **قائمة الأقسام** — والمقترح المطفي بينضاف لحتى يبيّن. */
  const options = (extraId?: string | null, extraName?: string | null) => (
    <>
      <option value="">{P.pickSection}</option>
      {extraId && !sections.some((s) => s.id === extraId) && (
        <option value={extraId}>{extraName ?? ""}</option>
      )}
      {sections.map((s) => (
        <option key={s.id} value={s.id}>
          {s.name}
        </option>
      ))}
    </>
  );

  const allPicked = items.length > 0 && items.every((it) => picked.has(it.id));

  return (
    <Card>
      <div className="flex flex-col gap-3 p-4" data-testid="sync-sections">
        <h3 className="heading-card">{P.sortTitle}</h3>
        <p className="text-sm font-bold">
          {pending > 0 ? fill(P.sortWaiting, { n: fmtNum(pending) }) : P.sortNone}
        </p>
        {pending > 0 && (
          <>
            <p className="text-xs text-ink-muted">{P.sortAutoHint}</p>
            <Button className="w-fit" onClick={() => void autoSort()} disabled={busy}>
              {P.sortAuto}
            </Button>
            {askRest ? (
              <div className="flex flex-col gap-2 rounded-card border border-line-soft p-3" role="group">
                <p className="text-sm">{fill(P.moveRestAsk, { n: fmtNum(pending) })}</p>
                <div className="flex flex-wrap gap-2">
                  <Button className="w-fit" onClick={() => void moveRest()} disabled={busy}>
                    {P.moveRestYes}
                  </Button>
                  <Button
                    className="w-fit"
                    variant="secondary"
                    onClick={() => setAskRest(false)}
                    disabled={busy}
                  >
                    {P.moveRestNo}
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                className="w-fit"
                variant="secondary"
                onClick={() => setAskRest(true)}
                disabled={busy}
              >
                {P.moveRest}
              </Button>
            )}
          </>
        )}
        {error && <Alert>{error}</Alert>}
        {note && <Alert tone="success">{note}</Alert>}

        {(pending > 0 || q) && (
          <div className="flex flex-col gap-2 border-t border-line-soft pt-3">
            <h4 className="font-bold">{P.reviewTitle}</h4>
            <Input
              id="sync-pending-search"
              type="search"
              placeholder={P.search}
              aria-label={P.search}
              value={q}
              onChange={(e) => setQ(e.target.value)}
            />
            <div className="flex flex-wrap items-end gap-2">
              <Checkbox
                id="sync-pick-all"
                checked={allPicked}
                onChange={(e) =>
                  setPicked(e.target.checked ? new Set(items.map((it) => it.id)) : new Set())
                }
                label={P.selectAll}
              />
              <span className="text-xs text-ink-muted">
                {fill(P.selected, { n: fmtNum(picked.size) })}
              </span>
              <Select
                id="sync-bulk-section"
                label={P.moveTo}
                value={bulkSec}
                onChange={(e) => setBulkSec(e.target.value)}
              >
                {options()}
              </Select>
              <Button
                onClick={() => void move([...picked], bulkSec)}
                disabled={busy || picked.size === 0 || !bulkSec}
              >
                {P.move}
              </Button>
            </div>
            {items.length === 0 ? (
              <p className="text-sm text-ink-muted">{P.reviewEmpty}</p>
            ) : (
              <ul className="flex flex-col divide-y divide-line-soft">
                {items.map((it) => (
                  <li key={it.id} className="flex flex-wrap items-center gap-2 py-2">
                    <Checkbox
                      id={`sync-pick-${it.id}`}
                      checked={picked.has(it.id)}
                      onChange={(e) =>
                        setPicked((old) => {
                          const n = new Set(old);
                          if (e.target.checked) n.add(it.id);
                          else n.delete(it.id);
                          return n;
                        })
                      }
                      label={
                        <span className="flex flex-col">
                          <span>{it.name}</span>
                          <span className="text-xs text-ink-muted">
                            {[it.external_code, it.external_group].filter(Boolean).join(" · ")}
                          </span>
                        </span>
                      }
                    />
                    <div className="ms-auto flex items-end gap-2">
                      <Select
                        id={`sync-sec-${it.id}`}
                        aria-label={P.itemSection}
                        value={rowSec[it.id] ?? ""}
                        onChange={(e) => setRowSec({ ...rowSec, [it.id]: e.target.value })}
                      >
                        {options(it.suggested_section_id, it.suggested_section_name)}
                      </Select>
                      <Button
                        variant="ghost"
                        onClick={() => void move([it.id], rowSec[it.id] ?? "")}
                        disabled={busy || !rowSec[it.id]}
                      >
                        {P.move}
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
            {next && (
              <Button variant="ghost" className="w-fit" onClick={() => loadItems(next, q)} disabled={busy}>
                {P.more} ({fmtNum(items.length)} / {fmtNum(total)})
              </Button>
            )}
          </div>
        )}

        {groups.length > 0 && (
          <div className="flex flex-col gap-2 border-t border-line-soft pt-3">
            <h4 className="font-bold">{P.groupsTitle}</h4>
            <p className="text-xs text-ink-muted">{P.groupsHint}</p>
            <ul className="flex flex-col divide-y divide-line-soft">
              {groups.map((g) => (
                <li key={g.group_key} className="flex flex-wrap items-center gap-2 py-2">
                  <span className="flex flex-col">
                    <span className="font-bold">{g.group_name}</span>
                    <span className="text-xs text-ink-muted">
                      {P.groupCount}: {fmtNum(g.items)}
                    </span>
                  </span>
                  <div className="ms-auto">
                    <Select
                      id={`sync-group-${g.group_key}`}
                      aria-label={P.groupSection}
                      value={choice[g.group_key] ?? ""}
                      onChange={(e) => setChoice({ ...choice, [g.group_key]: e.target.value })}
                    >
                      {options(g.suggested_section_id, g.suggested_section_name)}
                    </Select>
                  </div>
                </li>
              ))}
            </ul>
            <Button className="w-fit" onClick={() => void saveGroups()} disabled={busy}>
              {P.groupSave}
            </Button>
          </div>
        )}
      </div>
    </Card>
  );
}
