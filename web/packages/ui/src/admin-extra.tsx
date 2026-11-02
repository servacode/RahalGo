"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عُدّةُ اللوحة — الجزءُ الثاني** (إعادةُ تصميم اللوحة، ٢٠٢٦-١١-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * الجزءُ الأوّل (`admin.tsx`) أعطى الصفحةَ قالبَها. وهذا الجزءُ يعطي **ما تشترك فيه كلُّ
 * الصفحات ولم يكن له مكان**: الكيانُ (طلب · زبون · سائق · متجر · سحب…) ببطاقةٍ واحدةٍ
 * بهيكلٍ واحد، وشريطُ «بدّه انتباه»، وأزرارُ الاتصال، وقصّةُ الطلب، والفعلُ الخطِر بتأكيد،
 * والحالاتُ الأربع، والبحثُ السريع (Ctrl+K)، والفلاترُ المحفوظة.
 *
 *   AdminEntityCard · AdminEntityGrid     بطاقةُ كيانٍ بهيكلٍ ثابت (رأس · معلومات · قدم) — أطوالُها متساوية
 *   AttentionStrip              «بدّه منك انتباه» — كلُّ عدّادٍ رابطٌ للحلّ، والصفرُ لا يُرسم
 *   ContactButtons              اتصالٌ وواتساب من رقمٍ واحد (`tel:` لا يُبنى إلّا هنا)
 *   StoryTimeline               «شو صار بالطلب» — من عمل ماذا ومتى
 *   ConfirmedMenu               «⋯» تُنفَّذ أفعالُها بعد تأكيدٍ للخطِر منها
 *   AdminAsync                  تحميل · خطأ · فارغ · محتوى — بشكلٍ واحد
 *   CommandPalette · useCtrlK   بحثٌ سريعٌ عام
 *   useRememberedFilters        فلاترُ الصفحة كما تركتَها
 *   Presence / TONE_TEXT        ألوانٌ بمعنى واحد (من status.tsx)
 *
 * **ولا مكوّنَ هنا يعرف مساراً أو واجهةً** — الصفحةُ تُمرّر الروابطَ والبياناتِ والدوالّ.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { getMessages, defaultLocale, fmtNum, fmtClockTime } from "@rahalgo/i18n";
import { Input } from "./components";
import type { ActionMenuItem } from "./ActionMenu";
import { AdminLink, telHref, waHref } from "./adminlink";
import { AdminEmpty, AdminLoading, AdminReload, MoreMenu, type StatTone } from "./admin";
import { Timeline, type TimelineNode } from "./timeline";
import { IconPhone, IconSearch } from "./icons";
import { IconWhatsApp } from "./brand-icons";
import { TONE_TEXT } from "./status";
import { Confirm } from "./feedback";

const m = getMessages(defaultLocale);
const K = m.admin.kit;

type IconT = ComponentType<{ size?: number; className?: string }>;

// ═════════════════════════════════════════════════════════════════════
//  بطاقةُ الكيان
// ═════════════════════════════════════════════════════════════════════

export interface EntityFact {
  key: string;
  label: string;
  value: ReactNode;
  /** **القيمةُ رابطٌ** — كلُّ رقمٍ ينكبس. */
  href?: string;
  /** **يمتدّ على العمودين**. */
  wide?: boolean;
}

/**
 * **بطاقةُ الكيان — هيكلٌ واحدٌ لكلّ ما في اللوحة.**
 *
 * - **الرأس**: صورةٌ/أيقونةٌ · سطرُ تعريفٍ صغير (`eyebrow`: رقمٌ ووقت) · العنوانُ (رابطٌ إن أُعطي `href`) · الحالةُ.
 * - **المعلومات**: شبكةُ `facts` بعمودين (تسميةٌ صغيرةٌ فوق قيمة) — وفي `children` ما لا يدخل الشبكة
 *   (أصنافُ الطلب، قصّةٌ قصيرة…).
 * - **القدم**: الفعلُ التالي ظاهراً (اثنان على الأكثر في `primary`) و«⋯» لما سواه (`more`).
 *
 * **والقدمُ مثبّتٌ أسفلَ البطاقة** فتتحاذى الأقدامُ في الصفّ مهما اختلف طولُ المحتوى.
 * **والنبرةُ حافّةٌ في جهة البداية** — وهي المكانُ الوحيدُ الذي يتكلّم فيه اللونُ على البطاقة.
 * **ولا تُنزع منها معلومة** لتخفّ: ما لا يتّسع يذهب في `children` لا إلى الحذف.
 */
export function AdminEntityCard({
  tone = "neutral",
  leading,
  eyebrow,
  title,
  href,
  status,
  facts,
  children,
  primary,
  more,
  className = "",
}: {
  tone?: StatTone;
  leading?: ReactNode;
  eyebrow?: ReactNode;
  title: ReactNode;
  href?: string;
  status?: ReactNode;
  facts?: readonly EntityFact[];
  children?: ReactNode;
  /** **الفعلُ التالي** — أزرارٌ `Button size="sm"`، اثنان على الأكثر. */
  primary?: ReactNode;
  /** **النادرُ والخطِر** — يُرسم «⋯» بتأكيدٍ للخطِر. */
  more?: readonly ConfirmItem[];
  className?: string;
}) {
  const heading = href ? (
    <AdminLink href={href} className="hover:underline">
      {title}
    </AdminLink>
  ) : (
    title
  );
  return (
    <article className={`admin-panel admin-ecard ${className}`} data-tone={tone}>
      <div className="admin-ecard-head">
        <div className="flex min-w-0 items-start gap-3">
          {leading && <span className="shrink-0">{leading}</span>}
          <div className="min-w-0">
            {eyebrow && <p className="admin-meta">{eyebrow}</p>}
            <h3 className="truncate font-bold">{heading}</h3>
          </div>
        </div>
        {status && <div className="shrink-0">{status}</div>}
      </div>
      <div className="admin-ecard-body">
        {facts && facts.length > 0 && (
          <dl className="admin-facts">
            {facts.map((f) => (
              <div key={f.key} className={`min-w-0 ${f.wide ? "col-span-2" : ""}`}>
                <dt className="text-xs text-ink-muted">{f.label}</dt>
                <dd className="mt-0.5 truncate text-sm font-medium tabular-nums">
                  {f.href ? (
                    <AdminLink href={f.href} className="hover:underline">
                      {f.value}
                    </AdminLink>
                  ) : (
                    f.value
                  )}
                </dd>
              </div>
            ))}
          </dl>
        )}
        {children}
      </div>
      {(primary || (more && more.length > 0)) && (
        <div className="admin-ecard-foot">
          {primary}
          {more && more.length > 0 && (
            <span className="ms-auto">
              <ConfirmedMenu items={more} />
            </span>
          )}
        </div>
      )}
    </article>
  );
}

/** **شبكةُ بطاقاتٍ متساويةُ الارتفاع** — تتحاذى الصفوفُ والأقدام. */
export function AdminEntityGrid({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`admin-grid ${className}`}>{children}</div>;
}

// ═════════════════════════════════════════════════════════════════════
//  «⋯» بتأكيد
// ═════════════════════════════════════════════════════════════════════

export interface ConfirmItem extends ActionMenuItem {
  /** **يُسأل قبل التنفيذ**. والخطِرُ بلا `confirm` يُسأل بالصيغة العامّة — لا يُنفَّذ خطرٌ بنقرةٍ واحدة. */
  confirm?: { title: string; body?: ReactNode; confirmLabel?: string };
}

/**
 * **قائمةُ «⋯» تحرس الخطِر.** كلُّ بندٍ `danger` (أو له `confirm`) يفتح تأكيداً قبل أن يُنفَّذ،
 * و`onSelect` قد يكون غيرَ متزامنٍ فيظهر «جارٍ…» على الزرّ حتى ينتهي.
 */
export function ConfirmedMenu({ items, label }: { items: readonly ConfirmItem[]; label?: string }) {
  const [pending, setPending] = useState<ConfirmItem | null>(null);
  const [busy, setBusy] = useState(false);
  const wrapped = useMemo<ActionMenuItem[]>(
    () =>
      items.map((it) => ({
        ...it,
        onSelect: () => {
          if (it.confirm || it.danger) setPending(it);
          else it.onSelect();
        },
      })),
    [items],
  );
  const run = async () => {
    if (!pending) return;
    setBusy(true);
    try {
      await pending.onSelect();
    } finally {
      setBusy(false);
      setPending(null);
    }
  };
  const c = pending?.confirm;
  return (
    <>
      <MoreMenu items={wrapped} label={label} />
      {pending && (
        <Confirm
          open
          tone={pending.danger ? "danger" : "primary"}
          title={c?.title ?? K.sure}
          body={c?.body}
          confirmLabel={c?.confirmLabel ?? K.confirm}
          onConfirm={run}
          onCancel={() => setPending(null)}
          busy={busy}
        />
      )}
    </>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  بدّه منك انتباه
// ═════════════════════════════════════════════════════════════════════

export interface AttentionItem {
  key: string;
  label: string;
  count: number;
  /** **أين الحلّ** — الرقمُ رابطٌ إليه. */
  href: string;
  /** `danger` لما يضيع به مالٌ أو زبون الآن؛ وغيرُه أصفر. */
  tone?: "attention" | "danger";
  /** **أسماءٌ خلف الرقم** (أوّل اثنين) — «أبو علي، محمد» بدل «٢». */
  names?: readonly string[];
}

/**
 * **شريطُ «بدّه منك انتباه».** لا يُرسم فيه إلّا ما عدّادُه فوق الصفر، **وكلُّ بطاقةٍ رابطٌ للحلّ.**
 * وحين لا شيء: سطرٌ هادئ «كل شي تمام». **الأحمرُ لا يُستعمل إلّا لما فيه مالٌ أو زبونٌ يضيع.**
 */
export function AttentionStrip({
  items,
  loading = false,
  title = K.attention.title,
}: {
  items: readonly AttentionItem[];
  loading?: boolean;
  title?: string;
}) {
  const live = items.filter((i) => i.count > 0);
  return (
    <section aria-label={title} className="space-y-2">
      <h2 className="admin-h2">{title}</h2>
      {loading && live.length === 0 ? (
        <AdminLoading rows={1} />
      ) : live.length === 0 ? (
        <p className="admin-panel flex items-center gap-2 p-4 text-sm font-bold text-success">{K.attention.clear}</p>
      ) : (
        <div className="admin-attn">
          {live.map((i) => (
            <AdminLink key={i.key} href={i.href} className="block min-w-0">
              <div className="admin-attn-item" data-tone={i.tone ?? "attention"}>
                <span className="admin-figure">{fmtNum(i.count)}</span>
                <span className="truncate text-sm font-bold">{i.label}</span>
                {i.names && i.names.length > 0 && (
                  <span className="truncate text-xs text-ink-muted">{i.names.slice(0, 2).join(K.listSep)}</span>
                )}
              </div>
            </AdminLink>
          ))}
        </div>
      )}
    </section>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  الاتصال
// ═════════════════════════════════════════════════════════════════════

/**
 * **زرّا الاتصال وواتساب من رقمٍ واحد.** الرقمُ يُعرض من اليسار لليمين، وبلا رقمٍ لا يُرسم شيء.
 * **`tel:` لا يُبنى يدويّاً في صفحة** — الحارسُ يمنع، وهنا مصدرُه.
 */
export function ContactButtons({
  phone,
  name,
  whatsapp = true,
  showNumber = false,
}: {
  phone?: string | null;
  name?: string;
  whatsapp?: boolean;
  showNumber?: boolean;
}) {
  if (!phone) return null;
  const btn =
    "inline-flex h-9 min-w-9 items-center justify-center gap-1.5 rounded-control border border-line-soft px-2.5 text-sm transition-colors hover:bg-row-hover";
  return (
    <span className="inline-flex items-center gap-1.5">
      {showNumber && (
        <span dir="ltr" className="text-sm tabular-nums text-ink-muted">
          {phone}
        </span>
      )}
      <a href={telHref(phone)} className={btn} aria-label={name ? `${K.callPhone} ${name}` : K.callPhone} title={K.callPhone}>
        <IconPhone size={15} />
      </a>
      {whatsapp && (
        <a
          href={waHref(phone)}
          target="_blank"
          rel="noopener noreferrer"
          className={btn}
          aria-label={name ? `${K.whatsapp} ${name}` : K.whatsapp}
          title={K.whatsapp}
        >
          <IconWhatsApp size={15} />
        </a>
      )}
    </span>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  قصّةُ الطلب
// ═════════════════════════════════════════════════════════════════════

export interface StoryEvent {
  id: string;
  /** وقتُ الحدث (ISO). */
  at: string;
  title: ReactNode;
  /** مين عملها. */
  actor?: string;
  /** دورُه (سائق · متجر · موظف…). */
  role?: string;
  note?: string;
  tone?: TimelineNode["tone"];
  icon?: TimelineNode["icon"];
}

/**
 * **«شو صار بالطلب»** — من عمل ماذا ومتى، بترتيب الوقوع. آخرُ حدثٍ يُنبَض إن كان الطلبُ جارياً (`live`).
 * (يغلّف `Timeline` — لا خطَّ زمنيّاً ثانياً.)
 */
export function StoryTimeline({ events, live = false }: { events: readonly StoryEvent[]; live?: boolean }) {
  if (events.length === 0) return <AdminEmpty title={K.story.empty} />;
  const nodes: TimelineNode[] = events.map((e, i) => ({
    id: e.id,
    state: live && i === events.length - 1 ? "current" : "done",
    title: e.title,
    icon: e.icon,
    tone: e.tone,
    detail: (
      <>
        {(e.actor || e.role) && (
          <span>{e.actor ? K.story.by.replace("{name}", [e.actor, e.role].filter(Boolean).join(" · ")) : e.role}</span>
        )}
        {e.note && <span className="block text-ink">{e.note}</span>}
      </>
    ),
    trailing: fmtClockTime(e.at),
  }));
  return <Timeline nodes={nodes} />;
}

// ═════════════════════════════════════════════════════════════════════
//  الحالاتُ الأربع
// ═════════════════════════════════════════════════════════════════════

/**
 * **تحميل · خطأ · فارغ · محتوى** في مكانٍ واحد. **التحديثُ الصامتُ لا يُظهر التحميلَ ثانيةً**
 * (`loading` يُهمَل ما دام عندك بياناتٌ سابقة: مرّر `hasData`).
 */
export function AdminAsync({
  loading,
  error = false,
  onRetry,
  empty = false,
  emptyTitle = K.emptyList,
  emptyIcon,
  emptyAction,
  hasData = false,
  rows,
  children,
}: {
  loading: boolean;
  error?: boolean;
  onRetry?: () => void;
  empty?: boolean;
  emptyTitle?: string;
  emptyIcon?: IconT;
  emptyAction?: ReactNode;
  hasData?: boolean;
  rows?: number;
  children: ReactNode;
}) {
  if (loading && !hasData) return <AdminLoading rows={rows} />;
  if (error && !hasData) return <AdminReload onRetry={onRetry} />;
  if (empty) return <AdminEmpty icon={emptyIcon} title={emptyTitle} action={emptyAction} />;
  return <>{children}</>;
}

// ═════════════════════════════════════════════════════════════════════
//  الفلاترُ المحفوظة
// ═════════════════════════════════════════════════════════════════════

/**
 * **فلاترُ الصفحة كما تركتَها.** تُحفظ في المتصفّح بمفتاح الصفحة، وتُستعاد عند الفتح.
 * `ready` يصير `true` بعد القراءة — **لا تجلبْ قبله** وإلّا جلبتَ بالافتراضيّ ثم أعدتَ بالمحفوظ.
 * قيمٌ نصّيّةٌ فقط (كما في الرابط). والرابطُ إن حمل فلتراً يغلب المحفوظَ: مرّر `initial` منه.
 */
export function useRememberedFilters<T extends Record<string, string>>(
  page: string,
  defaults: T,
  initial?: Partial<T>,
): { filters: T; set: (patch: Partial<T>) => void; reset: () => void; ready: boolean; active: number } {
  const key = `rg-admin-filters:${page}`;
  const defs = useRef(defaults);
  const [filters, setFilters] = useState<T>({ ...defaults, ...initial });
  const [ready, setReady] = useState(false);
  const hasInitial = !!initial && Object.keys(initial).length > 0;
  useEffect(() => {
    if (!hasInitial) {
      try {
        const raw = window.localStorage.getItem(key);
        if (raw) setFilters({ ...defs.current, ...(JSON.parse(raw) as Partial<T>) });
      } catch {
        /* التخزينُ غيرُ متاح — نبقى على الافتراضيّ */
      }
    }
    setReady(true);
  }, [key, hasInitial]);
  const persist = useCallback(
    (next: T) => {
      try {
        window.localStorage.setItem(key, JSON.stringify(next));
      } catch {
        /* لا شيء */
      }
    },
    [key],
  );
  const set = useCallback(
    (patch: Partial<T>) =>
      setFilters((cur) => {
        const next = { ...cur, ...patch };
        persist(next);
        return next;
      }),
    [persist],
  );
  const reset = useCallback(() => {
    setFilters({ ...defs.current });
    try {
      window.localStorage.removeItem(key);
    } catch {
      /* لا شيء */
    }
  }, [key]);
  const active = Object.keys(defs.current).filter((k) => filters[k] !== defs.current[k]).length;
  return { filters, set, reset, ready, active };
}

// ═════════════════════════════════════════════════════════════════════
//  البحثُ السريع
// ═════════════════════════════════════════════════════════════════════

export interface PaletteResult {
  key: string;
  /** عنوانُ المجموعة (طلبات · ناس · متاجر · صفحات). */
  group: string;
  title: ReactNode;
  meta?: ReactNode;
  href: string;
  icon?: IconT;
}

/** **Ctrl+K (و⌘K)** يفتح البحث من أيّ صفحة. */
export function useCtrlK(onOpen: () => void) {
  const cb = useRef(onOpen);
  cb.current = onOpen;
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        cb.current();
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, []);
}

/**
 * **لوحةُ البحث السريع.** حقلٌ واحد؛ النتائجُ مجمّعةٌ بعناوين؛ ↑↓ تتنقّل وEnter تفتح وEsc تُغلق.
 * **المكوّنُ لا يعرف من أين النتائج**: `search(q)` تجمعها الصفحةُ من الواجهات الموجودة، و`onPick` ينقل.
 * يبدأ البحثُ من حرفين، وبتأخيرٍ قصيرٍ كي لا يُطلب مع كلّ حرف.
 */
export function CommandPalette({
  open,
  onClose,
  search,
  onPick,
}: {
  open: boolean;
  onClose: () => void;
  search: (q: string) => Promise<PaletteResult[]>;
  onPick: (href: string) => void;
}) {
  const [q, setQ] = useState("");
  const [rows, setRows] = useState<PaletteResult[]>([]);
  const [busy, setBusy] = useState(false);
  const [sel, setSel] = useState(0);
  const seq = useRef(0);
  const P = K.palette;

  useEffect(() => {
    if (!open) {
      setQ("");
      setRows([]);
      setSel(0);
    }
  }, [open]);

  useEffect(() => {
    const term = q.trim();
    if (term.length < 2) {
      setRows([]);
      setBusy(false);
      return;
    }
    const mine = ++seq.current;
    setBusy(true);
    const t = window.setTimeout(() => {
      search(term)
        .then((r) => {
          if (mine === seq.current) {
            setRows(r);
            setSel(0);
          }
        })
        .catch(() => {
          if (mine === seq.current) setRows([]);
        })
        .finally(() => {
          if (mine === seq.current) setBusy(false);
        });
    }, 250);
    return () => window.clearTimeout(t);
  }, [q, search]);

  if (!open || typeof document === "undefined") return null;

  const pick = (r: PaletteResult) => {
    onPick(r.href);
    onClose();
  };
  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    else if (e.key === "ArrowDown") {
      e.preventDefault();
      setSel((s) => Math.min(rows.length - 1, s + 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSel((s) => Math.max(0, s - 1));
    } else if (e.key === "Enter" && rows[sel]) {
      e.preventDefault();
      pick(rows[sel]);
    }
  };

  let lastGroup = "";
  return createPortal(
    <div
      className="fixed inset-0 z-[80] flex items-start justify-center scrim p-4 pt-[12vh]"
      role="dialog"
      aria-modal="true"
      aria-label={P.open}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      onKeyDown={onKey}
    >
      <div className="admin-cmd">
        <div className="border-b border-line-soft p-3">
          <Input
            id="admin-cmd-input"
            type="search"
            autoFocus
            icon={<IconSearch size={15} />}
            aria-label={P.open}
            placeholder={P.placeholder}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto" role="listbox">
          {q.trim().length < 2 ? (
            <p className="p-4 text-sm text-ink-muted">{P.hint}</p>
          ) : busy && rows.length === 0 ? (
            <div className="p-4">
              <AdminLoading rows={2} />
            </div>
          ) : rows.length === 0 ? (
            <p className="p-4 text-sm text-ink-muted">{P.none}</p>
          ) : (
            rows.map((r, i) => {
              const head = r.group !== lastGroup ? r.group : "";
              lastGroup = r.group;
              const Icon = r.icon;
              return (
                <div key={r.key}>
                  {head && <p className="px-4 pt-3 pb-1 text-xs font-bold text-ink-muted">{head}</p>}
                  <button
                    type="button"
                    role="option"
                    aria-selected={i === sel}
                    className="admin-cmd-item"
                    onMouseEnter={() => setSel(i)}
                    onClick={() => pick(r)}
                  >
                    {Icon && <Icon size={16} className="shrink-0 text-ink-muted" />}
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-bold">{r.title}</span>
                      {r.meta && <span className="block truncate text-xs text-ink-muted">{r.meta}</span>}
                    </span>
                  </button>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}

// ═════════════════════════════════════════════════════════════════════
//  نقطةُ الحضور
// ═════════════════════════════════════════════════════════════════════

/** **نقطةٌ بلونٍ من القاموس** + نصٌّ: متصل · مشغول · فاضي · غير متصل. (النصُّ دائماً — لا لونَ وحدَه.) */
export function Presence({ state }: { state: "online" | "busy" | "free" | "offline" }) {
  const tone = state === "offline" ? "neutral" : state === "busy" ? "progress" : "good";
  const dot = state === "offline" ? "bg-ink-muted" : state === "busy" ? "bg-info" : "bg-success";
  return (
    <span className={`inline-flex items-center gap-1.5 text-sm ${TONE_TEXT[tone]}`}>
      <span aria-hidden className={`h-2 w-2 rounded-full ${dot}`} />
      {(K.presence as Record<string, string>)[state]}
    </span>
  );
}

/** **قبل كذا** من وقتٍ ISO — للأسطر الحيّة («تحدّث قبل ٣ د»). */
export function agoText(iso: string | number | Date, now: number = Date.now()): string {
  const s = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000));
  const U = K.unit;
  const t =
    s < 60
      ? `${fmtNum(s)} ${U.s}`
      : s < 3600
        ? `${fmtNum(Math.round(s / 60))} ${U.m}`
        : s < 86400
          ? `${fmtNum(Math.round(s / 3600))} ${U.h}`
          : `${fmtNum(Math.round(s / 86400))} ${U.d}`;
  return K.ago.replace("{t}", t);
}
