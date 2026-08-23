# ══════════════════════════════════════════════════════════════════════
# **اسمُ الملفّ ورقمُ السطر يبقيان بعد التشويش**
# ══════════════════════════════════════════════════════════════════════
#
# **بلاهما يصل تقريرُ الانهيار بأسماءٍ من حرفين**: `a.b(Unknown Source)`
# — **فيُعرف أنّ التطبيقَ سقط ولا يُعرف أين.** والانهيارُ الذي لا يُقرأ
# لا يُصلَح.
#
# **وهما لا يكشفان الشيفرة**: الأسماءُ تبقى مشوّشة، **والخريطةُ التي
# تفكّها تُرفع إلى لوحتك وحدَك** مع كلّ إصدار.
-keepattributes SourceFile,LineNumberTable
-renamesourcefileattribute SourceFile

# ══════════════════════════════════════════════════════════════════════
# **وأسماءُ الاستثناءات تبقى**
# ══════════════════════════════════════════════════════════════════════
#
# **رسالةُ الخطأ تذكر اسمَ صنف الاستثناء** حين لا يكون خطأَ خادمٍ
# معروفاً (`apiError`). **وبعد التشويش يصير `gs1`** — فيقرؤه صاحبُ
# التطبيق ولا يعني شيئاً، **ويقرؤه من يُصلح فلا يعرف أين يبحث.**
#
# **ووقع فعلاً ٢٠٢٦-٠٨-١٨**: عُرض «تعذّر إتمام الطلب (gs1)» ولم يُعرف
# أنّه `CancellationException` إلّا بقراءة سجلّ الجهاز.
#
# **والاسمُ وحدَه يبقى** (`keepnames`) — **لا الأصنافُ ولا أعضاؤها**:
# الاستثناءاتُ لا تحمل منطقاً، وأسماؤها لا تكشف شيفرة.
-keepnames class * extends java.lang.Throwable

# ══════════════════════════════════════════════════════════════════════
# **ما يُنادى بالانعكاس — ولا يراه المُشذِّب**
# ══════════════════════════════════════════════════════════════════════
#
# (قِيس ٢٠٢٦-٠٨-٢٣: بناءُ الإصدار ينهار قبل أن تُرسم شاشةٌ واحدة.)
#
#     FATAL EXCEPTION: main
#     NoSuchMethodException: androidx.work.impl.WorkDatabase_Impl.<init>
#
# **حذف R8 بانيَ قاعدةِ بيانات `WorkManager`** — **لأنّه لا يُنادى من
# شيفرتنا بل بالانعكاس عند الإقلاع.** والمُشذِّبُ يقرأ النداءاتِ
# المكتوبةَ ولا يقرأ الانعكاس، **فيحذف ما يظنّه ميّتاً وهو عصبُ
# الإقلاع.**
#
# **وبناءُ التطوير سليمٌ تماماً** — بلا تشويشٍ ولا ضغط. **فالعطبُ لا
# يظهر إلّا في ما يُرفع**، وهو أخطرُ أنواعه.

# ── Room و`WorkManager` ───────────────────────────────────────────────
#
# **وبناتُ `_Impl` تُولَّد وتُنادى بالانعكاس** — فتُحفظ بأسمائها.
-keep class * extends androidx.room.RoomDatabase { <init>(); }
-keep class androidx.work.impl.WorkDatabase_Impl { *; }
-keep class * extends androidx.work.Worker { <init>(...); }
-keep class * extends androidx.work.ListenableWorker { <init>(...); }
-keep class androidx.work.impl.** { *; }
-dontwarn androidx.work.**

# ── وبادئُ الإقلاع ────────────────────────────────────────────────────
#
# **`androidx.startup` ينادي مُهيّئاتِه بالانعكاس** — وهو الذي سقط
# عليه الإقلاعُ حرفيّاً.
-keep class * extends androidx.startup.Initializer { *; }
-keep class androidx.startup.** { *; }

# ── وما يُسلسَل ───────────────────────────────────────────────────────
#
# **ونماذجُ العقد تُقرأ بأسماء حقولها** — **واسمٌ يُشوَّش يجعل الردَّ
# يُقرأ فارغاً بلا خطأ**، وهو صمتٌ أسوأُ من انهيار.
-keepattributes *Annotation*, InnerClasses, Signature
-keepclassmembers class ** {
    @kotlinx.serialization.SerialName <fields>;
}
-keep,includedescriptorclasses class com.rahalgo.**$$serializer { *; }
-keepclassmembers class com.rahalgo.** {
    *** Companion;
    kotlinx.serialization.KSerializer serializer(...);
}
-keepclasseswithmembers class com.rahalgo.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# ── والخريطة ──────────────────────────────────────────────────────────
#
# **MapLibre تحمّل شيفرةً أصليّةً وتنادي أصنافاً بالانعكاس.**
-keep class org.maplibre.android.** { *; }
-dontwarn org.maplibre.android.**
