"use client";

/**
 * **الطوارئ** — بلاغاتُ السائقين المفتوحة.
 *
 * # لماذا شاشةٌ لا إشعارٌ وحدَه
 *
 * الإشعارُ يمرّ في الشريط ويُقرأ مرّةً. **وطارئٌ يُقرأ ولا يُغلق طارئٌ نُسي**:
 * من سأل عنه بعد يومين لم يجد من يقول ماذا جرى — **أوصل الطلبَ غيرُه؟ أطمأنّ
 * السائق؟ أدُفع له شيء؟**
 *
 * فالبلاغُ يبقى ظاهراً حتى يُغلقه إنسان. **ولا يختفي بمرور الوقت** — الوقتُ لا
 * يطمئنّ على أحد.
 */

import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Button,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  Card,
  useLiveData,
  IconWarning,
  IconPhone,
  IconLocation,
  IconStatus,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const E = m.admin.emergencies;

interface Emergency {
  id: string;
  driver_name: string;
  driver_phone: string;
  order_number: number | null;
  note: string;
  lat: number | null;
  lng: number | null;
  created_at: string;
}

export default function EmergenciesPage() {
  const { data, reload } = useLiveData<{ emergencies: Emergency[] }>(
    () => api("/api/v1/admin/emergencies"),
    ["driver", "order"],
  );
  if (!data) return <LoadingState />;
  const list = data.emergencies ?? [];

  async function resolve(id: string) {
    await api(`/api/v1/admin/emergencies/${id}/resolve`, { method: "POST" });
    reload();
  }

  return (
    <PageContainer>
      <PageHeader icon={IconWarning} title={E.title} subtitle={E.hint} />

      {list.length === 0 ? (
        <EmptyState icon={IconStatus} title={E.empty} />
      ) : (
        <div className="space-y-3">
          {list.map((x) => (
            <Card key={x.id}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="font-bold">
                    {x.driver_name || x.driver_phone}
                    {x.order_number !== null && (
                      <span className="ms-2 text-sm font-normal text-ink-muted" dir="ltr">
                        #{fmtNum(x.order_number)}
                      </span>
                    )}
                  </p>
                  {x.note && <p className="mt-1 text-sm">{x.note}</p>}
                  <p className="mt-1 text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(x.created_at)}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {/* **الاتّصالُ أوّلاً** — قبل أيّ زرٍّ آخر في هذه الشاشة. */}
                  <a
                    href={`tel:${x.driver_phone}`}
                    className="inline-flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm"
                  >
                    <IconPhone size={15} />
                    <span dir="ltr">{x.driver_phone}</span>
                  </a>
                  {x.lat !== null && x.lng !== null && (
                    <a
                      href={`https://www.google.com/maps/dir/?api=1&destination=${x.lat},${x.lng}`}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm"
                    >
                      <IconLocation size={15} />
                      {E.where}
                    </a>
                  )}
                  <Button variant="secondary" onClick={() => resolve(x.id)}>
                    {E.resolve}
                  </Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </PageContainer>
  );
}
