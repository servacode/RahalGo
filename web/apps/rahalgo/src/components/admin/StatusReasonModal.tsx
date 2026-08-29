"use client";

/**
 * **نافذةُ سبب الإيقاف والحظر — واحدةٌ لكلّ من يوقف حساباً.**
 *
 * (شهده المالك ٢٠٢٦-٠٨-١٠: «زرُّ إيقاف حساب لا يعمل… وزرُّ الحظر لا يعمل.
 *  طبعاً الحظرُ والإيقافُ لا تعملان بالكرت الخارجيّ، أمّا داخل التفاصيل
 *  فتعمل بشكلٍ ممتاز».)
 *
 * # ولماذا اجتمعت في ملفٍّ واحد
 *
 * **كانت ثلاثَ حالات**: نسخةٌ في صفحة تفاصيل الحساب، ونسخةٌ ثانيةٌ باسمٍ
 * آخرَ في جدول المندوبين، **ولا شيءَ في جدول الحسابات** — الزرُّ يُضغط
 * ويُخزَّن الاختيارُ في حالةٍ **لا يقرؤها أحد.**
 *
 * **ولا خطأ ولا سطرٌ في سجلّ**: يضغط الموظّفُ فلا يقع شيء، **فيضغط ثانيةً
 * وثالثة**، ثمّ يظنّ الحسابَ محظوراً وهو يعمل.
 *
 * **وهو ما يقع حين يُنسخ المكوّن**: تُصلَح نسخةٌ وتبقى الأخرى، **ويُنسى
 * الثالث فلا يُكتب أصلاً.**
 *
 * # والسببُ إلزاميّ
 *
 * **حسابٌ يُوقَف بلا سببٍ مكتوبٍ لا يُراجَع**: صاحبُه يسأل «لماذا؟» ولا
 * جوابَ عند من أوقفه بعد شهر. **ويُعرض للموقوف نفسِه** في شاشته — فيعرف ما
 * يُصلح.
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, Modal, FormActions} from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const U = m.admin.users;

export function StatusReasonModal({
  status,
  onSubmit,
  onClose,
}: {
  /** `suspended` إيقافٌ مؤقّت · `blocked` حظرٌ نهائيّ. */
  status: string;
  onSubmit: (reason: string) => void;
  onClose: () => void;
}) {
  const [reason, setReason] = useState("");
  const label = status === "suspended" ? U.suspend : U.block;

  return (
    <Modal open onClose={onClose} title={U.statusReasonTitle.replace("{action}", label)}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit(reason);
        }}
        className="space-y-4"
      >
        <Input
          id="status-reason"
          label={U.statusReasonLabel}
          required
          autoFocus
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {/* **والحظرُ أحمرُ والإيقافُ ليس كذلك** — فعلان في نافذةٍ واحدةٍ
            ومعناهما مختلف، **ولونٌ واحدٌ لهما يجعل الضغطةَ قرعة.** */}
        <FormActions
          submit
          onCancel={onClose}
          saveLabel={label}
          tone={status === "blocked" ? "danger" : undefined}
        />
      </form>
    </Modal>
  );
}

export default StatusReasonModal;
