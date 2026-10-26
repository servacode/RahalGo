"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبُ التوسّع بالمكان الإداريّ** (`CR`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ تجميعٌ لا قائمةُ صفوف
 *
 * **وقائمةُ طلباتِ التغطية قائمةٌ في الخريطة منذ ٠١٢٧** — **تقول «أين»
 * بالخلايا.** **وهذا يقول «في أيّ مدينةٍ ومحافظة»** — **وهو سؤالُ
 * التوسّع الأوّل**: «كم شخصاً طلب الخدمةَ في دمشق؟»
 *
 * **ومئةُ صفٍّ متطابقٍ لا تُقرأ** — **والقرارُ يُتَّخذ بالمحافظة لا
 * بالصفّ.**
 *
 * # وما لا يُعرَف يبقى بلا اسم
 *
 * **وطلبٌ في موضعٍ لا تعرفه المنصّةُ يُعرَض «غير محدَّد إداريّاً»** —
 * **ولا يُنسَب إلى محافظةٍ لم تُحَلّ**: **وطلبٌ في البادية لا يُحسَب
 * على دمشق.**
 *
 * # ولا يوسّع شيئاً
 *
 * **وكثافةُ الطلب لا تنشئ منطقةً ولا تفعّل مدينة** — **والقرارُ
 * قرارُ المالك**، وهذا يُخبره لا يقرّر عنه.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Card, Tabs, LoadingState, EmptyState, IconLocation } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const D = m.admin.placeDemand;

interface Place {
  governorate_id?: string;
  governorate_name?: string;
  city_id?: string;
  city_name?: string;
  kind: string;
  people: number;
  signals: number;
}

export default function PlaceDemandPanel() {
  const [kind, setKind] = useState("");
  const [rows, setRows] = useState<Place[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const q = kind ? `?kind=${kind}` : "";
      const res = await api<{ places: Place[] }>(
        `/api/v1/admin/ops-map/coverage-demand/places${q}`,
      );
      setRows(res.places ?? []);
    } catch (e) {
      setError(errorText(e, m));
    } finally {
      setLoading(false);
    }
  }, [kind]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <Card>
      <div className="space-y-3 p-4">
        <h3 className="heading-card">{D.title}</h3>
        <p className="text-sm text-ink-muted">{D.hint}</p>
        {error && <Alert>{error}</Alert>}

        <Tabs
          items={[
            { key: "", label: D.kindAll },
            { key: "coverage_request", label: D.kindCoverage },
            { key: "service_interest", label: D.kindInterest },
          ]}
          value={kind}
          onChange={setKind}
        />

        {loading ? (
          <LoadingState />
        ) : rows.length === 0 ? (
          <EmptyState title={D.empty} />
        ) : (
          // **بطاقاتٌ لا جدول** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — المكانُ وتحته عددُ الأشخاص والطلبات.
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
            {rows.map((r, i) => (
              <div key={i} className="flex items-center gap-3 rounded-card border border-line-soft p-3">
                <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-field text-ink-muted">
                  <IconLocation size={17} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate font-bold">
                    {r.city_name || r.governorate_name || <span className="text-ink-muted">{D.unresolved}</span>}
                  </p>
                  {r.city_name && r.governorate_name && (
                    <p className="truncate text-xs text-ink-muted">{r.governorate_name}</p>
                  )}
                </div>
                <div className="flex shrink-0 gap-3 text-center tabular-nums">
                  <span>
                    <span className="block font-bold">{fmtNum(r.people)}</span>
                    <span className="block text-2xs text-ink-muted">{D.people}</span>
                  </span>
                  <span>
                    <span className="block font-bold">{fmtNum(r.signals)}</span>
                    <span className="block text-2xs text-ink-muted">{D.signals}</span>
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </Card>
  );
}
