"use client";

/**
 * **الكلمةُ المؤقّتةُ على شاشة الموظّف** — حين لم تصل رسالةُ الواتساب.
 *
 * **والخادمُ لا يُرسلها إلّا في التجهيز** (`revealTemp`): الإنتاجُ يرسلها بالواتساب
 * وحدَه ولا يُظهرها أبداً. (قرارُ المالك ٢٠٢٦-١٠-٠٥.)
 */
import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "@rahalgo/ui";

const A = getMessages(defaultLocale).admin.acc;

export function TempPasswordNote({ value }: { value?: string | null }) {
  const [copied, setCopied] = useState(false);
  if (!value) return null;
  return (
    <div className="space-y-2 rounded-card border border-line bg-surface-soft p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-sm text-ink-soft">{A.tempPasswordLabel}</span>
        <code dir="ltr" className="select-all rounded-control bg-surface px-2 py-1 font-mono text-base">
          {value}
        </code>
        <Button
          variant="secondary"
          onClick={() => {
            void navigator.clipboard?.writeText(value).then(() => setCopied(true), () => setCopied(false));
          }}
        >
          {copied ? A.tempPasswordCopied : A.tempPasswordCopy}
        </Button>
      </div>
      <p className="text-xs text-ink-soft">{A.tempPasswordHint}</p>
    </div>
  );
}
