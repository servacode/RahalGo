"use client";

/**
 * **حارسُ التقييم — يسأل حيث كان الزبون لا حيث نريده.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٩: «تقييم الطلب يجب أن يظهر بشكلٍ تلقائيٍّ بعد تسليم
 *  الطلب لينتبه الزبون» ثمّ «نفّذه هذا أيضاً» — أن يظهر في أيّ صفحةٍ كان
 *  فيها.)
 *
 * # ولماذا لا يكفي أن يكون في صفحة الطلبات
 *
 * **من ينتظر طلبَه يتصفّح السوق** لا صفحةَ طلباته — **ولحظةُ التسليم هي
 * لحظةُ الرأي**: بعدها بساعةٍ يُنسى الطعامُ ولا يُميَّز طلبٌ عن طلب.
 *
 * **وحدثُ التسليم يصل الصفحاتِ كلَّها** — البثُّ الحيُّ يُعلن `order` لكلّ
 * من فتح الموقع. **فيُسأل حيث هو.**
 *
 * # ولا يُلحّ
 *
 * **من ردّها فقد أجاب**: يُحفظ رقمُ الطلب فلا تُفتح له ثانية — ونافذةٌ تعود
 * في كلّ زيارةٍ تجعل الزبونَ يتجنّب الموقعَ لا يقيّمه.
 *
 * **ولا تُفتح فوق شاشةٍ يعمل فيها**: من يملأ سلّتَه أو يكتب عنواناً **تُقطع
 * عليه خطوتُه** — فتُؤجَّل حتّى يفرغ. (صفحاتُ السلّة والدفع والحساب.)
 *
 * **ولا تُفتح لمن لم يدخل** — ولا في اللوحات.
 */

import { useCallback, useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import { useLiveRefresh } from "@rahalgo/ui";
import RatingModal from "@/components/RatingModal";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

interface RateInfo {
  order_id: string;
  number: number;
  items_preview: string;
  has_driver: boolean;
  rated: boolean;
  platform_stars: number;
  driver_stars: number | null;
  comment: string;
}

/** **صفحاتٌ لا تُقاطَع** — فيها خطوةٌ نصفُها مكتوب. */
const BUSY_PAGES = ["/cart", "/app", "/app", "/app", "/account", "/join"];

/**
 * **ما رُدَّ من نوافذ التقييم** — في المتصفّح لا في الخادم.
 *
 * **تفضيلُ عرضٍ لا بيانُ حساب**، وضياعُه بمسح المتصفّح لا يضرّ: تُفتح مرّةً
 * أخرى لا أكثر. **ومتصفّحٌ يمنع التخزين لا يكسر الموقع.**
 */
const SKIP_KEY = "rahalgo.rating.skipped";

function dismissed(id: string): boolean {
  try {
    return (localStorage.getItem(SKIP_KEY) ?? "").split(",").includes(id);
  } catch {
    // @empty-ok — منعُ التخزين لا يمنع الموقع.
    return false;
  }
}

function dismiss(id: string) {
  try {
    const cur = (localStorage.getItem(SKIP_KEY) ?? "").split(",").filter(Boolean);
    if (cur.includes(id)) return;
    /* **وعشرون آخِرُها تكفي** — قائمةٌ تنمو بلا حدٍّ في تخزينٍ محدود. */
    localStorage.setItem(SKIP_KEY, [...cur, id].slice(-20).join(","));
  } catch {
    // @empty-ok — انظر أعلاه.
  }
}

export default function RatingWatcher() {
  const { user, loading } = useAuth();
  const pathname = usePathname();
  const [pending, setPending] = useState<RateInfo | null>(null);

  const busyPage = BUSY_PAGES.some((p) => pathname === p || pathname.startsWith(p + "/"));

  const check = useCallback(() => {
    if (!isLoggedIn(user)) return;
    api<{ ratings: RateInfo[] }>("/api/v1/my/ratings")
      .then((res) => {
        const rs = res?.ratings ?? [];
        /* **والقائمةُ مرتّبةٌ بالأحدث من الخادم** — فأوّلُ غيرِ مقيَّمٍ هو
           ما يذكره الزبون. **ولا ثلاثُ نوافذَ لثلاثة طلبات.** */
        const next = rs.find((r) => !r.rated && !dismissed(r.order_id));
        if (next) setPending((cur) => cur ?? next);
      })
      /* @empty-ok — **وسؤالُ رأيٍ لا يُعرض خطؤه**: فشلُ النداء يعني ألّا
         تُفتح النافذة، لا رسالةً في وجه من يتصفّح. */
      .catch(() => undefined);
  }, [user]);

  /* **يُسأل عند الفتح** — لطلبٍ سُلّم والموقعُ مغلق. */
  useEffect(() => {
    if (loading) return;
    check();
  }, [loading, check]);

  /* **وعند كلّ حدثِ طلب** — وفيه لحظةُ التسليم. */
  useLiveRefresh(["order"], check);

  if (!pending || busyPage || !isLoggedIn(user)) return null;

  return (
    <RatingModal
      order={pending}
      onClose={() => {
        dismiss(pending.order_id);
        setPending(null);
      }}
      onRated={() => setPending(null)}
    />
  );
}
