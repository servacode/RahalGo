"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الدوام والعدّاد** — صفحة مراقبة السائقين (قرارات المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * بطاقةٌ لكلّ دوام: متى فُتح ومتى سُكّر ومين سكّره، وصورتا العدّاد ورقماهما، و**حسب العدّاد**
 * مقابل **حسب التطبيق** (قيادةٌ فعليّة: بالطلبات · بلا طلب)، و**مشى وهو خارج الدوام** (من سكّر
 * الليل لفتح الصبح — «لازم يكون صفر»). **والأحمرُ** حين يفترق العدّادُ عن التطبيق كثيراً، أو
 * سُكّر الدوامُ بلا صورة («ما صوّر عند الإغلاق»).
 *
 * **والأرقامُ كلُّها من الخادم** — هنا عرضٌ فقط، وتصحيحُ رقمٍ غلط فيه السائق بسببٍ مكتوب.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDamascusTime, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, FormActions, ImageViewer, Input, Modal } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { distText } from "@/components/admin/orders/CardSections";

const m = getMessages(defaultLocale);
const S = m.admin.driverMonitor.shifts;

/** **قراءةُ عدّاد** — الرقمُ كما كتبه، وتصحيحُ المكتب إن وُجد، وصورتُها موقَّعة. */
export interface OdoReading {
  id: string;
  km: number;
  corrected_km: number | null;
  effective_km: number;
  check_status: string;
  correction_note: string;
  photo_url: string | null;
  thumb_url: string | null;
  created_at: string;
  app_build: number | null;
}

export interface MonitorShift {
  id: string;
  opened_at: string;
  closed_at: string | null;
  close_reason: string;
  opened_build: number | null;
  migrated: boolean;
  drive_m: number;
  orders_m: number;
  no_order_m: number;
  not_driving_m: number;
  gap_min: number;
  orders_n: number;
  final: boolean;
  open_hours?: number;
  // ── العدّاد (يملؤه الخادم حين تكون القراءات) ──
  start_reading?: OdoReading | null;
  end_reading?: OdoReading | null;
  odometer_km?: number | null;
  approx_km?: number | null;
  off_before_km?: number | null;
  unseen_km?: number | null;
  exempt_by_name?: string;
  exempt_reason?: string;
  flags?: string[];
}

const km1 = (n: number) => S.km.replace("{n}", fmtNum(Math.round(n * 10) / 10));

/** **الأعلامُ الحمراء** — ما يُقرأ تنبيهاً لا معلومة. */
const RED = new Set(["big_gap", "no_close_photo", "off_duty_moved"]);

function Line({ label, value, tone }: { label: string; value: React.ReactNode; tone?: "danger" }) {
  return (
    <div className="flex items-center justify-between gap-2 text-sm">
      <span className="text-ink-muted">{label}</span>
      <span className={`font-bold ${tone === "danger" ? "text-danger" : ""}`}>{value}</span>
    </div>
  );
}

function ReadingBox({
  title,
  r,
  canCorrect,
  onCorrect,
}: {
  title: string;
  r: OdoReading | null | undefined;
  canCorrect: boolean;
  onCorrect: (r: OdoReading) => void;
}) {
  const [open, setOpen] = useState(false);
  if (!r) {
    return (
      <div className="rounded-control border border-line-soft p-2 text-sm">
        <p className="text-xs font-bold">{title}</p>
        <p className="text-ink-muted">{S.noPhoto}</p>
      </div>
    );
  }
  return (
    <div className="space-y-1 rounded-control border border-line-soft p-2 text-sm">
      <p className="text-xs font-bold">{title}</p>
      {r.thumb_url ? (
        <button type="button" onClick={() => setOpen(true)} className="block w-full" aria-label={S.openPhoto}>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={r.thumb_url} alt={S.openPhoto} className="h-24 w-full rounded-control object-cover" />
        </button>
      ) : (
        <p className="text-ink-muted">{S.photoGone}</p>
      )}
      <p className="font-bold">{S.reading.replace("{n}", fmtNum(r.effective_km))}</p>
      {r.corrected_km != null && (
        <p className="text-xs text-ink-muted">
          {S.corrected.replace("{old}", fmtNum(r.km)).replace("{note}", r.correction_note || "—")}
        </p>
      )}
      <p className="text-xs text-ink-muted">{fmtDamascusTime(r.created_at)}</p>
      {r.check_status === "jump_confirmed" && <Badge variant="warning">{S.flags.jump_confirmed}</Badge>}
      {canCorrect && (
        <Button variant="secondary" size="sm" onClick={() => onCorrect(r)}>
          {S.correct}
        </Button>
      )}
      {open && r.photo_url && <ImageViewer src={r.photo_url} alt={S.openPhoto} onClose={() => setOpen(false)} />}
    </div>
  );
}

/** **«صحّح الرقم»** — رقمٌ غلط فيه السائق بالكتابة، والأصلُ يبقى ويُسجَّل من صحّح وليش. */
function CorrectDialog({ r, onClose, onDone }: { r: OdoReading; onClose: () => void; onDone: () => void }) {
  const [km, setKm] = useState(String(r.effective_km));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <Modal open onClose={onClose} title={S.correctTitle}>
      <form
        className="space-y-3"
        onSubmit={(ev) => {
          ev.preventDefault();
          if (!note.trim()) {
            setError(S.correctNeedNote);
            return;
          }
          setBusy(true);
          setError("");
          void (async () => {
            try {
              await api(`/api/v1/admin/driver-odometer/${r.id}`, {
                method: "PATCH",
                body: JSON.stringify({ km: Number(km), note: note.trim() }),
              });
              onDone();
            } catch (err) {
              setError(errorText(err));
              setBusy(false);
            }
          })();
        }}
      >
        <p className="text-sm text-ink-muted">{S.correctHint.replace("{n}", fmtNum(r.km))}</p>
        <Input
          id="odo-km"
          label={S.correctKm}
          inputMode="decimal"
          dir="ltr"
          required
          value={km}
          onChange={(ev) => setKm(ev.target.value)}
        />
        <Input
          id="odo-note"
          label={S.correctNote}
          required
          maxLength={300}
          value={note}
          onChange={(ev) => setNote(ev.target.value)}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={S.correctSave} />
      </form>
    </Modal>
  );
}

function ShiftCard({ s, canCorrect, onChanged }: { s: MonitorShift; canCorrect: boolean; onChanged: () => void }) {
  const [fixing, setFixing] = useState<OdoReading | null>(null);
  const reasons = S.reasons as Record<string, string>;
  const flags = S.flags as Record<string, string>;
  const red = (s.flags ?? []).some((f) => RED.has(f));
  return (
    <div className={`surface space-y-2 p-3 ${red ? "border-danger-edge" : ""}`}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="font-bold">{S.opened.replace("{time}", fmtDamascusTime(s.opened_at))}</span>
        <span className="text-sm text-ink-muted">
          {s.closed_at ? S.closed.replace("{time}", fmtDamascusTime(s.closed_at)) : S.open}
        </span>
      </div>
      {s.closed_at && s.close_reason && (
        <p className="text-xs text-ink-muted">{reasons[s.close_reason] ?? s.close_reason}</p>
      )}
      {(s.flags ?? []).length > 0 && (
        <div className="flex flex-wrap gap-1">
          {(s.flags ?? []).map((f) => (
            <Badge key={f} variant={RED.has(f) ? "danger" : "warning"}>
              {(flags[f] ?? f).replace("{h}", fmtNum(s.open_hours ?? 0))}
            </Badge>
          ))}
        </div>
      )}
      <div className="grid grid-cols-2 gap-2">
        <ReadingBox title={S.startPhoto} r={s.start_reading} canCorrect={canCorrect} onCorrect={setFixing} />
        <ReadingBox title={S.endPhoto} r={s.end_reading} canCorrect={canCorrect} onCorrect={setFixing} />
      </div>
      <div className="space-y-1 rounded-control bg-field px-3 py-2">
        {s.odometer_km != null && <Line label={S.odo} value={km1(s.odometer_km)} />}
        {s.odometer_km == null && s.approx_km != null && (
          <Line label={S.approx} value={km1(s.approx_km)} />
        )}
        <Line label={S.app} value={distText(s.drive_m)} />
        <Line label={S.orders.replace("{n}", fmtNum(s.orders_n))} value={distText(s.orders_m)} />
        <Line label={S.noOrder} value={distText(s.no_order_m)} />
        {s.unseen_km != null && (
          <Line
            label={S.unseen}
            value={km1(s.unseen_km)}
            tone={(s.flags ?? []).includes("big_gap") ? "danger" : undefined}
          />
        )}
        <Line label={S.notDriving} value={distText(s.not_driving_m)} />
        {s.gap_min > 0 && <Line label={S.gap} value={S.minutes.replace("{n}", fmtNum(s.gap_min))} />}
        {s.off_before_km != null && (
          <Line
            label={S.offShift}
            value={km1(s.off_before_km)}
            tone={(s.flags ?? []).includes("off_duty_moved") ? "danger" : undefined}
          />
        )}
        {s.exempt_by_name && (
          <p className="text-xs text-ink-muted">
            {S.exempt.replace("{name}", s.exempt_by_name).replace("{reason}", s.exempt_reason || "—")}
          </p>
        )}
      </div>
      {fixing && (
        <CorrectDialog
          r={fixing}
          onClose={() => setFixing(null)}
          onDone={() => {
            setFixing(null);
            onChanged();
          }}
        />
      )}
    </div>
  );
}

/** **الدوام والعدّاد** — بطاقةٌ لكلّ دوامٍ مسّ اليوم. */
export function ShiftsSection({
  shifts,
  canCorrect,
  onChanged,
}: {
  shifts: MonitorShift[];
  canCorrect: boolean;
  onChanged: () => void;
}) {
  return (
    <div className="surface mt-3 p-3">
      <h2 className="mb-1 font-bold">{S.title}</h2>
      <p className="mb-2 text-xs text-ink-muted">{S.hint}</p>
      {shifts.length === 0 ? (
        <p className="text-sm text-ink-muted">{S.empty}</p>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {shifts.map((s) => (
            <ShiftCard key={s.id} s={s} canCorrect={canCorrect} onChanged={onChanged} />
          ))}
        </div>
      )}
    </div>
  );
}
