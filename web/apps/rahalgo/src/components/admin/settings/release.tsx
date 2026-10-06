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
import { Alert, PageHeader, Card, LoadingState, IconSettings, Switch } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const S = m.admin.release;

interface SettingRow {
  key: string;
  value: unknown;
  editable: boolean;
}

interface FileInfo {
  present: boolean;
  code: number;
  name: string;
  auto: boolean;
  editable: boolean;
}

// **وتطبيقُ رحّال غو أوّلاً** — هو ما يحمّله الناس.
const APPS = ["customer", "driver", "merchant", "rep"] as const;
type AppName = (typeof APPS)[number];

/**
 * **لوحُ التحديث — مفتاحٌ واحدٌ لكلّ تطبيق** (قرارُ المالك ٢٠٢٦-١٠-٠٦: «ما عاد لازم
 * أعدّل أيّ رقم إصدار بإيدي… هذول ما ظلّ إلهن داعي»).
 *
 * **ولا خانةَ رقمٍ يدويّة**: رقمُ النسخة يُقرأ من الملفّ المرفوع، **والفرضُ مفتاح**.
 * وبقي الحدُّ اليدويُّ (`app.min_version.*`) في المحرّك صفراً — بابَ طوارئ لا شاشة.
 */
export default function ReleasePanel() {
  const [file, setFile] = useState<Record<AppName, FileInfo> | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api<{ settings?: SettingRow[] } | SettingRow[]>("/api/v1/admin/settings");
      const rows: SettingRow[] = Array.isArray(res) ? res : (res.settings ?? []);
      const row = (k: string) => rows.find((r) => r.key === k);
      const fi = {} as Record<AppName, FileInfo>;
      for (const app of APPS) {
        const code = row(`release.${app}.version_code`)?.value;
        const name = row(`release.${app}.version`)?.value;
        const apk = row(`release.${app}.apk`)?.value;
        const auto = row(`release.${app}.auto_force`);
        fi[app] = {
          present: typeof apk === "string" && apk !== "",
          code: typeof code === "number" ? code : 0,
          name: typeof name === "string" ? name : "",
          auto: auto?.value !== false,
          editable: auto?.editable ?? false,
        };
      }
      setFile(fi);
      setError("");
    } catch (e) {
      setError(errorText(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggle(app: AppName, on: boolean) {
    setBusy(app);
    setError("");
    try {
      await api(`/api/v1/admin/settings/release.${app}.auto_force`, {
        method: "PUT",
        body: JSON.stringify({ value: on }),
      });
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy("");
    }
  }

  if (!file) {
    return error ? <Alert>{error}</Alert> : <LoadingState />;
  }

  return (
    <div>
      <PageHeader icon={IconSettings} title={S.title} />
      <p className="mb-4 text-sm text-ink-muted">{S.hint}</p>
      {error && <Alert className="mb-4">{error}</Alert>}

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {APPS.map((app) => {
          const f = file[app];
          return (
            <Card key={app} className="space-y-3 p-4">
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-bold">{(S.apps as Record<string, string>)[app]}</span>
                <span className="text-sm text-ink-muted" dir="ltr">
                  {f.present && f.code > 0 ? `${f.name} (${f.code})` : S.fileNone}
                </span>
              </div>
              <Switch
                checked={f.auto}
                onChange={(v: boolean) => void toggle(app, v)}
                label={S.autoLabel}
                hint={f.auto ? S.autoOn : S.autoOff}
                disabled={!f.editable || busy !== ""}
              />
            </Card>
          );
        })}
      </div>
    </div>
  );
}
