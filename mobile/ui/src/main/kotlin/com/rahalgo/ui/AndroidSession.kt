package com.rahalgo.ui

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.rahalgo.shared.net.SessionStore

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حفظ الجلسة — مشفَّرا لا نصا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **توكن التجديد مفتاح الحساب**: من ملكه دخل باسم صاحبه حتى يُبطَل.
 * **وتخزينه نصا في `SharedPreferences`** يجعله يُقرأ من نسخة احتياطية،
 * ومن جهاز مكسور الحماية، **ومن أي أداة تصل مجلّد التطبيق.**
 *
 * **والتشفير هنا بمفتاح في عتاد الجهاز** (`Keystore`) — لا يخرج منه،
 * **فنسخة الملفّ وحدها لا تُقرأ.**
 *
 * # ولماذا في التطبيق لا في `shared`
 *
 * (`GROUND-RULES.md` §7.2 البند ٢.)
 *
 * **هذا أندرويد خالص**: `Context` و`Keystore` و`MasterKey`. **و`shared`
 * لا تعرف أندرويد** — تعرّف الحاجة (`SessionStore`) ولا تعرف كيف تُلبّى.
 * **فيوم iOS يُكتب مقابله بـ`Keychain` ولا يتغيّر سطر في القلب.**
 *
 * **وموضعه المؤقّت هنا**: مكانه `core-android` حين تُنشأ — وهي الوحدة
 * المخصّصة لما يلمس النظام.
 */
class AndroidSession(context: Context) : SessionStore {

    private val prefs = run {
        val key = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "rahalgo_session",
            key,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    override fun accessToken(): String = prefs.getString(ACCESS, "").orEmpty()

    override fun refreshToken(): String = prefs.getString(REFRESH, "").orEmpty()

    override fun save(access: String, refresh: String) {
        prefs.edit().putString(ACCESS, access).putString(REFRESH, refresh).apply()
    }

    override fun clear() {
        prefs.edit().remove(ACCESS).remove(REFRESH).apply()
    }

    private companion object {
        const val ACCESS = "access"
        const val REFRESH = "refresh"
    }
}
