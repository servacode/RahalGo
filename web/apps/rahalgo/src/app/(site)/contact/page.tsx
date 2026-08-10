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

export const metadata = { title: L.contactTitle };

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
              <ul className="space-y-2">
                {WAYS.map(({ Icon, label, value, href }) => (
                  <li key={label}>
                    {/* **وما يُتصل به رابطٌ وما يُقرأ نصّ** — العنوانُ لا
                        يُضغط، والهاتفُ يُضغط فيفتح المتّصل. */}
                    <span className="flex items-start gap-3 rounded-control border border-line px-3 py-2.5">
                      <span className="mt-0.5 shrink-0 text-ink-muted">
                        <Icon size={17} />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block text-xs text-ink-muted">{label}</span>
                        {href ? (
                          <a
                            href={href}
                            target={href.startsWith("http") ? "_blank" : undefined}
                            rel="noreferrer noopener"
                            dir="ltr"
                            className="block text-sm font-medium text-accent-text"
                          >
                            {value}
                          </a>
                        ) : (
                          <span className="block text-sm font-medium">{value}</span>
                        )}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            </FormSection>
          )}

          {ACCOUNTS.length > 0 && (
            <FormSection title={L.followUs} icon={<IconLink />}>
              <div className="flex flex-wrap gap-2">
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
              <ContactMap lat={at[0]} lng={at[1]} />

              {/* **وزرٌّ يفتح الملاحة لا خريطةً ثانية.**

                  **وخريطتُنا تقول أين المكتب، وتطبيقُ الهاتف يقول كيف
                  تصل** — ومن أراد الطريقَ لا يرسمها بيده.

                  **والوجهةُ إحداثيّاتٌ لا نصُّ عنوان**: بحثٌ باسم شارعٍ في
                  مدينةٍ لا تُرقَّم شوارعُها يقع في غير موضعه، **والنقطةُ
                  تقع حيث وُضعت.** */}
              <a
                href={`https://www.google.com/maps/dir/?api=1&destination=${at[0]},${at[1]}`}
                target="_blank"
                rel="noreferrer noopener"
                className="mt-3 flex items-center justify-center gap-2 rounded-control border border-line px-3 py-2.5 text-sm font-medium transition-colors hover:border-primary-edge hover:text-accent-text"
              >
                <IconLocation size={16} />
                {L.contactGo}
              </a>
            </FormSection>
          )}
        </div>
      )}
    </PageContainer>
  );
}
