"use client";

// ══════════════════════════════════════════════════════════════════════
//  **ذيلُ النافذة — موضعُ «حفظ» لا يتبدّل**
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس ٢٠٢٦-٠٨-٢٧**: عشرون ذيلاً مكتوبةً بيدٍ في سبعةَ عشرَ ملفّاً،
// **وفيها عيبان يراهما المستخدمُ ولا يراهما بناء:**
//
// **١ · زرُّ الإلغاء بصيغتين** — `ghost` في خمسةِ مواضعَ و`secondary`
// في خمسةَ عشر. **فالزرُّ نفسُه يبدو زرّاً في شاشةٍ ونصفَ زرٍّ في
// أخرى.**
//
// **٢ · وترتيبُهما ينقلب** — «إلغاء ثمّ حفظ» في ستّة، و«حفظ ثمّ
// إلغاء» في ثلاثة. **فموضعُ «حفظ» يتنقّل بين نافذةٍ وأخرى.**
//
// **ومن حفظ موضعَ زرٍّ ضغطه بلا نظر** — وهو ما يفعله من يعمل على
// اللوحة يومَه كلَّه. **فيقع على «إلغاء» وقد كتب نموذجاً كاملاً.**
//
// # ولماذا «حفظ» أوّلاً
//
// **الفعلُ الأساسُ يقع حيث تبدأ القراءة** — واللوحةُ عربيّةٌ من اليمين.
// **وهو الأغلبُ في ما كان مكتوباً** (ستّةٌ من تسعة)، فلا يتغيّر ما
// اعتاده أحد.

import * as React from "react";
import { Button } from "./components";
import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

export function FormActions({
  /** **يُنادى عند «حفظ»** — أو اتركه ومرِّر `submit` لنموذجٍ حقيقيّ. */
  onSave,
  /** **يُنادى عند «إلغاء»**. */
  onCancel,
  /** **مشغولٌ الآن؟** — يُعطَّل «حفظ» وحدَه، **فالإلغاءُ لا يُحبَس.** */
  busy = false,
  /** **زرُّ «حفظ» من نوع `submit`** — لنموذجٍ يُرسَل بالإدخال. */
  submit = false,
  /** **نصٌّ آخرُ للفعل** — «أرسل» أو «احذف» ونحوهما. */
  saveLabel,
  /** **نصٌّ آخرُ للإلغاء.** */
  cancelLabel,
  /** **لهجةُ الفعل** — `danger` لحذفٍ لا رجعةَ فيه. */
  tone,
  saveIcon,
  cancelIcon,
  equal = false,
}: {
  onSave?: () => void;
  onCancel?: () => void;
  busy?: boolean;
  submit?: boolean;
  saveLabel?: string;
  cancelLabel?: string;
  tone?: "danger";
  /** **أيقونةٌ قبل النصّ** — لشريط أدواتٍ كلُّه أيقونات (نافذةُ قصّ الصورة). */
  saveIcon?: React.ReactNode;
  cancelIcon?: React.ReactNode;
  /** **عرضان متساويان على كامل العرض** — لنافذةٍ (ملاحظةُ المالك ٢٠٢٦-١١-١١: الزرّان ملتصقان ومكوّمان في جهة). */
  equal?: boolean;
}) {
  /* **وعرضٌ متساوٍ وفاصلٌ ١٢px في كلّ موضع** — على الجوّال يملآن العرضَ، وعلى الشاشة الواسعة بحدٍّ أدنى واحد؛
     و`equal` يُبقيهما متساويَين على كامل العرض دائماً. */
  const w = equal ? "flex-1" : "flex-1 sm:min-w-28 sm:flex-none";
  return (
    <div className="flex justify-end gap-3">
      <Button
        type={submit ? "submit" : "button"}
        disabled={busy}
        onClick={submit ? undefined : onSave}
        variant={tone === "danger" ? "danger" : undefined}
        className={saveIcon ? `inline-flex ${w} items-center justify-center gap-1.5` : w}
      >
        {saveIcon}
        {saveLabel ?? m.common.save}
      </Button>
      <Button
        type="button"
        variant="secondary"
        onClick={onCancel}
        className={cancelIcon ? `inline-flex ${w} items-center justify-center gap-1.5` : w}
      >
        {cancelIcon}
        {cancelLabel ?? m.common.cancel}
      </Button>
    </div>
  );
}
