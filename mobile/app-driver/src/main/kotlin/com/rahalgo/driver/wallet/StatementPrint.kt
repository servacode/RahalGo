package com.rahalgo.driver.wallet

import android.content.Context
import android.print.PrintAttributes
import android.print.PrintManager
import android.webkit.WebView
import android.webkit.WebViewClient
import com.rahalgo.driver.R
import com.rahalgo.driver.ui.money
import com.rahalgo.shared.model.WalletStatement

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طباعةُ كشف الحساب — أو حفظُه ورقةً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (سأل المالك ٢٠٢٦-٠٨-١٣: «وشو استفدنا من الكشف ما فيه طباعة؟».)
 *
 * # وكشفٌ لا يخرج من الشاشة ليس كشفا
 *
 * **كشفُ الحساب حجّةٌ تُقدَّم**: يُريه للمكتب حين يختلفان على رقم،
 * **أو يحفظه لنفسه.** وشاشةٌ تُقرأ ثمّ تُغلق **لا تصلح حجّة.**
 *
 * # ولماذا نظامُ الطباعة لا مشاركةُ نصّ
 *
 * **نظامُ أندرويد يعطي الاثنين بنداءٍ واحد**: طابعةً إن وُجدت،
 * **و«حفظ كـPDF» في كلّ جهاز** — ومن أراد إرسالَه على واتساب أرسل
 * الملفّ.
 *
 * **ومشاركةُ نصٍّ خام تفقد الترتيب**: أعمدةُ المبالغ تنهار في محادثة،
 * **ورقمٌ تحت رقمٍ يصير سطراً واحدا.**
 *
 * # و`WebView` لأنّه الطريقُ القصير
 *
 * **بناءُ `PrintDocumentAdapter` بيدٍ يعني رسمَ كلِّ سطرٍ على `Canvas`**
 * وحسابَ فواصل الصفحات. **و`WebView` يعطيه جاهزاً من HTML** — وهي
 * صفحةٌ من عشرين سطراً.
 *
 * # ويُحتفظ به حتّى تنتهي الطباعة
 *
 * **`WebView` محلّيٌّ يُجمع بعد خروج الدالّة** — وأندرويد ينادي المحوّلَ
 * بعدها بلحظات، **فتُطبع صفحةٌ بيضاء.** فيُمسك في حقلٍ حتّى تُغلق
 * النافذة.
 */
object StatementPrint {

    private var keep: WebView? = null

    fun print(context: Context, name: String, st: WalletStatement, period: String) {
        val manager = context.getSystemService(Context.PRINT_SERVICE) as? PrintManager
            ?: return

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
        web.loadDataWithBaseURL(null, html(context, name, st, period), "text/html", "UTF-8", null)
    }

    /**
     * **ورقةُ الكشف** — عربيّةٌ من اليمين.
     *
     * **والنصُّ يُهرَّب قبل أن يوضع في HTML**: ملاحظةٌ كتبها موظّفٌ فيها
     * `<` **تكسر الصفحة**، وأسوأُ منها ما يُحقن عمدا.
     */
    private fun html(context: Context, name: String, st: WalletStatement, period: String): String {
        val rows = st.transactions.joinToString("") { t ->
            val sign = if (t.amount >= 0) "+" else "−"
            val cls = if (t.amount >= 0) "in" else "out"
            """<tr>
                 <td>${esc(t.createdAt.take(10))}</td>
                 <td>${esc(kindName(context, t.kind))}</td>
                 <td>${t.orderNumber?.let { "#$it" } ?: ""}</td>
                 <td class="$cls">$sign${esc(money(kotlin.math.abs(t.amount)))}</td>
               </tr>"""
        }
        return """
<!doctype html><html dir="rtl" lang="ar"><head><meta charset="utf-8">
<style>
  body { font-family: sans-serif; padding: 18px; color: #07283A; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  .sub { color: #5A6B75; font-size: 12px; margin-bottom: 14px; }
  .sum { display: flex; justify-content: space-between;
         border: 1px solid #DDD; border-radius: 8px; padding: 10px; margin-bottom: 14px; }
  table { width: 100%; border-collapse: collapse; font-size: 12px; }
  th, td { text-align: right; padding: 6px 4px; border-bottom: 1px solid #EEE; }
  th { color: #5A6B75; font-weight: normal; }
  .in { color: #1E9E5A; } .out { color: #D64545; }
</style></head><body>
<h1>${esc(context.getString(R.string.wal_doc_title))}</h1>
<div class="sub">${esc(context.getString(R.string.wal_doc_for, name, period))}</div>
<div class="sum">
  <span>${esc(context.getString(R.string.wal_opening))}: ${esc(money(st.opening))}</span>
  <span>${esc(context.getString(R.string.wal_closing))}: ${esc(money(st.closing))}</span>
</div>
<table>$rows</table>
</body></html>"""
    }

    private fun esc(s: String): String = s
        .replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

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
