"use client";

import { useCallback, useEffect, useState } from "react";
import QRCode from "qrcode";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert,
  Confirm,
  PageHeader, Button, Badge, IconWhatsApp,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

interface WAStatus {
  provider: string;
  connected?: boolean;
  logged_in?: boolean;
  paired_as?: string;
  qr?: string;
  last_error?: string;
  /** **طُلب رمزٌ ولمّا يظهر** — بين الضغطة والرمز ثانيتان. */
  pair_wanted?: boolean;
  /** **انتهى آخرُ رمزٍ ولم يُمسح** — دعوةٌ لا إنذار. */
  qr_expired?: boolean;
}

export default function WhatsAppPanel() {
  const [status, setStatus] = useState<WAStatus | null>(null);
  const [qrDataURL, setQrDataURL] = useState("");
  const [busy, setBusy] = useState(false);
  /** **وما وقع يُقال** — وزرٌّ يُضغط بلا أثرٍ ظاهرٍ يُضغط مرّتين. */
  const [notice, setNotice] = useState("");
  /** **وسؤالُ الخروج بنافذة المنصّة** — لا بنافذة النظام. */
  const [asking, setAsking] = useState(false);

  async function unpair() {
    setAsking(false);
    setBusy(true);
    try {
      await api("/api/v1/admin/whatsapp/unpair", { method: "POST" });
      setNotice(m.admin.whatsappPage.unpairDone);
      await load();
    } finally {
      setBusy(false);
    }
  }

  /** **وطلبُ الرمز صريح** — (قرارُ المالك ٢٠٢٦-٠٨-١٠). */
  async function pair() {
    setBusy(true);
    setNotice("");
    try {
      await api("/api/v1/admin/whatsapp/pair", { method: "POST" });
      await load();
    } finally {
      setBusy(false);
    }
  }

  const load = useCallback(async () => {
    try {
      const st = await api<WAStatus>("/api/v1/admin/whatsapp");
      setStatus(st);
      if (st.qr) {
        setQrDataURL(await QRCode.toDataURL(st.qr, { width: 280, margin: 1 }));
      } else {
        setQrDataURL("");
      }
    } catch {
      setStatus(null);
    }
  }, []);

  useEffect(() => {
    void load();
    const t = setInterval(load, 10000); // تحديث تلقائي كل 10 ثوانٍ (رمز QR يتغير)
    return () => clearInterval(t);
  }, [load]);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <PageHeader icon={IconWhatsApp} title={m.admin.whatsappPage.title} />
        <div className="flex gap-2">
          <Button variant="secondary" onClick={load}>
            {m.admin.whatsappPage.refresh}
          </Button>
          {/* ══════════════════════════════════════════════════════════
              **وزرُّ إعادة الربط — ولا رمزَ بلا فكّ**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-١٠: «إذا تمّ فصلُ الاقتران لا يوجد زرٌّ
               لإعادة ربط الجهاز».)

              **ورمزُ الربط لا يُولَّد إلّا لجهازٍ بلا هويّة** — ومن فُصل من
              هاتفه تبقى هويّتُه مخزَّنة، **فتقول اللوحةُ «غير مقترن» ولا
              سبيلَ إلى الاقتران.**

              **ولا يظهر لمزوّد التطوير** — لا اقترانَ فيه أصلاً. */}
          {/* **والزرُّ يقول الحالَ لا الفعلَ المجرَّد** — (قرارُ المالك
              ٢٠٢٦-٠٨-١٠: «إذا كان متّصلاً الزرُّ يجب أن يكون تسجيلَ الخروج،
              وإذا منفصلٌ ربطَ الجهاز — ليكون كلُّ شيءٍ واضحاً»).

              **وزرٌّ اسمُه واحدٌ في حالين يجعل صاحبَه يخمّن**: أيفكّ ما هو
              مربوطٌ أم يربط ما هو مفكوك؟

              **والفعلُ واحدٌ في الحالين**: فكُّ الاقتران — لأنّ **الرمزَ لا
              يُولَّد إلّا لجهازٍ بلا هويّة.** فمن ضغط «ربط الجهاز» وهو
              مفصولٌ يُنظَّف ما بقي فيظهر الرمز. */}
          {status?.provider === "whatsapp" && (
            <Button
              variant={status?.logged_in ? "danger" : "primary"}
              disabled={busy}
              /* ══════════════════════════════════════════════════════
                 **والسؤالُ من المنصّة لا من المتصفّح**
                 ══════════════════════════════════════════════════════

                 (شكوى المالك ٢٠٢٦-٠٨-١٠: «لم تُطبّق رسالةَ التأكيد بشكلٍ
                  مركزيّ».)

                 **`confirm()` الأصليّةُ نافذةُ نظام**: بخطّه ولغته، تُلصق
                 «localhost:3001 يعرض» فوق النصّ، **وأزرارُها «موافق/إلغاء»
                 لا تقول ماذا سيقع.** وقد خرج المستخدمُ من المنصّة بصريّاً
                 في أخطر ضغطةٍ فيها.

                 **و`Confirm` المركزيّةُ موجودةٌ ومُصدَّرة** — فتجاوزُها هنا
                 كان سهواً لا قراراً. */
              onClick={() => {
                /* **وما لا يُفكّ لا يُسأل عنه** — من كان مفصولاً أصلاً
                   يضغط «ربط الجهاز»، **وسؤالُه «أتُسجّل الخروج؟» يُربكه.**

                   **والفعلان صارا فعلين لا فعلاً واحداً**: الربطُ يطلب
                   رمزاً، والخروجُ يمحو الجلسة. **وكانا واحداً لأنّ الرمزَ
                   كان يُولَّد وحدَه** — فما عاد. */
                if (status?.logged_in) setAsking(true);
                else void pair();
              }}
            >
              {status?.logged_in
                ? m.admin.whatsappPage.logout
                : m.admin.whatsappPage.unpair}
            </Button>
          )}
        </div>
      </div>

      <Confirm
        open={asking}
        title={m.admin.whatsappPage.logout}
        body={m.admin.whatsappPage.unpairConfirm}
        confirmLabel={m.admin.whatsappPage.logout}
        busy={busy}
        onConfirm={() => void unpair()}
        onCancel={() => setAsking(false)}
      />

      {notice && <Alert tone="success" className="mb-4">{notice}</Alert>}

      {status?.provider === "dev" ? (
        <div className="surface p-6 text-ink-muted">
          {m.admin.whatsappPage.devMode}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="space-y-4 surface p-6">
            {/* ══════════════════════════════════════════════════════════
                **وسطران يقولان ما الفرق بينهما**
                ══════════════════════════════════════════════════════════

                (سؤالُ المالك ٢٠٢٦-٠٨-١٠: «ما معنى متّصلٌ بخوادم واتساب
                 والرقمُ غيرُ مربوط؟ من حقّي أن أفهمها».)

                **صفّان أخضرُ وبرتقاليٌّ بلا كلمة يبدوان متناقضين** — ومن
                قرأ «متّصل» فهم أنّ البوت يعمل، ثمّ رأى «غير مقترن» فظنّ
                عطباً. **وهما شيئان مختلفان لا حالان لشيء.**

                **والشرحُ في المنصّة لا في محادثةٍ تُنسى** — من يفتح اللوحةَ
                بعد سنةٍ يقرأ الجواب في مكانه. */}
            <div className="flex items-start justify-between gap-3">
              <span>
                <span className="font-medium">{m.admin.whatsappPage.connection}</span>
                <span className="mt-0.5 block text-2xs text-ink-muted">
                  {m.admin.whatsappPage.connectionHint}
                </span>
              </span>
              <Badge variant={status?.connected ? "success" : "danger"}>
                {status?.connected
                  ? m.admin.whatsappPage.connected
                  : m.admin.whatsappPage.disconnected}
              </Badge>
            </div>
            <div className="flex items-start justify-between gap-3">
              <span>
                <span className="font-medium">{m.admin.whatsappPage.pairing}</span>
                <span className="mt-0.5 block text-2xs text-ink-muted">
                  {m.admin.whatsappPage.pairingHint}
                </span>
              </span>
              {/* **ولا يُقال «مقترنٌ بالرقم» ولا رقمَ بعدها.**

                  (لقطةُ المالك ٢٠٢٦-٠٨-١٠: «مقترن بالرقم: +» — علامةٌ خضراء
                   وشرطةٌ فارغة.)

                  **الجلسةُ تمرّ بحالٍ وسطى**: المكتبةُ تقول «مسجَّل» قبل أن
                  تصل هويّةُ الرقم. **فتقرأ اللوحةُ نجاحاً وليس في يدها ما
                  تُريه** — ومن رآها ظنّ الاقترانَ تمّ فأغلق الصفحة.

                  **فالعلامةُ الخضراءُ لا تُرفع إلّا برقمٍ يُقرأ.** */}
              {status?.logged_in && status.paired_as ? (
                <Badge variant="success">
                  {m.admin.whatsappPage.paired}: <span dir="ltr">+{status.paired_as}</span>
                </Badge>
              ) : (
                <Badge variant="warning">{m.admin.whatsappPage.notPaired}</Badge>
              )}
            </div>
            {/* **ورمزٌ انتهى دعوةٌ لا إنذار** — (سؤالُ المالك ٢٠٢٦-٠٨-١٠:
                «ما زال يظهر آخرُ خطأ، لماذا؟»).

                **والرمزُ ينتهي بطبعه** — ستّةٌ في دقيقتين ثمّ يمضي. **وعرضُ
                ذلك بحمرةٍ يقول للمالك إنّ في منصّته عطباً وليس فيها عطب**،
                فيبحث عن سببٍ لا وجودَ له. */}
            {status?.qr_expired && !status?.qr && (
              <Alert tone="warning">{m.admin.whatsappPage.qrExpired}</Alert>
            )}
            {status?.pair_wanted && !status?.qr && (
              <Alert tone="info">{m.admin.whatsappPage.requestingQR}</Alert>
            )}
            {status?.last_error && (
              <Alert>
                {m.admin.whatsappPage.lastError}: <span dir="ltr">{status.last_error}</span>
              </Alert>
            )}
          </div>

          {qrDataURL && !status?.logged_in && (
            <div className="flex flex-col items-center surface p-6">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={qrDataURL} alt="WhatsApp QR" loading="lazy" className="rounded-control" />
              <p className="mt-3 max-w-xs text-center text-sm text-ink-muted">
                {m.admin.whatsappPage.scanHint}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
