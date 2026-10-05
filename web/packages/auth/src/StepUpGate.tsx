"use client";

/**
 * **نافذةُ تأكيدِ الفعل الشديد — واحدةٌ للوحة كلِّها.** (`ADG-3` · `AQ-1`.)
 *
 * # ولماذا واحدة
 *
 * **نموذجُ كلمةٍ في كلّ صفحةٍ يعني عشرةَ نماذج** — **تُصلَح واحدةٌ
 * وتبقى تسع**، وهي علّةُ `StatusReasonModal` نفسُها حين كانت ثلاثاً.
 *
 * **والاعتراضُ في عميل الـAPI** — **فأيُّ فعلٍ يطلبه الخادمُ تأكيداً
 * يُسأل عنه**، ولو أُضيف غداً ولم تُمَسّ شاشتُه.
 *
 * # والحدُّ في الخادم لا هنا
 *
 * **هذه يسرُ استعمالٍ لا أمن** — **ونداءٌ مباشرٌ بلا إثباتٍ يُردّ**
 * سواءٌ مرّ من هنا أو لم يمرّ.
 *
 * # ولا تُحفَظ الكلمة
 *
 * **تُقرأ وتُرسَل إلى بابِ التأكيد وتُمحى** — **ولا مخزنَ متصفّحٍ ولا
 * حالةٌ تبقى بعد النافذة**، **ولا تُرسَل ثانيةً مع الفعل نفسِه.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";
import { Input, Modal, FormActions } from "@rahalgo/ui";
import { setStepUpAsker, requestStepUp, type StepUpNeed, type StepUpRequest } from "./client";

const m = getMessages(defaultLocale);

/** **ونصوصُ الأفعال في المعجم المركزيّ** — ولا نصَّ عربيٌّ في شيفرة. */
const S = m.admin.stepUp;

type Pending = {
  need: StepUpNeed;
  req: StepUpRequest;
  resolve: (grant: string | null) => void;
};

export function StepUpGate() {
  const [pending, setPending] = useState<Pending | null>(null);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setStepUpAsker(
      (need, req) =>
        new Promise<string | null>((resolve) => {
          setPassword("");
          setFailed(false);
          setPending({ need, req, resolve });
        }),
    );
    return () => setStepUpAsker(null);
  }, []);

  const close = useCallback(() => {
    pending?.resolve(null);
    setPending(null);
    setPassword("");
  }, [pending]);

  if (!pending) return null;

  const { need, req } = pending;
  const what = (S.actions as Record<string, string>)[need.action] ?? need.action;
  // **والهدفُ يُعرَض إن وُجد** — **ومن يؤكّد يرى على من يقع الفعل.**
  const on = need.target_id ? ` — ${need.target_id}` : "";
  // **وحقولُ التبديل الجوهريّةُ تُعرَض** — **فمن أكّد مبلغاً رآه.**
  const body = (req.body ?? {}) as Record<string, unknown>;
  // **وبأسمائها العربيّة لا برموزها** (فحصُ المال ٢٠٢٦-١٠-٠٥: كانت «amount: 25000»).
  const names: Record<string, Record<string, string>> = {
    role: m.terms.roleNames as Record<string, string>,
    capability: m.terms.capabilityNames as Record<string, string>,
    status: S.statusValues as Record<string, string>,
  };
  const factValue = (k: string, v: unknown): string => {
    if (k === "amount" && typeof v === "number") return fmtMoney(v);
    if (typeof v === "string") return names[k]?.[v] ?? v;
    return JSON.stringify(v);
  };
  const facts = ["amount", "role", "capability", "status", "value"]
    .filter((k) => body[k] !== undefined)
    .map((k) => `${(S.facts as Record<string, string>)[k] ?? k}: ${factValue(k, body[k])}`)
    .join(" · ");

  return (
    <Modal open onClose={close} title={S.title}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          if (busy) return;
          setBusy(true);
          setFailed(false);
          try {
            const grant = await requestStepUp(req, password);
            pending.resolve(grant);
            setPending(null);
            setPassword("");
          } catch {
            // **وكلمةٌ خاطئةٌ تُمحى ويُعاد السؤال** — **ولا يُشرَح
            // للمهاجم أين أخطأ.**
            setFailed(true);
            setPassword("");
          } finally {
            setBusy(false);
          }
        }}
      >
        <p>
          <strong>{what}</strong>
          {on}
        </p>
        {facts && <p>{facts}</p>}
        <p>{S.prompt}</p>
        <Input
          type="password"
          autoComplete="current-password"
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={S.placeholder}
        />
        {failed && <p role="alert">{S.failed}</p>}
        <FormActions
          submit
          busy={busy}
          onCancel={close}
          saveLabel={S.confirm}
          cancelLabel={m.common.cancel}
        />
      </form>
    </Modal>
  );
}
