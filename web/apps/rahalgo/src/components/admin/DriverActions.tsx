"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أفعالُ السائق — صندوقُه وتسويتُه وإنهاءُ ورديّته**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (نُقلت من شاشة السائقين حين حُذفت ٢٠٢٦-٠٨-١٥ — **بنوافذها ونداءاتها
 *  كما هي**: النقلُ لا يُضيّع شيئاً، **وحذفُ ملفٍّ وإعادةُ كتابته من
 *  الذاكرة هو ما يُضيّع.**)
 *
 * # ولماذا في بطاقة الحساب لا في ملفّه
 *
 * **«إنهاءُ الورديّة» و«التسوية» يُفعلان على عجل**: سائقٌ نسي ورديّتَه
 * بعد منتصف الليل، أو بيده مالٌ ينتظر. **ومن فتح ملفَّه ليضغط زرّاً
 * واحداً دفع ثمنَ صفحةٍ كاملة.**
 *
 * # ولا تعرف هذه النوافذُ من السائق إلّا معرّفَه واسمَه
 *
 * **فتُنادى من أيّ شاشة** — ولو حملت نوعَ صفِّ السائق لَما استُعملت
 * إلّا حيث ذاك الصفّ.
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Modal,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

/** **ومن تُفعل به** — معرّفٌ واسمٌ لا صفُّ جدول. */
export interface DriverRef {
  id: string;
  full_name: string;
  /** **ويُعرض حين لا اسمَ له** — حسابٌ بلا اسمٍ يُعرف برقمه. */
  phone?: string;
}

/** **ورسالةُ الخادم تُترجَم بمفتاحها** — نُقلت مع النوافذ كما هي. */
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

/**
 * إغلاقُ الدوام — **بكلمةٍ تصل صاحبَه.**
 *
 * **ومن أُغلق دوامُه بلا علمه يظنّ أنّ النظام أعطبه** — فيشتكي، أو يظنّ أن لا
 * طلباتِ اليوم فيمضي إلى بيته. **والكلمةُ ليست تجميلاً: هي الفرقُ بين إجراءٍ
 * وبين عطبٍ يبدو عشوائياً.**
 */
export function EndShiftModal({
  driver,
  onClose,
  onDone,
}: {
  driver: DriverRef;
  onClose: () => void;
  onDone: () => void;
}) {
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function submit() {
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/drivers/${driver.id}/end-shift`, {
        method: "POST",
        body: JSON.stringify({ note: note.trim() }),
      });
      onDone();
    } catch (e) {
      setErr(errText(e));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.drivers.endShift}: ${driver.full_name}`}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{m.admin.drivers.endShiftHint}</p>
        <Input
          id="end-shift-note"
          label={m.admin.drivers.endShiftNote}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        {err && <p className="text-sm text-danger">{err}</p>}
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button variant="danger" disabled={busy} onClick={() => void submit()}>
            {m.admin.drivers.endShift}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
