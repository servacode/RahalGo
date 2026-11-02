"use client";

/**
 * **رابطُ اللوحة الواحد** (إعادةُ تصميم اللوحة، ٢٠٢٦-١١-٠٣)
 *
 * حزمةُ الواجهة لا تعرف `next/link`، فكان كلُّ رقمٍ «ينكبس» يحتاج دالّةَ تنقّلٍ تُمرَّر يدويّاً.
 * **فتُحقَن مرّةً واحدةً في هيكل اللوحة** (`AdminLinkProvider`) ويستعملها كلُّ ما في العُدّة:
 * `Stat` و`AttentionStrip` و`AdminEntityCard`… وبلا مزوّدٍ يعود `<a>` عاديّاً فلا ينكسر شيء.
 */

import { createContext, useContext, type ComponentType, type ReactNode } from "react";

type LinkImpl = ComponentType<{
  href: string;
  className?: string;
  title?: string;
  children: ReactNode;
  onClick?: () => void;
}>;

const Plain: LinkImpl = ({ href, className, title, children, onClick }) => (
  <a href={href} className={className} title={title} onClick={onClick}>
    {children}
  </a>
);

const Ctx = createContext<LinkImpl>(Plain);

export function AdminLinkProvider({ Link, children }: { Link: LinkImpl; children: ReactNode }) {
  return <Ctx.Provider value={Link}>{children}</Ctx.Provider>;
}

export function AdminLink(props: {
  href: string;
  className?: string;
  title?: string;
  children: ReactNode;
  onClick?: () => void;
}) {
  const L = useContext(Ctx);
  return <L {...props} />;
}

// ═════════════════════════════════════════════════════════════════════
//  الهاتف — رابطا الاتصال وواتساب من رقمٍ واحد
// ═════════════════════════════════════════════════════════════════════

/** **أرقامٌ فقط، بمفتاح سوريا** — `09xx` و`00963` و`+963` كلُّها تصير `963…`. */
export function intlDigits(phone: string): string {
  let d = phone.replace(/[^\d]/g, "");
  if (d.startsWith("00")) d = d.slice(2);
  if (d.startsWith("0")) d = `963${d.slice(1)}`;
  return d;
}

/** **`tel:`** — يُبنى هنا وحدَه (الحارسُ يمنع بناءه في الصفحات). */
export function telHref(phone: string): string {
  return `tel:+${intlDigits(phone)}`;
}

/** **`wa.me`** — كذلك. */
export function waHref(phone: string): string {
  return `https://wa.me/${intlDigits(phone)}`;
}
