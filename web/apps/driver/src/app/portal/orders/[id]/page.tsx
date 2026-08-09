"use client";

/**
 * **حديثُ طلبٍ بعينه — بابُ التواصل مع الزبون.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن طريق
 *  المنصّة فقط».)
 *
 * # ولماذا صفحةٌ لا نافذةٌ في البطاقة
 *
 * **الكتابةُ تحتاج لوحةَ مفاتيح**، وهي تغطّي نصفَ الشاشة على الهاتف. **ونافذةٌ
 * فوق بطاقةٍ في قائمةٍ تُدفَع فوق حافّتها** — فيكتب السائقُ ولا يرى ما كتب.
 *
 * **وصفحةٌ بمسارٍ تُفتح من الإشعار مباشرةً** — والإشعارُ يحمل وجهةً، **ووجهةٌ
 * تفتح قائمةً تجعله يبحث عن الطلب الذي وصلته رسالتُه.**
 */

import Link from "next/link";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { OrderChat, PageContainer, PageHeader, IconChat, Button } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

export default function OrderChatPage() {
  const { id } = useParams<{ id: string }>();
  return (
    <PageContainer>
      <PageHeader
        icon={IconChat}
        title={m.chat.openChat}
        actions={
          <Link href="/portal">
            <Button variant="secondary">{m.driver.tasks.title}</Button>
          </Link>
        }
      />
      <OrderChat api={api} orderId={id} />
    </PageContainer>
  );
}
