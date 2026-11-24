"use client";

/**
 * **جوائز رحّال غو** — سحبٌ شهريٌّ على الدعوات (قرارُ المالك ٢٠٢٦-١١-٢٣).
 *
 * كلُّ N دعواتٍ بالشهر = فرصةٌ؛ والسحبُ تلقائيٌّ أوّلَ الشهر ١٢ ظهراً بتوقيت دمشق على الشهر السابق.
 * **والإعداداتُ كلُّها من صفحة الإعدادات** (`prizes.*` من فهرس المحرّك) — هنا قراءةٌ ورابطٌ لا غير.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDate, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  AdminEmpty,
  AdminLoading,
  AdminReload,
  Badge,
  Button,
  ButtonLink,
  Chips,
  PageContainer,
  Select,
  IconCatGifts,
  IconCheck,
  IconDate,
  IconSettings,
  IconStatus,
  IconWallet,
  phoneText,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const P = m.admin.prizes;

interface Entrant {
  user_id: string;
  name: string;
  phone: string;
  invites: number;
  entries: number;
}
interface Entries {
  month: string;
  invites_per_entry: number;
  entrants: Entrant[];
}
interface Winner {
  rank: number;
  user_id: string;
  name: string;
  phone: string;
  amount: number;
  spent: number;
  expired: number;
}
interface Draw {
  month: string;
  drawn_at: string;
  entrants: number;
  total_entries: number;
  awarded: number;
  spent: number;
  expired: number;
  winners: Winner[];
}
interface PrizeSettings {
  enabled: boolean;
  launchOn: string;
}

interface PrizeStore {
  id: string;
  name: string;
  items: number;
  available: number;
}

/**
 * **متجرُ «رحّال غو» للجوائز** (قرارُ المالك ٢٠٢٦-١١-٢٤) — يُختار من هون متجرٌ عاديٌّ فيصير متجرَ
 * الجوائز: **يختفي من السوق وتطلع أصنافُه بصفحة الجوائز وحدَها.** والأصنافُ من صفحة أصناف المتجر نفسِها.
 */
function PrizeStorePanel() {
  const canCall = useCanCall();
  const canSet = canCall("PUT", "/prizes/store");
  const canMenu = canCall("GET", "/merchants/{id}/menu");
  const [store, setStore] = useState<PrizeStore | null | undefined>(undefined);
  const [err, setErr] = useState("");
  const [list, setList] = useState<{ id: string; name: string; status: string }[]>([]);
  const [pick, setPick] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api<{ store: PrizeStore | null }>("/api/v1/admin/prizes/store")
      .then((r) => {
        setStore(r.store);
        setErr("");
      })
      .catch((e) => setErr(errorText(e)));
  }, []);
  useEffect(load, [load]);
  useEffect(() => {
    if (!canSet || store !== null) return;
    api<{ merchants: { id: string; name: string; status: string }[] }>("/api/v1/admin/merchants?per_page=100")
      .then((r) => setList((r.merchants ?? []).filter((x) => x.status === "active" || x.status === "inactive")))
      .catch((e) => setErr(errorText(e)));
  }, [canSet, store]);

  const save = (merchantId: string) => {
    setBusy(true);
    api<{ store: PrizeStore | null }>("/api/v1/admin/prizes/store", {
      method: "PUT",
      body: JSON.stringify({ merchant_id: merchantId }),
    })
      .then((r) => {
        setStore(r.store);
        setPick("");
        setErr("");
      })
      .catch((e) => setErr(errorText(e)))
      .finally(() => setBusy(false));
  };

  return (
    <Panel title={P.storeTitle}>
      {err && <p className="mb-2 text-sm text-danger">{err}</p>}
      {store === undefined ? (
        err ? <AdminReload label={err} onRetry={load} /> : <AdminLoading />
      ) : store ? (
        <div className="flex flex-col gap-3">
          <p className="text-sm">
            <span className="font-bold">{store.name}</span>
            <span className="text-ink-muted">
              {" "}
              · {P.storeItems.replace("{n}", fmtNum(store.items)).replace("{a}", fmtNum(store.available))}
            </span>
          </p>
          <p className="text-xs text-ink-muted">{P.storeHidden}</p>
          <div className="flex flex-wrap gap-2">
            {canMenu && (
              <ButtonLink href={`/dashboard/merchants/${store.id}/menu`}>
                <IconCatGifts size={14} />
                {P.storeOpenItems}
              </ButtonLink>
            )}
            {canSet && (
              <Button variant="secondary" disabled={busy} onClick={() => save("")}>
                {P.storeUnset}
              </Button>
            )}
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-sm text-ink-muted">{P.storeNone}</p>
          {canSet && (
            <div className="flex flex-wrap items-end gap-2">
              <div className="min-w-0 flex-1">
                <Select value={pick} onChange={(e) => setPick(e.target.value)} aria-label={P.storePick}>
                  <option value="">{P.storePick}</option>
                  {list.map((x) => (
                    <option key={x.id} value={x.id}>
                      {x.name}
                    </option>
                  ))}
                </Select>
              </div>
              <Button disabled={!pick || busy} onClick={() => save(pick)}>
                {P.storeSet}
              </Button>
            </div>
          )}
        </div>
      )}
    </Panel>
  );
}

/** «YYYY-MM» لشهرٍ ما. */
function monthKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}
/** الشهرُ الحاليّ والأشهرُ الماضية. */
function recentMonths(n: number): string[] {
  const now = new Date();
  return Array.from({ length: n }, (_, i) => monthKey(new Date(now.getFullYear(), now.getMonth() - i, 1)));
}
function monthLabel(key: string): string {
  const [y = "", mo = "1"] = key.split("-");
  return `${P.months[Number(mo) - 1] ?? mo} ${fmtNum(Number(y))}`;
}

/** السحبُ القادم: أوّلُ الشهر الذي يلي (اليومَ أو تاريخَ البدء، الأبعد). */
function nextDraw(launchOn: string): string {
  const launch = launchOn ? new Date(`${launchOn}T00:00:00`) : new Date(0);
  const ref = launch.getTime() > Date.now() ? launch : new Date();
  return `${monthKey(new Date(ref.getFullYear(), ref.getMonth() + 1, 1))}-01`;
}

function Person({ id, name, phone, canUser }: { id: string; name: string; phone: string; canUser: boolean }) {
  return (
    <>
      <OpenLink
        allowed={canUser}
        href={`/dashboard/users/${id}`}
        className={`font-bold ${canUser ? "text-primary hover:underline" : ""}`}
      >
        {name || "—"}
      </OpenLink>
      <span className="block text-xs text-ink-muted" dir="ltr">
        {phoneText(phone)}
      </span>
    </>
  );
}

export default function PrizesPage() {
  const open = useCanOpen();
  const months = useMemo(() => recentMonths(6), []);
  const [month, setMonth] = useState(months[0] ?? "");
  const [entries, setEntries] = useState<Entries | null>(null);
  const [entriesErr, setEntriesErr] = useState("");
  const [draws, setDraws] = useState<Draw[] | null>(null);
  const [drawsErr, setDrawsErr] = useState("");
  const [cfg, setCfg] = useState<PrizeSettings | null>(null);

  const loadEntries = useCallback(() => {
    setEntries(null);
    api<Entries>(`/api/v1/admin/prizes/entries?month=${month}`)
      .then((r) => {
        setEntries(r);
        setEntriesErr("");
      })
      .catch((e) => setEntriesErr(errorText(e)))
  }, [month]);
  useEffect(loadEntries, [loadEntries]);

  const loadDraws = useCallback(() => {
    api<{ draws: Draw[] }>("/api/v1/admin/prizes/draws")
      .then((r) => {
        setDraws(r.draws ?? []);
        setDrawsErr("");
      })
      .catch((e) => setDrawsErr(errorText(e)));
  }, []);
  useEffect(loadDraws, [loadDraws]);

  // الحالةُ من فهرس الإعدادات — وإن لم تُقرأ (بلا صلاحية) يُترك الرأسُ بلا حالة.
  useEffect(() => {
    api<{ settings: { key: string; value: unknown }[] }>("/api/v1/admin/settings")
      .then((r) => {
        const v = (k: string) => r.settings.find((s) => s.key === k)?.value;
        const on = v("prizes.enabled");
        setCfg({ enabled: on === true || on === "true", launchOn: String(v("prizes.launch_on") ?? "") });
      })
      .catch(() => setCfg(null));
  }, []);

  const rows = entries?.entrants ?? [];
  const totalEntries = rows.reduce((a, r) => a + r.entries, 0);

  return (
    <PageContainer>
      <PageHero icon={IconCatGifts} title={P.title}>
        <p className="px-4 pb-4 text-sm text-ink-muted">{P.rules}</p>
      </PageHero>

      <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-3">
        <Tile
          icon={IconStatus}
          label={P.status}
          value={cfg ? (cfg.enabled ? P.statusOn : P.statusOff) : "—"}
          tone={cfg?.enabled ? "success" : "muted"}
        />
        <Tile icon={IconDate} label={P.launchOn} value={cfg?.launchOn ? fmtDate(cfg.launchOn) : "—"} tone="muted" />
        <Tile
          icon={IconDate}
          label={P.nextDraw}
          value={cfg ? fmtDate(nextDraw(cfg.launchOn)) : "—"}
          sub={P.nextDrawHint}
          tone="accent"
        />
      </div>

      <div className="space-y-4">
        <PrizeStorePanel />

        <Panel title={P.entrantsTitle}>
          <div className="mb-3">
            <Chips wrap items={months.map((k) => ({ id: k, label: monthLabel(k) }))} value={month} onChange={setMonth} />
          </div>
          {entriesErr ? (
            <AdminReload label={entriesErr} onRetry={loadEntries} />
          ) : entries === null ? (
            <AdminLoading />
          ) : rows.length === 0 ? (
            <AdminEmpty icon={IconCatGifts} title={P.entrantsEmpty} />
          ) : (
            <>
              <p className="mb-2 text-xs text-ink-muted">
                {P.perEntry.replace("{n}", fmtNum(entries.invites_per_entry))} · {P.colEntries}: {fmtNum(totalEntries)}
              </p>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-start text-ink-muted">
                      <th className="p-2 text-start">{P.colName}</th>
                      <th className="p-2 text-start">{P.colInvites}</th>
                      <th className="p-2 text-start">{P.colEntries}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((r) => (
                      <tr key={r.user_id} className="border-t border-line-soft">
                        <td className="p-2">
                          <Person id={r.user_id} name={r.name} phone={r.phone} canUser={open.user} />
                        </td>
                        <td className="p-2 tabular-nums">{fmtNum(r.invites)}</td>
                        <td className="p-2 font-bold tabular-nums">{fmtNum(r.entries)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </Panel>

        <Panel title={P.drawsTitle}>
          {drawsErr ? (
            <AdminReload label={drawsErr} onRetry={loadDraws} />
          ) : draws === null ? (
            <AdminLoading />
          ) : draws.length === 0 ? (
            <AdminEmpty icon={IconCatGifts} title={P.drawsEmpty} />
          ) : (
            <div className="space-y-4">
              {draws.map((d) => (
                <section key={d.month} className="rounded-card border border-line-soft p-3">
                  <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <h3 className="heading-card">{monthLabel(d.month)}</h3>
                    <span className="text-xs text-ink-muted">
                      {P.drawnAt}: {fmtDate(d.drawn_at)} · {P.drawEntrants}: {fmtNum(d.entrants)} · {P.drawEntries}:{" "}
                      {fmtNum(d.total_entries)}
                    </span>
                  </div>
                  <div className="mb-3 grid grid-cols-1 gap-2 sm:grid-cols-3">
                    <Tile icon={IconWallet} label={P.awarded} value={fmtMoney(d.awarded)} tone="accent" />
                    <Tile icon={IconCheck} label={P.spent} value={fmtMoney(d.spent)} tone="success" />
                    <Tile
                      icon={IconStatus}
                      label={P.expired}
                      value={fmtMoney(d.expired)}
                      tone={d.expired > 0 ? "warning" : "muted"}
                    />
                  </div>
                  {d.winners.length === 0 ? (
                    <p className="text-sm text-ink-muted">{P.noWinners}</p>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="text-start text-ink-muted">
                            <th className="p-2 text-start">{P.colRank}</th>
                            <th className="p-2 text-start">{P.colName}</th>
                            <th className="p-2 text-start">{P.colAmount}</th>
                            <th className="p-2 text-start">{P.colSpent}</th>
                            <th className="p-2 text-start">{P.colExpired}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {d.winners.map((w) => (
                            <tr key={`${w.rank}-${w.user_id}`} className="border-t border-line-soft">
                              <td className="p-2">
                                <Badge variant="neutral">{fmtNum(w.rank)}</Badge>
                              </td>
                              <td className="p-2">
                                <Person id={w.user_id} name={w.name} phone={w.phone} canUser={open.user} />
                              </td>
                              <td className="p-2 font-bold" dir="ltr">
                                {fmtMoney(w.amount)}
                              </td>
                              <td className="p-2" dir="ltr">
                                {fmtMoney(w.spent)}
                              </td>
                              <td className="p-2" dir="ltr">
                                {fmtMoney(w.expired)}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </section>
              ))}
            </div>
          )}
        </Panel>

        <Panel title={P.settingsTitle}>
          <p className="mb-3 text-sm text-ink-muted">{P.settingsHint}</p>
          {open.settings && (
            <ButtonLink href="/dashboard/settings?topic=money" variant="secondary">
              <IconSettings size={14} />
              {P.settingsOpen}
            </ButtonLink>
          )}
        </Panel>
      </div>
    </PageContainer>
  );
}
