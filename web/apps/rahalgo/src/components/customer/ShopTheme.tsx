"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ثيمُ سوق الزبون — فاتحٌ وداكنٌ، يتبع النظامَ ما لم يختر صاحبُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **السمتان على جذر المستند لا على الغلاف** (`data-shop` و`data-shop-theme`): النوافذُ تُرسم في
 * `body` خارجَ الغلاف، فلو كانت التوكنزُ على الغلاف لخرجت بالثيم القديم. (انظر `theme.css ٨ب`.)
 *
 * **ونصٌّ صغيرٌ يُنفَّذ قبل أوّل رسمٍ** (`SHOP_THEME_SCRIPT`) يضعهما — وإلّا ومض الفاتحُ لمن نظامُه داكن.
 * **والاختيارُ يُحفظ في `localStorage` بحذر**: تصفّحٌ خاصٌّ في سفاري يرمي عند الكتابة فيه، ولا يُكسر السوقُ لذلك.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconMoon, IconSun } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const SK = m.shop.skin;

type Theme = "light" | "dark";
const KEY = "rahalgo_shop_theme";

/** **يُنفَّذ قبل الرسم** — يقرأ المحفوظَ وإلّا النظام، ويضع السمتين على الجذر. */
export const SHOP_THEME_SCRIPT = `(function(){var d=document.documentElement;d.setAttribute("data-shop","");try{var t=localStorage.getItem("${KEY}");if(t!=="dark"&&t!=="light"){t=window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"}d.setAttribute("data-shop-theme",t)}catch(e){}})();`;

function read(): { theme: Theme; chosen: boolean } {
  let saved: string | null = null;
  try {
    saved = window.localStorage.getItem(KEY);
  } catch {
    saved = null;
  }
  if (saved === "dark" || saved === "light") return { theme: saved, chosen: true };
  const dark = window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false;
  return { theme: dark ? "dark" : "light", chosen: false };
}

function apply(theme: Theme) {
  document.documentElement.setAttribute("data-shop", "");
  document.documentElement.setAttribute("data-shop-theme", theme);
}

/**
 * **يُركَّب مرّةً في الغلاف** — يثبّت السمتين ما دام السوقُ معروضاً ويرفعهما عند الخروج منه
 * (لوحةُ المتجر والإدارةُ في المسار نفسِه من التطبيق ولا تُلوَّن بثيم الزبون).
 */
export function useShopTheme(): { theme: Theme; toggle: () => void } {
  const [theme, setTheme] = useState<Theme>("light");

  useEffect(() => {
    const first = read();
    setTheme(first.theme);
    apply(first.theme);
    const mq = window.matchMedia?.("(prefers-color-scheme: dark)");
    const onSystem = () => {
      const now = read();
      if (!now.chosen) {
        setTheme(now.theme);
        apply(now.theme);
      }
    };
    mq?.addEventListener("change", onSystem);
    return () => {
      mq?.removeEventListener("change", onSystem);
      document.documentElement.removeAttribute("data-shop");
      document.documentElement.removeAttribute("data-shop-theme");
    };
  }, []);

  const toggle = useCallback(() => {
    setTheme((cur) => {
      const next: Theme = cur === "dark" ? "light" : "dark";
      apply(next);
      try {
        window.localStorage.setItem(KEY, next);
      } catch {
        /* **تصفّحٌ خاصّ** — يبقى الاختيارُ لهذه الصفحة وحدَها. */
      }
      return next;
    });
  }, []);

  return { theme, toggle };
}

export function ThemeToggle({ theme, onToggle }: { theme: Theme; onToggle: () => void }) {
  const dark = theme === "dark";
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-label={dark ? SK.toLight : SK.toDark}
      title={dark ? SK.toLight : SK.toDark}
      className="shop-iconbtn"
    >
      {dark ? <IconSun size={20} /> : <IconMoon size={20} />}
    </button>
  );
}
