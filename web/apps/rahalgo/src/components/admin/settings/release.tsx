"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إعداداتُ الإصدار — الحدُّ الأدنى لنسخةِ كلِّ تطبيق** (البند F، ٢٠٢٦-٠٩-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما هي
 *
 * **`app.min_version.<التطبيق>` تقول أيُّ نسخةٍ يُسمَح لها بالعمل** — ومن
 * دونها يُردُّ ٤٢٦ فتُظهر شاشةَ التحديث الإلزاميّ. **ورفعُها فعلٌ تشغيليٌّ
 * خطير**: يقفل تطبيقَ كلِّ من دونها.
 *
 * # الأمانُ من المحرّك لا من هنا
 *
 * **والكتابةُ عبر `PUT /admin/settings/{key}`** — يحرسها المحرّكُ بقدرةِ
 * المفتاح، **وخطوةِ تحقّقٍ** (`criticalSettingKey` ⇒ `StepUpGate` مركزيّاً)،
 * **وتدقيقٍ دائمٍ في المعاملة**، وتحقّقٍ من النوع (عددٌ صحيح). **فاللوحةُ
 * تعرض وتُرسل، والحَكَمُ الخادم.**
 *
 * # والسائقُ توزيعٌ مباشر
 *
 * **ولا زرَّ «متجر Play» للسائق** — حزمتُه توزيعٌ مباشرٌ (`ChannelDirect`)،
 * فشاشةُ تحديثِه تشير إلى التنزيل المباشر لا إلى المتجر.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  PageHeader,
  Button,
  Card,
  Input,
  LoadingState,
  IconSettings,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const S = m.admin.release;

interface SettingRow {
  key: string;
  value: unknown;
  editable: boolean;
}

// **أربعُ نسخٍ لأربعةِ أدوار** — والسائقُ أوّلاً فهو موضعُ الشهادة اليوم.
const APPS = ["driver", "customer", "merchant", "rep"] as const;
type AppName = (typeof APPS)[number];

export default function ReleasePanel() {
  const [current, setCurrent] = useState<Record<AppName, number> | null>(null);
  const [editable, setEditable] = useState<Record<AppName, boolean>>(
    {} as Record<AppName, boolean>,
  );
  const [draft, setDraft] = useState<Record<AppName, string>>(
    {} as Record<AppName, string>,
  );
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState("");

  const load = useCallback(async () => {
    try {
      // **والردُّ كائنٌ لا قائمة** (`{settings, groups, topics}`) — كان يُقرأ قائمةً
      // فيسقط اللوحُ على `rows.find` (كشفه فحصُ الإعدادات ٢٠٢٦-١٠-٠٤).
      const res = await api<{ settings?: SettingRow[] } | SettingRow[]>("/api/v1/admin/settings");
      const rows: SettingRow[] = Array.isArray(res) ? res : (res.settings ?? []);
      const cur = {} as Record<AppName, number>;
      const edit = {} as Record<AppName, boolean>;
      const dft = {} as Record<AppName, string>;
      for (const app of APPS) {
        const row = rows.find((r) => r.key === `app.min_version.${app}`);
        const n = typeof row?.value === "number" ? row.value : 0;
        cur[app] = n;
        edit[app] = row?.editable ?? false;
        dft[app] = String(n);
      }
      setCurrent(cur);
      setEditable(edit);
      setDraft(dft);
      setError("");
    } catch (e) {
      setError(errorText(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function save(app: AppName) {
    const raw = draft[app]?.trim() ?? "";
    const n = Number(raw);
    // **عددٌ صحيحٌ غيرُ سالب** — والباقي يرفضه الكتالوجُ في المحرّك أيضاً.
    if (!Number.isInteger(n) || n < 0) {
      setError(S.invalid);
      return;
    }
    setBusy(app);
    setError("");
    setNotice("");
    try {
      await api(`/api/v1/admin/settings/app.min_version.${app}`, {
        method: "PUT",
        body: JSON.stringify({ value: n }),
      });
      setNotice(S.saved);
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy("");
    }
  }

  if (!current) {
    return error ? <Alert>{error}</Alert> : <LoadingState />;
  }

  return (
    <div>
      <PageHeader icon={IconSettings} title={S.title} />
      <p className="mb-4 text-sm text-ink-muted">{S.hint}</p>

      {error && <Alert className="mb-4">{error}</Alert>}
      {notice && <Alert tone="success" className="mb-4">{notice}</Alert>}

      {/* **والسائقُ توزيعٌ مباشرٌ — لا متجر Play.** */}
      <Card className="mb-4 p-4">
        <p className="text-sm text-ink-muted">{S.directDistribution}</p>
      </Card>

      <div className="grid gap-3">
        {APPS.map((app) => (
          <Card key={app} className="p-4">
            <div className="mb-2 font-medium">
              {(S.apps as Record<string, string>)[app]}
            </div>
            <p className="mb-3 text-xs text-ink-muted">
              {S.currentLabel}: {current[app]}
            </p>
            <div className="flex items-end gap-2">
              <div className="flex-1">
                <Input
                  type="number"
                  inputMode="numeric"
                  label={S.newLabel}
                  value={draft[app] ?? ""}
                  disabled={!editable[app]}
                  onChange={(e) =>
                    setDraft((d) => ({ ...d, [app]: e.target.value }))
                  }
                />
              </div>
              <Button
                disabled={
                  !editable[app] ||
                  busy !== "" ||
                  draft[app]?.trim() === String(current[app])
                }
                onClick={() => void save(app)}
              >
                {S.apply}
              </Button>
            </div>
            {!editable[app] && (
              <p className="mt-2 text-xs text-ink-muted">{S.noPermission}</p>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}
