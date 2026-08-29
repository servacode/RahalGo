"use client";

/**
 * **طابورُ مراجعة القائمة.**
 *
 * `merchants.menu_requires_approval` كان مفتاحاً يَعِد ولا يفعل: تُشغّله فتظنّ
 * أنّ قوائمَ المتاجر تُراجَع قبل النشر، **وهي تُنشر كما هي.**
 *
 * **وطابورٌ لا يعلم به أحدٌ طابورٌ لا يُفرَغ** — فيُعرض هنا، ويصل المكتبَ إشعارٌ
 * بكلّ صنفٍ ينتظر.
 *
 * # ولا يظهر حين يكون المفتاحُ مُطفأً
 *
 * **قسمٌ فارغٌ دائماً يُتعلَّم تجاهلُه** — ثمّ يُرفع المفتاحُ يوماً فلا يُنظر
 * إليه.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Money,
  Alert,
  Badge,
  Button,
  Input,
  FormSection,
  EmptyState,
  IconStore,
  IconCheck,
  IconBlock,
  FormActions,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { MediaThumb } from "@/components/admin/ImageUpload";

const m = getMessages(defaultLocale);
const Q = m.admin.menuReview;

interface Pending {
  id: string;
  name: string;
  description: string;
  merchant_price: number;
  merchant_id: string;
  merchant_name: string;
  section_name: string;
  thumb_url: string | null;
  updated_at: string;
}

export default function MenuReviewQueue() {
  const [rows, setRows] = useState<Pending[] | null>(null);
  /** **وعددُ الطابور كلِّه وسقفُ العرض** — (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦).

      **وكان العنوانُ يعدّ المعروضَ** — فطابورٌ فيه ثلاثمئةٌ وأربعةَ عشرَ
      يقول «مئتان». **وسقفٌ صامتٌ في طابور مراجعةٍ أخطرُ من سواه**:
      **صنفٌ لا يُوافَق عليه لأنّ أحداً لم يره.** */
  const [count, setCount] = useState(0);
  const [rejecting, setRejecting] = useState<string>("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api<{ items: Pending[]; count: number }>(
        "/api/v1/admin/menu/pending",
      );
      setRows(res.items ?? []);
      setCount(res.count ?? 0);
    } catch {
      setRows([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function review(id: string, approve: boolean, reason = "") {
    setBusy(id);
    setError("");
    try {
      await api(`/api/v1/admin/menu/items/${id}/review`, {
        method: "POST",
        body: JSON.stringify({ approve, note: reason }),
      });
      setRejecting("");
      setNote("");
      await load();
    } catch (err) {
      setError(
        err instanceof ApiError
          ? ((m.errors as Record<string, string>)[
              err.body.message_key.split(".").pop() ?? ""
            ] ?? m.errors.internal)
          : m.errors.internal,
      );
    } finally {
      setBusy("");
    }
  }

  // **ولا يُعرض قسمٌ فارغ** — الطابورُ يظهر حين يكون فيه عمل.
  if (!rows || rows.length === 0) return null;

  return (
    <FormSection title={`${Q.title} (${fmtNum(count)})`} icon={<IconStore />}>
      {/* **والنقصُ يُعلَن** — **وسقفٌ صامتٌ يُقرأ «هذا كلُّ ما ينتظر»**،
          فيُظنّ الطابورُ نضب وفيه مئة. */}
      {count > rows.length && (
        <p className="mb-2 text-center text-xs text-ink-muted">
          {Q.showingLatest
            .replace("{n}", fmtNum(rows.length))
            .replace("{all}", fmtNum(count))}
        </p>
      )}
      <p className="mb-3 text-xs text-ink-muted">{Q.hint}</p>
      {error && (
        <Alert className="mb-3">{error}</Alert>
      )}
      <ul className="space-y-2">
        {rows.map((it) => (
          <li key={it.id} className="rounded-card border border-line bg-field p-3">
            <div className="flex flex-wrap items-center gap-3">
              {it.thumb_url && <MediaThumb url={it.thumb_url} alt={it.name} fallback={it.name.slice(0, 1)} />}
              <span className="min-w-0 flex-1">
                <span className="block font-bold">{it.name}</span>
                {it.description && (
                  <span className="block text-sm text-ink-muted">{it.description}</span>
                )}
                <span className="block text-xs text-ink-muted">
                  {it.merchant_name}
                  {it.section_name && ` · ${it.section_name}`} ·{" "}
                  <span dir="ltr">{fmtDateTime(it.updated_at)}</span>
                </span>
              </span>
              <Badge variant="neutral">
                <Money value={it.merchant_price} />
              </Badge>
              {rejecting !== it.id && (
                <span className="flex shrink-0 gap-2">
                  <Button disabled={busy !== ""} onClick={() => void review(it.id, true)}>
                    <span className="flex items-center gap-1.5">
                      <IconCheck size={15} />
                      {Q.approve}
                    </span>
                  </Button>
                  <Button variant="danger" disabled={busy !== ""} onClick={() => setRejecting(it.id)}>
                    <span className="flex items-center gap-1.5">
                      <IconBlock size={15} />
                      {Q.reject}
                    </span>
                  </Button>
                </span>
              )}
            </div>

            {/* **والردُّ يلزمه كلمة** — «رُدّ» بلا سببٍ يُعاد إرسالُه كما هو،
                فيدور المتجرُ والمكتبُ في حلقة. */}
            {rejecting === it.id && (
              <div className="mt-3 space-y-2 border-t border-line-soft pt-3">
                <Input
                  id={`reject-${it.id}`}
                  label={Q.rejectNote}
                  required
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
                <FormActions onSave={() => void review(it.id, false, note.trim())} onCancel={() => {
                      setRejecting("");
                      setNote("");
                    }} saveLabel={Q.rejectConfirm} tone="danger" />
              </div>
            )}
          </li>
        ))}
      </ul>
    </FormSection>
  );
}
