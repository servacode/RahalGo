/**
 * **صفحةُ التواصل** — بابٌ قائمٌ بذاته لا سطرٌ في ذيل صفحةٍ أخرى.
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٩: «صفحة تواصل معنا لازم صفحة خاصّة فيها خريطة
 *  المكتب وعنوان ووسائل سوشيال ميديا — مو مجرّد رقم وخلص».)
 *
 * # ولماذا صفحةٌ لا كتلة
 *
 * **كان التواصلُ كتلةً في ذيل صفحة المساعدة**: رقمٌ وعنوانٌ نصّاً. **ومن
 * أراد أن يزور المكتبَ لا يعرف أين هو** — عنوانٌ مكتوبٌ في مدينةٍ لا تُرقَّم
 * شوارعُها لا يقود أحداً. **والخريطةُ تقود.**
 *
 * **ومن أراد أن يشتكي في غير أوقات الدوام** لا يجد إلّا رقماً لا يردّ —
 * **وحساباتُ التواصل تستقبل في كلّ وقت.**
 *
 * # وكلُّ شيءٍ من الإعدادات
 *
 * **لا رقمَ ولا عنوانَ ولا رابطَ مكتوبٌ هنا** — يُبدَّل من اللوحة بلا نشر.
 * **وما لم يُضبط لا يُعرض**: بطاقةٌ فارغةٌ تقول «لا نريد أن نُكلَّم».
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  FormSection,
  EmptyState,
  IconPhone,
  IconWhatsApp,
  IconLocation,
  IconFacebook,
  IconInstagram,
  IconTelegram,
  IconLink,
  fetchPlatform,
} from "@rahalgo/ui";
import ContactMap from "./ContactMap";

const m = getMessages(defaultLocale);
const L = m.site.legal;

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";


/** **«عرض,طول» ← رقمان** — وفارغٌ يعني «لا خريطة». */
function geoOf(v: string): [number, number] | null {
  const p = (v ?? "").split(",");
  if (p.length !== 2) return null;
  const la = Number(p[0]);
  const ln = Number(p[1]);
  return Number.isFinite(la) && Number.isFinite(ln) ? [la, ln] : null;
}

export default async function Page() {
  const p = await fetchPlatform(API);
  const at = geoOf(p.location);
  const wa = p.social.whatsapp.replace(/\D/g, "");

  const WAYS = [
    p.supportPhone && {
      Icon: IconPhone,
      label: L.contactPhone,
      value: p.supportPhone,
      href: `tel:${p.supportPhone}`,
    },
    wa && {
      Icon: IconWhatsApp,
      label: L.contactWhats,
      value: p.social.whatsapp,
      href: `https://wa.me/${wa}`,
    },
    p.address && {
      Icon: IconLocation,
      label: L.contactAddress,
      value: p.address,
      href: "",
    },
  ].filter(Boolean) as { Icon: typeof IconPhone; label: string; value: string; href: string }[];

  const ACCOUNTS = [
    { href: p.social.facebook, Icon: IconFacebook, label: "Facebook" },
    { href: p.social.instagram, Icon: IconInstagram, label: "Instagram" },
    { href: p.social.telegram, Icon: IconTelegram, label: "Telegram" },
  ].filter((a) => a.href);

  const bare = WAYS.length === 0 && ACCOUNTS.length === 0 && !at;

  return (
    <PageContainer>
      <PageHeader title={L.contactTitle} subtitle={L.contactSubtitle} />

      {bare ? (
        /* **وفراغٌ يُقال ولا يُترك** — صفحةٌ بيضاءُ تُقرأ عطباً. */
        <EmptyState icon={<IconLink size={28} />} title={L.contactNoWay} />
      ) : (
        <div className="space-y-4">
          {WAYS.length > 0 && (
            <FormSection title={L.contactTitle} icon={<IconPhone />}>
              {/* ══════════════════════════════════════════════════════
                  **ومربّعاتٌ صغيرةٌ متوسّطةٌ لا صفوفٌ ممتدّة**
                  ══════════════════════════════════════════════════════

                  (طلبُ المالك ٢٠٢٦-٠٨-١٧: «رقمُ الدعم والواتساب مربّعاتٌ
                   صغيرةٌ أنيقة، وليست طويلةً مزعجة».)

                  **وصفٌّ يمتدّ بعرض الشاشة لرقمٍ من عشرة أرقام** يترك
                  خلفَه فراغاً بطولِ ذراع — **والعينُ تقطعه لتصل إلى ما
                  يُقرأ.**

                  **والمربّعُ يأخذ قدرَ ما فيه** ويقف مع أخيه في الوسط. */}
              <ul className="flex flex-wrap justify-center gap-3">
                {WAYS.map(({ Icon, label, value, href }) => (
                  <li key={label}>
                    {/* **وما يُتصل به رابطٌ وما يُقرأ نصّ** — العنوانُ لا
                        يُضغط، والهاتفُ يُضغط فيفتح المتّصل. */}
                    {href ? (
                      <a
                        href={href}
                        target={href.startsWith("http") ? "_blank" : undefined}
                        rel="noreferrer noopener"
                        className="site-card lift flex min-w-[9rem] flex-col items-center gap-1.5 px-5 py-4 text-center"
                      >
                        <Icon size={22} className="text-accent-text" />
                        <span className="text-2xs text-ink-muted">{label}</span>
                        <span dir="ltr" className="text-sm font-bold text-accent-text">
                          {value}
                        </span>
                      </a>
                    ) : (
                      <span className="site-card flex min-w-[9rem] flex-col items-center gap-1.5 px-5 py-4 text-center">
                        <Icon size={22} className="text-ink-muted" />
                        <span className="text-2xs text-ink-muted">{label}</span>
                        <span className="text-sm font-bold text-ink">{value}</span>
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </FormSection>
          )}

          {ACCOUNTS.length > 0 && (
            <FormSection title={L.followUs} icon={<IconLink />}>
              <div className="flex flex-wrap justify-center gap-2">
                {ACCOUNTS.map(({ href, Icon, label }) => (
                  <a
                    key={label}
                    href={href}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="flex items-center gap-2 rounded-control border border-line px-3 py-2 text-sm transition-colors hover:border-primary-edge hover:text-accent-text"
                  >
                    <Icon size={17} />
                    <span dir="ltr">{label}</span>
                  </a>
                ))}
              </div>
            </FormSection>
          )}

          {at && (
            <FormSection title={L.contactMapHint} icon={<IconLocation />}>
              {/* ══════════════════════════════════════════════════════
                  **والعنوانُ فوق الخريطة لا في بطاقةٍ أخرى**
                  ══════════════════════════════════════════════════════

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «بصفحة تواصل معنا فقط يُعرض
                   الخريطة وفوقها لازم العنوان… اذهب للعنوان بالخريطة يجب
                   أن يكون التفصيليّ».)

                  **ودبّوسٌ على خريطةٍ لا يقول شارعاً ولا معلَماً** — ومن
                  أراد أن يزور يقرأ العنوانَ ثمّ ينظر أين هو. */}
              {p.address && (
                <p className="mb-3 text-sm leading-relaxed">{p.address}</p>
              )}

              {/* **والخريطةُ تُصيَّر في المتصفّح وحدَه** — `leaflet` يقرأ
                  `window` عند تحميله، **ومكوّنُ خادمٍ يستورده يسقط.** */}
              {/* ══════════════════════════════════════════════════════
                  **والزرُّ في وسط الخريطة لا تحتها**
                  ══════════════════════════════════════════════════════

                  (طلبُ المالك ٢٠٢٦-٠٨-١٧: «بدل اذهب إلى العنوان، زرٌّ
                   بوسط الخريطة يقول افتح الخريطة».)

                  **وزرٌّ تحت الخريطة يُقرأ عنصراً ثالثاً في الصفحة** —
                  **وفوقها يُقرأ فعلاً عليها.**

                  **والخريطةُ تُصيَّر في المتصفّح وحدَه** — `leaflet` يقرأ
                  `window` عند تحميله، ومكوّنُ خادمٍ يستورده يسقط. */}
              <div className="relative">
                <ContactMap lat={at[0]} lng={at[1]} />
                {/* **والوجهةُ إحداثيّاتٌ لا نصُّ عنوان**: بحثٌ باسم شارعٍ
                    في مدينةٍ لا تُرقَّم شوارعُها يقع في غير موضعه،
                    **والنقطةُ تقع حيث وُضعت.** */}
                <a
                  href={`https://www.google.com/maps/dir/?api=1&destination=${at[0]},${at[1]}`}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="site-card lift absolute left-1/2 top-1/2 z-[400] flex -translate-x-1/2 -translate-y-1/2 items-center gap-2 px-5 py-3 text-sm font-bold text-accent-text"
                >
                  <IconLocation size={18} />
                  {L.contactGo}
                </a>
              </div>
            </FormSection>
          )}
        </div>
      )}
    </PageContainer>
  );
}
