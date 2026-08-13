"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جلدُ السائق وسمتُه — لوحتُه كتطبيقه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن تكون نسخةُ الويب مطابقةً لنسخة
 *  التطبيق بشكلٍ كامل، حتّى الثيمُ والستايلُ والألوان».)
 *
 * # ما يفعله
 *
 * **يضع وسمين على جذر الصفحة**: `data-skin="driver"` فتُبدَّل قيمُ
 * التوكنز إلى لوحة التطبيق، **و`data-theme` فتُختار الفاتحةُ أو
 * الغامقة.**
 *
 * **وعلى الجذر لا على عنصرٍ داخليّ**: النوافذُ والقوائمُ المنسدلةُ
 * تُرسم في `body` خارج شجرة الصفحة (`portal`) — **فجلدٌ يوضع على حاوية
 * داخليّةٍ لا يصلها**، فتخرج نافذةٌ بألوان الموقع القديمة فوق لوحةٍ
 * بألوان التطبيق.
 *
 * # ولا وميضَ عند الفتح
 *
 * **الوسمُ يُكتب قبل أوّل رسم** — بنصٍّ يعمل في رأس الصفحة. **ولو كُتب
 * بعد التحميل لَرأى صاحبُه الأبيضَ ثمّ الأسود** في كلّ فتحة.
 *
 * # ويُنظَّف عند الخروج
 *
 * **من مضى إلى السوق زبونًا يخرج من الجلد** — وإلّا حمل لوحةَ السائق
 * إلى صفحاتٍ لم تُصمَّم لها.
 */

import { useEffect, useState } from "react";
import { IconMoon, IconSun } from "./icons";

const KEY = "rahalgo.driver.theme";

export type SkinTheme = "light" | "dark";

/** **ما اختاره صاحبُه، وإلّا ما يقوله نظامُه.** */
function initial(): SkinTheme {
  if (typeof window === "undefined") return "light";
  try {
    const saved = localStorage.getItem(KEY);
    if (saved === "dark" || saved === "light") return saved;
  } catch {
    // @empty-ok **جهازٌ يمنع التخزين لا يُسقط الشاشة** — يُقرأ نظامُه.
  }
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

/**
 * **يُلبس الصفحةَ جلدَ السائق** — ويردّ مبدّلَ السمة.
 *
 * @returns الحالُ ودالّةُ التبديل، ليضعهما الغلافُ حيث شاء.
 */
export function useDriverSkin(): [SkinTheme, () => void] {
  const [theme, setTheme] = useState<SkinTheme>("light");

  useEffect(() => {
    const t = initial();
    setTheme(t);
    const root = document.documentElement;
    root.setAttribute("data-skin", "driver");
    root.setAttribute("data-theme", t);
    return () => {
      // **ويُنزع عند الخروج** — لا يُحمل إلى سوقٍ ولا إلى لوحةِ إدارة.
      root.removeAttribute("data-skin");
      root.removeAttribute("data-theme");
    };
  }, []);

  const flip = () => {
    const next: SkinTheme = theme === "dark" ? "light" : "dark";
    setTheme(next);
    document.documentElement.setAttribute("data-theme", next);
    try {
      localStorage.setItem(KEY, next);
    } catch {
      // @empty-ok — انظر أعلاه.
    }
  };

  return [theme, flip];
}

/**
 * **زرُّ السمة** — هلالٌ في الفاتحة وشمسٌ في الغامقة.
 *
 * **والأيقونةُ تُظهر الوجهةَ لا الحال** — وهو ما في التطبيق حرفًا:
 * **من رأى الهلالَ عرف أنّ الضغطةَ تُغمّق.**
 */
export function ThemeToggle({
  theme,
  onFlip,
  label,
}: {
  theme: SkinTheme;
  onFlip: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onFlip}
      aria-label={label}
      title={label}
      className="grid h-9 w-9 place-items-center rounded-badge text-ink-muted transition hover:bg-field hover:text-ink"
    >
      {theme === "dark" ? <IconSun size={18} /> : <IconMoon size={18} />}
    </button>
  );
}
