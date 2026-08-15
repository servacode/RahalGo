"use client";

/**
 * **رمزٌ يُنسخ بضغطة.**
 *
 * (نُقل من شاشة المندوبين حين حُذفت ٢٠٢٦-٠٨-١٥ — **وكان مكتوباً فيها
 *  وحدَها**، فلمّا نزل رمزُ الدعوة إلى جدول الحسابات كان سيُنسخ معه.)
 *
 * **ورمزٌ يُقرأ من شاشةٍ ويُكتب بيدٍ يُخطئ فيه حرف** — ويُملى على
 * مندوبٍ في الهاتف، **فالنسخُ بضغطةٍ ليس ترفاً.**
 *
 * **وجوابُ الضغطة في الحبّة نفسِها** — علامةُ صحٍّ ثانيةً ونصفاً:
 * **من ضغط ولم يقع شيءٌ يعيد الضغط.**
 *
 * **و`dir="ltr"` لأنّه رمزٌ لاتينيّ** — **وفي صفحةٍ عربيّةٍ تُقلب
 * حروفُه فيُنسخ صحيحاً ويُقرأ مقلوبا.**
 */

import { useState } from "react";
import { Badge } from "./components";
import { IconCheck, IconCopy } from "./icons";

export function CopyCode({ code, title }: { code: string; title?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={(e) => {
        // **ولا يفتح الصفَّ الذي هو فيه** — البطاقةُ كلُّها تُضغط لتفتح
        // الملفّ، **ومن أراد النسخَ لا يريد الانتقال.**
        e.stopPropagation();
        void navigator.clipboard.writeText(code);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }}
      dir="ltr"
      title={title}
    >
      <Badge variant="accent" className="gap-1.5 font-mono font-bold">
        {code}
        <span className="font-normal">
          {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
        </span>
      </Badge>
    </button>
  );
}
