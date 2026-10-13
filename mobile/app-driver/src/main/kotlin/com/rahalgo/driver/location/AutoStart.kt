package com.rahalgo.driver.location

import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.Settings

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«التشغيلُ التلقائيّ» في جوّالات شاومي وأوبو وفيفو وهواوي** (٢٠٢٦-١٠-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قِيس على الإنتاج: سائقٌ في الدوام بلا موضعٍ أربعَ ساعات.) **هذه الجوّالاتُ
 * تقتل خدمةَ الموقع ولو استُثني التطبيقُ من توفير البطارية** — ولها مفتاحٌ
 * خاصٌّ بها لا يفتحه أندرويد. **فتُفتح صفحتُه إن وُجدت، وإلّا صفحةُ التطبيق.**
 */
object AutoStart {
    private const val PREFS = "driver_autostart"

    private val brands = setOf("xiaomi", "redmi", "poco", "oppo", "realme", "oneplus", "vivo", "iqoo", "huawei", "honor", "tecno", "infinix", "itel")

    private val screens = listOf(
        ComponentName("com.miui.securitycenter", "com.miui.permcenter.autostart.AutoStartManagementActivity"),
        ComponentName("com.coloros.safecenter", "com.coloros.safecenter.permission.startup.StartupAppListActivity"),
        ComponentName("com.coloros.safecenter", "com.coloros.safecenter.startupapp.StartupAppListActivity"),
        ComponentName("com.oplus.safecenter", "com.oplus.safecenter.permission.startup.StartupAppListActivity"),
        ComponentName("com.vivo.permissionmanager", "com.vivo.permissionmanager.activity.BgStartUpManagerActivity"),
        ComponentName("com.iqoo.secure", "com.iqoo.secure.ui.phoneoptimize.AddWhiteListActivity"),
        ComponentName("com.huawei.systemmanager", "com.huawei.systemmanager.startupmgr.ui.StartupNormalAppListActivity"),
        ComponentName("com.huawei.systemmanager", "com.huawei.systemmanager.optimize.process.ProtectActivity"),
        ComponentName("com.transsion.phonemaster", "com.cyin.himgr.autostart.AutoStartActivity"),
    )

    /** **أجوّالُه من هذه العائلة؟** */
    fun applies(): Boolean = Build.MANUFACTURER.lowercase() in brands || Build.BRAND.lowercase() in brands

    /** **أفتح الصفحةَ من قبل؟** — فلا تبقى البطاقةُ بعد أن فعل. */
    fun done(context: Context): Boolean =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean("done", false)

    fun open(context: Context) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putBoolean("done", true).apply()
        for (c in screens) {
            val i = Intent().setComponent(c).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            if (runCatching { context.startActivity(i) }.isSuccess) return
        }
        runCatching {
            context.startActivity(
                Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:" + context.packageName))
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            )
        }
    }
}
