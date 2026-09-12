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
import { getMessages, defaultLocale, errorText} from "@rahalgo/i18n";
import { Alert, Button, Chips, Input, Modal, FormActions} from "@rahalgo/ui";
import { api, type AuthUser } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { listRoles, roleLabel, type Role } from "@/lib/rbac";
import { assignmentGroups } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);

/**
 * **والأدوارُ تُقرأ من المحرّك** — دورةُ ٧٠ب-و١.
 *
 * **وكانت `Object.keys(ROLE_LABELS)`** — **معجمَ نصوصٍ في الواجهة**:
 * **سبعةٌ تُعرَض والقاعدةُ فيها أحدَ عشر**، **ودورٌ يُنشَأ اليومَ لا
 * يظهر حتّى تُبنى الواجهةُ من جديد.** **وحقيقةٌ ثانيةٌ في العميل
 * تفترق عن الأولى يوماً**، وقد افترقت.
 *
 * **والمعجمُ باقٍ معيناً للعرض** — انظر `roleLabel`.
 */

/** **ورسالةُ الخادم تُترجَم بمفتاحها** — كما في بقيّة الشاشات. */
function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
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
  const [roles, setRoles] = useState<Role[]>([]);
  // **وأدوارُ المشغّل تُقرَّر بها الأهليّةُ المعروضة** — **فنقرةٌ
  // يردُّها المحرّكُ لا تُعرَض** (بندُ ط): `admin` للمالك وحدَه.
  const { user: me } = useAuth();
  const actorRoles = me?.roles ?? [];

  useEffect(() => {
    setCurrent(user?.roles ?? []);
    setError("");
    setPending(null);
    setReason("");
  }, [user]);

  // **وتُقرأ الأدوارُ عند فتح النافذة** — **فدورٌ أُنشئ قبل لحظةٍ
  // يظهر بلا إعادةِ تحميلِ الصفحة.**
  useEffect(() => {
    if (!user) return;
    let alive = true;
    void listRoles()
      .then((r) => {
        if (alive) setRoles(r);
      })
      .catch(() => {
        // **وتعذّرُ القراءة لا يُغلق الشاشة** — يبقى ما يملكه الحساب
        // مرئيّاً، **ولا تُخترَع قائمةٌ من الواجهة بديلاً.**
        if (alive) setRoles([]);
      });
    return () => {
      alive = false;
    };
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
      setError(errorText(err));
    }
  }
  function toggle(role: string) {
    // **والمحميُّ لا يُبدَّل من هنا حتّى لو نُقر** — **و`disabled` في
    // الحبّة لطفٌ بالعين، وهذا هو المنعُ في المنطق.** (والمنعُ الحقيقيُّ
    // في المحرّك: `roles.manage` وتأكيدٌ وقيدُ تدقيق.)
    const locked = assignmentGroups(roles, current, actorRoles).some((g) =>
      g.roles.some((r) => r.code === role && r.locked),
    );
    if (locked) return;
    setPending({ role, adding: !current.includes(role) });
    setReason("");
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.users.rolesFor}: ${user.full_name || user.phone}`}>
      {/* ══════════════════════════════════════════════════════════
          **مجموعاتٌ تلتفّ — لا صفٌّ واحدٌ ينزلق**

          **وهذا هو الإصلاح**: كان `Chips` واحداً بلا `wrap`، **فثمانيةَ
          عشرَ دوراً في ٣٩٨ بكسل ⇒ ثلاثٌ مرئيّةٌ وخمسةَ عشرَ خارجَ
          الإطار** — **وشريطُ التمرير مخفيٌّ بالأنماط**، فلا شيءَ يدلّ
          على الغائب. **و«مراقبة التشغيل» كانت على `-820px`.**

          **والوجودُ يبقى من المحرّك** — `roles` هي جوابُه، **والسياسةُ
          ترتّب ما جاء ولا تخترع.** ══════════════════════════════════ */}
      {assignmentGroups(roles, current, actorRoles).map((g) => (
        <div key={g.cls} className="mb-4">
          <p className="mb-1 text-sm font-bold">{g.title}</p>
          <p className="mb-2 text-xs text-ink-muted">{g.note}</p>
          <Chips
            wrap
            items={g.roles.map((r) => ({
              id: r.code,
              disabled: r.locked,
              /* **والرمزُ التقنيُّ ظاهرٌ ثانويّاً** — **و`ops` و
                 `operations` اسمُهما العربيُّ واحدٌ «العمليات»**،
                 فبلا الرمز لا يعرف الموظّفُ أيَّهما يُسند. (قِيس في
                 النافذة: حبّتان بنصٍّ واحد.) */
              label: (
                <span className="flex items-center gap-1.5">
                  {r.label}
                  <span className="font-mono text-[10px] opacity-60">{r.code}</span>
                </span>
              ),
            }))}
            value={current}
            onChange={toggle}
          />
        </div>
      ))}
      {/* **وتعذّرُ القراءة يُقال بنصّه** — **و«حدث خطأ غير متوقع» هي
          الرسالةُ التي أضلّت المالكَ في عطب التأكيد**، فلا تُعاد هنا. */}
      {roles.length === 0 && <Alert className="mt-3">{m.admin.users.rolesUnavailable}</Alert>}
      {pending && (
        <div className="mt-4 rounded-control border border-primary-edge bg-primary-tint p-3">
          <p className="mb-2 text-sm font-medium">
            {m.admin.users.roleReasonTitle}:{" "}
            {roleLabel(
              roles.find((r) => r.code === pending.role) ?? {
                code: pending.role,
                name_key: "",
              },
            )}
          </p>
          <Input
            id="role-reason"
            label={m.admin.users.roleReasonLabel}
            required
            autoFocus
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <FormActions onSave={apply} onCancel={() => setPending(null)} saveLabel={m.common.confirm} />
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
