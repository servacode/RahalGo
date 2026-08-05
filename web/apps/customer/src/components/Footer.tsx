/**
 * **التذييل** — بابُ الصفحات التي لا تُطلب كلَّ يومٍ وتُطلب حين تُطلب.
 *
 * # ولماذا لا في الشريط
 *
 * الشريطُ العلويُّ لما يُستعمل في كلّ زيارة: السلّةُ والطلباتُ والمحفظة.
 * **والشروطُ تُقرأ مرّةً والمساعدةُ عند حيرة** — ووضعُهما في الشريط يزاحم ما
 * يُضغط يوميّاً، **وهو ضيّقٌ على الجوّال أصلاً.**
 *
 * # وأسفلَ الصفحة حيث يُبحث عنها
 *
 * **من فرغ من الصفحة ولم يجد جواباً ينزل** — وهي العادةُ المتّفق عليها في
 * كلّ موقع، **ومخالفتُها تجعل من يبحث لا يجد.**
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);
const L = m.site.legal;

const LINKS = [
  { href: "/help", label: L.helpTitle },
  { href: "/terms", label: L.termsTitle },
  { href: "/privacy", label: L.privacyTitle },
];

export default function Footer() {
  return (
    <footer className="mt-4 flex flex-wrap items-center justify-center gap-x-5 gap-y-2 px-3 py-4 text-xs text-ink-muted">
      {LINKS.map((l) => (
        <Link key={l.href} href={l.href} className="transition-colors hover:text-accent-text">
          {l.label}
        </Link>
      ))}
      {/* **والسنةُ تُكتب ولا تُحسب في المتصفّح**: خادمٌ يقول ٢٠٢٦ ومتصفّحٌ
          ضُبط على ٢٠٢٧ يختلفان في أوّل رسمٍ فيصرخ React. */}
      <span>© {m.common.appName}</span>
    </footer>
  );
}
