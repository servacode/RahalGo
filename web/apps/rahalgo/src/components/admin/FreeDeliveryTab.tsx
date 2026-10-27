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
 *
 * (تدقيقُ العروض ٢٠٢٦-١٠-٢٧:)
 *  - «عرض جديد» يبدأ من الآن بمدّةٍ تُختار في نافذة التأكيد — **كان يأخذ نهايةَ العرض القديم**، فإن
 *    كان منتهياً جاءت النهايةُ قبل البداية وردّ الخادم. **والخادمُ يُطفئ القديمَ معه** (لا عرضَ مخفيٌّ
 *    يبقى يُعطي توصيلاتٍ مجانيّة).
 *  - «مو شغّال» يقول لماذا: خلصت مدّته · لسا ما بلّش · مطفي.
 *  - التاريخُ المحدّد يُطبَّق بزرٍّ لا مع كلّ حرف — **وتغييرٌ وقع والحفظُ السابقُ شغّالٌ يُصفّ لا يُرمى.**
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDate, fmtTime, damascusDay, errorText } from "@rahalgo/i18n";
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
function leftLabel(endIso: string, now: number): string {
  const ms = new Date(endIso).getTime() - now;
  if (ms <= 0) return F.ended;
  const h = Math.floor(ms / 3_600_000);
  const d = Math.floor(h / 24);
  const rh = h % 24;
  if (d > 0) return F.leftDays.replace("{d}", fmtNum(d)).replace("{h}", fmtNum(rh));
  if (h > 0) return F.leftHours.replace("{h}", fmtNum(h));
  return F.leftMinutes.replace("{m}", fmtNum(Math.max(1, Math.floor(ms / 60_000))));
}

/** **«اليوم الساعة ٣:٠٠ م» · «بكرا الساعة ٩:٠٠ ص»** — والأبعدُ بتاريخه. */
function whenText(iso: string, now: number): string {
  const day = damascusDay(iso);
  const name =
    day === damascusDay(now)
      ? F.today
      : day === damascusDay(now + 86_400_000)
        ? F.tomorrow
        : day === damascusDay(now - 86_400_000)
          ? F.yesterday
          : fmtDate(iso);
  return F.dayAt.replace("{day}", name).replace("{time}", fmtTime(iso));
}

const chipCls = (on: boolean) =>
  `rounded-full px-3 py-1.5 text-sm font-bold transition-colors ${
    on ? "bg-accent text-on-bright" : "border border-line-soft text-ink-muted hover:bg-row-hover hover:text-ink"
  }`;

export default function FreeDeliveryTab() {
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [form, setForm] = useState<Form>(blank);
  const [loaded, setLoaded] = useState(false);
  const [state, setState] = useState<"" | "saving" | "saved">("");
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [custom, setCustom] = useState(false);
  const [dStart, setDStart] = useState("");
  const [dEnd, setDEnd] = useState("");
  const [askNew, setAskNew] = useState(false);
  const [newHours, setNewHours] = useState(168);
  const [newActive, setNewActive] = useState(true);
  const [now, setNow] = useState(() => Date.now());
  const busy = useRef(false);
  const campaignRef = useRef<Campaign | null>(null);
  /** **التغييرُ الذي وقع والحفظُ السابقُ شغّال** — يُحفظ بعده لا يُرمى. */
  const queued = useRef<{ next: Form; asNew: boolean } | null>(null);

  // **والعدّادُ «باقيله…» يتحدّث لحاله** — كلَّ دقيقة.
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(t);
  }, []);

  const fill = useCallback((c: Campaign | null, keepForm = false) => {
    setCampaign(c);
    campaignRef.current = c;
    if (c && !keepForm) {
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
      .then((r) => {
        fill(r.campaign);
        setNow(Date.now());
      })
      .catch((e) => setError(errorText(e)))
      .finally(() => setLoaded(true));
  }, [fill]);
  useEffect(load, [load]);

  async function run(next: Form, asNew: boolean) {
    busy.current = true;
    setState("saving");
    setError("");
    const body = {
      active: next.active,
      starts_at: next.starts,
      ends_at: next.ends,
      kinds: next.kinds,
      max_fee: next.maxFee.trim() ? Number(next.maxFee) : null,
    };
    const current = campaignRef.current;
    try {
      const r =
        current && !asNew
          ? await api<{ campaign: Campaign }>(`/api/v1/admin/free-delivery/${current.id}`, {
              method: "PATCH",
              body: JSON.stringify(body),
            })
          : await api<{ campaign: Campaign }>("/api/v1/admin/free-delivery", { method: "POST", body: JSON.stringify(body) });
      fill(r.campaign, queued.current !== null);
      setNow(Date.now());
      setState("saved");
    } catch (e) {
      setError(errorText(e));
      setState("");
      queued.current = null;
      load();
    } finally {
      busy.current = false;
    }
    const q = queued.current;
    if (q) {
      queued.current = null;
      void run(q.next, q.asNew);
    }
  }

  /** **يُحفظ لحظتَه** — أوّلُ تغييرٍ بلا عرضٍ يُنشئه، وما بعده يعدّله. */
  function persist(next: Form, asNew = false) {
    setForm(next);
    setNote("");
    if (busy.current) {
      queued.current = { next, asNew: asNew || (queued.current?.asNew ?? false) };
      return;
    }
    void run(next, asNew);
  }

  function pickDuration(hours: number) {
    const t = new Date();
    setCustom(false);
    persist({ ...form, starts: t.toISOString(), ends: new Date(t.getTime() + hours * 3_600_000).toISOString() });
  }

  function toggleKind(k: string, on: boolean) {
    const kinds = on ? [...form.kinds, k] : form.kinds.filter((x) => x !== k);
    if (kinds.length === 0) {
      setNote(F.lastKind);
      return;
    }
    persist({ ...form, kinds });
  }

  function openCustom() {
    if (custom) {
      setCustom(false);
      return;
    }
    setDStart(toLocal(form.starts));
    setDEnd(toLocal(form.ends));
    setCustom(true);
  }

  function applyDates() {
    if (!dStart || !dEnd) return;
    const s = new Date(dStart);
    const e = new Date(dEnd);
    if (!(e.getTime() > s.getTime())) {
      setNote(F.datesInvalid);
      return;
    }
    persist({ ...form, starts: s.toISOString(), ends: e.toISOString() });
  }

  function openNew() {
    const span = Math.round((new Date(form.ends).getTime() - new Date(form.starts).getTime()) / 3_600_000);
    setNewHours(DURATIONS.some((d) => d.hours === span) ? span : 168);
    setNewActive(true);
    setAskNew(true);
  }

  if (!loaded) return <LoadingState variant="stats" />;

  const startsMs = new Date(form.starts).getTime();
  const endsMs = new Date(form.ends).getTime();
  // **والحالُ من الساعة لا من آخر قراءة** — عرضٌ خلص وأنت على الصفحة يقول «خلص».
  const running = campaign != null && form.active && now >= startsMs && now < endsMs;
  const why = !campaign || running
    ? ""
    : !form.active
      ? F.whyOff
      : now >= endsMs
        ? F.whyEnded
        : F.whyNotStarted.replace("{when}", whenText(form.starts, now));
  const spanHours = Math.round((endsMs - startsMs) / 3_600_000);
  const datesDirty = custom && (dStart !== toLocal(form.starts) || dEnd !== toLocal(form.ends));
  return (
    <div className="space-y-4">
      {error && <Alert tone="error">{error}</Alert>}

      {/* ── الحالُ بنظرة: شغّال أو لا ولماذا، ومن إيمتى لإيمتى، وكم باقي — والمفتاح ── */}
      <section className="surface space-y-3 p-4">
        <div className="flex flex-wrap items-center gap-4">
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
            {why && <p className="text-sm font-bold text-warning">{why}</p>}
            {campaign ? (
              <p className="text-sm">
                {F.fromTo.replace("{from}", whenText(form.starts, now)).replace("{to}", whenText(form.ends, now))}
                {running && <b className="text-success"> · {leftLabel(form.ends, now)}</b>}
              </p>
            ) : (
              <p className="text-sm font-medium">{F.none}</p>
            )}
            <p className="text-xs text-ink-muted">{F.hint}</p>
          </div>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-control bg-field px-3 py-2.5">
          <p className="min-w-0 flex-1 text-xs text-ink-muted">{F.activeHint}</p>
          <Switch checked={form.active} onChange={(v) => persist({ ...form, active: v })} label={F.active} />
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

        {note && <Alert tone="info">{note}</Alert>}

        {/* ── المدّة ── */}
        <div className="space-y-2">
          <p className="flex items-center gap-1.5 text-sm font-bold">
            <IconDate size={15} />
            {F.durationTitle}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {DURATIONS.map((d) => (
              <button
                key={d.hours}
                type="button"
                onClick={() => pickDuration(d.hours)}
                className={chipCls(!custom && campaign != null && Math.abs(spanHours - d.hours) < 1)}
              >
                {d.label}
              </button>
            ))}
            <button type="button" onClick={openCustom} className={chipCls(custom)}>
              {F.durCustom}
            </button>
          </div>
          <p className="text-xs text-ink-muted">{F.durationHint}</p>
          {campaign && !form.active && <p className="text-xs font-bold text-warning">{F.offNote}</p>}
          {custom && (
            <div className="space-y-3 rounded-control bg-field p-3">
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <Input type="datetime-local" label={F.startsAt} value={dStart} onChange={(e) => setDStart(e.target.value)} />
                <Input type="datetime-local" label={F.endsAt} value={dEnd} onChange={(e) => setDEnd(e.target.value)} />
              </div>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="min-w-0 flex-1 text-xs text-ink-muted">{F.datesHint}</p>
                <Button onClick={applyDates} disabled={!datesDirty || !dStart || !dEnd}>
                  {F.applyDates}
                </Button>
              </div>
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
              if (form.maxFee.trim() !== was) persist(form);
            }}
          />
          <p className="mt-1 text-xs text-ink-muted">{F.maxFeeHint}</p>
        </div>

        {campaign && (
          <div className="space-y-2 border-t border-line-soft pt-4">
            <p className="text-xs text-ink-muted">{F.createNewHint}</p>
            <Button variant="secondary" className="w-full" onClick={openNew}>
              {F.createNew}
            </Button>
          </div>
        )}
      </section>

      <Confirm
        open={askNew}
        tone="primary"
        title={F.createNew}
        body={
          <div className="space-y-3">
            <p>{F.newBody}</p>
            <p className="text-sm font-bold">{F.newDuration}</p>
            <div className="flex flex-wrap gap-1.5">
              {DURATIONS.map((d) => (
                <button key={d.hours} type="button" onClick={() => setNewHours(d.hours)} className={chipCls(newHours === d.hours)}>
                  {d.label}
                </button>
              ))}
            </div>
            <Switch checked={newActive} onChange={setNewActive} label={F.newStartNow} />
          </div>
        }
        confirmLabel={F.createNew}
        onCancel={() => setAskNew(false)}
        onConfirm={() => {
          setAskNew(false);
          setCustom(false);
          const t = new Date();
          persist(
            { ...form, active: newActive, starts: t.toISOString(), ends: new Date(t.getTime() + newHours * 3_600_000).toISOString() },
            true,
          );
        }}
      />
    </div>
  );
}
