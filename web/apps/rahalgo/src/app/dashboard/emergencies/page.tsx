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

import { useState } from "react";
import { getMessages, defaultLocale, fmtRef, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Modal,
  Tabs,
  Textarea,
  PageContainer,
  Pagination,
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
  /** **وماذا جرى** — (قرارُ المالك ٢٠٢٦-٠٨-١٦)، في المُعالَج وحدَه. */
  resolution: string;
  resolved_at: string | null;
  resolved_by: string;
}

export default function EmergenciesPage() {
  /** **الصفحةُ المعروضة** — (قرارُ المالك ٢٠٢٦-٠٨-١٠).

      **والمفتوحُ منها لا يُقفل نفسَه**: يتراكم حتّى تعالجه العملياتُ سطراً
      سطراً. **ومئةٌ صامتةٌ تعني أنّ سائقاً في ضائقةٍ لا يراه أحد** — وهو
      آخرُ ما يُحتمل صمتُه في هذه المنصّة. */
  const [page, setPage] = useState(1);
  /** ══════════════════════════════════════════════════════════════════
   *  **والمُعالَجُ تبويبٌ — وهو ما وُعدت به الشاشةُ ولم تفِ**
   *  ══════════════════════════════════════════════════════════════════
   *
   *  (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
   *
   *  **مكتوبٌ في رأسها**: «من سأل عنه بعد يومين لم يجد من يقول ماذا
   *  جرى». **وكانت تعرض المفتوحَ وحدَه** — فما إن يُغلق حتّى يختفي،
   *  **وهو عينُ ما كُتبت لمنعه.**
   */
  const [tab, setTab] = useState<"open" | "resolved">("open");
  /** **والبلاغُ يُغلق بكلمة** — لا بضغطةٍ صامتة. */
  const [closing, setClosing] = useState<Emergency | null>(null);
  const [error, setError] = useState("");
  const { data, reload } = useLiveData<{
    emergencies: Emergency[];
    total: number;
    per_page: number;
  }>(
    () => api(`/api/v1/admin/emergencies?page=${page}&status=${tab}`),
    ["driver", "order"],
    [page, tab],
  );
  if (!data) return <LoadingState />;
  const list = data.emergencies ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconWarning} title={E.title} subtitle={E.hint} />

      {/* **والمفتوحُ أوّلاً** — هو ما تُفتح الشاشةُ لأجله. */}
      <Tabs
        items={[
          { key: "open" as const, label: E.tabOpen, icon: IconWarning },
          { key: "resolved" as const, label: E.tabResolved, icon: IconStatus },
        ]}
        value={tab}
        onChange={(k) => {
          setTab(k);
          setPage(1);
        }}
        className="mb-4"
      />
      {error && <Alert className="mb-3">{error}</Alert>}

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
                        #{fmtRef(x.order_number)}
                      </span>
                    )}
                  </p>
                  {x.note && <p className="mt-1 text-sm">{x.note}</p>}
                  <p className="mt-1 text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(x.created_at)}
                  </p>
                  {/* **وماذا جرى** — **وبلاغٌ يُقال «أُغلق» ولا يُقال كيف
                      يُنسى كما لو لم يُغلق.** */}
                  {x.resolved_at && (
                    <p className="mt-1 text-xs text-success">
                      {x.resolution || E.noResolution}
                      <span className="ms-2 text-ink-muted">
                        {E.by}: {x.resolved_by || "—"} · {fmtDateTime(x.resolved_at)}
                      </span>
                    </p>
                  )}
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
                  {tab === "open" && (
                    <Button variant="secondary" onClick={() => setClosing(x)}>
                      {E.resolve}
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* **والترقيمُ من المكوّن المشترك** — ولا يظهر لصفحةٍ واحدة. */}
      {data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}
      {closing && (
        <ResolveModal
          item={closing}
          onClose={() => setClosing(null)}
          onDone={() => {
            setClosing(null);
            setError("");
            reload();
          }}
          onError={setError}
        />
      )}
    </PageContainer>
  );
}

/**
 * **إغلاقُ البلاغ — بكلمةٍ تقول ماذا جرى.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
 *
 * **والشاشةُ تسأل**: «أوصل الطلبَ غيرُه؟ أطمأنّ السائق؟ أدُفع له شيء؟» —
 * **ولم يكن ثمّة موضعٌ للجواب.**
 *
 * **وفشلُ الإغلاق يُقال**: كان النداءُ بلا مِسكة — **فيضغط المكتبُ ولا يقع
 * شيء، فيظنّ الزرَّ معطّلاً**، والبلاغُ مفتوحٌ وهو يحسبه مغلقاً.
 */
function ResolveModal({
  item,
  onClose,
  onDone,
  onError,
}: {
  item: Emergency;
  onClose: () => void;
  onDone: () => void;
  onError: (msg: string) => void;
}) {
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit() {
    if (busy) return;
    setBusy(true);
    try {
      await api(`/api/v1/admin/emergencies/${item.id}/resolve`, {
        method: "POST",
        body: JSON.stringify({ resolution: text.trim() }),
      });
      onDone();
    } catch (err) {
      onError(errorText(err));
      onClose();
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={E.resolveTitle}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{E.resolveHint}</p>
        <Textarea
          id="emg-resolution"
          label={E.resolution}
          rows={3}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <div className="flex gap-2">
          <Button variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          {/* **والكلمةُ مطلوبةٌ هنا** — **وإغلاقٌ بلا كلمةٍ يُعيد المسألةَ
              إلى ما كانت.** */}
          <Button disabled={busy || !text.trim()} onClick={() => void submit()}>
            {busy ? m.common.loading : E.resolve}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
