package com.rahalgo.ui

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.core.content.FileProvider
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import java.net.HttpURLConnection
import java.net.URL

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التحديثُ من داخل التطبيق** (طلبُ المالك ٢٠٢٦-١٠-٠٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «التحديث من جوّا التطبيق أحلى وأركز، وما يضطر الشخص إذا ما عنده خبرة
 * بالتطبيقات يتخبّط يمين ويسار.» — **كان الزرُّ يفتح المتصفّح**، فيُنزَّل الملفُّ
 * ويضيع في «التنزيلات»، **ولا يعرف صاحبُه أين يكبس.**
 *
 * **الآن**: يُنزَّل الملفُّ هنا بشريط تقدّم، **ثمّ تُفتح شاشةُ «تثبيت» في أندرويد
 * مباشرة.** وتلك الكبسةُ وحدَها لا تُلغى — **أندرويد لا يسمح لتطبيقٍ أن يثبّت
 * نفسَه بلا موافقة صاحب الجوال.** ومن لم يأذن بعدُ بالتثبيت من هذا التطبيق يُفتح
 * له مفتاحُ الإذن أوّلاً.
 *
 * **ولتطبيقات التوزيع المباشر وحدَها** (السائق والمتجر والمندوب) — فهي التي
 * تعلن `REQUEST_INSTALL_PACKAGES`؛ وتطبيقُ الزبون يُحدَّث من بلاي.
 */
object SelfUpdate {

    private const val DIR = "updates"
    private const val FILE = "update.apk"

    /** **رابطُ الملفّ في المحرّك** — `‎/api/v1/public/app/<key>`. */
    fun apkUrl(appKey: String): String =
        AppCore.get().baseUrl.trimEnd('/') + "/api/v1/public/app/" + appKey

    /**
     * **ينزّل الملفَّ إلى ذاكرة التطبيق** — ويقول التقدّمَ من ٠ إلى ١.
     * يتبع التحويلات (بين http وhttps أيضاً) حتّى خمس قفزات.
     */
    suspend fun download(ctx: Context, url: String, onProgress: (Float) -> Unit): File =
        withContext(Dispatchers.IO) {
            val dir = File(ctx.cacheDir, DIR).apply { mkdirs() }
            val out = File(dir, FILE)
            val part = File(dir, "$FILE.part")
            var target = url
            var conn: HttpURLConnection? = null
            for (hop in 0 until 5) {
                val c = (URL(target).openConnection() as HttpURLConnection).apply {
                    instanceFollowRedirects = false
                    connectTimeout = 15_000
                    readTimeout = 30_000
                }
                val code = c.responseCode
                if (code in 300..399) {
                    val loc = c.getHeaderField("Location") ?: error("redirect without location")
                    target = URL(URL(target), loc).toString()
                    c.disconnect()
                    continue
                }
                if (code != HttpURLConnection.HTTP_OK) {
                    c.disconnect()
                    error("http $code")
                }
                conn = c
                break
            }
            val c = conn ?: error("too many redirects")
            try {
                val total = c.contentLengthLong
                c.inputStream.use { input ->
                    part.outputStream().use { output ->
                        val buf = ByteArray(64 * 1024)
                        var read = 0L
                        while (true) {
                            val n = input.read(buf)
                            if (n < 0) break
                            output.write(buf, 0, n)
                            read += n
                            if (total > 0) onProgress((read.toFloat() / total).coerceIn(0f, 1f))
                        }
                    }
                }
            } finally {
                c.disconnect()
            }
            if (out.exists()) out.delete()
            if (!part.renameTo(out)) error("rename failed")
            onProgress(1f)
            out
        }

    /** **أيملك هذا التطبيقُ إذنَ التثبيت؟** — وإلّا يُفتح مفتاحُه أوّلاً. */
    fun canInstall(ctx: Context): Boolean =
        Build.VERSION.SDK_INT < Build.VERSION_CODES.O || ctx.packageManager.canRequestPackageInstalls()

    /** **يفتح مفتاحَ «السماح بالتثبيت من هذا التطبيق»** في الإعدادات. */
    fun openInstallPermission(ctx: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        runCatching {
            ctx.startActivity(
                Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:" + ctx.packageName))
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            )
        }
    }

    /** **يفتح شاشةَ «تثبيت» في أندرويد** على الملفّ المنزَّل. */
    fun install(ctx: Context, apk: File): Boolean {
        val uri = FileProvider.getUriForFile(ctx, ctx.packageName + ".photos", apk)
        return runCatching {
            val i = Intent(Intent.ACTION_VIEW)
                .setDataAndType(uri, "application/vnd.android.package-archive")
                .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            if (ctx !is Activity) i.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            ctx.startActivity(i)
        }.isSuccess
    }
}
