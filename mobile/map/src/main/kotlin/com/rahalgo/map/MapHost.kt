package com.rahalgo.map

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import org.maplibre.android.MapLibre
import org.maplibre.android.WellKnownTileServer
import org.maplibre.android.maps.MapView

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أساسُ الخريطة — تهيئةٌ ولوحٌ ودورةُ حياة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٠ من خطّة الملاحة، بأمر المالك ٢٠٢٦-٠٨-٢٠: **فصلٌ معماريٌّ
 *  بحتٌ بلا أيّ تغييرٍ وظيفيّ.**)
 *
 * # ما هذه الوحدة وما ليست
 *
 * **`map-core` ترسم ولا تقرّر.** لا تعرف طلباً ولا سائقاً ولا رحلة —
 * **ومن أراد منطقَ ملاحةٍ فليضعه في `driver-navigation`.**
 *
 * **ولا تستورد هذه الوحدةُ وحدةَ الملاحة أبداً** — والاتّجاهُ في جهةٍ
 * واحدة، **وحارسٌ يمنع عكسَه** (`check-map-core-pure.mjs`).
 *
 * # ولماذا جُمعت هنا
 *
 * **كانت هذه الأسطرُ نفسُها في موضعين**: `MapCanvas` في هذه الوحدة،
 * و`TripMap` في تطبيق السائق. **ونسختان من دورةِ حياةٍ تفترقان يومَ
 * يُضاف `onLowMemory` في إحداهما** — فتُسرّب الأخرى.
 */

/**
 * **تُهيَّأ المكتبةُ من مكانٍ واحدٍ يناديه الجميع.**
 *
 * **وكانت تُهيَّأ عند بناء الخريطة وحدَها** — فحين تُقرأ المناطقُ
 * المنزَّلةُ قبل أن تُفتح خريطةٌ يسقط التطبيق:
 *
 *     MapLibreConfigurationException
 *
 * **ولم يظهر على المحاكي** لأنّ الخريطةَ كانت تُفتح أوّلاً. **وظهر على
 * أوّل جهازٍ حقيقيّ** (٢٠٢٦-٠٨-١٢).
 *
 * **والمكتبةُ تتجاهل النداءَ الثاني** — فالنداءُ الزائدُ لا يضرّ.
 *
 * **ولا مفتاح**: MapLibre لا تطلب حساباً. **أمّا نوعُ خادم البلاطات فلا
 * يقبل الفراغ** — مُرِّر فارغاً مرّةً فسقط التطبيقُ عند أوّل فتح.
 */
fun ensureMapLibre(context: Context) {
    MapLibre.getInstance(context.applicationContext, null, WellKnownTileServer.MapLibre)
}

/**
 * **لوحُ خريطةٍ يُبنى مرّةً في عمر الشاشة.**
 *
 * **و`MapView` تُبنى في كلّ رسمٍ تُسرّب السياقَ والذاكرةَ الأصليّة** —
 * فتُذكَر (`remember`).
 */
@Composable
fun rememberMapView(): MapView {
    val context = LocalContext.current
    return remember {
        ensureMapLibre(context)
        MapView(context)
    }
}

/**
 * **دورةُ حياةِ اللوح — تُربط بدورة حياة الشاشة.**
 *
 * **ولوحٌ لا يُوقَف عند مغادرة الشاشة يبقى يرسم** — يستنزف البطّاريّةَ
 * ويُبقي مؤشّرَ الموقع حيّاً.
 *
 * **ولا يُنادى `onCreate` هنا**: `MapCanvas` تناديه في بانيها
 * و`TripMap` لا تناديه — **وتوحيدُهما تغييرُ سلوك**، وهو خارجَ المرحلة
 * ٠. مسجَّلٌ في تقريرها.
 */
@Composable
fun MapLifecycle(view: MapView) {
    DisposableEffect(Unit) {
        view.onStart()
        view.onResume()
        onDispose {
            view.onPause()
            view.onStop()
            view.onDestroy()
        }
    }
}
