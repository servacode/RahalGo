"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إدارةُ أدوار حساب — بابٌ نادرٌ لا زرٌّ في كلّ بطاقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «زرُّ الأدوار بلا معنًى داخل الكرت — نحدّد
 *  دورَ المستخدم بداية تسجيل حسابٍ جديد فقط».)
 *
 * # ولماذا لا يفعل شيئاً في أكثر الحالات
 *
 * **دورُ الزبون ممنوحٌ تلقائيّاً لكلّ عامل** (قرارُ المالك ٢٠٢٦-٠٨-١٠):
 * السائقُ والمندوبُ وصاحبُ المتجر **يدخلون بأرقامهم إلى تطبيق الزبون
 * زبائنَ** — فلا يُفتح لهم حسابٌ ثانٍ برقمٍ ثانٍ.
 *
 * **ودورٌ ميدانيٌّ ثانٍ يرفضه المحرّك** (`ErrRoleConflict`): مندوبٌ هو
 * صاحبُ متجرٍ يمنح متجرَه عمولةَ نفسِه، **وسائقٌ هو مندوبٌ يُسند
 * لنفسه.**
 *
 * **ودورُ التاجر لا يُمنح بيدٍ أصلاً** (`ErrMerchantNeedsStore`).
 *
 * **فلا يبقى إلّا حالةٌ واحدة**: زبونٌ صار سائقاً أو مندوباً — **تقع
 * مرّةً في العمر**، فموضعُها ملفُّ الشخص لا بطاقتُه.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, Button, Chips, Input, Modal } from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";

const m = getMessages(defaultLocale);

/** **وأسماءُ الأدوار من المعجم** — لا نصَّ مكتوبٌ في شاشة. */
const ROLE_LABELS: Record<string, string> = m.terms.roleNames;
const ALL_ROLES = Object.keys(ROLE_LABELS);

/** **ورسالةُ الخادم تُترجَم بمفتاحها** — كما في بقيّة الشاشات. */
function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

export function ManageRolesModal({
  user,
  onClose,
  onChanged,
}: {
  user: AuthUser | null;
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [error, setError] = useState("");
  const [current, setCurrent] = useState<string[]>([]);
  const [pending, setPending] = useState<{ role: string; adding: boolean } | null>(null);
  const [reason, setReason] = useState("");

  useEffect(() => {
    setCurrent(user?.roles ?? []);
    setError("");
    setPending(null);
    setReason("");
  }, [user]);

  if (!user) return null;

  async function apply() {
    if (!user || !pending) return;
    setError("");
    try {
      if (!pending.adding) {
        await api(
          `/api/v1/admin/users/${user.id}/roles/${pending.role}?reason=${encodeURIComponent(reason)}`,
          { method: "DELETE" }
        );
        setCurrent((p) => p.filter((r) => r !== pending.role));
      } else {
        await api(`/api/v1/admin/users/${user.id}/roles`, {
          method: "POST",
          body: JSON.stringify({ role: pending.role, reason }),
        });
        setCurrent((p) => [...p, pending.role]);
      }
      setPending(null);
      setReason("");
      await onChanged();
    } catch (err) {
      setError(errText(err));
    }
  }
  function toggle(role: string) {
    setPending({ role, adding: !current.includes(role) });
    setReason("");
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.users.rolesFor}: ${user.full_name || user.phone}`}>
      <Chips
        items={ALL_ROLES.map((r) => ({ id: r, label: ROLE_LABELS[r] ?? r }))}
        value={current}
        onChange={toggle}
      />
      {pending && (
        <div className="mt-4 rounded-control border border-primary-edge bg-primary-tint p-3">
          <p className="mb-2 text-sm font-medium">
            {m.admin.users.roleReasonTitle}: {ROLE_LABELS[pending.role]}
          </p>
          <Input
            id="role-reason"
            label={m.admin.users.roleReasonLabel}
            required
            autoFocus
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <div className="mt-3 flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setPending(null)}>
              {m.common.cancel}
            </Button>
            <Button disabled={!reason.trim()} onClick={apply}>
              {m.common.confirm}
            </Button>
          </div>
        </div>
      )}
      {error && (
        <Alert className="mt-3">{error}</Alert>
      )}
      <div className="mt-5 flex justify-end">
        <Button variant="secondary" onClick={onClose}>
          {m.common.back}
        </Button>
      </div>
    </Modal>
  );
}
