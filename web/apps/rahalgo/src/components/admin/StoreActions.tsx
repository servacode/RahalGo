"use client";

/**
 * **أفعالُ المتجر — حيثما عُرض المتجر.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «خياراتُ المتجر والأزرار أيضاً يجب أن تنتقل
 *  لنحذف تبويبَ المتاجر لاحقاً».)
 *
 * # لماذا مكوّنٌ واحدٌ لا نسختان
 *
 * **الشرطُ قبل حذف أيّ قسم**: أن يستوعب الملفُّ **كلَّ فعلٍ** كان فيه —
 * «ولا نريد خسارةَ أيّ ميزة» (قرارُ المالك ٢٠٢٦-٠٨-٠٣). **وأربعةُ أقسامٍ
 * حُذفت على هذا الشرط** قبل هذا.
 *
 * **ونسختان من الأزرار تفترقان يوماً**: يُضاف فعلٌ في الشاشة ويُنسى في
 * الملفّ، **فيُحذف القسمُ وقد ضاع الفعل** — ولا يُكتشف إلّا حين يُطلب.
 *
 * # وما ليس هنا
 *
 * **«ملفّ المتجر» شاشةٌ كاملة** (الأصنافُ والأقسامُ والصور) لا نافذة —
 * **فيبقى انتقالاً**، وهو ما تفعله الشاشةُ نفسُها.
 *
 * # وإيقافُ المتجر غيرُ إيقاف صاحبه
 *
 * **في الملفّ زرّا إيقافٍ وحظرٍ للحساب أعلى الصفحة** — **وزرّان متشابهان
 * لمعنيين يُخلطان.** فهذه في سطر المتجر نفسِه، **وباسمٍ يقول «المتجر».**
 */

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText} from "@rahalgo/i18n";
import { Button, Alert, IconEdit, IconDate, IconOrder as IconMenu } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { HoursModal } from "@/components/admin/HoursModal";
import ViolationsModal from "@/components/admin/ViolationsModal";
import { MerchantModal, type Merchant } from "@/components/admin/MerchantModal";

const m = getMessages(defaultLocale);
const M = m.admin.merchants;

/** **والصفُّ كاملاً** — **ونافذةُ التعديل تحتاجه كلَّه**، فلا يُختصر. */
export type StoreTarget = Merchant;


export function StoreActions({
  store,
  onChanged,
}: {
  store: StoreTarget;
  /** **ويُعاد الجلبُ بعد كلّ فعل** — وإلّا بقي السطرُ يقول ما بطل. */
  onChanged: () => void;
}) {
  const router = useRouter();
  const [hours, setHours] = useState(false);
  const [edit, setEdit] = useState(false);
  const [violations, setViolations] = useState(false);
  const [error, setError] = useState("");

  async function call(path: string, init: RequestInit) {
    try {
      await api(`/api/v1/admin/merchants/${store.id}${path}`, init);
      setError("");
      onChanged();
    } catch (err) {
      setError(errorText(err));
    }
  }

  // **و`suspended` غيرُ `inactive`**: الثانيةُ يملكها المتجر — إجازةٌ أو
  // ترميم — **والأولى حكمُ المنصّة عليه.** وخلطُهما يجعل رفعَ الحظر
  // يُقرأ فتحاً، **ويفتح متجراً حظرناه.**
  const banned = store.status === "suspended";

  return (
    <>
      <div className="flex flex-wrap items-center gap-1.5">
        {/* **والملفُّ انتقالٌ لا نافذة** — شاشةٌ كاملةٌ فيها أصنافُه. */}
        <Button
          variant="secondary"
          onClick={() => router.push(`/dashboard/merchants/${store.id}`)}
          className="flex items-center gap-1.5 !px-2.5"
        >
          <IconMenu size={14} />
          {M.openProfile}
        </Button>
        <Button
          variant="ghost"
          onClick={() => setHours(true)}
          className="flex items-center gap-1.5 !px-2.5"
        >
          <IconDate size={14} />
          {m.admin.hours.manageHours}
        </Button>
        <Button
          variant="ghost"
          onClick={() => setEdit(true)}
          className="flex items-center gap-1.5 !px-2.5"
        >
          <IconEdit size={14} />
          {M.edit}
        </Button>
        <Button variant="ghost" onClick={() => setViolations(true)} className="!px-2.5">
          {M.violationsLog.viewLog}
        </Button>
        {/* **والعفوُ يُعرض إن كان عمّا يُعفى** — زرٌّ لا يفعل شيئاً
            يزاحم ما يفعل. */}
        {store.violations > 0 && (
          <Button
            variant="ghost"
            className="!px-2.5"
            onClick={() => void call("/clear-violations", { method: "POST" })}
          >
            {M.forgive}
          </Button>
        )}
        <Button
          variant={store.status === "active" ? "secondary" : "primary"}
          className="!px-2.5"
          onClick={() =>
            void call("", {
              method: "PATCH",
              body: JSON.stringify({
                status: store.status === "active" ? "inactive" : "active",
              }),
            })
          }
        >
          {store.status === "active" ? M.deactivate : M.activate}
        </Button>
        <Button
          variant={banned ? "secondary" : "danger"}
          className="!px-2.5"
          onClick={() =>
            void call("/suspend", {
              method: "POST",
              body: JSON.stringify({ suspended: !banned, note: "" }),
            })
          }
        >
          {banned ? M.unban : M.ban}
        </Button>
      </div>
      {error && <Alert>{error}</Alert>}

      {hours && (
        <HoursModal
          merchant={store}
          onClose={() => setHours(false)}
          onChanged={onChanged}
        />
      )}
      {violations && (
        <ViolationsModal
          merchant={store}
          onClose={() => setViolations(false)}
          onChanged={onChanged}
        />
      )}
      {edit && (
        <MerchantModal
          merchant={store}
          onClose={() => setEdit(false)}
          onSaved={() => {
            setEdit(false);
            onChanged();
          }}
        />
      )}
    </>
  );
}
