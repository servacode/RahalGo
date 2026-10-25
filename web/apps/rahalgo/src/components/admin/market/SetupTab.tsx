"use client";

/**
 * **تبويبُ إعداد المتاجر — مَن ضبط ومَن لم يضبط** (طلبُ المالك ٢٠٢٦-١٠-٠٨، وشكلُه ٢٠٢٦-١٠-٢٦:
 * «بطريقة احترافية»).
 *
 * الدوامُ والعنوانُ والموقعُ، **وتأكيدُ طريقة المستحقّات وقبول الاسترداد** — ومن اختار (صاحبُ
 * المتجر من تطبيقه أو الموظّفُ من صفحة المتجر).
 *
 * **ورأسٌ بأعدادٍ تُرشِّح بضغطة**، وكلُّ متجرٍ بطاقةٌ فيها نسبةُ اكتماله وقائمةُ بنوده بعلامة صحٍّ
 * أو نقص. **والتذكيرُ بيد الموظّف لا بالبوت**: الواتساب يفتح المحادثة على جهازه والرسالةُ جاهزة،
 * والإشعارُ يرنّ في تطبيق صاحب المتجر.
 */

import { useState, type ComponentType } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Button,
  LoadingState,
  EmptyState,
  useLiveData,
  useToast,
  IconHourglass,
  IconStore,
  IconLocation,
  IconWallet,
  IconStatus,
  IconCheck,
  IconClose,
  IconBell,
  IconWhatsApp,
  IconCheckAll,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const U = m.admin.market.setup;

interface SetupRow {
  id: string;
  name: string;
  status: string;
  phone: string;
  has_hours: boolean;
  has_address: boolean;
  has_location: boolean;
  settlement_method: "cash" | "wallet";
  settlement_confirmed_by: "merchant" | "admin" | null;
  accepts_returns: boolean;
  returns_confirmed_by: "merchant" | "admin" | null;
  missing: string[];
  message: string;
  whatsapp_url: string;
}

type Key = "hours" | "address" | "location" | "settlement" | "returns";
type IconT = ComponentType<{ size?: number; className?: string }>;
const KEYS: { key: Key; label: string; Icon: IconT }[] = [
  { key: "hours", label: U.hours, Icon: IconHourglass },
  { key: "address", label: U.address, Icon: IconStore },
  { key: "location", label: U.location, Icon: IconLocation },
  { key: "settlement", label: U.settlement, Icon: IconWallet },
  { key: "returns", label: U.returns, Icon: IconStatus },
];

/** **سطرُ بندٍ في بطاقة المتجر** — علامةُ صحٍّ خضراء أو نقصٍ حمراء، والتفصيلُ بجانبه. */
function Item({ ok, label, Icon, detail }: { ok: boolean; label: string; Icon: IconT; detail?: string }) {
  return (
    <li className="flex items-center gap-2 text-sm">
      <span
        className={`inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-on-solid ${
          ok ? "bg-success-solid" : "bg-danger-solid"
        }`}
      >
        {ok ? <IconCheck size={13} /> : <IconClose size={13} />}
      </span>
      <Icon size={15} className="shrink-0 text-ink-muted" />
      <span className={ok ? "" : "font-bold"}>{label}</span>
      {detail && <span className="min-w-0 truncate text-xs text-ink-muted">· {detail}</span>}
    </li>
  );
}

export default function SetupTab() {
  const { data, reload } = useLiveData<{ stores: SetupRow[] }>(
    () => api("/api/v1/admin/merchants-setup"),
    ["merchant", "catalog"],
  );
  /** "missing" = الناقصين بس · "all" = الكل · أو بندٌ بعينه. */
  const [filter, setFilter] = useState<"missing" | "all" | Key>("missing");
  const [busy, setBusy] = useState<string | null>(null);
  const { push, Toaster } = useToast();

  if (!data) return <LoadingState />;

  const all = data.stores;
  const incomplete = all.filter((s) => s.missing.length > 0);
  const done = all.length - incomplete.length;
  const count = (k: Key) => all.filter((s) => s.missing.includes(k)).length;
  const rows =
    filter === "all" ? all : filter === "missing" ? incomplete : all.filter((s) => s.missing.includes(filter));
  const pct = all.length ? Math.round((done / all.length) * 100) : 100;

  async function remind(s: SetupRow) {
    setBusy(s.id);
    try {
      const res = await api<{ sent: boolean }>(`/api/v1/admin/merchants/${s.id}/setup-reminder`, {
        method: "POST",
      });
      push(res.sent ? U.sent : U.notSent, res.sent ? "success" : "error");
      reload();
    } catch (err) {
      push(errorText(err), "error");
    } finally {
      setBusy(null);
    }
  }

  const iconBtn = "!h-10 !w-10 inline-flex items-center justify-center !min-h-0 !min-w-0 !p-0 rounded-full";

  return (
    <div className="space-y-4">
      {/* **الرأس**: نسبةُ الاكتمال بشريط، وتحتها عدُّ كلّ بندٍ ناقص — والضغطُ عليه يرشّح. */}
      <section className="surface space-y-4 p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 space-y-1">
            <h2 className="heading-section">{U.title}</h2>
            <p className="max-w-3xl text-xs text-ink-muted">{U.hint}</p>
          </div>
          <div className="flex shrink-0 overflow-hidden rounded-full border border-line text-sm" role="radiogroup">
            {(
              [
                ["missing", U.onlyMissing, incomplete.length],
                ["all", U.showAll, all.length],
              ] as const
            ).map(([k, label, n]) => (
              <button
                key={k}
                type="button"
                role="radio"
                aria-checked={filter === k}
                onClick={() => setFilter(k)}
                className={`px-4 py-1.5 font-bold ${filter === k ? "bg-accent text-on-bright" : "text-ink hover:bg-row-hover"}`}
              >
                {label} <span className="tabular-nums">({fmtNum(n)})</span>
              </button>
            ))}
          </div>
        </div>

        <div className="space-y-1.5">
          <div className="flex items-baseline justify-between text-sm">
            <span className="font-bold">{U.completeOf.replace("{done}", fmtNum(done)).replace("{total}", fmtNum(all.length))}</span>
            <span className="tabular-nums text-ink-muted">{`${fmtNum(pct)}%`}</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-field">
            <div className="h-full rounded-full bg-success-solid transition-all" style={{ width: `${pct}%` }} />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          {KEYS.map(({ key, label, Icon }) => {
            const n = count(key);
            const on = filter === key;
            return (
              <button
                key={key}
                type="button"
                onClick={() => setFilter(on ? "missing" : key)}
                aria-pressed={on}
                className={`flex items-center gap-3 rounded-card border p-3 text-start transition-colors ${
                  on ? "border-accent bg-row-hover" : "border-line hover:bg-row-hover"
                }`}
              >
                <span
                  className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${
                    n > 0 ? "bg-warning-solid text-on-solid" : "bg-success-solid text-on-solid"
                  }`}
                >
                  <Icon size={17} />
                </span>
                <span className="min-w-0">
                  <span className="figure block leading-none">{fmtNum(n)}</span>
                  <span className="block truncate text-xs text-ink-muted">{U.missingOf.replace("{x}", label)}</span>
                </span>
              </button>
            );
          })}
        </div>
      </section>

      {rows.length === 0 ? (
        <EmptyState icon={IconCheckAll} title={U.allDone} />
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {rows.map((s) => {
            const total = KEYS.length;
            const got = total - s.missing.filter((k) => KEYS.some((x) => x.key === k)).length;
            const p = Math.round((got / total) * 100);
            return (
              <div key={s.id} className="surface flex flex-col gap-3 p-4">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <Link className="block truncate font-bold hover:underline" href={`/dashboard/merchants/${s.id}`}>
                      {s.name}
                    </Link>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {s.phone || U.noPhone}
                    </span>
                  </div>
                  <span
                    className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-bold tabular-nums text-on-solid ${
                      got === total ? "bg-success-solid" : "bg-warning-solid"
                    }`}
                  >
                    {fmtNum(got)} / {fmtNum(total)}
                  </span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-field">
                  <div
                    className={`h-full rounded-full ${got === total ? "bg-success-solid" : "bg-warning-solid"}`}
                    style={{ width: `${p}%` }}
                  />
                </div>

                <ul className="space-y-1.5">
                  <Item ok={s.has_hours} label={U.hours} Icon={IconHourglass} />
                  <Item ok={s.has_address} label={U.address} Icon={IconStore} />
                  <Item ok={s.has_location} label={U.location} Icon={IconLocation} />
                  <Item
                    ok={!!s.settlement_confirmed_by}
                    label={U.settlement}
                    Icon={IconWallet}
                    detail={
                      s.settlement_confirmed_by
                        ? `${s.settlement_method === "wallet" ? U.wallet : U.cash} · ${
                            s.settlement_confirmed_by === "merchant" ? U.byMerchant : U.byStaff
                          }`
                        : U.notAsked
                    }
                  />
                  <Item
                    ok={!!s.returns_confirmed_by}
                    label={U.returns}
                    Icon={IconStatus}
                    detail={
                      s.returns_confirmed_by
                        ? `${s.accepts_returns ? U.returnsYes : U.returnsNo} · ${
                            s.returns_confirmed_by === "merchant" ? U.byMerchant : U.byStaff
                          }`
                        : U.notAsked
                    }
                  />
                </ul>

                {s.missing.length > 0 && (
                  <div className="mt-auto flex items-center justify-end gap-1.5 border-t border-line-soft pt-3">
                    {s.whatsapp_url && (
                      <a
                        href={s.whatsapp_url}
                        target="_blank"
                        rel="noreferrer"
                        title={U.whatsapp}
                        aria-label={U.whatsapp}
                        className="inline-flex h-10 w-10 items-center justify-center rounded-full bg-success-solid text-on-solid hover:opacity-90"
                      >
                        <IconWhatsApp size={19} />
                      </a>
                    )}
                    <Button
                      variant="ghost"
                      title={U.appNotify}
                      aria-label={U.appNotify}
                      disabled={busy === s.id}
                      onClick={() => void remind(s)}
                      className={`${iconBtn} text-ink-muted hover:text-ink`}
                    >
                      <IconBell size={18} />
                    </Button>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
      <Toaster />
    </div>
  );
}
