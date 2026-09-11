"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأدوارُ والصلاحيّات — ما كان يُدار بيدٍ في القاعدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (دورةُ ٧٠ب-و١.)
 *
 * # لماذا وُجدت
 *
 * **وعقدُ المالك**: `NO-CODE FOR OPERATIONS · CODE FOR NEW CAPABILITIES`
 * — **معجمُ القدرات في الشيفرة، ودورٌ ← قدرة في القاعدة من اللوحة.**
 *
 * **ولم يكن للنصف الثاني شاشة.** **فقدرةٌ تُضاف في الشيفرة لا تجد
 * دوراً ضيّقاً يحملها**: إمّا تُمنَح لدورٍ يملك خمساً وعشرين قدرةً
 * أصلاً، وإمّا يُكتب صفٌّ بيدٍ في القاعدة. **وكلاهما نقضُ العقد.**
 *
 * # والدورُ وعاءٌ والصلاحيّةُ ما يملؤه
 *
 * **ودورٌ يُنشَأ اليومَ لا يملك شيئاً** — وهو الافتراضُ في `ADG-1`.
 * **ثمّ تُمنَح قدراتُه واحدةً واحدةً**، **كلُّ واحدةٍ بفعلٍ مؤكَّدٍ
 * يُقيَّد في السجلّ** — فيُقرأ ما مُنح ومتى ولمن.
 *
 * # ولا حقيقةَ ثانيةً في الواجهة
 *
 * **وكلُّ ما يُعرَض هنا مقروءٌ من المحرّك** (`lib/rbac.ts`) — **لا
 * قائمةَ أدوارٍ مثبَّتةٌ ولا معجمَ قدراتٍ مكتوبٌ في شاشة.** **ودورٌ
 * يُنشَأ يظهر بلا بناءِ واجهةٍ جديد.**
 *
 * # والحراسةُ في المحرّك
 *
 * **وإخفاءُ شاشةٍ لطفٌ بالعين لا حراسة** — **و`roles.manage` تحرسها
 * سياسةُ الجدول المركزيّة**، **وكلُّ فعلٍ هنا يمرّ ببوّابة التأكيد.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  Input,
  LoadingState,
  Modal,
  Select,
  PageContainer,
  PageHeader,
  FormActions,
  IconRoles,
} from "@rahalgo/ui";
import {
  createRole,
  grantCapability,
  listCapabilities,
  listRoles,
  revokeCapability,
  roleLabel,
  type Capability,
  type Role,
} from "@/lib/rbac";

const m = getMessages(defaultLocale);
const t = m.admin.roles;

export default function RolesPage() {
  const [roles, setRoles] = useState<Role[] | null>(null);
  const [caps, setCaps] = useState<Capability[]>([]);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const [createOpen, setCreateOpen] = useState(false);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");

  const [openRole, setOpenRole] = useState<string | null>(null);
  const [pick, setPick] = useState("");

  /**
   * **وتُقرأ الاثنتان معاً بعد كلّ تبديل** — **وواجهةٌ تُحدّث حالتَها
   * من عندها تُظهر ما لم يقع.** (`AQ-4`: القراءةُ من المصدر.)
   */
  const load = useCallback(async () => {
    try {
      const [r, c] = await Promise.all([listRoles(), listCapabilities()]);
      setRoles(r);
      setCaps(c);
      setErr("");
    } catch (e) {
      setErr(errorText(e));
      setRoles([]);
    }
  }, []);

  // **ويُقرأ مرّةً عند الفتح** — ثمّ بعد كلّ تبديلٍ من `act`.
  useEffect(() => {
    void load();
  }, [load]);

  async function act(fn: () => Promise<unknown>) {
    setBusy(true);
    setErr("");
    try {
      await fn();
      await load();
      return true;
    } catch (e) {
      // **ورسالةُ المحرّك تُقرأ بمفتاحها** — **ومنها طلبُ التأكيد**،
      // فتلتقطه بوّابةُ `StepUpGate` المركزيّة ولا يُعاد بناؤها هنا.
      setErr(errorText(e));
      return false;
    } finally {
      setBusy(false);
    }
  }

  const current = roles?.find((r) => r.code === openRole) ?? null;
  const available = current
    ? caps.filter((c) => !current.capabilities.includes(c.code))
    : [];

  return (
    <PageContainer>
      <PageHeader
        title={t.title}
        subtitle={t.subtitle}
        actions={
          <Button onClick={() => setCreateOpen(true)}>{t.create}</Button>
        }
      />

      {err ? <Alert>{err}</Alert> : null}

      {roles === null ? (
        <LoadingState variant="text" />
      ) : roles.length === 0 ? (
        <EmptyState icon={<IconRoles size={32} />} title={t.empty} />
      ) : (
        <div className="grid gap-3">
          {roles.map((r) => (
            <button
              key={r.code}
              type="button"
              onClick={() => {
                setOpenRole(r.code);
                setPick("");
              }}
              className="flex items-center justify-between gap-3 rounded-card border p-3 text-start"
            >
              <span className="flex flex-col gap-1">
                <span className="font-medium">{roleLabel(r)}</span>
                <span className="text-xs opacity-60">{r.code}</span>
              </span>
              <span className="flex items-center gap-2">
                <Badge>
                  {t.capabilities}: {r.capabilities.length}
                </Badge>
                <Badge>
                  {t.members}: {r.members}
                </Badge>
              </span>
            </button>
          ))}
        </div>
      )}

      {/* ── إنشاءُ دور ────────────────────────────────────────────── */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t.createTitle}
      >
        <div className="grid gap-3">
          <div className="grid gap-1">
            <Input
              label={t.code}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              dir="ltr"
            />
            <p className="text-xs opacity-60">{t.codeHint}</p>
          </div>
          <div className="grid gap-1">
            <Input
              label={t.name}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <p className="text-xs opacity-60">{t.nameHint}</p>
          </div>
          <FormActions
            busy={busy || !code.trim() || !name.trim()}
            saveLabel={t.save}
            cancelLabel={t.cancel}
            onCancel={() => setCreateOpen(false)}
            onSave={async () => {
              if (await act(() => createRole(code.trim(), name.trim()))) {
                setCreateOpen(false);
                setCode("");
                setName("");
              }
            }}
          />
        </div>
      </Modal>

      {/* ── صلاحيّاتُ دور ─────────────────────────────────────────── */}
      <Modal
        open={openRole !== null}
        onClose={() => setOpenRole(null)}
        title={current ? roleLabel(current) : ""}
      >
        {current ? (
          <div className="grid gap-3">
            <p className="text-xs opacity-60">{t.backendTruth}</p>

            {current.capabilities.length === 0 ? (
              <p className="text-sm opacity-70">{t.noCapabilities}</p>
            ) : (
              <ul className="grid gap-2">
                {current.capabilities.map((c) => (
                  <li key={c} className="flex items-center justify-between gap-2">
                    <code className="text-sm" dir="ltr">
                      {c}
                    </code>
                    <Button
                      variant="secondary"
                      disabled={busy}
                      onClick={() => act(() => revokeCapability(current.code, c))}
                    >
                      {t.removeCapability}
                    </Button>
                  </li>
                ))}
              </ul>
            )}

            <div className="flex items-end gap-2">
              <Select
                label={t.addCapability}
                id="cap-pick"
                className="flex-1"
                dir="ltr"
                value={pick}
                onChange={(e) => setPick(e.target.value)}
              >
                <option value="">{t.pickCapability}</option>
                {available.map((c) => (
                  <option key={c.code} value={c.code}>
                    {c.code}
                    {c.description ? ` — ${c.description}` : ""}
                  </option>
                ))}
              </Select>
              <Button
                disabled={busy || !pick}
                onClick={async () => {
                  if (await act(() => grantCapability(current.code, pick))) {
                    setPick("");
                  }
                }}
              >
                {t.save}
              </Button>
            </div>
          </div>
        ) : null}
      </Modal>
    </PageContainer>
  );
}
