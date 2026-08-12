"use client";

/**
 * جرس الإشعارات المركزي — يُركَّب مرة واحدة في الهيكل الموحّد فترثه كل اللوحات.
 * يجمع بين الصندوق الدائم (لما فات) والبث الحي (لما يقع الآن)، فلا يحتاج أحد
 * تحديث الصفحة ليعرف ما استجدّ.
 */

import { useCallback, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { CountBadge } from "./components";
import { IconBell, IconClose } from "./icons";

const m = getMessages(defaultLocale);
const N = m.shared.notifications;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type LinkType = ComponentType<{
  href: string;
  className?: string;
  children: ReactNode;
  onClick?: () => void;
}>;

export interface AppNotification {
  id: string;
  kind: string;
  title: string;
  body: string;
  entity: string;
  entity_id: string;
  read: boolean;
  created_at: string;
  /**
   * **عابرٌ — يرنّ ولا يُحفَظ.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٢.) **ويقولها المحرّك** — لا تُخمَّن من
   * النوع: **شرطٌ يُكتب في الشاشة يفترق عن شرطٍ في الخادم**، فيُعرض
   * سطرٌ لا وجودَ له في الصندوق أو يختفي خبرٌ محفوظ.
   */
  transient?: boolean;
}


// ---------- ناقل الأحداث المركزي ----------
// أي صفحة تصبح حيّة بسطر واحد: useLiveRefresh(["lead"], reload)
// المصدر واحد (قناة البث في الهيكل الموحّد) فلا يفتح كل صفحة اتصالاً خاصاً بها.

// المشتركون يهمّهم *نوع* ما وقع لا شكله: الإشعار المحفوظ والحدث الخام كلاهما
// يصل هنا بنفس النوع (order/ticket/wallet/…) فتتحدّث الصفحة في الحالتين.
type Listener = (kind: string) => void;
const listeners = new Set<Listener>();

/** أي رسالة تصل من قناة البث — الإشعارات وغيرها (تحديث حالة طلب مثلاً). */
export interface LiveEvent {
  type: string;
  [k: string]: unknown;
}
type EventListener_ = (e: LiveEvent) => void;
const eventListeners = new Set<EventListener_>();

function fan<T>(set: Set<(v: T) => void>, v: T) {
  set.forEach((fn) => {
    try {
      fn(v);
    } catch {
      /* تجاهل */
    }
  });
}

const emitKind = (kind: string) => fan(listeners, kind);
const emitEvent = (e: LiveEvent) => fan(eventListeners, e);

/**
 * emitLocal يبثّ حدثاً **محلياً** (بلا خادم) لبقية أجزاء التطبيق المفتوح.
 * ليس كل تغيير يمرّ عبر البث الحي: تغيير المستخدم صورته أو اسمه يخصّ متصفحه
 * وحده، وكان الشريط العلوي لا يعلم به فتبقى الصورة القديمة حتى تحديث الصفحة.
 */
export function emitLocal(kind: string) {
  emitKind(kind);
}

/** حدث «تغيّرت حالة قراءة الإشعارات» — يبطل الجرس والصفحة معاً. */
export const READ_EVENT = "notifications:read";

/** يستقبل رسائل البث الخام — لمن يحتاج أدق من الإشعارات (تتبّع طلب مثلاً). */
export function useLiveEvent(onEvent: (e: LiveEvent) => void) {
  const fn = useRef(onEvent);
  fn.current = onEvent;
  useEffect(() => {
    const listener: EventListener_ = (e) => fn.current(e);
    eventListeners.add(listener);
    return () => {
      eventListeners.delete(listener);
    };
  }, []);
}

// حالة الاتصال — مصدر واحد تقرأه أي صفحة تريد إظهار مؤشر «حي»
let live = false;
const statusListeners = new Set<(b: boolean) => void>();
function setLive(b: boolean) {
  if (live === b) return;
  live = b;
  fan(statusListeners, b);
}

/** هل قناة البث متصلة الآن؟ */
export function useLiveStatus() {
  const [state, setState] = useState(live);
  useEffect(() => {
    setState(live);
    statusListeners.add(setState);
    return () => {
      statusListeners.delete(setState);
    };
  }, []);
  return state;
}

/** يعيد تحميل بيانات الصفحة عند وصول حدث من الأنواع المذكورة (أو أي حدث). */
export function useLiveRefresh(kinds: string[], onEvent: () => void) {
  useEffect(() => {
    const fn: Listener = (kind) => {
      if (kinds.length === 0 || kinds.includes(kind)) onEvent();
    };
    listeners.add(fn);
    return () => {
      listeners.delete(fn);
    };
  }, [kinds.join(","), onEvent]); // eslint-disable-line react-hooks/exhaustive-deps
}

/**
 * useLiveData هو الطريقة المركزية لجلب بيانات أي صفحة: يجلب مرة عند الفتح،
 * ثم يعيد الجلب تلقائياً كلما وقع حدث من الأنواع المذكورة — فلا يحتاج أحد
 * تحديث الصفحة. سطر واحد يغني عن useEffect يدوي في كل صفحة.
 *
 * # و`deps` — **ما يُعيد الجلبَ حين يتغيّر**
 *
 * دالّةُ التحميل محفوظةٌ في مرجعٍ كي لا يُعاد الاشتراكُ مع كلّ رسم. **وثمنُ
 * ذلك أنّ ما تلتقطه الدالّةُ من حالةٍ لا يُلاحَظ تغيّرُه**: صفحةٌ تبني عنوانَها
 * من مُرشِّح (`?kind=` أو `?status=`) **تُغيّر المُرشِّحَ ولا تُنادي الشبكة.**
 *
 * **فتُضيء الشريحةُ ولا يتغيّر شيء** — والمستخدمُ يظنّ أن لا نتائج، **ويظنّ
 * المطوّرُ أنّ الفلترَ يعمل لأنّه رأى اللونَ يتحرّك.**
 *
 * **ووقع في موضعين**: مُرشِّحُ أنواع الإشعارات (شهده المالك ٢٠٢٦-٠٨-٠٣:
 * «الإشعارات يوجد أنواع لكن الفلتر وهميّ لا يعمل») **ومُرشِّحُ حالة طلبات
 * السحب في اللوحة** — ولم يشتكِ منه أحدٌ بعد.
 *
 * **ومن مرّر مُرشِّحاً في العنوان يمرّره هنا.**
 */
export function useLiveData<T>(load: () => Promise<T>, kinds: string[] = [], deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  // نحفظ الدالة في مرجع كي لا يُعاد الاشتراك مع كل إعادة رسم
  const fn = useRef(load);
  fn.current = load;

  const reload = useCallback(() => {
    let alive = true;
    fn.current()
      .then((d) => {
        if (alive) {
          setData(d);
          setError(false);
        }
      })
      .catch(() => {
        if (alive) setError(true);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => reload(), [reload]);
  useLiveRefresh(kinds, reload);

  return { data, loading, error, reload, setData };
}

/**
 * useLiveNotifications يفتح قناة البث ويُبقي الصندوق محدّثاً لحظياً.
 * يعيد الجلب عند إعادة الاتصال كي لا تضيع الأحداث أثناء الانقطاع.
 */
export function useLiveNotifications(api: ApiFn, wsUrl: string, token: string | null) {
  const [items, setItems] = useState<AppNotification[]>([]);
  const [unread, setUnread] = useState(0);
  const [toast, setToast] = useState<AppNotification | null>(null);

  const refresh = useCallback(async () => {
    try {
      const d = await api<{ items: AppNotification[]; unread: number }>("/api/v1/me/notifications");
      setItems(d.items);
      setUnread(d.unread);
    } catch {
      /* تجاهل */
    }
  }, [api]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  // الجرس وصفحة الإشعارات حالتان منفصلتان لنفس البيانات: تعليم القراءة في
  // إحداهما كان لا يصل الأخرى، فيبقى العدّاد كما هو حتى تحديث الصفحة.
  // حدث مركزي واحد يبطل الحالتين معاً.
  useLiveRefresh([READ_EVENT], refresh);

  useEffect(() => {
    if (!token) return;
    let ws: WebSocket | null = null;
    let closed = false;
    let retry = 1000;
    let timer: ReturnType<typeof setTimeout>;

    const connect = () => {
      if (closed) return;
      ws = new WebSocket(`${wsUrl}?token=${encodeURIComponent(token)}`);
      ws.onopen = () => {
        retry = 1000;
        setLive(true);
        void refresh(); // تعويض ما فات أثناء الانقطاع
      };
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string);
          if (!msg?.type) return;
          emitEvent(msg as LiveEvent); // كل رسالة تُبثّ للمشتركين بالرسائل الخام
          if (msg.type === "notification" && msg.notification) {
            const n = msg.notification as AppNotification;
            // ══════════════════════════════════════════════════════════
            // **والعابرُ يرنّ ولا يدخل الصندوق**
            // ══════════════════════════════════════════════════════════
            //
            // (قرارُ المالك ٢٠٢٦-٠٨-١٢: «تقدّمُ حالات الطلب تظهر
            //  كإشعاراتٍ ثمّ تختفي، بدون أن تبقى موجودةً في
            //  الإشعارات».)
            //
            // **وأربعةُ إشعاراتٍ تتكرّر مع كلّ طلب** — قُبل، يُجهَّز،
            // في الطريق، سُلّم. **وعشرون طلباً في الشهر ثمانون سطراً
            // في صندوقه**، وما عداها يقع مرّةً أو مرّتين. **فمن كفّ عن
            // قراءة صندوقه لا يقرأ ما يهمّ.**
            //
            // **ولا يزيد العدّاد**: رقمٌ أحمرُ لخبرٍ لا يجده صاحبُه في
            // الصندوق **يُقرأ عطباً في المنصة.**
            setToast(n);
            emitKind(n.kind);
            if (!n.transient) {
              setItems((prev) => [n, ...prev].slice(0, 30));
              setUnread((u) => u + 1);
            }
          } else {
            // حدث خام (تغيّر حالة طلب، تعديل منطقة…): تحديث صامت للصفحات فقط،
            // بلا صفّ في الصندوق — ليس كل تغيّر يستحق إشعاراً في وجه المستخدم.
            emitKind(msg.type);
          }
        } catch {
          /* تجاهل */
        }
      };
      ws.onclose = () => {
        setLive(false);
        if (closed) return;
        timer = setTimeout(connect, retry);
        retry = Math.min(retry * 2, 15000);
      };
    };
    connect();
    return () => {
      closed = true;
      setLive(false);
      clearTimeout(timer);
      ws?.close();
    };
  }, [wsUrl, token, refresh]);

  const markRead = useCallback(
    async (id?: string) => {
      setUnread((u) => (id ? Math.max(0, u - 1) : 0));
      setItems((prev) => prev.map((n) => (!id || n.id === id ? { ...n, read: true } : n)));
      try {
        await api("/api/v1/me/notifications/read", {
          method: "POST",
          body: JSON.stringify({ id: id ?? "" }),
        });
        emitKind(READ_EVENT); // بقية الشاشات تلتقط التغيير فوراً
      } catch {
        /* تجاهل */
      }
    },
    [api],
  );

  return { items, unread, toast, dismissToast: () => setToast(null), markRead, refresh };
}

/**
 * LiveNotifications هي نقطة التركيب الوحيدة: قناة بث واحدة + جرس + تنبيه عابر.
 * تُركَّب مرة في هيكل كل تطبيق (لوحة أو موقع الزبون) فتصير كل صفحاته حيّة.
 */
export function LiveNotifications({
  api,
  wsUrl,
  token,
  Link,
  allHref,
}: {
  api: ApiFn;
  wsUrl: string;
  token: string | null;
  Link: LinkType;
  /** مسار صفحة الإشعارات الكاملة في هذا التطبيق */
  allHref?: string;
}) {
  const notif = useLiveNotifications(api, wsUrl, token);
  return (
    <>
      <NotificationBell unread={notif.unread} Link={Link} allHref={allHref} />
      <NotificationToast notification={notif.toast} onDismiss={notif.dismissToast} />
    </>
  );
}

/** تنبيه عابر يظهر فور وصول حدث جديد. */
export function NotificationToast({
  notification,
  onDismiss,
}: {
  notification: AppNotification | null;
  onDismiss: () => void;
}) {
  useEffect(() => {
    if (!notification) return;
    const t = setTimeout(onDismiss, 6000);
    return () => clearTimeout(t);
  }, [notification, onDismiss]);

  if (!notification) return null;
  return (
    /* ══════════════════════════════════════════════════════════════════
       **والتنبيهُ العابر فوق الشريط السفليّ لا تحته**
       ══════════════════════════════════════════════════════════════════

       (شهده المالك ٢٠٢٦-٠٨-١١: «انظر كيف ظهر الإشعارُ على الجوّال —
        مقصوصٌ وكبيرٌ وغيرُ مناسب».)

       **ثلاثةٌ مجتمعة:**

       **الأوّل** أنّه يجلس على `bottom-4` — **وشريطُ أقسام الزبون ثابتٌ
       هناك بارتفاع ٦٠ بكسلاً**، فيقع التنبيهُ خلفه أو عليه.

       **والثاني** أنّ `truncate` تقطع العنوانَ والنصَّ سطراً واحداً —
       **فيصل الخبرُ نصفَه**: «طلبُك رقم…». وهو تنبيهٌ يعيش ستَّ ثوانٍ،
       **ومن لم يقرأه فيها لم يقرأه.**

       **والثالث** أنّ `w-80` تُقاس بالحاسوب، **و`max-w-[90vw]` تجعله
       لوحاً يغطّي عرضَ الهاتف** وتترك خمسةَ بالمئة من كلّ جهةٍ فيُقرأ
       زحفاً لا بطاقة.

       **فيرتفع عن الشريط ويمتدّ من حافّةٍ إلى حافّةٍ بهامشٍ متساوٍ**،
       **ويلتفُّ نصُّه سطرين.** وعلى الحاسوب يعود بطاقةً في الزاوية. */
    <div className="fixed inset-x-3 bottom-[4.5rem] z-[100] surface p-3 elev-3 sm:inset-x-auto sm:bottom-4 sm:end-4 sm:w-80 sm:p-4 md:bottom-4">
      <div className="flex items-start gap-2">
        <IconBell size={18} className="mt-0.5 shrink-0 text-primary" />
        <div className="min-w-0 flex-1">
          <p className="line-clamp-2 font-bold">{notification.title}</p>
          {notification.body && (
            <p className="mt-0.5 line-clamp-2 text-sm text-ink-muted">{notification.body}</p>
          )}
        </div>
        <button onClick={onDismiss} className="text-ink-muted hover:text-ink" aria-label={m.common.cancel}>
          <IconClose size={16} />
        </button>
      </div>
    </div>
  );
}

/**
 * جرسُ الإشعارات — **رابطٌ إلى الصفحة لا قائمةٌ منسدلة.**
 *
 * كانت القائمةُ تعرض آخرَ ما وصل ثم تُذيّل بـ«عرض الكلّ» — **خطوتان إلى مكانٍ
 * واحد**: من فتحها إمّا وجد ما يريد في ثلاثة أسطر مقتطعة، وإمّا ضغط مرّةً
 * ثانية. **ونافذةٌ صغيرة تعرض بعضَ الخبر تُغري بالاكتفاء بالبعض.**
 *
 * والصفحةُ تفعل ما تفعله القائمةُ وزيادة: تُظهر الكلَّ، وتُعلّم بالقراءة،
 * وتُصفّي. **فبقاؤهما معاً شاشتان لخبرٍ واحد تفترقان يوماً.**
 *
 * ويبقى **العدّادُ الأحمر** فوق الجرس: هو ما يُقرأ من بعيد، وهو وحده ما كان
 * يلزم من القائمة. والتنبيهُ العابر يبقى كذلك — يقول «وصل الآن» بلا ضغطة.
 */
export function NotificationBell({
  unread,
  Link,
  allHref,
}: {
  unread: number;
  Link: LinkType;
  /** مسار صفحة الإشعارات — بلاه لا وجهةَ للجرس */
  allHref?: string;
}) {
  /* ══════════════════════════════════════════════════════════════════
     **والعدّادُ فوق الجرس — وحشوةُ الزرّ هي ما يمنع قصَّه**
     ══════════════════════════════════════════════════════════════════

     (قرارُ المالك ٢٠٢٦-٠٨-١١: «بحياتي ما شفت عدّادَ إشعاراتٍ بجانب
      الجرس — يجب أن يكون فوقه، النظامُ المستخدَمُ عالمِيّاً».)

     **وهو الصواب**: العدّادُ فوق الجرس اصطلاحٌ يعرفه كلُّ من حمل هاتفاً،
     **وما يخالف الاصطلاحَ يُقرأ عطباً حتّى لو كان أنظفَ رسماً.** وكنتُ
     أزحتُه جانباً هرباً من القصّ — **وذاك علاجُ العَرَض لا العلّة.**

     # والعلّةُ أنّه كان يخرج من صندوق الزرّ

     **صفُّ أدوات الشريط فيه `overflow-x: auto`** ليمنع الشريطَ أن يجرّ
     الصفحةَ على الجوّال — **وهي تجرّ `overflow-y: auto` بحكم المواصفة.**
     فما خرج علوّاً يُقصّ، **والعدّادُ أوّلُ ما يخرج.**

     # فيُعلَّق على الأيقونة داخلَ زرٍّ حشوتُه تحتمله

     **الأيقونةُ تُلَفُّ بمرجعٍ في مقاسها** (`relative`) — فالإزاحةُ
     السالبةُ تُقاس من حافّتها لا من حافّة الزرّ. **والفقّاعةُ تخرج ستّاً**
     (أربعةٌ إزاحةً واثنتان حلقة)، **والحشوةُ ثمانٍ.**

     **فتبقى داخلَ صندوق الزرّ ببكسلين** — قِيس بمتصفّحٍ على قوالب المشروع
     نفسِها: **قبلَه تخرج ستّاً، وبعدَه تدخل بكسلين.** **ولا يبلغها مِقصُّ
     أيِّ سلفٍ** مهما تبدّل ما حولَها في الصفّ.

     **والحلقةُ تبقى**: هي ما يفصل الفقّاعةَ عن خطوط الجرس تحتها. */
  const cls =
    "flex items-center rounded-control p-2 text-ink-muted transition-colors hover:text-accent-text";
  const inner = (
    <>
      {/* **اسمٌ مقروءٌ للقارئ الصوتيّ**: أيقونةٌ وحدها رابطٌ بلا اسم.
          و`title` لا يمرّ عبر `LinkType` — وتوسيعُ نوعٍ مشتركٍ لأجل صفةٍ
          واحدة يُثقل كلَّ من يستعمله. */}
      <span className="sr-only">{N.title}</span>
      <span className="relative flex">
        <IconBell size={19} />
        {unread > 0 && <CountBadge count={unread} float max={9} />}
      </span>
    </>
  );

  // **بلا وجهةٍ لا زرّ**: جرسٌ يُضغط ولا يحدث شيء يُعلّم المستخدمَ ألّا يثق
  // بالأيقونات. وكلُّ تطبيقاتنا تمرّر المسار — وهذا لِما بعدها.
  if (!allHref) {
    return (
      <span className={cls}>{inner}</span>
    );
  }
  return (
    <Link href={allHref} className={cls}>
      {inner}
    </Link>
  );
}
