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
import { Alert, Card, Tabs, LoadingState, EmptyState } from "@rahalgo/ui";
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
        <h3 className="font-medium">{D.title}</h3>
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
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-start text-ink-muted">
                  <th className="p-2 text-start">{D.governorate}</th>
                  <th className="p-2 text-start">{D.city}</th>
                  <th className="p-2 text-start">{D.people}</th>
                  <th className="p-2 text-start">{D.signals}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r, i) => (
                  <tr key={i} className="border-t border-line-soft">
                    <td className="p-2">
                      {r.governorate_name || (
                        <span className="text-ink-muted">{D.unresolved}</span>
                      )}
                    </td>
                    <td className="p-2">{r.city_name || "—"}</td>
                    <td className="p-2 font-medium">{fmtNum(r.people)}</td>
                    <td className="p-2">{fmtNum(r.signals)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Card>
  );
}
