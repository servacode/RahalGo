"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أوقاتُ منطقةِ توصيل — في محرّر المنطقة نفسِه** (`ZH`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ومنطقةٌ مُغطّاةٌ جغرافيّاً قد لا يُوصَّل إليها الآن** — **والساعةُ
 * هي المانع لا العنوان.** **فالجدولُ صفةٌ من صفات المنطقة كالرسمِ
 * ونصفِ القطر.**
 *
 * **والسريانُ رايةٌ لكلّ منطقةٍ لا رايةٌ عامّة** — **منطقةٌ تعمل ليلَ
 * نهارٍ وأخرى تُغلق السادسةَ وثالثةٌ بفترتين.**
 *
 * **ومطفأةً تبقى المنطقةُ متاحةً على مدار الساعة** — **وهو حالُ كلّ
 * منطقةٍ قائمةٍ اليوم**، **فلا يتبدّل شيءٌ حتّى يُفعَّل بيدٍ.**
 *
 * **ومحرّرُ الأسبوع مشتركٌ مع دوام المنصّة** (`WeekHours`) — **ونسختان
 * تفترقان يوماً.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, Switch, LoadingState } from "@rahalgo/ui";
import WeekHours, { type Win } from "./WeekHours";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const Z = m.admin.zoneHours;
const P = m.admin.platformHours;

interface Props {
  zoneID: string;
  may: boolean;
}

export default function ZoneHoursCard({ zoneID, may }: Props) {
  const [wins, setWins] = useState<Win[]>([]);
  const [enforced, setEnforced] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const h = await api<{ windows: Win[]; enforced: boolean }>(
        `/api/v1/admin/zones/${zoneID}/hours`,
      );
      setWins(h.windows ?? []);
      setEnforced(Boolean(h.enforced));
    } catch (e) {
      setError(errorText(e, m));
    } finally {
      setLoading(false);
    }
  }, [zoneID]);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    setError("");
    try {
      await api(`/api/v1/admin/zones/${zoneID}/hours`, {
        method: "PUT",
        body: JSON.stringify({ enforced, windows: wins }),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      await load();
    } catch (e) {
      setError(errorText(e, m));
    }
  }

  if (loading) return <LoadingState />;

  return (
    <div className="mt-4 space-y-3 border-t border-line-soft pt-4">
      <h4 className="font-medium">{Z.title}</h4>
      <p className="text-sm text-ink-muted">{Z.hint}</p>
      {error && <Alert>{error}</Alert>}

      {/* **والسريانُ أوّلُ ما يُرى** — **وجدولٌ مكتوبٌ لا يسري يُظَنّ
          سارياً.** */}
      <Switch
        checked={enforced}
        onChange={(v: boolean) => setEnforced(v)}
        label={Z.enforced}
        hint={enforced ? Z.enforcedHint : Z.alwaysOpen}
        disabled={!may}
      />

      {enforced && (
        <>
          <p className="text-xs text-ink-muted">{P.boundary}</p>
          <p className="text-xs text-ink-muted">{P.timezone}</p>
          <WeekHours windows={wins} onChange={setWins} disabled={!may} />
        </>
      )}

      {may && (
        <div className="flex items-center gap-3">
          <Button onClick={save}>{Z.save}</Button>
          {saved && <span className="text-sm text-ink-muted">{Z.saved}</span>}
        </div>
      )}
    </div>
  );
}
