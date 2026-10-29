"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/menu/prices — «تغيير الأسعار»** (تصميمُ ٢٠٢٦-١٠-٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `BulkPriceEditor` و`BulkPriceState` في `ui/menu/MenuTools.kt`.)
 *
 * مربّعان «زيادة / تخفيض»، ثمّ «نسبة ٪ / مبلغ ثابت ل.س»، ثمّ القيمة، ثمّ
 * «على:»، ثمّ «تطبيق». **ولا خطوةَ معاينة**: يُسأل المحرّكُ بلا كتابةٍ عن
 * العدد (`confirm: false`)، **وتطلع نافذةُ تأكيدٍ واحدة** «سيتغيّر سعر N
 * صنفاً»، ثمّ يُطبَّق بالقيمة التي عُدَّ عليها لا بما كُتب بعدها.
 *
 * **والمبلغُ الثابتُ `amount`** بالليرة (`catalog.BulkPriceInput`) — ولا
 * يُرسَل معه `percent`.
 */

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Confirm, FormActions, IconAdd, IconCheck, IconClose, IconMinus, Radio } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { SectionRef } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";
import { ChoiceBox, Segmented, ToolHeader } from "@/components/store/ToolParts";

const m = getMessages(defaultLocale);
const B = m.store.bulk;
const BACK = "/store/menu";

interface Body {
  percent?: number;
  amount?: number;
  platform_section_id?: string;
}

export default function BulkPricePage() {
  const { store } = useStore();
  const router = useRouter();
  const toast = useShopToast();
  const id = store?.id ?? "";
  const [sections, setSections] = useState<SectionRef[]>([]);
  const [up, setUp] = useState(true);
  const [byAmount, setByAmount] = useState(false);
  const [value, setValue] = useState("");
  const [section, setSection] = useState("");
  const [asking, setAsking] = useState<{ body: Body; count: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  /** **رقمُ الطلب** — عددٌ قديمٌ يصل بعد تبديل القيمة يُرمى. */
  const seq = useRef(0);

  useEffect(() => {
    if (!id) return;
    api<{ sections: SectionRef[] }>(`/api/v1/merchant/stores/${id}/sections`)
      .then((r) => setSections(r.sections ?? []))
      .catch(() => undefined);
  }, [id]);

  const changed = () => {
    seq.current++;
    setAsking(null);
    setError("");
  };

  function input(): Body | null {
    const n = Number(value.replace(/\D/g, ""));
    const scope = section ? { platform_section_id: section } : {};
    if (byAmount) {
      if (!value || !(n > 0) || n > 10_000_000) {
        setError(B.badAmount);
        return null;
      }
      return { amount: up ? n : -n, ...scope };
    }
    if (!value || !(n > 0) || (up && n > 100) || (!up && n > 50)) {
      setError(B.badPercent);
      return null;
    }
    return { percent: up ? n : -n, ...scope };
  }

  const run = (body: Body, confirm: boolean) =>
    api<{ count: number }>(`/api/v1/merchant/stores/${id}/menu/prices`, {
      method: "POST",
      body: JSON.stringify({ ...body, confirm }),
    });

  /** **«تطبيق»** — يعدّ بلا كتابة، ثمّ يفتح نافذةَ التأكيد. */
  async function submit() {
    if (busy || !id) return;
    const body = input();
    if (!body) return;
    const mine = seq.current;
    setBusy(true);
    try {
      const r = await run(body, false);
      if (mine !== seq.current) return;
      if (r.count === 0) setError(B.none);
      else setAsking({ body, count: r.count });
    } catch (e) {
      if (mine === seq.current) setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function apply() {
    if (!asking || busy) return;
    const body = asking.body;
    setAsking(null);
    setBusy(true);
    try {
      const r = await run(body, true);
      toast(fill(B.done, { n: fmtNum(r.count) }), "success");
      router.push(BACK);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!store) return null;

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-4" data-testid="bulk-price">
      <ToolHeader title={B.title} href={BACK} />

      <div role="radiogroup" aria-label={B.title} className="flex gap-2">
        <ChoiceBox icon={IconAdd} label={B.up} tone="success" on={up} disabled={busy} onPick={() => { setUp(true); changed(); }} testId="bulk-up" />
        <ChoiceBox icon={IconMinus} label={B.down} tone="danger" on={!up} disabled={busy} onPick={() => { setUp(false); changed(); }} testId="bulk-down" />
      </div>

      <Segmented
        label={B.title}
        options={[
          ["percent", B.modePercent],
          ["amount", B.modeAmount],
        ]}
        value={byAmount ? "amount" : "percent"}
        disabled={busy}
        onChange={(v) => {
          setByAmount(v === "amount");
          setValue("");
          changed();
        }}
      />

      <label className="flex flex-col gap-1">
        <span className="text-sm font-medium">{byAmount ? B.amount : B.percent}</span>
        <span className="flex h-12 items-center gap-2 rounded-control border border-line bg-field px-3 focus-within:border-accent">
          <input
            value={value}
            onChange={(e) => {
              setValue(e.target.value.replace(/\D/g, "").slice(0, byAmount ? 8 : 3));
              changed();
            }}
            inputMode="numeric"
            dir="ltr"
            disabled={busy}
            data-testid="bulk-value"
            className="min-w-0 flex-1 bg-transparent text-base text-ink outline-none"
          />
          <span className="shrink-0 font-bold text-ink-muted">{byAmount ? B.unitSyp : B.unitPct}</span>
        </span>
      </label>

      <fieldset className="flex flex-col gap-1">
        <legend className="mb-1 text-sm font-bold">{B.scope}</legend>
        <div className="surface flex flex-col gap-1 rounded-card border border-line p-2">
          <Radio
            id="bulk-sec-all"
            name="bulk-section"
            className="min-h-11 px-1"
            checked={section === ""}
            disabled={busy}
            onChange={() => { setSection(""); changed(); }}
            label={B.all}
          />
          {sections.map((s) => (
            <Radio
              key={s.id}
              id={`bulk-sec-${s.id}`}
              name="bulk-section"
              className="min-h-11 px-1"
              checked={section === s.id}
              disabled={busy}
              onChange={() => { setSection(s.id); changed(); }}
              label={s.name}
            />
          ))}
        </div>
      </fieldset>

      {error && <Alert>{error}</Alert>}
      <FormActions
        onSave={() => void submit()}
        onCancel={() => router.push(BACK)}
        busy={busy || !value}
        saveLabel={B.go}
        saveIcon={<IconCheck size={18} />}
        cancelIcon={<IconClose size={18} />}
      />

      <Confirm
        open={asking != null}
        title={B.title}
        body={asking ? fill(B.askCount, { n: fmtNum(asking.count) }) : ""}
        confirmLabel={m.common.confirm}
        tone="primary"
        busy={busy}
        onConfirm={() => void apply()}
        onCancel={() => setAsking(null)}
      />
    </div>
  );
}
