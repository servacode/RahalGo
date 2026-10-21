"use client";

/**
 * **«انضمّ متجر جديد إلى عائلة رحّال غو»** (قرارُ المالك ٢٠٢٦-١٠-٢١).
 *
 * يُعلَن عن المتجر لكلّ الزبائن مرّةً واحدةً حين يصير ظاهراً لهم. **و«لا ترسل إشعاراً»**
 * لمتجرٍ تجريبيٍّ مثلاً — فلا يصل الزبائنَ الحقيقيّين خبرُ متجرٍ ليس حقيقيّاً.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDate, errorText } from "@rahalgo/i18n";
import { Alert } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const A = m.admin.merchantProfile.announce;

interface AnnounceState {
  announced_at: string | null;
  skip: boolean;
}

export function MerchantAnnounce({ merchantId, canManage }: { merchantId: string; canManage: boolean }) {
  const [st, setSt] = useState<AnnounceState | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setSt(await api<AnnounceState>(`/api/v1/admin/merchants/${merchantId}/announce`));
    } catch (e) {
      setError(errorText(e));
    }
  }, [merchantId]);

  useEffect(() => {
    void load();
  }, [load]);

  const setSkip = useCallback(
    async (skip: boolean) => {
      setSaving(true);
      setError("");
      try {
        setSt(
          await api<AnnounceState>(`/api/v1/admin/merchants/${merchantId}/announce`, {
            method: "PATCH",
            body: JSON.stringify({ skip }),
          }),
        );
      } catch (e) {
        setError(errorText(e));
      } finally {
        setSaving(false);
      }
    },
    [merchantId],
  );

  if (!st) return error ? <Alert tone="error">{error}</Alert> : null;

  return (
    <div className="flex flex-col gap-2 rounded-card border border-line p-4">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm text-ink-muted">{A.title}</span>
        <span className="text-sm font-medium">
          {st.announced_at ? A.sentAt.replace("{date}", fmtDate(st.announced_at)) : A.notYet}
        </span>
      </div>
      {!st.announced_at && (
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm text-ink-muted">{A.skipLabel}</span>
          <div className="flex gap-1">
            {[false, true].map((v) => (
              <button
                key={String(v)}
                type="button"
                disabled={saving || !canManage}
                onClick={() => void setSkip(v)}
                className={`rounded-control border px-3 py-1 text-sm transition-colors ${
                  st.skip === v
                    ? "border-accent bg-accent-tint font-medium"
                    : "border-line text-ink-muted hover:border-accent-edge"
                }`}
              >
                {v ? A.skipYes : A.skipNo}
              </button>
            ))}
          </div>
        </div>
      )}
      {error && <Alert tone="error">{error}</Alert>}
    </div>
  );
}
