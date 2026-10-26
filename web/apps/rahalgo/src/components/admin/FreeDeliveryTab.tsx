"use client";

/**
 * **«التوصيل المجاني»** — تبويبٌ في صفحة العروض (قرارُ المالك ٢٠٢٦-١٠-٢٣).
 *
 * مفتاحٌ وبدايةٌ ونهاية، وما يشمله العرض، وسقفٌ اختياريّ. **مرّةٌ لكلّ زبونٍ بالعرض**،
 * و«عرض جديد» يُرجع للجميع حقَّ توصيلة. والعدّادُ: كم انعطى وكم دفعت المنصّة.
 *
 * **ولا زرَّ حفظ** (قرارُ المالك ٢٠٢٦-١٠-٢٧: «ما يلزم زر حفظ، في خيار تشغيل وإطفاء والمعلومات تحفظ
 * تلقائي») — كلُّ تغييرٍ يُحفظ لحظتَه. **والمدّةُ تُختار بالساعات أو الأيّام** («نحط كم يوم العرض أو كم
 * ساعة») وتبدأ من الآن، والتاريخُ يُقرأ جملةً مفهومة؛ و«تاريخ محدّد» لمن أراد البدايةَ والنهايةَ بيده.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Confirm,
  Input,
  LoadingState,
  Switch,
  IconWallet,
  IconPromos,
  IconMoto,
  IconDate,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const F = m.admin.promos.freeDelivery;

const KINDS = [
  { key: "standard", label: F.kindStandard },
  { key: "purchase", label: F.kindPurchase },
  { key: "amanah", label: F.kindAmanah },
  { key: "merchant", label: F.kindMerchant },
] as const;

/** **مُدَدٌ جاهزة بالساعات** — والعرضُ يبدأ من لحظة الاختيار. */
const DURATIONS: { hours: number; label: string }[] = [
  { hours: 3, label: F.dur3h },
  { hours: 6, label: F.dur6h },
  { hours: 12, label: F.dur12h },
  { hours: 24, label: F.dur1d },
  { hours: 72, label: F.dur3d },
  { hours: 168, label: F.dur1w },
  { hours: 720, label: F.dur1m },
];

interface Campaign {
  id: string;
  active: boolean;
  starts_at: string;
  ends_at: string;
  kinds: string[];
  max_fee: number | null;
  running: boolean;
  claims: number;
  waived: number;
}

interface Form {
  active: boolean;
  starts: string; // ISO
  ends: string; // ISO
  kinds: string[];
  maxFee: string;
}

/** **تاريخُ الحقل بتوقيت الجهاز** — `datetime-local` لا يقرأ المنطقة. */
function toLocal(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function blank(): Form {
  const now = new Date();
  return {
    active: false,
    starts: now.toISOString(),
    ends: new Date(now.getTime() + 7 * 24 * 3600 * 1000).toISOString(),
    kinds: ["standard"],
    maxFee: "",
  };
}

/** **كم باقي** — «يومين و٥ ساعات» · «٣ ساعات» · «انتهى». */
function leftLabel(endIso: string): string {
  const ms = new Date(endIso).getTime() - Date.now();
  if (ms <= 0) return F.ended;
  const h = Math.floor(ms / 3_600_000);
  const d = Math.floor(h / 24);
  const rh = h % 24;
  if (d > 0) return F.leftDays.replace("{d}", fmtNum(d)).replace("{h}", fmtNum(rh));
  if (h > 0) return F.leftHours.replace("{h}", fmtNum(h));
  return F.leftMinutes.replace("{m}", fmtNum(Math.max(1, Math.floor(ms / 60_000))));
}

export default function FreeDeliveryTab() {
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [form, setForm] = useState<Form>(blank);
  const [loaded, setLoaded] = useState(false);
  const [state, setState] = useState<"" | "saving" | "saved">("");
  const [error, setError] = useState("");
  const [custom, setCustom] = useState(false);
  const [askNew, setAskNew] = useState(false);
  const busy = useRef(false);

  const fill = useCallback((c: Campaign | null) => {
    setCampaign(c);
    if (c) {
      setForm({
        active: c.active,
        starts: c.starts_at,
        ends: c.ends_at,
        kinds: c.kinds,
        maxFee: c.max_fee ? String(c.max_fee) : "",
      });
    }
  }, []);

  const load = useCallback(() => {
    api<{ campaign: Campaign | null }>("/api/v1/admin/free-delivery")
      .then((r) => fill(r.campaign))
      .catch((e) => setError(errorText(e)))
      .finally(() => setLoaded(true));
  }, [fill]);
  useEffect(load, [load]);

  /** **يُحفظ لحظتَه** — أوّلُ تغييرٍ بلا عرضٍ يُنشئه، وما بعده يعدّله. */
  async function persist(next: Form, asNew = false) {
    if (busy.current) return;
    busy.current = true;
    setForm(next);
    setState("saving");
    setError("");
    const body = {
      active: next.active,
      starts_at: next.starts,
      ends_at: next.ends,
      kinds: next.kinds,
      max_fee: next.maxFee.trim() ? Number(next.maxFee) : null,
    };
    try {
      const r =
        campaign && !asNew
          ? await api<{ campaign: Campaign }>(`/api/v1/admin/free-delivery/${campaign.id}`, {
              method: "PATCH",
              body: JSON.stringify(body),
            })
          : await api<{ campaign: Campaign }>("/api/v1/admin/free-delivery", { method: "POST", body: JSON.stringify(body) });
      fill(r.campaign);
      setState("saved");
    } catch (e) {
      setError(errorText(e));
      setState("");
      load();
    } finally {
      busy.current = false;
    }
  }

  function pickDuration(hours: number) {
    const now = new Date();
    setCustom(false);
    void persist({ ...form, starts: now.toISOString(), ends: new Date(now.getTime() + hours * 3_600_000).toISOString() });
  }

  function toggleKind(k: string, on: boolean) {
    const kinds = on ? [...form.kinds, k] : form.kinds.filter((x) => x !== k);
    if (kinds.length === 0) return;
    void persist({ ...form, kinds });
  }

  if (!loaded) return <LoadingState variant="stats" />;

  const running = campaign?.running ?? false;
  const spanHours = Math.round((new Date(form.ends).getTime() - new Date(form.starts).getTime()) / 3_600_000);
  return (
    <div className="space-y-4">
      {error && <Alert tone="error">{error}</Alert>}

      {/* ── الحالُ بنظرة: شغّال أو لا، ومن إيمتى لإيمتى، وكم باقي ── */}
      <section className="surface flex flex-wrap items-center gap-4 p-4">
        <span
          className={`inline-flex h-14 w-14 shrink-0 items-center justify-center rounded-full ${
            running ? "bg-success-solid text-on-solid" : "bg-field text-ink-muted"
          }`}
        >
          <IconMoto size={26} />
        </span>
        <div className="min-w-0 flex-1 space-y-1">
          <h2 className="heading-card flex flex-wrap items-center gap-2">
            {m.admin.promos.tabFreeDelivery}
            <Badge variant={running ? "success" : "neutral"}>{running ? F.running : F.notRunning}</Badge>
          </h2>
          {campaign ? (
            <p className="text-sm">
              {F.fromTo.replace("{from}", fmtDateTime(form.starts)).replace("{to}", fmtDateTime(form.ends))}
              {running && <b className="text-success"> · {leftLabel(form.ends)}</b>}
            </p>
          ) : (
            <p className="text-sm font-medium">{F.none}</p>
          )}
          <p className="text-xs text-ink-muted">{F.hint}</p>
        </div>
        <div className="flex shrink-0 items-center gap-3 rounded-control bg-field px-3 py-2">
          <Switch checked={form.active} onChange={(v) => void persist({ ...form, active: v })} label={F.active} />
        </div>
      </section>

      {campaign && (
        <div className="grid grid-cols-2 gap-2">
          <Tile icon={IconPromos} label={F.claims} value={fmtNum(campaign.claims)} tone={campaign.claims > 0 ? "accent" : "muted"} />
          <Tile icon={IconWallet} label={F.waived} value={fmtMoney(campaign.waived)} tone={campaign.waived > 0 ? "success" : "muted"} />
        </div>
      )}

      <section className="surface space-y-5 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="heading-card">{F.formTitle}</h2>
          <span className={`text-xs ${state === "saved" ? "text-success" : "text-ink-muted"}`}>
            {state === "saving" ? F.autoSaving : state === "saved" ? F.autoSaved : F.autoSaveHint}
          </span>
        </div>

        {/* ── المدّة ── */}
        <div className="space-y-2">
          <p className="flex items-center gap-1.5 text-sm font-bold">
            <IconDate size={15} />
            {F.durationTitle}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {DURATIONS.map((d) => {
              const on = !custom && campaign != null && Math.abs(spanHours - d.hours) < 1;
              return (
                <button
                  key={d.hours}
                  type="button"
                  onClick={() => pickDuration(d.hours)}
                  className={`rounded-full px-3 py-1.5 text-sm font-bold transition-colors ${
                    on ? "bg-accent text-on-bright" : "border border-line-soft text-ink-muted hover:bg-row-hover hover:text-ink"
                  }`}
                >
                  {d.label}
                </button>
              );
            })}
            <button
              type="button"
              onClick={() => setCustom((c) => !c)}
              className={`rounded-full px-3 py-1.5 text-sm font-bold transition-colors ${
                custom ? "bg-accent text-on-bright" : "border border-line-soft text-ink-muted hover:bg-row-hover hover:text-ink"
              }`}
            >
              {F.durCustom}
            </button>
          </div>
          <p className="text-xs text-ink-muted">{F.durationHint}</p>
          {custom && (
            <div className="grid grid-cols-1 gap-3 rounded-control bg-field p-3 sm:grid-cols-2">
              <Input
                type="datetime-local"
                label={F.startsAt}
                value={toLocal(form.starts)}
                onChange={(e) => e.target.value && void persist({ ...form, starts: new Date(e.target.value).toISOString() })}
              />
              <Input
                type="datetime-local"
                label={F.endsAt}
                value={toLocal(form.ends)}
                onChange={(e) => e.target.value && void persist({ ...form, ends: new Date(e.target.value).toISOString() })}
              />
            </div>
          )}
        </div>

        {/* ── ما يشمله ── */}
        <div className="space-y-2">
          <p className="text-sm font-bold">{F.kinds}</p>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {KINDS.map((k) => (
              <div key={k.key} className="rounded-control border border-line-soft px-3 py-2">
                <Checkbox
                  id={`fd-${k.key}`}
                  label={k.label}
                  checked={form.kinds.includes(k.key)}
                  onChange={(e) => toggleKind(k.key, e.target.checked)}
                />
              </div>
            ))}
          </div>
        </div>

        {/* ── السقف ── يُحفظ عند الخروج من الخانة */}
        <div>
          <Input
            type="number"
            inputMode="numeric"
            min={1}
            label={F.maxFee}
            value={form.maxFee}
            onChange={(e) => setForm((f) => ({ ...f, maxFee: e.target.value }))}
            onBlur={() => {
              const was = campaign?.max_fee ? String(campaign.max_fee) : "";
              if (form.maxFee.trim() !== was) void persist(form);
            }}
          />
          <p className="mt-1 text-xs text-ink-muted">{F.maxFeeHint}</p>
        </div>

        {campaign && (
          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-soft pt-4">
            <p className="min-w-0 flex-1 text-xs text-ink-muted">{F.createNewHint}</p>
            <Button variant="secondary" onClick={() => setAskNew(true)}>
              {F.createNew}
            </Button>
          </div>
        )}
      </section>

      <Confirm
        open={askNew}
        tone="primary"
        title={F.createNew}
        body={F.createNewHint}
        confirmLabel={F.createNew}
        onCancel={() => setAskNew(false)}
        onConfirm={() => {
          setAskNew(false);
          void persist({ ...form, starts: new Date().toISOString() }, true);
        }}
      />
    </div>
  );
}
