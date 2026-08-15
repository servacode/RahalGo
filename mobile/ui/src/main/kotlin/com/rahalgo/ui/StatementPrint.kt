package com.rahalgo.ui

import android.content.Context
import android.print.PrintAttributes
import android.print.PrintManager
import android.webkit.WebView
import android.webkit.WebViewClient
import com.rahalgo.shared.model.WalletStatement

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ورقةُ كشف الحساب — بقالب الويب نفسِه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (سأل المالك ٢٠٢٦-٠٨-١٣: «وشو استفدنا من الكشف ما فيه طباعة؟» ثمّ:
 *  «قالبُ الطباعة المستخدم بالويب لازم يُطبَّق بالتطبيق، مشان الكشف
 *  يكون رسميّاً للمنصّة — مو مجرّد كشفٍ عاديّ. يبني ثقةً كبيرةً مع
 *  العملاء والموظّفين والناس وقت تشوفه».)
 *
 * # وورقتان من دارٍ واحدة
 *
 * **الترويسةُ نفسُها التي تحملها فاتورةُ الويب وكشفُه** (`SheetHeader`):
 * العلامةُ يساراً، وتاريخُ الطباعة يميناً، **وشريطُ العلامة تحتهما.**
 *
 * **وكشفٌ يخرج بشكلٍ آخرَ من التطبيق يُقرأ ورقةً أخرى** — والسائقُ
 * يُريه لمن يختلف معه، **فورقتان بشكلين تُضعفان الاثنتين.**
 *
 * # والاسمُ من الإعدادات لا من الشيفرة
 *
 * (قاعدةُ المالك: «لا أريد أن تكتب اسم المنصة بأيّ مكانٍ أبدا».)
 *
 * **يبدّله من لوحته فتتبدّل الورقةُ في الويب والتطبيق معا.**
 *
 * # والرصيدُ الجاري عمودٌ لا حاشية
 *
 * **كشفٌ يقول «‎+٣٠٠» ولا يقول «فصار ٩٠٠»** يُجبر قارئَه على الجمع
 * بالورقة. **والعمودُ الجاري هو ما يجعله كشفَ حسابٍ لا قائمةَ حركات.**
 *
 * # ويُحتفظ بالعارض حتّى تنتهي الطباعة
 *
 * **`WebView` محلّيٌّ يُجمع بعد خروج الدالّة** — وأندرويد ينادي المحوّلَ
 * بعدها بلحظات، **فتُطبع صفحةٌ بيضاء.**
 */
object StatementPrint {

    private var keep: WebView? = null

    fun print(
        context: Context,
        name: String,
        phone: String,
        st: WalletStatement,
        period: String,
        platform: String,
        support: String,
        logo: String,
    ) {
        // **وجهازٌ بلا خدمة طباعةٍ يقول ذلك** — وضغطةٌ تُبتلَع بلا ردٍّ
        // **تُقرأ عطبا**، فيعيدها ثلاثاً ثمّ يترك.
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
                val title = context.getString(R.string.wal_doc_title)
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
        web.loadDataWithBaseURL(
            null,
            html(context, name, phone, st, period, platform, support, logo),
            "text/html",
            "UTF-8",
            null,
        )
    }

    private fun html(
        c: Context,
        name: String,
        phone: String,
        st: WalletStatement,
        period: String,
        platform: String,
        support: String,
        logo: String,
    ): String {
        // **والرصيدُ الجاري يُبنى من الافتتاحيّ صعودا** — والمحرّكُ يضمن
        // أنّ الافتتاحيَّ زائدَ المعروض يساوي الختاميّ.
        var running = st.opening
        var totalIn = 0L
        var totalOut = 0L
        val rows = StringBuilder()
        for (t in st.transactions.reversed()) {
            running += t.amount
            if (t.amount >= 0) totalIn += t.amount else totalOut += -t.amount
            val sign = if (t.amount >= 0) "+" else "−"
            val cls = if (t.amount >= 0) "in" else "out"
            rows.append(
                """<tr>
                     <td class="cell"><span class="n">${esc(t.createdAt.take(10))}</span></td>
                     <td>${esc(kindName(c, t.kind))}</td>
                     <td class="note">${esc(t.note.ifEmpty { t.orderNumber?.let { "#$it" } ?: "" })}</td>
                     <td class="cell $cls">${amount(kotlin.math.abs(t.amount), sign)}</td>
                     <td class="cell">${amount(running)}</td>
                   </tr>""",
            )
        }

        // ══════════════════════════════════════════════════════════════
        // **والشعارُ صورةُ المنصّة — لا حرفاً في مربّع**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٣.)
        //
        // **والحرفُ احتياطٌ لا أصل**: من لم يرفع شعاراً بعدُ يرى أوّلَ
        // حرفٍ من اسمه — **كما تفعل `BrandMark` في الويب.**
        val markHtml = if (logo.isNotEmpty()) {
            """<img class="logo" src="$logo" alt="">"""
        } else {
            """<div class="mark">${esc(platform.trim().take(1))}</div>"""
        }
        val today = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.US)
            .format(java.util.Date())
        val now = java.text.SimpleDateFormat("HH:mm", java.util.Locale.US)
            .format(java.util.Date())

        return """
<!doctype html><html dir="rtl" lang="ar"><head><meta charset="utf-8">
<style>
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
</style></head><body>

<div class="head">
  $markHtml
  <div class="when">
    ${esc(c.getString(R.string.sheet_printed_at))}
    <b>${esc(today)}</b>${esc(now)}
  </div>
</div>
<div class="rule"></div>

<p class="title">${esc(c.getString(R.string.wal_doc_title))}</p>
<div class="meta">
  <span><b>${esc(name)}</b> ${esc(phone)}</span>
  <span>${esc(c.getString(R.string.sheet_period))} ${esc(period)}</span>
</div>

<div class="sum">
  <span>${esc(c.getString(R.string.sheet_opening))}: <b>${amount(st.opening)}</b></span>
  <span>${esc(c.getString(R.string.sheet_closing))}: <b>${amount(st.closing)}</b></span>
</div>

${if (st.truncated) """<div class="warn">${esc(c.getString(R.string.sheet_truncated))}</div>""" else ""}

${
            if (st.transactions.isEmpty()) {
                """<p style="text-align:center;color:#5A6B75;padding:24px 0">""" +
                    esc(c.getString(R.string.sheet_empty)) + "</p>"
            } else {
                """<table>
  <thead><tr>
    <th class="cell">${esc(c.getString(R.string.sheet_col_date))}</th>
    <th>${esc(c.getString(R.string.sheet_col_kind))}</th>
    <th>${esc(c.getString(R.string.sheet_col_note))}</th>
    <th class="cell">${esc(c.getString(R.string.sheet_col_amount))}</th>
    <th class="cell">${esc(c.getString(R.string.sheet_col_running))}</th>
  </tr></thead>
  <tbody>$rows</tbody>
  <tfoot>
    <tr>
      <td colspan="3">${esc(c.getString(R.string.sheet_total_in))}</td>
      <td class="cell in">${amount(totalIn)}</td><td></td>
    </tr>
    <tr>
      <td colspan="3">${esc(c.getString(R.string.sheet_total_out))}</td>
      <td class="cell out">${amount(totalOut)}</td><td></td>
    </tr>
    <tr>
      <td colspan="3">${esc(c.getString(R.string.sheet_net))}</td>
      <td class="cell">${amount(totalIn - totalOut)}</td>
      <td class="cell">${amount(st.closing)}</td>
    </tr>
  </tfoot>
</table>"""
            }
        }

<p class="thanks">${esc(c.getString(R.string.sheet_thanks, platform))}</p>
${
            if (support.isNotEmpty()) {
                """<p class="support">""" +
                    esc(c.getString(R.string.sheet_support)) +
                    " <b dir=\"ltr\">" + esc(support) + "</b></p>"
            } else {
                ""
            }
        }
<p class="foot">${esc(c.getString(R.string.sheet_footer, platform))}</p>
</body></html>"""
    }

    /**
     * **والنصُّ يُهرَّب قبل أن يوضع في HTML**: ملاحظةٌ كتبها موظّفٌ فيها
     * `<` **تكسر الصفحة**، وأسوأُ منها ما يُحقن عمدا.
     */
    private fun esc(s: String): String = s
        .replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

    /**
     * **مبلغٌ للورقة — الرقمُ ملفوفٌ والرمزُ خارجَه.**
     *
     * **ونظيرُ `Money` في الويب حرفيّا** (`money.tsx`): لفّةٌ للرقم
     * وحدَه، **ويبقى ترتيبُه مع الرمز عربيّاً خارجَها.**
     */
    private fun amount(value: Long, sign: String = ""): String {
        val text = money(value)
        val i = text.lastIndexOf(' ')
        if (i <= 0) return esc(text)
        val digits = text.substring(0, i)
        val symbol = text.substring(i + 1)
        return """<span class="n">${esc(sign + digits)}</span> ${esc(symbol)}"""
    }

    /** **اسمُ النوع بالعربيّة** — والمجهولُ بمفتاحه ليُعرف. */
    private fun kindName(c: Context, kind: String): String = when (kind) {
        "payout" -> c.getString(R.string.wal_k_payout)
        "driver_earning" -> c.getString(R.string.wal_k_driver_earning)
        "commission" -> c.getString(R.string.wal_k_commission)
        "compensation" -> c.getString(R.string.wal_k_compensation)
        "penalty" -> c.getString(R.string.wal_k_penalty)
        "refund" -> c.getString(R.string.wal_k_refund)
        "adjustment" -> c.getString(R.string.wal_k_adjustment)
        "reward" -> c.getString(R.string.wal_k_reward)
        "settlement" -> c.getString(R.string.wal_k_settlement)
        else -> kind
    }
}
