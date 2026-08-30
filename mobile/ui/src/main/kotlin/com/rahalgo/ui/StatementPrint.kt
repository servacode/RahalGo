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
                     <td class="cell"><span class="n">${DocPrint.esc(t.createdAt.take(10))}</span></td>
                     <td>${DocPrint.esc(kindName(c, t.kind))}</td>
                     <td class="note">${DocPrint.esc(t.note.ifEmpty { t.orderNumber?.let { "#$it" } ?: "" })}</td>
                     <td class="cell $cls">${DocPrint.amount(kotlin.math.abs(t.amount), sign)}</td>
                     <td class="cell">${DocPrint.amount(running)}</td>
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

        return """
<!doctype html><html dir="rtl" lang="ar"><head><meta charset="utf-8">
${DocPrint.STYLES}</head><body>

${DocPrint.head(c, platform, logo)}

<p class="title">${DocPrint.esc(c.getString(R.string.wal_doc_title))}</p>
<div class="meta">
  <span><b>${DocPrint.esc(name)}</b> ${DocPrint.esc(phone)}</span>
  <span>${DocPrint.esc(c.getString(R.string.sheet_period))} ${DocPrint.esc(period)}</span>
</div>

<div class="sum">
  <span>${DocPrint.esc(c.getString(R.string.sheet_opening))}: <b>${DocPrint.amount(st.opening)}</b></span>
  <span>${DocPrint.esc(c.getString(R.string.sheet_closing))}: <b>${DocPrint.amount(st.closing)}</b></span>
</div>

${if (st.truncated) """<div class="warn">${DocPrint.esc(c.getString(R.string.sheet_truncated))}</div>""" else ""}

${
            if (st.transactions.isEmpty()) {
                """<p style="text-align:center;color:#5A6B75;padding:24px 0">""" +
                    DocPrint.esc(c.getString(R.string.sheet_empty)) + "</p>"
            } else {
                """<table>
  <thead><tr>
    <th class="cell">${DocPrint.esc(c.getString(R.string.sheet_col_date))}</th>
    <th>${DocPrint.esc(c.getString(R.string.sheet_col_kind))}</th>
    <th>${DocPrint.esc(c.getString(R.string.sheet_col_note))}</th>
    <th class="cell">${DocPrint.esc(c.getString(R.string.sheet_col_amount))}</th>
    <th class="cell">${DocPrint.esc(c.getString(R.string.sheet_col_running))}</th>
  </tr></thead>
  <tbody>$rows</tbody>
  <tfoot>
    <tr>
      <td colspan="3">${DocPrint.esc(c.getString(R.string.sheet_total_in))}</td>
      <td class="cell in">${DocPrint.amount(totalIn)}</td><td></td>
    </tr>
    <tr>
      <td colspan="3">${DocPrint.esc(c.getString(R.string.sheet_total_out))}</td>
      <td class="cell out">${DocPrint.amount(totalOut)}</td><td></td>
    </tr>
    <tr>
      <td colspan="3">${DocPrint.esc(c.getString(R.string.sheet_net))}</td>
      <td class="cell">${DocPrint.amount(totalIn - totalOut)}</td>
      <td class="cell">${DocPrint.amount(st.closing)}</td>
    </tr>
  </tfoot>
</table>"""
            }
        }

${DocPrint.foot(c, platform, support)}
</body></html>"""
    }

    /**
     * **والنصُّ يُهرَّب قبل أن يوضع في HTML**: ملاحظةٌ كتبها موظّفٌ فيها
     * `<` **تكسر الصفحة**، وأسوأُ منها ما يُحقن عمدا.
     */

    /**
     * **مبلغٌ للورقة — الرقمُ ملفوفٌ والرمزُ خارجَه.**
     *
     * **ونظيرُ `Money` في الويب حرفيّا** (`money.tsx`): لفّةٌ للرقم
     * وحدَه، **ويبقى ترتيبُه مع الرمز عربيّاً خارجَها.**
     */

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
