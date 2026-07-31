"use client";

/**
 * جرس الإشعارات المركزي — يُركَّب مرة واحدة في الهيكل الموحّد فترثه كل اللوحات.
 * يجمع بين الصندوق الدائم (لما فات) والبث الحي (لما يقع الآن)، فلا يحتاج أحد
 * تحديث الصفحة ليعرف ما استجدّ.
 */

import { useCallback, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconBell, IconClose } from "./icons";

const m = getMessages(defaultLocale);
const N = m.shared.notifications;

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type LinkType = ComponentType<{ href: string; className?: string; children: ReactNode }>;

export interface AppNotification {
  id: string;
  kind: string;
  title: string;
  body: string;
  entity: string;
  entity_id: string;
  href: string;
  read: boolean;
  created_at: string;
}


// ---------- ناقل الأحداث المركزي ----------
// أي صفحة تصبح حيّة بسطر واحد: useLiveRefresh(["lead"], reload)
// المصدر واحد (قناة البث في الهيكل الموحّد) فلا يفتح كل صفحة اتصالاً خاصاً بها.

type Listener = (n: AppNotification) => void;
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

const emit = (n: AppNotification) => fan(listeners, n);
const emitEvent = (e: LiveEvent) => fan(eventListeners, e);

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
    const fn: Listener = (n) => {
      if (kinds.length === 0 || kinds.includes(n.kind)) onEvent();
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
 */
export function useLiveData<T>(load: () => Promise<T>, kinds: string[] = []) {
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
  }, []);

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
          if (msg?.type) emitEvent(msg as LiveEvent); // كل رسالة تُبثّ للمشتركين
          if (msg?.type === "notification" && msg.notification) {
            const n = msg.notification as AppNotification;
            setItems((prev) => [n, ...prev].slice(0, 30));
            setUnread((u) => u + 1);
            setToast(n);
            emit(n); // إعلام الصفحات المشتركة لتحدّث بياناتها
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
}: {
  api: ApiFn;
  wsUrl: string;
  token: string | null;
  Link: LinkType;
}) {
  const notif = useLiveNotifications(api, wsUrl, token);
  return (
    <>
      <NotificationBell
        items={notif.items}
        unread={notif.unread}
        markRead={notif.markRead}
        Link={Link}
      />
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
    <div className="fixed bottom-4 end-4 z-[100] w-80 max-w-[90vw] rounded-card border border-line bg-surface p-4 shadow-lg">
      <div className="flex items-start gap-2">
        <IconBell size={18} className="mt-0.5 shrink-0 text-primary" />
        <div className="min-w-0 flex-1">
          <p className="truncate font-bold">{notification.title}</p>
          {notification.body && (
            <p className="mt-0.5 truncate text-sm text-ink-muted">{notification.body}</p>
          )}
        </div>
        <button onClick={onDismiss} className="text-ink-muted hover:text-ink" aria-label={m.common.cancel}>
          <IconClose size={16} />
        </button>
      </div>
    </div>
  );
}

/** جرس الإشعارات مع عدّاد غير المقروء وقائمة منسدلة. */
export function NotificationBell({
  items,
  unread,
  markRead,
  Link,
}: {
  items: AppNotification[];
  unread: number;
  markRead: (id?: string) => void;
  Link: LinkType;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((o) => !o)}
        className="relative flex items-center rounded-control p-1.5 text-ink-muted transition-colors hover:bg-page hover:text-ink"
        title={N.title}
        aria-label={N.title}
      >
        <IconBell size={19} />
        {unread > 0 && (
          <span className="absolute -top-0.5 -end-0.5 flex h-4 min-w-4 items-center justify-center rounded-badge bg-danger px-1 text-[10px] font-bold text-white">
            {unread > 9 ? "9+" : unread}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute end-0 mt-1 max-h-96 w-80 overflow-y-auto rounded-card border border-line bg-surface shadow-lg">
          <div className="flex items-center justify-between border-b border-line px-3 py-2">
            <span className="text-sm font-bold">{N.title}</span>
            {unread > 0 && (
              <button
                onClick={() => markRead()}
                className="text-xs text-primary hover:underline"
              >
                {N.markAllRead}
              </button>
            )}
          </div>
          {items.length === 0 ? (
            <p className="px-3 py-8 text-center text-sm text-ink-muted">{N.empty}</p>
          ) : (
            <ul>
              {items.map((n) => {
                const row = (
                  <span className="flex items-start gap-2">
                    {!n.read && <span className="mt-1.5 h-2 w-2 shrink-0 rounded-badge bg-primary" />}
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium">{n.title}</span>
                      {n.body && (
                        <span className="block truncate text-xs text-ink-muted">{n.body}</span>
                      )}
                    </span>
                  </span>
                );
                return (
                  <li key={n.id} className="border-b border-line last:border-0">
                    {n.href ? (
                      <Link
                        href={n.href}
                        className={`block px-3 py-2 hover:bg-page ${n.read ? "" : "bg-primary-light/30"}`}
                      >
                        {row}
                      </Link>
                    ) : (
                      <div
                        onClick={() => markRead(n.id)}
                        className={`cursor-pointer px-3 py-2 hover:bg-page ${n.read ? "" : "bg-primary-light/30"}`}
                      >
                        {row}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
