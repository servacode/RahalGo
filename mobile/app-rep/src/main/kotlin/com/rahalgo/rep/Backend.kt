package com.rahalgo.rep

import android.content.Context
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Core

/**
 * ══════════════════════════════════════════════════════════════════════
 * **وصلةُ تطبيق المندوب بالمحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وكلُّ ما فيها في `ui.Core`** — الجلسةُ والدخولُ و`me` والحسابُ
 * والمحفظةُ والبثُّ الحيّ. **وما يخصّ المندوبَ في `RepApi`.**
 */
object Backend {

    const val BASE_URL = "https://rahalgo-api.onrender.com"

    /**
     * **نوعُ العميل — كما تعرفه قائمةُ المحرّك المغلقة.**
     *
     * (`identity/client_kind.go`: `{android|ios}-{customer|driver|merchant|rep}`.)
     *
     * **وجلسةٌ لكلّ نوع**: فمن دخل التطبيقَ لا يُخرج نفسَه من المتصفّح.
     */
    const val CLIENT = "android-rep"

    fun of(context: Context): Core = AppCore.install(context, BASE_URL, CLIENT)
}
