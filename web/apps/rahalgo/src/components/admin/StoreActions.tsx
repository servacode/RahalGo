"use client";

/**
 * **أفعالُ المتجر — حيثما عُرض المتجر.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «خياراتُ المتجر والأزرار أيضاً يجب أن تنتقل
 *  لنحذف تبويبَ المتاجر لاحقاً».)
 *
 * # ومكوّنٌ واحدٌ لا نسختان
 *
 * **ونسختان من الأزرار تفترقان يوماً**: يُضاف فعلٌ في الشاشة ويُنسى في الملفّ.
 *
 * # زرّان ظاهران والباقي في «⋯» (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 *
 * **كانت سبعةَ أزرارٍ متلاصقةٍ بألوانٍ مختلفة تنكسر على الجوّال، بلا فحصِ صلاحيّة**:
 * «الملفّ» و«تعديل» يُرَيان، والباقي في قائمةٍ **كلُّ بندٍ فيها بقدرة بابه**
 * (`useCanCall`)، **والخطرُ بالأحمر في آخرها.**
 *
 * # والحظرُ بابٌ واحدٌ بسببٍ وتأكيدٍ وإشعار
 *
 * **كان «فتح» على متجرٍ محظورٍ يرفع الحظرَ من الباب الخلفيّ** — بلا سببٍ وبصلاحية
 * «إدارة المتاجر» لا «السلامة». **فـ«فتح/إغلاق» لا يُعرض على محظور** (والمحرّكُ
 * يردّه: `merchant_ban_locked`)، **والحظرُ ورفعُه بسببٍ يصل صاحبَه.**
 */

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Button,
  Alert,
  Confirm,
  Modal,
  Textarea,
  FormActions,
  ActionMenu,
  type ActionMenuItem,
  IconEdit,
  IconDate,
  IconOrder as IconMenu,
  IconWarning,
  IconBlock,
  IconUnblock,
  IconCheck,
  IconStore,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { HoursModal } from "@/components/admin/HoursModal";
import ViolationsModal from "@/components/admin/ViolationsModal";
import { MerchantModal, type Merchant } from "@/components/admin/MerchantModal";

const m = getMessages(defaultLocale);
const M = m.admin.merchants;
const A = m.admin.acc;

/** **والصفُّ كاملاً** — **ونافذةُ التعديل تحتاجه كلَّه**، فلا يُختصر. */
export type StoreTarget = Merchant;

/** **لونُ حال المتجر** — فعّالٌ أخضر · مغلقٌ رماديّ · محظورٌ أحمر (قرارُ المالك ٢٠٢٦-١٠-٠٤). */
export function storeStatusVariant(status: string): "success" | "neutral" | "danger" {
  return status === "active" ? "success" : status === "suspended" ? "danger" : "neutral";
}

export function StoreActions({
  store,
  onChanged,
}: {
  store: StoreTarget;
  /** **ويُعاد الجلبُ بعد كلّ فعل** — وإلّا بقي السطرُ يقول ما بطل. */
  onChanged: () => void;
}) {
  const router = useRouter();
  const canCall = useCanCall();
  const [hours, setHours] = useState(false);
  const [edit, setEdit] = useState(false);
  const [violations, setViolations] = useState(false);
  const [confirm, setConfirm] = useState<"open" | "close" | "forgive" | null>(null);
  const [ban, setBan] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function call(path: string, init: RequestInit): Promise<boolean> {
    setBusy(true);
    try {
      await api(`/api/v1/admin/merchants/${store.id}${path}`, init);
      setError("");
      onChanged();
      return true;
    } catch (err) {
      setError(errorText(err));
      return false;
    } finally {
      setBusy(false);
    }
  }

  // **و`suspended` غيرُ `inactive`**: الثانيةُ يملكها المتجر — إجازةٌ أو ترميم —
  // **والأولى حكمُ المنصّة عليه.**
  const banned = store.status === "suspended";
  const canEdit = canCall("PATCH", "/merchants/{id}");
  const canSafety = canCall("POST", "/merchants/{id}/suspend");

  const items: ActionMenuItem[] = [];
  if (canCall("PUT", "/merchants/{id}/hours")) {
    items.push({ key: "hours", label: m.admin.hours.manageHours, icon: IconDate, onSelect: () => setHours(true) });
  }
  if (canCall("GET", "/merchants/{id}/violations")) {
    items.push({ key: "violations", label: M.violationsLog.viewLog, icon: IconWarning, onSelect: () => setViolations(true) });
  }
  if (store.violations > 0 && canCall("POST", "/merchants/{id}/clear-violations")) {
    items.push({ key: "forgive", label: M.forgive, icon: IconCheck, onSelect: () => setConfirm("forgive") });
  }
  if (canEdit && !banned) {
    items.push(
      store.status === "active"
        ? { key: "close", label: A.storeClose, icon: IconStore, onSelect: () => setConfirm("close") }
        : { key: "open", label: A.storeOpen, icon: IconStore, onSelect: () => setConfirm("open") },
    );
  }
  if (canSafety) {
    items.push({
      key: "ban",
      label: banned ? M.unban : M.ban,
      icon: banned ? IconUnblock : IconBlock,
      danger: !banned,
      onSelect: () => {
        setReason("");
        setBan(true);
      },
    });
  }

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
        {canEdit && (
          <Button variant="ghost" onClick={() => setEdit(true)} className="flex items-center gap-1.5 !px-2.5">
            <IconEdit size={14} />
            {M.edit}
          </Button>
        )}
        <ActionMenu label={A.more} variant="ghost" items={items} ariaLabel={A.more} />
      </div>
      {error && <Alert className="mt-1">{error}</Alert>}

      <Confirm
        open={confirm !== null}
        tone={confirm === "close" ? "danger" : "primary"}
        title={confirm === "close" ? A.storeCloseTitle : confirm === "open" ? A.storeOpenTitle : A.forgiveTitle}
        body={confirm === "close" ? A.storeCloseBody : confirm === "open" ? A.storeOpenBody : A.forgiveBody}
        confirmLabel={confirm === "close" ? A.storeClose : confirm === "open" ? A.storeOpen : M.forgive}
        busy={busy}
        onCancel={() => setConfirm(null)}
        onConfirm={() => {
          const which = confirm;
          setConfirm(null);
          if (which === "forgive") void call("/clear-violations", { method: "POST" });
          else
            void call("", {
              method: "PATCH",
              body: JSON.stringify({ status: which === "close" ? "inactive" : "active" }),
            });
        }}
      />

      {ban && (
        <Modal open onClose={() => setBan(false)} title={banned ? A.storeUnbanTitle : A.storeBanTitle}>
          <form
            className="space-y-3"
            onSubmit={async (e) => {
              e.preventDefault();
              const ok = await call("/suspend", {
                method: "POST",
                body: JSON.stringify({ suspended: !banned, note: reason.trim() }),
              });
              if (ok) setBan(false);
            }}
          >
            <p className="text-sm text-ink-muted">
              <span className="font-bold text-ink">{store.name}</span> — {banned ? A.storeUnbanHint : A.storeBanHint}
            </p>
            <Textarea
              id="store-ban-reason"
              label={A.reason}
              required
              rows={3}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            {error && <Alert>{error}</Alert>}
            <FormActions
              submit
              onCancel={() => setBan(false)}
              busy={busy}
              saveLabel={banned ? M.unban : M.ban}
            />
          </form>
        </Modal>
      )}

      {hours && <HoursModal merchant={store} onClose={() => setHours(false)} onChanged={onChanged} />}
      {violations && (
        <ViolationsModal merchant={store} onClose={() => setViolations(false)} onChanged={onChanged} />
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
