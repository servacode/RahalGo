"use client";

/**
 * الإعدادات — أخطر شاشة في المنصة.
 *
 * كانت محرّر JSON خاماً: اسمٌ لاتينيّ (`drivers.share_value`) وقيمةٌ في مربّع
 * نصٍّ حرّ. ويُطلب من صاحب المنصة — رجلٍ في الرقة لا مبرمج — أن يكتب JSON
 * صحيحاً في حقلٍ يحكم رواتب سائقيه. وحرفٌ زائد يكسر خطّ التوصيل كلَّه.
 *
 * والتعريف كلُّه من الخادم: نوع الحقل ومداه وخياراته. **المدى الذي يحرسه
 * الخادم هو المدى الذي يعرضه الحقل** — ولو كُتب هنا لانحرف عنه يوماً، فيرى
 * المالك حقلاً يقبل ما يرفضه الحفظ.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  PageHeader, Button, Input, Select, Checkbox, Badge, Card,
  IconSettings, IconWarning, IconCheck,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const S = m.admin.settings;

type Kind = "int" | "money" | "bool" | "choice" | "text";

interface Setting {
  key: string;
  group: string;
  kind: Kind;
  min?: number;
  max?: number;
  options?: string[];
  unit?: string;
  default: unknown;
  sensitive?: boolean;
  value: unknown;
  updated_at: string | null;
  updated_by: string | null;
}

const label = (k: string) =>
  (S.keys as Record<string, { label: string; hint: string }>)[k]?.label ?? k;
const hint = (k: string) =>
  (S.keys as Record<string, { label: string; hint: string }>)[k]?.hint ?? "";
const unitText = (u?: string) => (u ? (S.units as Record<string, string>)[u] ?? "" : "");
const choiceText = (c: string) => (S.choices as Record<string, string>)[c] ?? c;

export default function SettingsPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const [list, setList] = useState<Setting[] | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setList(await api<Setting[]>("/api/v1/admin/settings"));
      setError("");
    } catch {
      setError(m.errors.internal);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // **نمطُ الهامش يُقرأ مرّةً للصفحة** — تتبعه لافتةُ حقل القيمة.
  const marginMode = String(
    list?.find((x) => x.key === "pricing.margin_mode")?.value ?? "percent",
  );

  // المجموعات بترتيب الخادم لا بترتيب أبجديّ: المفاتيح مجموعةٌ بالموضوع،
  // وبعثرتُها تفصل «مهلة القبول» عن «مهلة التوصيل».
  const groups = useMemo(() => {
    if (!list) return [];
    const seen: string[] = [];
    for (const s of list) if (!seen.includes(s.group)) seen.push(s.group);
    return seen.map((g) => ({ g, items: list.filter((s) => s.group === g) }));
  }, [list]);

  if (!list) return <p className="p-6 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div>
      <PageHeader icon={IconSettings} title={m.admin.settingsPage.title} />
      <p className="mb-6 text-sm text-ink-muted">{m.admin.settingsPage.hint}</p>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <div className="space-y-6">
        {groups.map(({ g, items }) => (
          <section key={g}>
            <h2 className="mb-2 font-bold">
              {(S.groups as Record<string, string>)[g] ?? g}
            </h2>
            <div className="space-y-3">
              {items.map((s) => (
                <SettingRow key={s.key} s={s} editable={isAdmin} onSaved={load} marginMode={marginMode} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

/**
 * صفٌّ واحد — يحرّر نفسه في مكانه.
 *
 * ولا نافذة منبثقة: النافذة تُخفي بقية الإعدادات، ومن يضبط «مهلة القبول» يريد
 * أن يرى «مهلة السائق» وهو يضبطها. والحفظ لكل صفٍّ على حدة لا زرّ واحد في
 * الأسفل — كي لا يحفظ من غيّر رقماً واحداً عشرين رقماً بلا قصد.
 */
function SettingRow({
  s,
  editable,
  onSaved,
  marginMode,
}: {
  s: Setting;
  editable: boolean;
  onSaved: () => void;
  /** نمطُ الهامش — **تتبعه لافتةُ حقل القيمة**: «٪» أو «ل.س». */
  marginMode: string;
}) {
  const [draft, setDraft] = useState<string>(() => String(s.value ?? ""));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    setDraft(String(s.value ?? ""));
  }, [s.value]);

  const numeric = s.kind === "int" || s.kind === "money";
  /**
   * **حسابُ الخزينة يُختار من قائمة لا يُكتب معرّفُه بالحروف.**
   *
   * كان حقلَ نصّ يطلب من المالك أن يبحث عن معرّفٍ في مكانٍ آخر ويلصقه —
   * **وحرفٌ ناقصٌ فيه يعني خزينةً على حسابٍ لا وجود له، فتمضي المنصةُ بلا
   * دفترٍ ولا تقول شيئاً.**
   *
   * ولا يُعرض إلّا الأدمن والمالية: **الخزينةُ على حساب سائقٍ ليست خطأً
   * يُكتشف، هي مالٌ يُقيَّد لمن لا يخصّه.**
   */
  const isTreasury = s.key === "platform.treasury_user_id";
  const [holders, setHolders] = useState<{ id: string; full_name: string; phone: string }[]>([]);
  useEffect(() => {
    if (!isTreasury) return;
    void (async () => {
      try {
        // **نقطةٌ مخصّصة**: قائمةُ المستخدمين تستبعد الأدمن عمداً، وتوسيعُها
        // لأجل هذه القائمة يُضعف حارساً قائماً لأجل راحةٍ عابرة.
        const res = await api<{ users: { id: string; full_name: string; phone: string }[] }>(
          "/api/v1/admin/treasury-candidates",
        );
        setHolders(res.users ?? []);
      } catch {
        setHolders([]);
      }
    })();
  }, [isTreasury]);
  const dirty =
    s.kind === "bool" ? false : draft !== String(s.value ?? "");

  async function save(raw: unknown) {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/settings/${s.key}`, {
        method: "PUT",
        body: JSON.stringify({ value: raw }),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      onSaved();
    } catch (err) {
      // الخادم يردّ `validation` لكل رفض — والمدى معروفٌ هنا، فنقول السبب
      setError(
        err instanceof ApiError && numeric
          ? S.range.replace("{min}", fmtNum(s.min ?? 0)).replace("{max}", fmtNum(s.max ?? 0))
          : m.errors.validation,
      );
    } finally {
      setBusy(false);
    }
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (numeric) {
      const n = Number(draft);
      if (!Number.isInteger(n)) return setError(m.errors.validation);
      // الفحص قبل الشبكة: رحلةٌ إلى الخادم لتُردّ برسالة نعرفها سلفاً بطءٌ بلا سبب
      if ((s.min !== undefined && n < s.min) || (s.max !== undefined && n > s.max)) {
        return setError(
          S.range.replace("{min}", fmtNum(s.min ?? 0)).replace("{max}", fmtNum(s.max ?? 0)),
        );
      }
      return void save(n);
    }
    void save(draft);
  }

  return (
    <Card>
      <form onSubmit={submit}>
        <div className="mb-1 flex flex-wrap items-center gap-2">
          <span className="font-medium">{label(s.key)}</span>
          {s.sensitive && (
            <Badge variant="warning">
              <span className="flex items-center gap-1">
                <IconWarning size={12} />
                {S.sensitive}
              </span>
            </Badge>
          )}
          {saved && (
            <Badge variant="success">
              <span className="flex items-center gap-1">
                <IconCheck size={12} strokeWidth={3} />
                {S.saved}
              </span>
            </Badge>
          )}
        </div>
        <p className="mb-3 text-xs leading-relaxed text-ink-muted">{hint(s.key)}</p>

        <div className="flex flex-wrap items-end gap-2">
          {s.kind === "bool" ? (
            <Checkbox
              id={s.key}
              label={label(s.key)}
              checked={s.value === true}
              disabled={!editable || busy}
              onChange={(e) => void save(e.target.checked)}
            />
          ) : isTreasury ? (
            <Select
              id={s.key}
              value={draft}
              disabled={!editable || busy}
              onChange={(e) => {
                setDraft(e.target.value);
                void save(e.target.value);
              }}
              className="min-w-64"
            >
              {/* **الفراغُ خيارٌ صريح**: «لم تُختَر بعد» حالةٌ يعمل فيها كلُّ
                  شيء ويبقى الدفترُ ناقصَ طرف — وإخفاؤها يجعل أوّلَ فتحٍ
                  للصفحة يختار حساباً بلا قصد. */}
              <option value="">{m.admin.settings.treasuryNone}</option>
              {holders.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.full_name || u.phone}
                </option>
              ))}
            </Select>
          ) : s.kind === "choice" ? (
            <Select
              id={s.key}
              value={draft}
              disabled={!editable || busy}
              onChange={(e) => {
                setDraft(e.target.value);
                void save(e.target.value);
              }}
              className="min-w-52"
            >
              {(s.options ?? []).map((o) => (
                <option key={o} value={o}>
                  {choiceText(o)}
                </option>
              ))}
            </Select>
          ) : (
            <>
              <Input
                id={s.key}
                type={numeric ? "number" : "text"}
                inputMode={numeric ? "numeric" : undefined}
                min={s.min}
                max={s.max}
                value={draft}
                disabled={!editable || busy}
                onChange={(e) => {
                  setDraft(e.target.value);
                  setError("");
                }}
                className={numeric ? "w-40" : "w-full"}
              />
              {/* **ولافتةُ الهامش تتبع نمطَه.**

                  مفتاحٌ واحدٌ بمعنيين: «٣٠٠٠» ثلاثةُ آلاف ليرةٍ في الثابت،
                  **وواحدٌ وثلاثون ضعفاً في النسبة.** وقد وقع فعلاً في تجربةٍ
                  حيّة: كُتب ٣٠٠٠ قصداً للّيرة **فبِيع طلبٌ تكلفتُه ٦٥ ألفاً
                  بمليونين** — والحقلُ لا يقول أيَّهما يُكتب. */}
              {s.key === "pricing.margin_value" ? (
                <span className="pb-2 text-sm font-medium text-primary-dark">
                  {marginMode === "fixed" ? m.common.currency : "%"}
                </span>
              ) : (
                s.unit && (
                  <span className="pb-2 text-sm text-ink-muted">{unitText(s.unit)}</span>
                )
              )}
              {editable && dirty && (
                <Button type="submit" disabled={busy}>
                  {m.common.save}
                </Button>
              )}
            </>
          )}
        </div>

        {numeric && s.min !== undefined && s.max !== undefined && (
          <p className="mt-1.5 text-xs text-ink-muted">
            {S.range.replace("{min}", fmtNum(s.min)).replace("{max}", fmtNum(s.max))}
            {" · "}
            {S.defaultIs.replace("{v}", fmtNum(Number(s.default)))}
          </p>
        )}

        {error && <p className="mt-1.5 text-xs text-danger">{error}</p>}

        {/* من غيّره ومتى: إعدادٌ يحكم المال يجب أن يُعرف صاحبُ قراره */}
        <p className="mt-2 text-xs text-ink-muted">
          {s.updated_at
            ? S.lastChange
                .replace("{who}", s.updated_by ?? m.admin.audit.system)
                .replace("{when}", fmtDateTime(s.updated_at))
            : S.neverChanged}
        </p>

        {s.sensitive && dirty && (
          <p className="mt-2 rounded-control bg-warning/10 px-3 py-2 text-xs text-warning">
            {S.sensitiveHint}
          </p>
        )}
      </form>
    </Card>
  );
}
