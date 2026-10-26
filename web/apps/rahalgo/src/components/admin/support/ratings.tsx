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
 *
 * # تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦)
 *
 * ثلاثةُ أرقامٍ تُضغط، ثمّ كلُّ التقييمات بفلاترها في بطاقةٍ واحدة، ثمّ السائقون والمتاجرُ
 * بطاقاتٍ بدائرةٍ ونجومٍ مرسومة. **ولا جدول.**
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useCanCall } from "@/lib/policy";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  PageContainer,
  Select,
  Badge,
  LoadingState,
  IconStar,
  IconDriver,
  IconWarning,
  IconOrder,
  IconStore,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { RatingsList, StarRow, EmptyCircle, starTone } from "./RatingsList";

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
  created_at: string;
}

/** **متجرٌ كما يراه السائقون** — نجمتان لا واحدة. */
interface Store {
  id: string;
  name: string;
  count: number;
  speed: number;
  conduct: number;
  low: number;
}

interface Data {
  total: number;
  average: number;
  has_average: boolean;
  low: number;
  drivers: Party[];
  stores: Store[];
  comments: Comment[];
}

/** لونُ دائرة السائق أو المتجر بمتوسّطه. */
function avatarTint(avg: number): string {
  return avg < 3
    ? "bg-danger-solid text-on-solid"
    : avg < 4
      ? "bg-warning-solid text-on-solid"
      : "bg-success-solid text-on-solid";
}

export function RatingsView() {
  const router = useRouter();
  // **وملفُّ الحساب لمن يقرأ الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥): موظّفُ العمليّات
  // لا يملك `users.read` — **فلا يُرسَم له رابطٌ يُردّ ٤٠٣.**
  const canProfile = useCanCall()("GET", "/users/{id}");
  const [min, setMin] = useState("3");
  const [low, setLow] = useState(false);
  const [data, setData] = useState<Data | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await api<Data>(`/api/v1/admin/ratings?min=${min}`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [min]);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <p className="py-10 text-center text-danger">{error}</p>;
  if (!data) return <LoadingState />;

  return (
    <PageContainer>
      {/* **رأسٌ واحدٌ احترافيّ** (المالك ٢٠٢٦-١٠-٢٧) — متوسّطُ المنصّة كبيراً بنجومه، وعددُ التقييمات
          والضعيفةُ شارتان تُضغطان. */}
      <section className="surface mb-4 flex flex-wrap items-center gap-4 p-4">
        <span
          className={`inline-flex h-16 w-16 shrink-0 flex-col items-center justify-center rounded-full ${
            !data.has_average
              ? "bg-field text-ink-muted"
              : data.average < 3
                ? "bg-danger-solid text-on-solid"
                : data.average < 4
                  ? "bg-warning-solid text-on-solid"
                  : "bg-success-solid text-on-solid"
          }`}
        >
          <span className="figure leading-none" dir="ltr">
            {data.has_average ? data.average.toFixed(1) : "\u2014"}
          </span>
          <span className="text-2xs">{R.outOf5}</span>
        </span>
        <div className="min-w-0 flex-1 space-y-1">
          <h2 className="heading-card">{R.headTitle}</h2>
          {data.has_average && <StarRow n={data.average} />}
          <p className="text-sm text-ink-muted">{R.hint}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => setLow(false)}
            className={`rounded-full px-3 py-1.5 text-sm font-bold ${
              !low ? "bg-accent text-on-bright" : "border border-line-soft text-ink-muted hover:bg-row-hover"
            }`}
          >
            {R.headTotal.replace("{n}", fmtNum(data.total))}
          </button>
          <button
            type="button"
            onClick={() => setLow(true)}
            className={`rounded-full px-3 py-1.5 text-sm font-bold ${
              low
                ? "bg-danger-solid text-on-solid"
                : data.low > 0
                  ? "border border-danger text-danger hover:bg-row-hover"
                  : "border border-line-soft text-ink-muted"
            }`}
          >
            {R.headLow.replace("{n}", fmtNum(data.low))}
          </button>
        </div>
      </section>

      {/* **كلُّ تقييمٍ بسطره أوّلاً** (المالك ٢٠٢٦-١٠-٢٥) — ثمّ المعدّلاتُ بالسائق وبالمتجر. */}
      <RatingsList canProfile={canProfile} low={low} onLow={setLow} />

      <section className="space-y-3">
        <div className="surface flex flex-wrap items-center justify-between gap-3 p-4">
          <span className="min-w-0 flex-1">
            <h2 className="heading-section">{R.driversTitle}</h2>
            <span className="block text-xs text-ink-muted">{R.driversHint}</span>
          </span>
          <div className="w-full sm:w-56">
            <Select value={min} onChange={(e) => setMin(e.target.value)} aria-label={R.driversTitle}>
              {["1", "3", "5", "10"].map((v) => (
                <option key={v} value={v}>
                  {R.minCount.replace("{n}", v)}
                </option>
              ))}
            </Select>
          </div>
        </div>

        {data.drivers.length === 0 ? (
          <EmptyCircle icon={IconDriver} title={R.noDrivers} hint={R.noDriversHint} />
        ) : (
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {data.drivers.map((d) => (
              <li key={d.id}>
                <button
                  type="button"
                  onClick={canProfile ? () => router.push(`/dashboard/users/${d.id}`) : undefined}
                  disabled={!canProfile}
                  className={`surface flex h-full w-full flex-col gap-3 p-4 text-start enabled:hover:bg-row-hover disabled:cursor-default ${
                    d.average < 3 ? "border-danger" : ""
                  }`}
                >
                  <span className="flex w-full items-center gap-3">
                    <span
                      className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full font-bold ${avatarTint(d.average)}`}
                    >
                      {(d.name || d.phone || "?").trim().charAt(0)}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-bold">{d.name}</span>
                      <span dir="ltr" className="block text-2xs text-ink-muted">
                        {d.phone}
                      </span>
                    </span>
                    {/* **المتوسّطُ برقمه** — والنجومُ تُقرَّب فتكذب قليلاً. */}
                    <span dir="ltr" className={`figure shrink-0 ${starTone(d.average)}`}>
                      {d.average.toFixed(1)}
                    </span>
                  </span>
                  <span className="flex w-full flex-wrap items-center justify-between gap-2 rounded-control bg-field px-3 py-2">
                    <StarRow n={Math.round(d.average)} />
                    <span className="text-xs text-ink-muted">{R.ofCount.replace("{n}", fmtNum(d.count))}</span>
                    {d.low > 0 && <Badge variant="danger">{R.lowBadge.replace("{n}", fmtNum(d.low))}</Badge>}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* ══════════════════════════════════════════════════════════════
          **والمتاجرُ كما يراها السائقون**
          ══════════════════════════════════════════════════════════════

          (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)

          **والتبويبُ وُضع لسؤال «من يشكو منه الناس؟»** — وكان يجيبه عن
          السائقين وحدَهم، **فيبقى السؤالُ عن المتاجر بلا جوابٍ إلّا بفتح
          عشرين ملفّاً**، وهو ما بُني ليمنعه.

          **ونجمتان لا متوسّط**: «بطيءٌ في التجهيز» و«سيّئُ التعامل»
          عيبان لا يُعالجان بشيءٍ واحد — **ومتوسّطُهما يخفي أيَّهما هو.**

          **ولا يُرسم فارغاً** — قسمٌ صفريٌّ يُقرأ عطباً، **وهو فارغٌ حتّى
          يبدأ السائقون بالتقييم.** */}
      {data.stores.length > 0 && (
        <section className="space-y-3">
          <div>
            <h2 className="heading-section">{R.storesTitle}</h2>
            <p className="text-xs text-ink-muted">{R.storesHint}</p>
          </div>
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {data.stores.map((x) => (
              <li key={x.id}>
                <button
                  type="button"
                  onClick={() => router.push(`/dashboard/merchants/${x.id}`)}
                  className={`surface flex h-full w-full flex-col gap-3 p-4 text-start hover:bg-row-hover ${
                    Math.min(x.speed, x.conduct) < 3 ? "border-danger" : ""
                  }`}
                >
                  <span className="flex w-full items-center gap-3">
                    <span
                      className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${avatarTint(Math.min(x.speed, x.conduct))}`}
                    >
                      <IconStore size={20} />
                    </span>
                    <span className="min-w-0 flex-1 truncate font-bold">{x.name}</span>
                    {x.low > 0 && <Badge variant="danger">{R.lowBadge.replace("{n}", fmtNum(x.low))}</Badge>}
                  </span>
                  <span className="w-full space-y-1.5 rounded-control bg-field px-3 py-2 text-xs">
                    <span className="flex items-center justify-between gap-2">
                      <span className="text-ink-muted">{R.speed}</span>
                      <StarRow n={x.speed} />
                    </span>
                    <span className="flex items-center justify-between gap-2">
                      <span className="text-ink-muted">{R.conduct}</span>
                      <StarRow n={x.conduct} />
                    </span>
                  </span>
                  <span className="text-2xs text-ink-muted">{R.ofCount.replace("{n}", fmtNum(x.count))}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="space-y-3">
        <div>
          <h2 className="heading-section">{R.lowTitle}</h2>
          <p className="text-xs text-ink-muted">{R.lowHint}</p>
        </div>
        {data.comments.length === 0 ? (
          <EmptyCircle icon={IconStar} title={R.noLow} hint={R.noLowHint} />
        ) : (
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {data.comments.map((c, i) => (
              <li key={i} className="surface flex flex-col gap-2 border-danger p-4 text-sm">
                <div className="flex items-start justify-between gap-2">
                  <span className="min-w-0">
                    <span dir="ltr" className="flex items-center gap-1 font-bold tabular-nums">
                      <IconOrder size={13} />#{fmtRef(c.order_number)}
                    </span>
                    <span className="block truncate text-ink-muted">{c.customer}</span>
                  </span>
                  {c.driver && <Badge variant="neutral">{c.driver}</Badge>}
                </div>
                <div className="space-y-1 rounded-control bg-field px-3 py-2 text-xs">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-ink-muted">{R.platform}</span>
                    <StarRow n={c.platform_stars} />
                  </div>
                  {c.driver_stars != null && (
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-ink-muted">{R.driver}</span>
                      <StarRow n={c.driver_stars} />
                    </div>
                  )}
                </div>
                <p dir="ltr" className="text-2xs text-ink-muted">
                  {fmtDateTime(c.created_at)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </PageContainer>
  );
}
