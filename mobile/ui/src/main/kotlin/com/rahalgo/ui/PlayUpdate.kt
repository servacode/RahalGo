package com.rahalgo.ui

import android.app.Activity
import com.google.android.play.core.appupdate.AppUpdateManagerFactory
import com.google.android.play.core.appupdate.AppUpdateOptions
import com.google.android.play.core.install.model.AppUpdateType
import com.google.android.play.core.install.model.UpdateAvailability

/**
 * **تحديثُ بلاي من داخل التطبيق** (طلبُ المالك ٢٠٢٦-١٠-٠٨) — شاشةُ بلاي تنزّل
 * النسخةَ وتعيد تشغيلَ التطبيق، **بلا خروجٍ إلى المتجر والبحث عن الزرّ.**
 *
 * **ولا يعمل إلّا لمن نزّل من بلاي** — ومن نزّل من الموقع (أو بلاي لم يعلم بعدُ
 * بالنسخة) يُفتح له المتجرُ كما كان (`onUnavailable`).
 */
object PlayUpdate {
    fun start(activity: Activity, onUnavailable: () -> Unit) {
        val manager = runCatching { AppUpdateManagerFactory.create(activity) }.getOrNull()
            ?: return onUnavailable()
        manager.appUpdateInfo
            .addOnSuccessListener { info ->
                val can = info.updateAvailability() == UpdateAvailability.UPDATE_AVAILABLE ||
                    info.updateAvailability() == UpdateAvailability.DEVELOPER_TRIGGERED_UPDATE_IN_PROGRESS
                if (can && info.isUpdateTypeAllowed(AppUpdateType.IMMEDIATE)) {
                    val started = runCatching {
                        manager.startUpdateFlow(info, activity, AppUpdateOptions.defaultOptions(AppUpdateType.IMMEDIATE))
                    }.isSuccess
                    if (!started) onUnavailable()
                } else {
                    onUnavailable()
                }
            }
            .addOnFailureListener { onUnavailable() }
    }
}
