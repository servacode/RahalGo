"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الترحيبُ ومعالجُ الإعداد الأوّل** (مرآةُ `SetupWizard.kt`، قرارُ المالك ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يظهر ملءَ الصفحة متى نقص المتجرَ إعدادٌ إلزاميّ** — ولا تُفتح الأقسامُ حتّى
 * يكتمل. **وسؤالٌ واحدٌ في كلّ بطاقة**، بشريط تقدّمٍ و«السابق/التالي».
 *
 * **والخطواتُ هي ما ينقص وحدَه** (`missingRequired`) — ومتجرٌ كاملٌ لا يراه أبداً.
 * والشعارُ اختياريٌّ بزرّ «لاحقاً».
 *
 * **و«التالي» لا يتقدّم حتّى يردّ الخادمُ بالحفظ** — وكلُّ حفظٍ بنداء محرّره القائم.
 * **وفي آخره لا أزرار**: «✓ متجرك أصبح جاهزاً» ثانيتين ونصفاً ثمّ يُفتح «متجري».
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { BrandMark, Button, Confirm, IconBlock, IconCash, IconCheck, IconReturn, IconWallet, LoadingState, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { DayHours, SectionRef } from "@/lib/store/types";
import type { SetupStep } from "@/lib/store/setup";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { LogoCard } from "./StoreSettings";
import { saveAddress } from "./AddressEditor";
import {
  clock12,
  HoursPlanner,
  IconChoicePair,
  nightOpen,
  putHours,
  SectionGrid,
  StorePointMap,
  useHoursPlan,
  useStoreHours,
} from "./SetupParts";

const m = getMessages(defaultLocale);
const W = m.store.wizard;
const H = m.store.hours;
const A = m.store.address;

/** **كم تبقى شاشةُ «جاهز» قبل أن يُفتح «متجري».** */
const READY_MS = 2_500;

const LABEL: Record<SetupStep, string> = {
  logo: W.stepLogo,
  sections: W.stepSections,
  address: W.stepAddress,
  hours: W.stepHours,
  settlement: W.stepSettlement,
  returns: W.stepReturns,
};

interface Sections {
  all: SectionRef[];
  mine: SectionRef[];
  save: (ids: string[]) => Promise<void>;
}

// ── الترحيب ─────────────────────────────────────────────────────────────

function Welcome({ steps, onStart }: { steps: SetupStep[]; onStart: () => void }) {
  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex flex-col gap-3 rounded-b-card bg-primary-solid px-6 pb-8 pt-10 text-on-solid">
        <span className="flex h-18 w-18 items-center justify-center rounded-card bg-paper">
          <BrandMark size={60} />
        </span>
        <h1 className="heading-page">{W.welcomeTitle}</h1>
        <p className="text-base opacity-85">{W.welcomeBody}</p>
      </header>
      <ol className="mx-auto flex w-full max-w-xl flex-1 flex-col gap-4 px-6 py-6">
        <li className="font-bold text-ink-muted">{fill(W.welcomeCount, { n: fmtNum(steps.length) })}</li>
        {steps.map((s, i) => (
          <li key={s} className="flex items-center gap-3">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-tint text-sm font-bold text-primary">
              {fmtNum(i + 1)}
            </span>
            <span className="min-w-0 flex-1 text-base">{LABEL[s]}</span>
            {s === "logo" && <span className="text-sm text-ink-muted">{W.optional}</span>}
          </li>
        ))}
      </ol>
      <div className="mx-auto w-full max-w-xl px-6 pb-6">
        <Button size="lg" className="w-full font-bold" onClick={onStart} data-testid="wizard-start">
          {W.start}
        </Button>
      </div>
    </div>
  );
}

// ── قالبُ الخطوة ────────────────────────────────────────────────────────

function StepFrame({
  index,
  total,
  title,
  children,
  ok,
  busy,
  nextLabel,
  onPrev,
  onNext,
}: {
  index: number;
  total: number;
  title: string;
  children: ReactNode;
  ok: boolean;
  busy: boolean;
  nextLabel?: string;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex min-h-screen flex-col">
      <header className="mx-auto w-full max-w-2xl px-4 pb-2 pt-4">
        <div className="flex items-center gap-2 text-sm">
          <span className="min-w-0 flex-1 text-ink-muted">{W.header}</span>
          <span className="font-bold text-primary">{fill(W.progress, { i: fmtNum(index + 1), n: fmtNum(total) })}</span>
        </div>
        <div className="mt-3 flex gap-1.5" aria-hidden>
          {Array.from({ length: total }, (_, i) => (
            <span key={i} className={`h-1.5 flex-1 rounded-badge ${i <= index ? "bg-primary-solid" : "bg-line"}`} />
          ))}
        </div>
      </header>
      <main className="mx-auto w-full max-w-2xl flex-1 px-4 py-3">
        <section className="surface flex flex-col gap-4 rounded-card p-4 elev-1" data-testid="wizard-step">
          <h2 className="heading-section">{title}</h2>
          {children}
        </section>
      </main>
      <footer className="mx-auto flex w-full max-w-2xl gap-2 px-4 pb-6 pt-2">
        <Button variant="secondary" size="lg" className="w-28 font-bold" onClick={onPrev} disabled={busy}>
          {W.prev}
        </Button>
        <Button size="lg" className="flex-1 font-bold" onClick={onNext} disabled={!ok || busy} data-testid="wizard-next">
          {nextLabel ?? W.next}
        </Button>
      </footer>
    </div>
  );
}

// ── الخطوات ─────────────────────────────────────────────────────────────

interface StepProps {
  index: number;
  total: number;
  onPrev: () => void;
  advance: () => void;
}

/** **حفظٌ ثمّ تقدّم** — والخطأُ يُقال كما قاله الخادم ولا يتقدّم. */
function useRun() {
  const toast = useShopToast();
  const [busy, setBusy] = useState(false);
  async function run(fn: () => Promise<void>) {
    setBusy(true);
    try {
      await fn();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
    }
  }
  return { busy, run };
}

function LogoStep(p: StepProps) {
  const { store } = useStore();
  const has = !!store?.logo_thumb_url;
  return (
    <StepFrame {...p} title={W.logoQ} ok busy={false} nextLabel={has ? undefined : W.later} onNext={p.advance}>
      <LogoCard />
    </StepFrame>
  );
}

function SectionsStep(p: StepProps & { sec: Sections }) {
  const { sec } = p;
  const current = new Set(sec.mine.map((s) => s.id));
  const [picked, setPicked] = useState<Set<string>>(() => current);
  const { busy, run } = useRun();
  const same = picked.size === current.size && [...picked].every((id) => current.has(id));
  return (
    <StepFrame
      {...p}
      title={W.sectionsQ}
      ok={picked.size > 0}
      busy={busy}
      onNext={() => {
        if (same) return p.advance();
        void run(async () => {
          await sec.save([...picked]);
          p.advance();
        });
      }}
    >
      {sec.all.length === 0 ? (
        <p className="text-sm text-danger">{W.sectionsNone}</p>
      ) : (
        <SectionGrid
          all={sec.all}
          picked={picked}
          disabled={busy}
          onToggle={(id) =>
            setPicked((s) => {
              const n = new Set(s);
              if (n.has(id)) n.delete(id);
              else n.add(id);
              return n;
            })
          }
        />
      )}
    </StepFrame>
  );
}

function AddressStep(p: StepProps) {
  const { store, patch } = useStore();
  const [text, setText] = useState(store?.address_text ?? "");
  const [point, setPoint] = useState<{ lat: number; lng: number } | null>(null);
  const { busy, run } = useRun();
  if (!store) return null;
  const lat = point?.lat ?? store.lat;
  const lng = point?.lng ?? store.lng;
  const unchanged = text.trim() === store.address_text && point == null && store.lat != null;
  return (
    <StepFrame
      {...p}
      title={W.addressQ}
      ok={!!text.trim() && lat != null && lng != null}
      busy={busy}
      onNext={() =>
        unchanged
          ? p.advance()
          : void run(async () => {
              patch(await saveAddress(store, text, point));
              p.advance();
            })
      }
    >
      <Textarea
        id="wizard-address"
        label={A.title}
        placeholder={A.hint}
        rows={2}
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <StorePointMap lat={lat} lng={lng} disabled={busy} onPick={(la, ln) => setPoint({ lat: la, lng: ln })} />
    </StepFrame>
  );
}

function HoursPlan({ saved, ...p }: StepProps & { saved: DayHours[] }) {
  const { store, patch } = useStore();
  const plan = useHoursPlan(saved, !!store?.has_hours);
  const { busy, run } = useRun();
  const [night, setNight] = useState<DayHours[] | null>(null);
  if (!store) return null;
  const save = (days: DayHours[]) =>
    void run(async () => {
      await putHours(store.id, days);
      patch({ has_hours: true });
      p.advance();
    });
  const nightDay = night?.find(nightOpen);
  return (
    <StepFrame
      {...p}
      title={W.hoursQ}
      ok={plan.chosen}
      busy={busy}
      onNext={() => {
        const days = plan.days();
        if (days.some(nightOpen)) setNight(days);
        else save(days);
      }}
    >
      <HoursPlanner plan={plan} disabled={busy} />
      <Confirm
        open={!!night}
        title={H.nightTitle}
        body={nightDay ? fill(H.nightText, { open: clock12(nightDay.open_time), close: clock12(nightDay.close_time) }) : ""}
        confirmLabel={H.nightYes}
        tone="primary"
        busy={busy}
        onConfirm={() => {
          const d = night;
          setNight(null);
          if (d) save(d);
        }}
        onCancel={() => setNight(null)}
      />
    </StepFrame>
  );
}

function HoursStep(p: StepProps) {
  const { store } = useStore();
  const { saved, error, reload } = useStoreHours(store?.id ?? "");
  if (!saved) {
    return (
      <StepFrame {...p} title={W.hoursQ} ok={false} busy={false} onNext={() => undefined}>
        {error ? (
          <Button variant="secondary" onClick={() => void reload()}>
            {errorText(error)}
          </Button>
        ) : (
          <LoadingState variant="list" rows={3} />
        )}
      </StepFrame>
    );
  }
  return <HoursPlan {...p} saved={saved} />;
}

function SettlementStep(p: StepProps) {
  const { store, patch } = useStore();
  const [pick, setPick] = useState<"" | "cash" | "wallet">(() =>
    store?.settlement_confirmed ? (store.settlement_method === "wallet" ? "wallet" : "cash") : "",
  );
  const { busy, run } = useRun();
  if (!store) return null;
  return (
    <StepFrame
      {...p}
      title={W.settlementQ}
      ok={pick !== ""}
      busy={busy}
      onNext={() =>
        void run(async () => {
          const r = await api<{ settlement_method: string }>(`/api/v1/merchant/stores/${store.id}/settlement-method`, {
            method: "PATCH",
            body: JSON.stringify({ method: pick }),
          });
          patch({ settlement_method: r.settlement_method, settlement_confirmed: true });
          p.advance();
        })
      }
    >
      <IconChoicePair
        disabled={busy}
        items={[
          { icon: IconCash, title: W.payCash, selected: pick === "cash", onClick: () => setPick("cash") },
          { icon: IconWallet, title: W.payWallet, selected: pick === "wallet", onClick: () => setPick("wallet") },
        ]}
      />
    </StepFrame>
  );
}

function ReturnsStep(p: StepProps) {
  const { store, patch } = useStore();
  const [pick, setPick] = useState<"" | "yes" | "no">(() =>
    store?.returns_confirmed ? (store.accepts_returns ? "yes" : "no") : "",
  );
  const { busy, run } = useRun();
  if (!store) return null;
  return (
    <StepFrame
      {...p}
      title={W.returnsQ}
      ok={pick !== ""}
      busy={busy}
      onNext={() =>
        void run(async () => {
          const accepts = pick === "yes";
          await api(`/api/v1/merchant/stores/${store.id}/settings`, {
            method: "PATCH",
            body: JSON.stringify({ accepts_returns: accepts }),
          });
          patch({ accepts_returns: accepts, returns_confirmed: true });
          p.advance();
        })
      }
    >
      <IconChoicePair
        disabled={busy}
        items={[
          { icon: IconReturn, title: W.returnsYes, selected: pick === "yes", onClick: () => setPick("yes") },
          { icon: IconBlock, title: W.returnsNo, selected: pick === "no", onClick: () => setPick("no") },
        ]}
      />
    </StepFrame>
  );
}

// ── «متجرك جاهز» ────────────────────────────────────────────────────────

/** **علامةُ صحٍّ كبيرةٌ ثمّ يُفتح «متجري» وحدَه** — بلا أزرار. */
function Ready({ onFinish }: { onFinish: () => void }) {
  const done = useRef(onFinish);
  done.current = onFinish;
  useEffect(() => {
    const t = setTimeout(() => done.current(), READY_MS);
    return () => clearTimeout(t);
  }, []);
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-6 p-6" role="status" data-testid="wizard-ready">
      <span className="flex h-32 w-32 animate-[pulse_1s_ease-out_1] items-center justify-center rounded-badge bg-success-solid text-on-solid">
        <IconCheck size={72} />
      </span>
      <p className="heading-page text-center text-success">{W.ready}</p>
    </div>
  );
}

// ── المعالج ─────────────────────────────────────────────────────────────

export default function SetupWizard({
  steps,
  index,
  setIndex,
  sec,
  onFinish,
}: {
  steps: SetupStep[];
  index: number;
  setIndex: (i: number) => void;
  sec: Sections;
  onFinish: () => void;
}) {
  if (index < 0) return <Welcome steps={steps} onStart={() => setIndex(0)} />;
  if (index >= steps.length) return <Ready onFinish={onFinish} />;
  const p: StepProps = {
    index,
    total: steps.length,
    onPrev: () => setIndex(index - 1),
    advance: () => setIndex(index + 1),
  };
  switch (steps[index]) {
    case "logo":
      return <LogoStep key="logo" {...p} />;
    case "sections":
      return <SectionsStep key="sections" {...p} sec={sec} />;
    case "address":
      return <AddressStep key="address" {...p} />;
    case "hours":
      return <HoursStep key="hours" {...p} />;
    case "settlement":
      return <SettlementStep key="settlement" {...p} />;
    default:
      return <ReturnsStep key="returns" {...p} />;
  }
}
