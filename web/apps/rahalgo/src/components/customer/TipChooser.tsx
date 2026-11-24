"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إكرامية الكابتن — اختيارٌ واحدٌ يُستعمل بثلاثة أماكن** (السلّة · تأكيد السعر · تبديل الدفع)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **اختياريّةٌ وافتراضُها «لا شكراً»**، وبتنحجز من المحفظة لا من الطلب. **ولا تظهر** إلا
 * والمنصّةُ مفعّلتها (`tips.enabled`) والدفعُ من المحفظة — **والمتحكّم بالشرط الأبُ** عبر
 * `tipOffered`، والمحرّك يحرس القيمة (`tip_invalid` · `tip_not_allowed`).
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";
import { Button, Input } from "@rahalgo/ui";
import { useCustomer } from "@/lib/customer/context";
import type { CustomerPlatform } from "@/lib/customer/types";

const T = getMessages(defaultLocale).shop.tip;

/** **أتُعرض الإكرامية؟** — المنصّةُ مفعّلتها ومحفظةٌ (كاملةً أو توصيلاً) ولا واتساب. */
export function tipOffered(platform: CustomerPlatform | null, scope: "none" | "full" | "delivery"): boolean {
  return platform?.tips?.enabled === true && scope !== "none";
}

export function TipChooser({
  value,
  onChange,
  disabled,
}: {
  value: number;
  onChange: (n: number) => void;
  disabled?: boolean;
}) {
  const { platform } = useCustomer();
  const tips = platform?.tips;
  const presets = tips?.presets ?? [];
  const max = tips?.max ?? 0;
  const [custom, setCustom] = useState(() => value > 0 && !presets.includes(value));
  const [text, setText] = useState(() => (value > 0 && !presets.includes(value) ? String(value) : ""));
  if (!tips?.enabled) return null;

  const pick = (n: number) => {
    setCustom(false);
    setText("");
    onChange(n);
  };
  const typed = (raw: string) => {
    const digits = raw.replace(/\D/g, "").slice(0, 7);
    setText(digits);
    const n = Number(digits);
    onChange(Number.isFinite(n) && n >= 1 && n <= max ? n : 0);
  };
  const over = custom && text !== "" && Number(text) > max;

  return (
    <div className="flex flex-col gap-rg-sm" role="group" aria-label={T.title}>
      <p className="text-card-title">{T.title}</p>
      <div className="flex flex-wrap gap-2">
        {presets.map((p) => (
          <Button
            key={p}
            type="button"
            size="sm"
            variant={!custom && value === p ? "primary" : "secondary"}
            aria-pressed={!custom && value === p}
            disabled={disabled}
            onClick={() => pick(p)}
          >
            {fmtMoney(p)}
          </Button>
        ))}
        <Button
          type="button"
          size="sm"
          variant={custom ? "primary" : "secondary"}
          aria-pressed={custom}
          disabled={disabled}
          onClick={() => setCustom(true)}
        >
          {T.other}
        </Button>
        <Button
          type="button"
          size="sm"
          variant={!custom && value === 0 ? "primary" : "secondary"}
          aria-pressed={!custom && value === 0}
          disabled={disabled}
          onClick={() => pick(0)}
        >
          {T.none}
        </Button>
      </div>
      {custom && (
        <Input
          id="tip-custom"
          label={fillMax(T.customLabel, max)}
          inputMode="numeric"
          value={text}
          disabled={disabled}
          onChange={(e) => typed(e.target.value)}
          error={over ? fillMax(T.tooMuch, max) : undefined}
        />
      )}
      <p className="text-caption text-ink-muted">{T.hint}</p>
    </div>
  );
}

const fillMax = (s: string, max: number) => s.replace("{max}", fmtMoney(max));
