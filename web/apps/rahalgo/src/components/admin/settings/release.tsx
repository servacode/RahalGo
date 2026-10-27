"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التطبيقاتُ والتنزيل — بطاقةٌ لكلّ تطبيق** (البند F ٢٠٢٦-٠٩-٢٧، وتدقيقُ ٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت ١٧ بطاقةً مبعثرةً ومكرّرة**: ملفٌّ هنا ورابطُ بلاي هناك، ورقمُ النسخة واسمُها
 * بطاقتان للقراءة، والفرضُ في لوحٍ ثالث. **فصار لكلّ تطبيقٍ بطاقةٌ واحدة**: النسخةُ الحاليّة
 * برقمٍ كبير، ورفعُ الملفّ، ورابطُ غوغل بلاي، ومفتاحُ فرض التحديث **بتأكيد**.
 *
 * # الأمانُ من المحرّك لا من هنا
 *
 * **والكتابةُ عبر `PUT /admin/settings/{key}`** — يحرسها المحرّكُ بقدرةِ المفتاح وتدقيقٍ
 * دائمٍ في المعاملة. **وفرضُ التحديث يقفل تطبيقَ كلِّ من دون النسخة** (`updateGateSetting`)،
 * **فلا يُقلب بلمسةٍ واحدة** — يُسأل قبله ويُقال اسمُ التطبيق والنسخة.
 *
 * **ولا خانةَ رقمٍ يدويّة** (قرارُ المالك ٢٠٢٦-١٠-٠٦): رقمُ النسخة يُقرأ من الملفّ المرفوع.
 */

import { useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Card, Confirm, IconSmartphone, Switch } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const S = m.admin.release;

interface SettingLite {
  key: string;
  value: unknown;
  editable?: boolean;
}

// **وتطبيقُ رحّال غو أوّلاً** — هو ما يحمّله الناس.
const APPS = ["customer", "driver", "merchant", "rep"] as const;
type AppName = (typeof APPS)[number];

const appName = (a: AppName) => (S.apps as Record<string, string>)[a] ?? a;

export default function ReleasePanel({
  list,
  onSaved,
  renderKey,
}: {
  list: SettingLite[];
  onSaved: () => void;
  /** **بطاقةُ مفتاحٍ بلا إطار** — رفعُ الملفّ ورابطُ بلاي يُحرَّران بمحرّر الإعدادات نفسِه. */
  renderKey: (key: string) => ReactNode;
}) {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [ask, setAsk] = useState<{ app: AppName; on: boolean } | null>(null);

  const row = (k: string) => list.find((r) => r.key === k);
  const info = (app: AppName) => {
    const code = row(`release.${app}.version_code`)?.value;
    const name = row(`release.${app}.version`)?.value;
    const apk = row(`release.${app}.apk`)?.value;
    const auto = row(`release.${app}.auto_force`);
    return {
      present: typeof apk === "string" && apk !== "",
      code: typeof code === "number" ? code : 0,
      name: typeof name === "string" ? name : "",
      auto: auto?.value !== false,
      editable: auto?.editable ?? false,
    };
  };
  const versionText = (app: AppName) => {
    const f = info(app);
    if (!f.present || f.code <= 0) return S.noFileYet;
    return f.name ? `${f.name} (${fmtNum(f.code)})` : fmtNum(f.code);
  };

  async function toggle(app: AppName, on: boolean) {
    setBusy(app);
    setError("");
    try {
      await api(`/api/v1/admin/settings/release.${app}.auto_force`, {
        method: "PUT",
        body: JSON.stringify({ value: on }),
      });
      onSaved();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy("");
      setAsk(null);
    }
  }

  return (
    <section className="space-y-3">
      <div>
        <h2 className="heading-card">{S.title}</h2>
        <p className="text-sm text-ink-muted">{S.hint}</p>
      </div>
      {error && <Alert>{error}</Alert>}

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {APPS.map((app) => {
          const f = info(app);
          return (
            <Card key={app} className="space-y-4 p-4">
              <div className="flex items-center gap-3">
                <span
                  className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${
                    f.present ? "bg-accent text-on-bright" : "bg-field text-ink-muted"
                  }`}
                >
                  <IconSmartphone size={20} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-bold">{appName(app)}</span>
                  <span className="block text-2xs text-ink-muted">{S.versionLabel}</span>
                  <span className={`figure block ${f.present ? "" : "text-ink-muted"}`} dir="ltr">
                    {versionText(app)}
                  </span>
                </span>
              </div>

              <div className="space-y-3 border-t border-line-soft pt-3">
                {renderKey(`release.${app}.apk`)}
              </div>
              <div className="space-y-3 border-t border-line-soft pt-3">
                {renderKey(`release.${app}.play_url`)}
              </div>

              <div className="border-t border-line-soft pt-3">
                <Switch
                  checked={f.auto}
                  onChange={(v: boolean) => setAsk({ app, on: v })}
                  label={S.autoLabel}
                  hint={f.auto ? S.autoOn : S.autoOff}
                  disabled={!f.editable || busy !== ""}
                />
                {!f.editable && <p className="mt-1.5 text-xs text-ink-muted">{S.noPermission}</p>}
              </div>
            </Card>
          );
        })}
      </div>

      <Confirm
        open={ask !== null}
        title={(ask?.on ? S.confirmOnTitle : S.confirmOffTitle).replace("{app}", ask ? appName(ask.app) : "")}
        body={
          ask
            ? (ask.on ? S.confirmOnBody : S.confirmOffBody)
                .replace(/\{app\}/g, appName(ask.app))
                .replace("{v}", versionText(ask.app))
            : ""
        }
        confirmLabel={ask?.on ? S.confirmOn : S.confirmOff}
        tone={ask?.on ? "danger" : "primary"}
        busy={busy !== ""}
        onConfirm={() => ask && void toggle(ask.app, ask.on)}
        onCancel={() => setAsk(null)}
      />
    </section>
  );
}
