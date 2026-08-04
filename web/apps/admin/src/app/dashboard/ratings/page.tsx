"use client";

/**
 * **التقييمات — من يشكو منه الناس؟**
 *
 * كانت تُقرأ في ملفّ كلّ إنسانٍ على حدة، **فسؤالُ «أيُّ سائقٍ يشكو منه الناس؟»
 * لا جوابَ له إلّا بفتح عشرين ملفّاً** — فلا يُفتح، فلا يُعرف.
 *
 * **ونجمتان لسائقٍ في ملفّه رقمٌ يخصّه، ونجمتان بين خمسةٍ متوسّطُهم أربعٌ
 * مسألةٌ تخصّ المنصة.**
 *
 * # والعددُ مع المتوسّط
 *
 * سائقٌ أوصل طلبين وأخذ نجمةً في أحدهما متوسّطُه ثلاث، **وسائقٌ أوصل مئتين
 * ومتوسّطُه ثلاثٌ حالةٌ أخرى تماماً.** فالترتيبُ بالأسوأ **بشرط عددٍ أدنى**
 * يملك المالكُ رفعَه.
 *
 * # والكلامُ مع الرقم
 *
 * **رقمٌ بلا كلامٍ لا يُصلح شيئاً**: من رأى «٢٫٤» لا يعرف أهو تأخّرٌ أم سوءُ
 * خلق. **ولا تُعرض التعليقاتُ كلُّها** — الرضا الصامتُ لا يحتاج قراءة، وما
 * يُقرأ هو ما فيه كلمةٌ أو نجمتان فأقلّ.
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  PageHeader,
  Select,
  Badge,
  Stars,
  StatGrid,
  StatCard,
  EmptyState,
  LoadingState,
  IconStar,
  IconDriver,
  IconWarning,
  IconOrder,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const R = m.admin.ratingsPage;

interface Party {
  id: string;
  name: string;
  phone: string;
  count: number;
  average: number;
  low: number;
}

interface Comment {
  order_number: number;
  customer: string;
  driver: string;
  platform_stars: number;
  driver_stars: number | null;
  comment: string;
  created_at: string;
}

interface Data {
  total: number;
  average: number;
  has_average: boolean;
  low: number;
  drivers: Party[];
  comments: Comment[];
}

export default function RatingsPage() {
  const router = useRouter();
  const [min, setMin] = useState("3");
  const [data, setData] = useState<Data | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await api<Data>(`/api/v1/admin/ratings?min=${min}`));
      setError("");
    } catch {
      setError(m.errors.internal);
    }
  }, [min]);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!data) return <LoadingState />;

  return (
    <div>
      <PageHeader icon={IconStar} title={R.title} />
      <p className="mb-4 text-sm text-ink-muted">{R.hint}</p>

      <StatGrid>
        <StatCard label={R.total} value={fmtNum(data.total)} icon={IconStar} />
        <StatCard
          label={R.average}
          value={data.has_average ? data.average.toFixed(1) : "—"}
          icon={IconStar}
        />
        <StatCard label={R.lowCount} value={fmtNum(data.low)} icon={IconWarning} />
      </StatGrid>

      <section className="mt-6">
        <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
          <h2 className="font-bold">{R.driversTitle}</h2>
          <div className="w-56">
            <Select value={min} onChange={(e) => setMin(e.target.value)}>
              {["1", "3", "5", "10"].map((v) => (
                <option key={v} value={v}>
                  {R.minCount.replace("{n}", v)}
                </option>
              ))}
            </Select>
          </div>
        </div>

        {data.drivers.length === 0 ? (
          <EmptyState icon={IconDriver} title={R.noDrivers} />
        ) : (
          <ul className="divide-y divide-line rounded-card border border-line bg-surface">
            {data.drivers.map((d) => (
              <li key={d.id}>
                <button
                  onClick={() => router.push(`/dashboard/users/${d.id}`)}
                  className="flex w-full flex-wrap items-center gap-3 px-3 py-2.5 text-start hover:bg-page/60"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block font-medium">{d.name}</span>
                    <span dir="ltr" className="block text-xs text-ink-muted">
                      {d.phone}
                    </span>
                  </span>
                  <Stars value={Math.round(d.average)} />
                  {/* **المتوسّطُ برقمه** — والنجومُ تُقرَّب فتكذب قليلاً. */}
                  <span
                    dir="ltr"
                    className={`shrink-0 font-bold tabular-nums ${
                      d.average < 3 ? "text-danger" : d.average < 4 ? "text-warning" : "text-success"
                    }`}
                  >
                    {d.average.toFixed(1)}
                  </span>
                  <span className="shrink-0 text-xs text-ink-muted">
                    {R.ofCount.replace("{n}", fmtNum(d.count))}
                  </span>
                  {d.low > 0 && (
                    <Badge variant="danger">{R.lowBadge.replace("{n}", fmtNum(d.low))}</Badge>
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="mt-6">
        <h2 className="mb-3 font-bold">{R.commentsTitle}</h2>
        <p className="mb-3 text-xs text-ink-muted">{R.commentsHint}</p>
        {data.comments.length === 0 ? (
          <EmptyState icon={IconStar} title={R.noComments} />
        ) : (
          <ul className="space-y-2">
            {data.comments.map((c, i) => (
              <li key={i} className="rounded-card border border-line bg-surface p-3">
                <div className="mb-1 flex flex-wrap items-center gap-2 text-sm">
                  <span dir="ltr" className="flex items-center gap-1 font-bold tabular-nums">
                    <IconOrder size={13} />#{fmtRef(c.order_number)}
                  </span>
                  <span className="text-ink-muted">{c.customer}</span>
                  {c.driver && <Badge variant="neutral">{c.driver}</Badge>}
                  <span className="ms-auto flex items-center gap-2">
                    <span className="text-xs text-ink-muted">{R.platform}</span>
                    <Stars value={c.platform_stars} />
                    {c.driver_stars != null && (
                      <>
                        <span className="text-xs text-ink-muted">{R.driver}</span>
                        <Stars value={c.driver_stars} />
                      </>
                    )}
                  </span>
                </div>
                {c.comment && <p className="text-sm">{c.comment}</p>}
                <p dir="ltr" className="mt-1 text-xs text-ink-muted">
                  {fmtDateTime(c.created_at)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
