package com.rahalgo.ui

import android.app.DownloadManager
import android.content.Context
import android.net.Uri
import android.os.Build
import android.os.Environment

/**
 * **تنزيلُ النسخة على الجهاز مباشرة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لمن نزّل
 * تطبيقَ الزبون من موقعنا: لا صفحةٌ في المتصفّح، **منزّلُ أندرويد يُظهر إشعاراً
 * فيكبس عليه ويثبّت بيده.** ولا إذنَ تثبيتٍ في التطبيق — غوغل بلاي يمنعه.
 */
object ApkDownload {
    fun start(context: Context, url: String, fileName: String): Boolean = runCatching {
        val dm = context.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
        val req = DownloadManager.Request(Uri.parse(url))
            .setMimeType("application/vnd.android.package-archive")
            .setTitle(context.getString(R.string.update_download_title))
            .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            req.setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, fileName)
        }
        dm.enqueue(req)
        Flash.ok(context.getString(R.string.update_download_started))
    }.isSuccess
}
