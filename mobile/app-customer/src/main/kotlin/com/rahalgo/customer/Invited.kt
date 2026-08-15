package com.rahalgo.customer

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.util.Log
import androidx.core.content.edit
import com.android.installreferrer.api.InstallReferrerClient
import com.android.installreferrer.api.InstallReferrerStateListener

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رمزُ من دعاه — من الرابط أو من المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (سؤالُ المالك ٢٠٢٦-٠٨-١٤: «رابطُ الدعوة يجب أن يوجّهنا إلى أين لكي
 *  يتمّ تنزيل البرنامج مسبقاً، ثمّ الرابطُ يفتح تسجيلَ حسابٍ جديدٍ من
 *  التطبيق؟».)
 *
 * # وطريقان لا طريق
 *
 * **١ · والتطبيقُ مثبَّت** — الرابطُ يفتحه مباشرةً عند شاشة التسجيل
 * والرمزُ فيه (`https://…/signup?ref=CODE`).
 *
 * **٢ · وغيرُ مثبَّت** — يفتح الويبَ، **ومن هناك يُنزّله من المتجر.**
 * **والرمزُ يضيع هنا لولا `InstallReferrer`**: يمشي مع رابط المتجر،
 * **فيقرؤه التطبيقُ في أوّل إقلاعٍ بعد التثبيت.**
 *
 * **وهذه هي «الوجهةُ المؤجَّلة»** — بلاها يُنزّل المدعوُّ التطبيقَ
 * ويُسجّل **فلا يُنسب لمن دعاه**، ولا يأخذ أحدُهما مكافأة.
 *
 * # ولا يُقرأ مرّتين
 *
 * **يُكتب أنّه قُرئ** — ونداءُ المتجر يُجيب في كلّ إقلاع، **ورمزٌ
 * يُعاد ملؤه بعد أن سجّل صاحبُه يُربكه.**
 */
object Invited {

    private const val FILE = "rahalgo_invite"
    private const val CODE = "code"
    private const val ASKED = "asked_store"

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)

    /** **الرمزُ المحفوظ** — أو فارغ. */
    fun code(context: Context): String = prefs(context).getString(CODE, "").orEmpty()

    fun clear(context: Context) {
        prefs(context).edit { remove(CODE) }
    }

    /**
     * **يقرأ الرمزَ من رابطٍ فُتح به التطبيق.**
     *
     * **ويقبل شكلين**: `?ref=CODE` كما يبنيه المحرّك، **و`/i/CODE`**
     * لرابطٍ قصيرٍ إن أُضيف يوما.
     */
    fun fromLink(context: Context, intent: Intent?) {
        val data: Uri = intent?.data ?: return
        val code = data.getQueryParameter("ref")
            ?: data.pathSegments?.takeIf { it.size == 2 && it[0] == "i" }?.get(1)
            ?: return
        if (code.isBlank()) return
        prefs(context).edit { putString(CODE, code.trim()) }
        Log.i("RahalGo/invite", "رمزُ دعوةٍ من الرابط")
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ومن نزّله من المتجر يُسأل المتجرُ عن رمزه**
     * ══════════════════════════════════════════════════════════════════
     *
     * **ويُنادى مرّةً في العمر** — أوّلَ إقلاعٍ بعد التثبيت: **الجوابُ
     * لا يتبدّل، ونداءٌ في كلّ إقلاعٍ وصلةٌ إلى المتجر بلا فائدة.**
     *
     * **وفشلُه لا يُسقط شيئا**: من نزّله من ملفٍّ مباشرٍ لا متجرَ له،
     * **ومن دخل بلا دعوةٍ حسابُه كامل.**
     */
    fun fromStore(context: Context) {
        val p = prefs(context)
        if (p.getBoolean(ASKED, false) || code(context).isNotEmpty()) return
        val client = InstallReferrerClient.newBuilder(context).build()
        runCatching {
            client.startConnection(object : InstallReferrerStateListener {
                override fun onInstallReferrerSetupFinished(code: Int) {
                    runCatching {
                        if (code == InstallReferrerClient.InstallReferrerResponse.OK) {
                            // **والقيمةُ سلسلةُ استعلامٍ كما كُتبت في
                            // رابط المتجر** — `ref=CODE&utm_source=…`.
                            val raw = client.installReferrer.installReferrer.orEmpty()
                            val ref = Uri.parse("x://x?$raw").getQueryParameter("ref").orEmpty()
                            if (ref.isNotBlank()) {
                                p.edit { putString(CODE, ref.trim()) }
                                Log.i("RahalGo/invite", "رمزُ دعوةٍ من المتجر")
                            }
                        }
                        p.edit { putBoolean(ASKED, true) }
                        client.endConnection()
                    }
                }

                override fun onInstallReferrerServiceDisconnected() {
                    // **ولا تُعاد المحاولة** — تُسأل في الإقلاع التالي.
                }
            })
        }.onFailure {
            Log.w("RahalGo/invite", "تعذّر سؤالُ المتجر عن الدعوة", it)
        }
    }
}
