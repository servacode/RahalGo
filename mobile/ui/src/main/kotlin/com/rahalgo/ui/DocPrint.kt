package com.rahalgo.ui

import android.content.Context
import android.print.PrintAttributes
import android.print.PrintManager
import android.webkit.WebView
import android.webkit.WebViewClient

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ورقُ المنصّة — هيكلٌ واحدٌ لكلّ ما يُطبع**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٩: «طباعة التقرير يجب أن يكون القالب مثل
 *  الطباعة بالمحفظة أو كشف الحساب».)
 *
 * # ولماذا هيكلٌ لا نسخة
 *
 * **كشفُ الحساب أُتقن على ثلاث شكاوى** — المحاذاةُ (٢٠٢٦-٠٨-١٣)،
 * و«ل.س» تقع بغير موضعها في نصٍّ عربيّ، **وترتيبةُ الفاتورة بلوغو
 * وعباراتِ ثقة.** وتقريرُ المتجر كان `‎<h1>` وجدولاً بعمودين.
 *
 * **ونسخُ القالب يعني أن تُعاد الشكاوى الثلاث** — فمن أصلح المحاذاةَ
 * في الكشف لن يفتح التقرير.
 *
 * **فالهيكلُ هنا مرّةً**: الأنماطُ والترويسةُ والشريطُ والذيل. **وكلُّ
 * ورقةٍ تضع جسمَها بينهما.**
 */
object DocPrint {

    /** **يُبقي الصفحةَ حيّةً حتّى تُسلَّم للطابعة** — وإلّا جُمعت. */
    private var keep: WebView? = null

    /**
     * **الأنماط** — منقولةٌ حرفاً بحرف من كشف الحساب.
     *
     * **ولا تُحسَّن هنا**: كلُّ سطرٍ فيها جوابُ شكوى، **ومن هذّبها بلا
     * شكوى أعادها.**
     */
    const val STYLES = """<style>
  * { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  body { font-family: sans-serif; padding: 20px; color: #07283A; }

  /* **الترويسةُ كترويسة الويب** — علامةٌ وتاريخُ طباعة. */
  .head { display: flex; align-items: center; justify-content: space-between; }
  .mark { width: 62px; height: 62px; border: 2px solid #07283A; border-radius: 12px;
          display: flex; align-items: center; justify-content: center;
          font-size: 30px; font-weight: bold; }
  .logo { width: 72px; height: 72px; object-fit: contain; }
  .when { text-align: left; font-size: 11px; color: #5A6B75; }
  .when b { display: block; color: #07283A; }

  /* **وشريطُ العلامة** — من الفيروزيّ إلى البرتقاليّ كما في `brand-rule`. */
  .rule { height: 3px; border-radius: 3px; margin: 10px 0 14px;
          background: linear-gradient(to left, #02678F 0%, #02678F 45%, #FE9501 100%); }

  .title { font-size: 17px; font-weight: bold; margin: 0 0 2px; }
  .meta { display: flex; justify-content: space-between; gap: 12px;
          font-size: 12px; color: #5A6B75;
          border-bottom: 1px solid #E4E9EC; padding-bottom: 10px; margin-bottom: 12px; }
  .meta b { color: #07283A; }

  .sum { display: flex; justify-content: space-between; gap: 10px;
         border: 1px solid #E4E9EC; border-radius: 10px;
         padding: 10px 12px; margin-bottom: 14px; font-size: 12px; }

  /* ══════════════════════════════════════════════════════════════
     **ورأسُ العمود يحاذي ما تحته**
     ══════════════════════════════════════════════════════════════

     (شكوى المالك ٢٠٢٦-٠٨-١٣: «وفي عندك المحاذاة أيضاً مو مزبوطة
      بالكشف والطباعة».)

     **كانت الرؤوسُ كلُّها يميناً وخاناتُ المبالغ يسارا** — فيقع
     «الحركة» فوق فراغٍ ورقمُه في الطرف الآخر، **والعينُ تمسح عموداً
     فلا تجد رأسَه فوقه.**

     **والويبُ يفعلها بـ`text-start` للنصّ و`text-end` للمبلغ** —
     ورأسُ كلِّ عمودٍ بصنف عمودِه. */
  table { width: 100%; border-collapse: collapse; font-size: 12px; }
  th, td { padding: 7px 5px; }
  th { text-align: right; font-weight: normal;
       color: #5A6B75; border-bottom: 2px solid #E4E9EC; font-size: 11px; }
  td { text-align: right; border-bottom: 1px solid #F0F3F5; }
  th.cell, td.cell { text-align: left; }
  /* ══════════════════════════════════════════════════════════════
     **واللفّةُ للرقم وحدَه — لا للمبلغ كلِّه**
     ══════════════════════════════════════════════════════════════

     (شكوى المالك ٢٠٢٦-٠٨-١٣: «في مشكلة ل.س ما إنك مصلّحها بالتطبيق
      والكشف والطباعة».)

     **وهي عائلةُ العطب نفسُها التي أُصلحت في الويب خمسَ مرّات**:
     الرقمُ يُلَفّ بـ`ltr` لأنّ فاصلةَ الآلاف تقع بغير موضعها في نصٍّ
     عربيّ، **فإذا وقع الرمزُ داخل اللفّة صار ترتيبُهما من اليسار** —
     فتُقرأ «ل.س ٦٠٠».

     **فالخانةُ تبقى عربيّةً** (`.cell`) **والرقمُ وحدَه يُلَفّ**
     (`.n`). */
  .cell { text-align: left; white-space: nowrap; }
  .n { unicode-bidi: isolate; direction: ltr; }
  .note { color: #5A6B75; }
  .in { color: #1E9E5A; } .out { color: #D64545; }

  tfoot td { border-top: 2px solid #E4E9EC; border-bottom: none;
             padding-top: 9px; font-weight: bold; }
  /* ══════════════════════════════════════════════════════════════
     **وعبارةُ الثقة — لا حاشيةَ زينة**

     (قرارُ المالك ٢٠٢٦-٠٨-١٣: «شوف ترتيبة الفاتورة بالطلب كيف مرتّبة
      وأنيقة — لوغو وعباراتُ ثقةٍ وغيره».)

     **والورقةُ تُقرأ من غير صاحبها**: يُريها للمكتب أو لمن يختلف
     معه. **وسطرٌ يشكر ويعطي رقمَ الشكوى يقول إنّ خلفَها دارا** —
     لا جدولاً خرج من هاتف. */
  .thanks { margin-top: 18px; text-align: center; font-size: 12px;
            font-weight: bold; color: #02678F; }
  .support { text-align: center; font-size: 11px; color: #5A6B75; margin: 4px 0 0; }
  .foot { margin-top: 14px; font-size: 10px; color: #5A6B75;
          line-height: 1.7; text-align: center; }
  .warn { border: 1px solid #FE9501; background: #FFF6E9; border-radius: 8px;
          padding: 8px 10px; font-size: 11px; margin-bottom: 12px; }
</style>"""

    /** **ترويسةٌ وشريطُ علامة** — العلامةُ يميناً وتاريخُ الطباعة يساراً. */
    fun head(c: Context, platform: String, logo: String): String {
        val mark = if (logo.isNotEmpty()) {
            """<img class="logo" src="$logo" alt="">"""
        } else {
            """<div class="mark">${esc(platform.trim().take(1))}</div>"""
        }
        val today = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.US)
            .format(java.util.Date())
        val now = java.text.SimpleDateFormat("HH:mm", java.util.Locale.US)
            .format(java.util.Date())
        return """
<div class="head">
  $mark
  <div class="when">
    ${esc(c.getString(R.string.sheet_printed_at))}
    <b>${esc(today)}</b>${esc(now)}
  </div>
</div>
<div class="rule"></div>"""
    }

    /** **الذيل** — شكرٌ ورقمُ الدعم وسطرُ المنصّة. */
    fun foot(c: Context, platform: String, support: String): String {
        val line = if (support.isNotEmpty()) {
            """<p class="support">""" +
                esc(c.getString(R.string.sheet_support)) +
                " <b dir=\"ltr\">" + esc(support) + "</b></p>"
        } else {
            ""
        }
        return """
<p class="thanks">${esc(c.getString(R.string.sheet_thanks, platform))}</p>
$line
<p class="foot">${esc(c.getString(R.string.sheet_footer, platform))}</p>"""
    }

    /** **يلفّ الجسمَ بورقةٍ كاملة.** */
    fun page(body: String): String =
        """<!doctype html><html dir="rtl" lang="ar"><head><meta charset="utf-8">
$STYLES</head><body>
$body
</body></html>"""

    /**
     * **يُسلّم الورقةَ إلى طابعة النظام.**
     *
     * **ويقول إن تعذّر** — جهازٌ بلا خدمةِ طباعةٍ يترك صاحبَه ينتظر
     * شاشةً لا تأتي.
     */
    fun print(context: Context, title: String, html: String) {
        val manager = context.getSystemService(Context.PRINT_SERVICE) as? PrintManager
        if (manager == null) {
            android.widget.Toast.makeText(
                context,
                context.getString(R.string.wal_print_failed),
                android.widget.Toast.LENGTH_LONG,
            ).show()
            return
        }
        val web = WebView(context)
        web.webViewClient = object : WebViewClient() {
            override fun onPageFinished(view: WebView, url: String) {
                manager.print(
                    title,
                    view.createPrintDocumentAdapter(title),
                    PrintAttributes.Builder()
                        .setMediaSize(PrintAttributes.MediaSize.ISO_A4)
                        .build(),
                )
                keep = null
            }
        }
        keep = web
        web.loadDataWithBaseURL(null, html, "text/html", "UTF-8", null)
    }

    /** **يحيّد ما يكسر HTML.** */
    fun esc(s: String): String = s
        .replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

    /**
     * **مبلغٌ برقمٍ معزول** — الفاصلةُ تقع بغير موضعها في نصٍّ عربيّ،
     * **فيُلَفّ الرقمُ وحدَه ويبقى الرمزُ خارجه.**
     */
    fun amount(value: Long, sign: String = ""): String {
        val text = money(value)
        val i = text.lastIndexOf(' ')
        if (i <= 0) return esc(text)
        return """<span class="n">${esc(sign + text.substring(0, i))}</span> """ +
            esc(text.substring(i + 1))
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **اللوغو يُدرَج بايتاتٍ لا عنواناً**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٩: «بطباعة التقارير لوغو رحال غو لا يظهر،
     *  يظهر حرف ر فقط».)
     *
     * **وصفحةُ الطباعة تُحمَّل بـ`loadDataWithBaseURL(null, …)`** — بلا
     * أصلٍ تُنسَب إليه العناوين، **فأيُّ `‎<img src="https://…">` لا
     * يُجلَب**، وتسقط الورقةُ إلى الحرف الاحتياطيّ.
     *
     * **فتُقرأ الصورةُ بايتاتٍ وتُدرَج في الصفحة نفسِها** — وهي كذلك
     * تُطبع بلا إنترنت.
     */
    suspend fun dataUri(api: com.rahalgo.shared.net.ApiClient, url: String): String {
        if (url.isEmpty()) return ""
        val raw = runCatching { api.bytes(url) }.getOrDefault(ByteArray(0))
        if (raw.isEmpty()) return ""
        val mime = when {
            url.endsWith(".svg", true) -> "image/svg+xml"
            url.endsWith(".png", true) -> "image/png"
            url.endsWith(".webp", true) -> "image/webp"
            else -> "image/jpeg"
        }
        return "data:" + mime + ";base64," +
            android.util.Base64.encodeToString(raw, android.util.Base64.NO_WRAP)
    }
}
