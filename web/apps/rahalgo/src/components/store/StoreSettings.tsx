"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقاتُ «متجري» — كما في شاشة المتجر بالتطبيق بطاقةً بطاقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreScreen.kt` و`StoreViewModel.kt`.)
 *
 * **كلُّ بطاقةٍ تحفظ وحدَها** (`PATCH /settings` بحقلها وحدَه) — كما في
 * التطبيق: لا زرَّ «حفظ الكلّ» يُنسى، **ولا حقلٌ يُرسَل لم يلمسه صاحبُه.**
 *
 * **والمحرّكُ هو الحَكَم** — والخطأُ يُقال كما قاله (`errorText`).
 */

import { useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Button, Confirm, IconBlock, IconEdit, ImageUpload, Input, Modal, FormActions } from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { parseUsd, usdInput, usdText } from "@/lib/store/usd";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const S = m.store;
const T = S.settings;

/** **سؤالٌ بخيارين كبيرين** — مرآةُ `BigChoice`: لا جوابَ بعدُ يُقال «اختار وحدة». */
export function BigChoice({
  question,
  hint,
  options,
  disabled,
  onPick,
  children,
}: {
  question: string;
  hint?: string;
  options: readonly { label: string; on: boolean }[];
  disabled?: boolean;
  onPick: (i: number) => void;
  children?: ReactNode;
}) {
  return (
    <section className="surface flex flex-col gap-3 rounded-card p-4" aria-label={question}>
      <h3 className="heading-card">{question}</h3>
      {hint && <p className="text-xs text-ink-muted">{hint}</p>}
      <div className="grid grid-cols-2 gap-2">
        {options.map((o, i) => (
          <button
            key={o.label}
            type="button"
            aria-pressed={o.on}
            disabled={disabled}
            onClick={() => onPick(i)}
            className={`rounded-control border px-3 py-3 text-sm font-bold transition-colors disabled:opacity-60 ${
              o.on ? "border-2 border-accent bg-accent-tint" : "border-line bg-field hover:bg-row-hover"
            }`}
          >
            {o.label}
          </button>
        ))}
      </div>
      {options.every((o) => !o.on) && <p className="text-xs text-danger">{T.chooseOne}</p>}
      {children}
    </section>
  );
}

/** **حفظُ حقلٍ واحدٍ من إعدادات المتجر** — `PATCH /settings` ثمّ تحديثٌ محلّيّ. */
function useSaveSettings() {
  const { store, patch, reload } = useStore();
  const toast = useShopToast();
  const [busy, setBusy] = useState(false);
  async function save(body: Record<string, unknown>, ok: string, local: Record<string, unknown>): Promise<boolean> {
    if (!store) return false;
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${store.id}/settings`, { method: "PATCH", body: JSON.stringify(body) });
      patch(local);
      toast(ok, "success");
      void reload();
      return true;
    } catch (e) {
      toast(errorText(e), "error");
      return false;
    } finally {
      setBusy(false);
    }
  }
  return { busy, save };
}

/** **اسمُ المتجر وقلمُ تعديله** — مرآةُ `renaming` في `StoreScreen.kt`. */
export function NameCard() {
  const { store } = useStore();
  const { busy, save } = useSaveSettings();
  const [draft, setDraft] = useState<string | null>(null);
  if (!store) return null;
  const suspended = store.status === "suspended";
  if (draft == null) {
    return (
      <div className="flex items-center gap-2">
        <h2 className="heading-page min-w-0 flex-1 truncate">{store.name}</h2>
        {!suspended && (
          <Button variant="ghost" aria-label={T.rename} title={T.rename} onClick={() => setDraft(store.name)}>
            <IconEdit size={18} className="text-accent-text" />
          </Button>
        )}
      </div>
    );
  }
  return (
    <form
      className="flex flex-col gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        const name = draft.trim();
        if (!name) return;
        void save({ name }, T.nameSaved, { name }).then((ok) => ok && setDraft(null));
      }}
    >
      <Input
        id="store-name"
        label={T.storeName}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        error={draft.trim() ? undefined : T.vStoreName}
        autoFocus
      />
      <FormActions submit onCancel={() => setDraft(null)} busy={busy} />
    </form>
  );
}

/**
 * **شعارُ المتجر بيد صاحبه** (اقتراحٌ وافق عليه المالك ٢٠٢٦-١٠-١٩) — مرآةُ زرّ الشعار في
 * تطبيق المتجر: يُرفع بنوع الشعار ثمّ يُحفظ في الإعدادات.
 */
export function LogoCard() {
  const { store, reload } = useStore();
  const { save } = useSaveSettings();
  if (!store || store.status === "suspended") return null;
  return (
    <ImageUpload
      kind="merchant_logo"
      label={T.logo}
      initialUrl={store.logo_thumb_url}
      onChange={(id) => {
        if (id) void save({ logo_media_id: id }, T.logoSaved, {}).then((ok) => { if (ok) void reload(); });
      }}
      api={api}
      mediaUrl={mediaUrl}
      errorText={errorText}
      path="/api/v1/merchant/media"
    />
  );
}

/** **«إغلاق طارئ»** — مرآةُ زرّ `store_emergency` وسؤالِه. */
export function EmergencyButton() {
  const { store, patch, reload } = useStore();
  const toast = useShopToast();
  const [ask, setAsk] = useState(false);
  const [busy, setBusy] = useState(false);
  if (!store || store.emergency_closed || store.status === "suspended") return null;
  async function close() {
    if (!store) return;
    setBusy(true);
    try {
      const r = await api<{ emergency_closed: boolean }>(`/api/v1/merchant/stores/${store.id}/emergency`, {
        method: "POST",
        body: JSON.stringify({ closed: true }),
      });
      patch({ emergency_closed: r.emergency_closed });
      toast(T.closedOk, "success");
      void reload();
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
      setAsk(false);
    }
  }
  return (
    <>
      <Button variant="danger" className="flex w-full items-center justify-center gap-2" onClick={() => setAsk(true)}>
        <IconBlock size={18} />
        {T.emergency}
      </Button>
      <Confirm
        open={ask}
        title={T.emergency}
        body={T.emergencyHint}
        confirmLabel={T.emergencyDo}
        busy={busy}
        onConfirm={() => void close()}
        onCancel={() => setAsk(false)}
      />
    </>
  );
}

/** **طريقةُ استلام المستحقّات** — والتبديلُ بسؤال «ينطبق على الطلبات الجديدة فقط». */
export function SettlementCard() {
  const { store, patch } = useStore();
  const toast = useShopToast();
  const [ask, setAsk] = useState<"" | "cash" | "wallet">("");
  const [busy, setBusy] = useState(false);
  if (!store) return null;
  const confirmed = store.settlement_confirmed;

  async function save(method: "cash" | "wallet") {
    if (!store) return;
    setBusy(true);
    try {
      const r = await api<{ settlement_method: string }>(`/api/v1/merchant/stores/${store.id}/settlement-method`, {
        method: "PATCH",
        body: JSON.stringify({ method }),
      });
      patch({ settlement_method: r.settlement_method, settlement_confirmed: true });
      toast(S.settlement.saved, "success");
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy(false);
      setAsk("");
    }
  }

  const keys = ["cash", "wallet"] as const;
  return (
    <BigChoice
      question={S.settlement.q}
      options={[
        { label: S.settlement.cash, on: confirmed && store.settlement_method === "cash" },
        { label: S.settlement.wallet, on: confirmed && store.settlement_method === "wallet" },
      ]}
      disabled={busy || store.status === "suspended"}
      onPick={(i) => {
        const k = keys[i] ?? "cash";
        if (!confirmed || store.settlement_method !== k) setAsk(k);
      }}
    >
      {store.settlement_method === "cash" && store.unpaid_cash_due > 0 && (
        <p className="flex items-center justify-between text-sm">
          <span>{S.settlement.unpaid}</span>
          <span className="font-bold text-accent-text">{fmtMoney(store.unpaid_cash_due)}</span>
        </p>
      )}
      <Confirm
        open={!!ask}
        title={S.settlement.confirmTitle}
        body={S.settlement.confirmBody}
        confirmLabel={S.settlement.confirmYes}
        tone="primary"
        busy={busy}
        onConfirm={() => ask && void save(ask)}
        onCancel={() => setAsk("")}
      />
    </BigChoice>
  );
}

/** **«تقبل تسترد بضاعتك؟»** — مرآةُ `setAcceptsReturns`. */
export function ReturnsCard() {
  const { store } = useStore();
  const { busy, save } = useSaveSettings();
  if (!store) return null;
  const c = store.returns_confirmed;
  return (
    <BigChoice
      question={T.returnsQ}
      hint={T.returnsHint}
      options={[
        { label: T.yes, on: c && store.accepts_returns },
        { label: T.no, on: c && !store.accepts_returns },
      ]}
      disabled={busy || store.status === "suspended"}
      onPick={(i) => {
        const accepts = i === 0;
        if (accepts === store.accepts_returns && c) return;
        void save({ accepts_returns: accepts }, T.returnsSaved, { accepts_returns: accepts, returns_confirmed: true });
      }}
    />
  );
}

interface RateJump {
  rate: number;
  old: string;
  next: string;
  samples: string;
}

/**
 * **سعرُ الصرف وعملةُ العرض** — مرآةُ بطاقة `store_rate` و`q_display`.
 *
 * **والقفزةُ الكبيرةُ تُسأل** (`exchange_rate_jump`): المحرّكُ يردّها بالقديم
 * والجديد وأمثلةٍ من الأصناف، **ويُعاد الإرسالُ بـ`confirm` إن أكّد.**
 *
 * **ويُحفظ بزرٍّ لا بعد ثانيةٍ من الكتابة** كما في التطبيق — لوحةُ مفاتيح
 * الحاسوب تكتب أسرعَ من المهلة، **ونصفُ رقمٍ يُحفظ سعراً لكلّ الأصناف.**
 */
export function RateCard() {
  const { store, patch, reload } = useStore();
  const toast = useShopToast();
  const { busy: savingDisplay, save } = useSaveSettings();
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [jump, setJump] = useState<RateJump | null>(null);
  useEffect(() => setText(usdText(store?.exchange_rate ?? null)), [store?.id, store?.exchange_rate]);
  if (!store) return null;
  const suspended = store.status === "suspended";
  const value = parseUsd(text);
  const dirty = value != null && value > 0 && value !== store.exchange_rate;

  async function saveRate(rate: number, confirm: boolean) {
    if (!store) return;
    if (!(rate > 0)) {
      toast(T.rateInvalid, "error");
      return;
    }
    setBusy(true);
    try {
      const r = await api<{ exchange_rate: number; items_repriced: number }>(
        `/api/v1/merchant/stores/${store.id}/exchange-rate`,
        { method: "PATCH", body: JSON.stringify({ rate, confirm }) },
      );
      setJump(null);
      patch({ exchange_rate: rate });
      toast(fill(T.rateSaved, { n: fmtNum(r.items_repriced ?? 0) }), "success");
      void reload();
    } catch (e) {
      const body = e instanceof ApiError ? (e.body as { code?: string; details?: Record<string, unknown> }) : null;
      if (body?.code === "exchange_rate_jump") {
        const d = body.details ?? {};
        setJump({
          rate,
          old: String(d.old_rate ?? ""),
          next: String(d.new_rate ?? rate),
          samples: String(d.samples ?? ""),
        });
      } else {
        toast(errorText(e), "error");
      }
    } finally {
      setBusy(false);
    }
  }

  const usd = store.display_currency === "USD";
  return (
    <section className="surface flex flex-col gap-3 rounded-card p-4" aria-label={T.rate}>
      <h3 className="heading-card">{T.rate}</h3>
      <form
        className="flex items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          if (value != null) void saveRate(value, false);
        }}
      >
        <Input
          id="store-rate"
          label={T.rateField}
          inputMode="decimal"
          dir="ltr"
          value={text}
          disabled={suspended}
          onChange={(e) => setText(usdInput(e.target.value))}
          wrapperClassName="min-w-0 flex-1"
        />
        <Button type="submit" disabled={!dirty || busy || suspended} data-testid="store-rate-save">
          {T.rateSave}
        </Button>
      </form>
      <p className="text-xs text-ink-muted">{T.rateHint}</p>
      <h3 className="heading-card pt-2">{T.displayQ}</h3>
      <div className="grid grid-cols-2 gap-2">
        {(
          [
            ["SYP", T.displaySyp, !usd],
            ["USD", T.displayUsd, usd],
          ] as const
        ).map(([code, label, on]) => (
          <button
            key={code}
            type="button"
            aria-pressed={on}
            disabled={savingDisplay || suspended}
            onClick={() => {
              if (on) return;
              if (code === "USD" && !((store.exchange_rate ?? 0) > 0)) {
                toast(T.displayNeedRate, "error");
                return;
              }
              void save({ display_currency: code }, T.displaySaved, { display_currency: code });
            }}
            className={`rounded-control border px-3 py-3 text-sm font-bold transition-colors disabled:opacity-60 ${
              on ? "border-2 border-accent bg-accent-tint" : "border-line bg-field hover:bg-row-hover"
            }`}
          >
            {label}
          </button>
        ))}
      </div>
      <p className="text-xs text-ink-muted">{T.displayHint}</p>
      <Modal open={!!jump} onClose={() => setJump(null)} title={T.rateJumpTitle}>
        {jump && (
          <div className="flex flex-col gap-3">
            <p className="text-sm">{fill(T.rateJumpBody, { old: jump.old, new: jump.next })}</p>
            {jump.samples.trim() && (
              <div className="text-sm">
                <p className="font-bold">{T.rateJumpSamples}</p>
                <p className="whitespace-pre-line text-ink-muted">{jump.samples}</p>
              </div>
            )}
            <FormActions
              onSave={() => void saveRate(jump.rate, true)}
              onCancel={() => setJump(null)}
              busy={busy}
              saveLabel={T.rateJumpYes}
              cancelLabel={T.rateJumpNo}
            />
          </div>
        )}
      </Modal>
    </section>
  );
}

/** **مدّةُ التحضير** — ناقصٌ وزائدٌ بخمس دقائق، كما في التطبيق. */
export function PrepCard() {
  const { store } = useStore();
  const { busy, save } = useSaveSettings();
  if (!store) return null;
  const suspended = store.status === "suspended";
  const minutes = store.default_prep_minutes;
  const set = (n: number) => {
    if (n <= 0) return;
    void save({ default_prep_minutes: n }, T.prepSaved, { default_prep_minutes: n });
  };
  return (
    <section className="surface flex items-center gap-2 rounded-card p-4" aria-label={T.prep}>
      <span className="min-w-0 flex-1 text-sm font-bold">{T.prep}</span>
      <Button
        variant="secondary"
        aria-label={T.prepLess}
        disabled={busy || suspended || minutes <= 5}
        onClick={() => set(minutes - 5)}
      >
        −
      </Button>
      <span className="min-w-20 text-center font-bold tabular-nums">
        {fmtNum(minutes)} {T.prepUnit}
      </span>
      <Button variant="secondary" aria-label={T.prepMore} disabled={busy || suspended} onClick={() => set(minutes + 5)}>
        +
      </Button>
    </section>
  );
}
