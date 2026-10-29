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

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Button, Confirm, IconBlock, IconCash, IconReturn, IconWallet, ImageUpload, Input, Modal, FormActions } from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { parseUsd, usdInput, usdText } from "@/lib/store/usd";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { IconChoicePair } from "./SetupParts";

const m = getMessages(defaultLocale);
const S = m.store;
const T = S.settings;
const W = S.wizard;

/** **حفظُ حقلٍ واحدٍ من إعدادات المتجر** — `PATCH /settings` ثمّ تحديثٌ محلّيّ. */
export function useSaveSettings() {
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
      crop
    />
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

  const pick = (k: "cash" | "wallet") => {
    if (!confirmed || store.settlement_method !== k) setAsk(k);
  };
  return (
    <section className="surface flex flex-col gap-3 rounded-card p-4" aria-label={S.settlement.q}>
      <h3 className="heading-card">{S.settlement.q}</h3>
      <IconChoicePair
        disabled={busy || store.status === "suspended"}
        items={[
          {
            icon: IconCash,
            title: W.payCash,
            selected: confirmed && store.settlement_method === "cash",
            onClick: () => pick("cash"),
          },
          {
            icon: IconWallet,
            title: W.payWallet,
            selected: confirmed && store.settlement_method === "wallet",
            onClick: () => pick("wallet"),
          },
        ]}
      />
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
    </section>
  );
}

/** **«تقبل تسترد بضاعتك؟»** — مرآةُ `setAcceptsReturns`. */
export function ReturnsCard() {
  const { store } = useStore();
  const { busy, save } = useSaveSettings();
  if (!store) return null;
  const c = store.returns_confirmed;
  const pick = (accepts: boolean) => {
    if (accepts === store.accepts_returns && c) return;
    void save({ accepts_returns: accepts }, T.returnsSaved, { accepts_returns: accepts, returns_confirmed: true });
  };
  return (
    <section className="surface flex flex-col gap-3 rounded-card p-4" aria-label={W.returnsQ}>
      <h3 className="heading-card">{W.returnsQ}</h3>
      <IconChoicePair
        disabled={busy || store.status === "suspended"}
        items={[
          { icon: IconReturn, title: W.returnsYes, selected: c && store.accepts_returns, onClick: () => pick(true) },
          { icon: IconBlock, title: W.returnsNo, selected: c && !store.accepts_returns, onClick: () => pick(false) },
        ]}
      />
    </section>
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
/** **أدنى سعرِ صرفٍ يُحفظ** — دونه كتابةٌ لم تكتمل لا سعر (مرآةُ `MIN_RATE` في التطبيق، تدقيقُ ٢٠٢٦-١٠-٢٧). */
const MIN_RATE = 10;

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
    if (rate < MIN_RATE) {
      toast(T.rateTooLow, "error");
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
          error={value != null && value > 0 && value < MIN_RATE ? T.rateTooLow : undefined}
          wrapperClassName="min-w-0 flex-1"
        />
        <Button type="submit" disabled={!dirty || busy || suspended} data-testid="store-rate-save">
          {T.rateSave}
        </Button>
      </form>
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
