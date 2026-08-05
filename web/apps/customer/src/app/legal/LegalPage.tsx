"use client";

/**
 * **الصفحاتُ الثابتة** — الشروطُ والخصوصيةُ والمساعدة.
 *
 * # ولماذا هيكلٌ واحدٌ لثلاث
 *
 * ثلاثُ صفحاتٍ من عناوينَ وفقرات، **ولو كُتبت كلٌّ منها وحدَها لَافترقت**:
 * تُصلَح مسافةٌ في إحداها وتبقى الأخريان، **فتُقرأ الثلاثُ من ثلاث منصّات.**
 *
 * # والنصُّ في المعجم لا في الشيفرة
 *
 * **ومن أراد تصحيحَ سطرٍ في الشروط لا يفتح ملفَّ واجهة** — والنصوصُ كلُّها في
 * `ar.json` كبقيّة المنصة.
 *
 * # وهويّةُ المنصة من الخادم
 *
 * **الاسمُ والرقمُ والعنوان يتغيّرون**، وما كُتب في الشيفرة لا يُبدَّل إلّا
 * بنشر — **فيبقى الرقمُ القديمُ معروضاً شهراً ومن اتّصل به لم يجد أحداً.**
 *
 * **وفارغُها يُحذف لا يُعرض**: سطرٌ يقول «الهاتف: —» يُقرأ عطباً في المنصة.
 */

import { useEffect, useState } from "react";
import type { ComponentType } from "react";
import { PageContainer, PageHeader } from "@rahalgo/ui";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export interface Contact {
  legal_name: string;
  support_phone: string;
  address: string;
}

export interface Block {
  /** عنوانُ الفقرة — **وفارغُه يعني نصّاً بلا عنوان**. */
  h?: string;
  /** فقراتُها — **سطرٌ لكلّ معنًى**، وفقرةٌ من عشرة أسطرٍ لا تُقرأ. */
  p: string[];
}

/**
 * حقولُ الهويّة التي تُملأ في النصّ — `{name}` و`{phone}` و`{address}`.
 *
 * **ولا يُترك قالبٌ ظاهراً أبداً.**
 *
 * كان يعود بالنصّ كما هو حين لا هويّةَ بعد — **فيقرأ الزائرُ «{name} منصّةُ
 * توصيل»** في أوّل رسمٍ قبل أن يصل الردّ، **وإلى الأبد إن تعثّر الطلب.**
 *
 * **ووثيقةٌ قانونيةٌ فيها قوسٌ لم يُملأ تُقرأ منصّةً غيرَ جاهزة** — وهي أوّلُ
 * ما يفتحه من يريد أن يطمئنّ.
 *
 * **واسمُ العلامة يكفي حتّى يصل الاسمُ المسجَّل**: هو صحيحٌ في الحالين، ولا
 * يقول شيئاً كاذباً.
 */
function fill(text: string, c: Contact | null): string {
  return text
    .replace(/\{name\}/g, c?.legal_name || m.common.appName)
    .replace(/\{phone\}/g, c?.support_phone ?? "")
    .replace(/\{address\}/g, c?.address ?? "");
}

export function LegalPage({
  icon,
  title,
  subtitle,
  blocks,
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  title: string;
  subtitle?: string;
  blocks: Block[];
}) {
  const [contact, setContact] = useState<Contact | null>(null);

  useEffect(() => {
    api<Contact>("/api/v1/public/contact")
      .then(setContact)
      .catch(() => setContact(null));
  }, []);

  return (
    <PageContainer width="medium">
      <PageHeader icon={icon} title={title} subtitle={subtitle} />

      {/* **وسطرٌ يُقرأ في نصف الشاشة** — والنصُّ القانونيُّ بعرض اللابتوب
          كاملاً يُفقد العينَ أوّلَ السطر عند آخره، فتُعاد قراءتُه أو يُترك. */}
      <div className="space-y-6 leading-relaxed">
        {blocks.map((b, i) => (
          <section key={i}>
            {b.h && <h2 className="mb-2 font-bold">{fill(b.h, contact)}</h2>}
            {b.p.map((line, j) => (
              <p key={j} className="mb-2 text-sm text-ink-muted last:mb-0">
                {fill(line, contact)}
              </p>
            ))}
          </section>
        ))}

        {/* **والتواصلُ في الذيل** — من قرأ ولم يجد جواباً يريد باباً. */}
        {contact && (contact.support_phone || contact.address) && (
          <section className="rounded-card border border-line bg-surface p-4">
            <h2 className="mb-2 font-bold">{L.contactTitle}</h2>
            {contact.support_phone && (
              <p className="text-sm">
                {L.phone}:{" "}
                <a href={`tel:${contact.support_phone}`} dir="ltr" className="text-accent">
                  {contact.support_phone}
                </a>
              </p>
            )}
            {contact.address && <p className="mt-1 text-sm text-ink-muted">{contact.address}</p>}
          </section>
        )}
      </div>
    </PageContainer>
  );
}
