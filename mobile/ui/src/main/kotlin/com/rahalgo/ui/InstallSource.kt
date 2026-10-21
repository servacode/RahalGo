package com.rahalgo.ui

import android.content.Context
import android.os.Build

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من أين ثُبِّت التطبيق؟** (قرارُ المالك ٢٠٢٦-١٠-٢١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «يلي منزّل من رابطنا بيحدّث من رابطنا، ويلي من غوغل بلاي من غوغل». **والملفُّ
 * واحدٌ في الطريقين** (يُسحَب من إصدارات غوغل بلاي)، **فالمثبِّتُ وحدَه يفرّق.**
 *
 * **وسياسةُ غوغل** تمنع نسخةَ المتجر من تحديث نفسها من خارجه — فالمثبَّتُ من
 * المتجر يُحدَّث من المتجر، **والمثبَّتُ من رابطنا يفتح صفحةَ تنزيلنا.**
 */
fun installedFromPlay(context: Context): Boolean = runCatching {
    val pm = context.packageManager
    val installer = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
        pm.getInstallSourceInfo(context.packageName).installingPackageName
    } else {
        @Suppress("DEPRECATION")
        pm.getInstallerPackageName(context.packageName)
    }
    installer == "com.android.vending"
}.getOrDefault(false)
