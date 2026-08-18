package com.rahalgo.ui

import android.content.Context
import coil3.ImageLoader
import coil3.SingletonImageLoader
import coil3.network.okhttp.OkHttpNetworkFetcherFactory
import coil3.request.crossfade

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جالبُ الصور يُسجَّل بيدنا — لا يُترك لـ`ServiceLoader`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شكوى المالك ٢٠٢٦-٠٨-١٨ بصورةِ شاشة: لافتاتُ السلايدر إطاراتٌ فارغة.)
 *
 * # ما كان يقع
 *
 * **`coil3` تكتشف جالبَ الشبكة بـ`ServiceLoader`** — سطرٌ في
 * `META-INF/services` يشير إلى صنفٍ **لا تناديه شيفرةٌ أبدا.**
 *
 * **وR8 يحذف ما لا يُنادى.** فيُبنى الإصدارُ بلا جالبِ شبكة: **كلُّ
 * صورةٍ بعيدةٍ تفشل** — ولا انهيارَ ولا رسالة، **بل إطارٌ فارغ.**
 *
 * **وفي التطوير تعمل** (`isMinifyEnabled = false`) — **وهو أخبثُ ما
 * يكون**: يُبنى ويُجرَّب فيُرى سليماً، **ولا يظهر العطبُ إلّا في
 * النسخة التي تصل الناس.**
 *
 * # ولماذا لم يُكتشف قبل اليوم
 *
 * **لم تكن في المنصّة صورةٌ بعيدةٌ تُعرض فعلاً**: أقسامُ السوق بلا
 * صور، **وصورةُ الحساب لم يرفعها أحد** — فكان يُرى حرفُها الاحتياطيُّ
 * ويُظنّ صواباً. **واللافتاتُ أوّلُ صورةٍ حقيقيّة، فكشفتها.**
 *
 * # ولماذا تسجيلٌ صريحٌ لا قاعدةُ إبقاء
 *
 * **قاعدةٌ في `proguard-rules` تُبقي أصنافاً بالاسم** — وتُكسر يومَ
 * تُبدَّل المكتبةُ أسماءَها الداخليّة، **بلا أن يصرخ شيء.**
 *
 * **والتسجيلُ الصريحُ نداءٌ في الشيفرة** — يُكسَر عند البناء لا عند
 * التشغيل.
 */
object Images {

    @Volatile private var wired = false

    /** **يُنادى مرّةً في `onCreate`** — قبل أوّل صورةٍ تُطلب. */
    fun install(context: Context) {
        if (wired) return
        wired = true
        val app = context.applicationContext
        SingletonImageLoader.setSafe {
            ImageLoader.Builder(app)
                .components { add(OkHttpNetworkFetcherFactory()) }
                // **وذوبانٌ خفيفٌ عند الوصول** — **وصورةٌ تظهر فجأةً
                // فوق أرضٍ باهتةٍ تُقرأ ومضة.**
                .crossfade(true)
                .build()
        }
    }
}
