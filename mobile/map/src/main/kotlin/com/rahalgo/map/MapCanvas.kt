package com.rahalgo.map

import kotlinx.coroutines.withContext
import kotlinx.coroutines.Dispatchers
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.Alignment
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.getValue
import androidx.compose.material3.Text
import androidx.compose.material3.MaterialTheme
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.background
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import com.rahalgo.map.data.MapRuntime
import com.rahalgo.map.data.MapSourceResolver
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.Style

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحُ الخريطة — نمطٌ متّجهٌ من مستودعٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٢٢ و٢٣.)
 *
 * **كان يجلب نمطاً راستراً من المحرّك** (`/api/v1/public/map-style.json`)
 * **بينما `TripMap` تقرأ ملفّاً راستراً من الحزمة.** فشاشتان ترسمان
 * المدينةَ نفسَها من مصدرين، **وتصحيحُ واحدةٍ لا يبلغ الأخرى.**
 *
 * **وصارتا على `MapStyleRepository`** — والاختلافُ في الطبقات فوقَه
 * لا في النمط.
 *
 * # ولا يُقرأ العنوانُ إلّا حين تستقرّ
 *
 * **`onCameraIdle` لا `onCameraMove`** — **ونداءٌ في كلّ إطارٍ يُغرق
 * المزوّد**: تحريكةٌ واحدةٌ بالإصبع تُطلق عشرين نداء.
 *
 * # وإعادةُ تحميل النمط تمحو ما فوقه
 *
 * **البند ١٩** — فما يُرسم فوق الخريطة يُسجَّل في `surface.overlays`
 * **ويُعاد تركيبُه بعد كلّ تحميل.** ولوحُ الالتقاط لا طبقاتِ له،
 * **لكنّ الطريقَ واحدٌ لكلّ الشاشات فلا استثناء.**
 */
@Composable
fun MapCanvas(
    start: LatLng,
    onSettle: (LatLng) -> Unit,
    modifier: Modifier = Modifier,
    jumpTo: LatLng? = null,
    onJumped: () -> Unit = {},
    online: Boolean = true,
    /**
     * **درجاتُ تكبيرٍ تُطلب من خارج اللوحة** — زرّا `+` و`−`.
     *
     * **(بلاغُ المالك ٢٠٢٦-٠٨-٣١:** «تصغيرُ وتكبيرُ الخريطة صعبٌ
     * جدّاً».) **ومن يمسك هاتفَه بيدٍ ويفتح البابَ بالأخرى لا يملك
     * إصبعين** — فالقرصةُ وحدَها لا تكفي.
     *
     * **ورقمٌ يتبدّل هو الإشارة** — لا القيمةُ نفسُها: `+1` ثمّ `+1`
     * لا يتحرّكان مرّتين لو قُرئت القيمة.
     */
    zoomTick: Int = 0,
    zoomStep: Double = 0.0,
) {
    // ══════════════════════════════════════════════════════════════════
    // **والسمةُ تُستدلّ من لوحة التطبيق لا من النظام**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وللتطبيق مفتاحٌ يدويّ** («السمة الغامقة» في القائمة) — **ومن
    // قرأ سمةَ النظام خالفه**: يختار الغامقةَ وجهازُه فاتحٌ فتبقى
    // الخريطةُ بيضاءَ بين شريطين غامقين.
    //
    // **ولونُ الأرض أصدقُ من أيّ مفتاح** — هو ما يُرسم فعلاً. وأرضٌ
    // ضوؤها دون النصف ليلٌ.
    val canvas = com.rahalgo.design.Rahal.colors.canvas
    val night = canvas.luminance() < 0.5f
    val surface = rememberMapSurface()

    // ══════════════════════════════════════════════════════════════════
    // **والفهرسُ يُجلب قبل الربط — لا يُفترض موجوداً**
    // ══════════════════════════════════════════════════════════════════
    //
    // (إغلاقُ `TD-CUSTOMER-MAP-NO-MANIFEST`، قِيس ٢٠٢٦-٠٨-٢٢.)
    //
    // **كان الربطُ يقع في `remember` مباشرةً** — فإن لم يكن ثمّ فهرسٌ
    // عاد `Unavailable` **وخرجت `LaunchedEffect` صامتةً، فبقيت الشاشةُ
    // بيضاء.** وذلك ما شكا منه المالك في تطبيق الزبون: **لا رسالةَ ولا
    // سجلّ، خريطةٌ فارغةٌ فقط.**
    //
    // **والآن**: يُجلب الفهرسُ (طلبٌ واحدٌ ١٫٤ ك.ب يُخزَّن)، ثمّ يُربط،
    // **ومن سقط يُقال له.**
    var bound by remember(online) { mutableStateOf<MapRuntime.Binding?>(null) }
    LaunchedEffect(online) {
        MapStyleRepository.ensureManifest()
        // ══════════════════════════════════════════════════════════════
        // **والربطُ يقرأ القرصَ — فلا يقع في الخيط الرئيسيّ**
        // ══════════════════════════════════════════════════════════════
        //
        // **(بلاغُ المالك ٢٠٢٦-٠٩-٠٢:** «التطبيق لا يستجيب» — بصورةٍ من
        // شاشته: **نافذةُ أندرويد «تجريبي لا يستجيب»**.)
        //
        // **و«لا يستجيب» أخطرُ من انهيار**: الانهيارُ يُرى ويُبلَّغ عنه،
        // **وهذا يترك السائقَ ينظر إلى شاشةٍ ميّتةٍ وهو يقود.**
        //
        // **و`bind` تنادي `installedRegions()`** — تسرد مجلّدات الخرائط
        // وتقرأ ملفَّ وصفٍ لكلّ حزمة. **وكانت تُنادى في `LaunchedEffect`
        // بلا خيطٍ خلفيّ، أي في الخيط الرئيسيّ.**
        //
        // **وقرصٌ بطيءٌ أو حزمةٌ بأربعمئة ميغا يجعلانها مئاتِ
        // الميلّي‑ثانية** — وخمسُ ثوانٍ تكفي أندرويد ليقول «لا يستجيب».
        bound = withContext(Dispatchers.IO) {
            MapStyleRepository.bind(
            purpose = MapSourceResolver.Purpose.PICK_POINT,
            online = online,
                lat = start.latitude,
                lng = start.longitude,
            )
        }
    }

    val started = remember { booleanArrayOf(false) }

    LaunchedEffect(bound) {
        val now = bound ?: return@LaunchedEffect
        if (started[0]) return@LaunchedEffect
        started[0] = true
        val ready = now as? MapRuntime.Binding.Ready ?: return@LaunchedEffect
        surface.view.getMapAsync { map ->
            /**
             * **والنمطُ نصٌّ لا عنوان** — `fromJson` لا `fromUri`.
             *
             * **فالربطُ وقع في الشيفرة** (البند ٢٣): بلاطاتٌ وحروفٌ
             * وأيقونات. **ولو مُرّر عنوانٌ لأعادت MapLibre جلبَه
             * وربطَه بنفسها**، وضاع ما قرّرناه.
             */
            // ══════════════════════════════════════════════════════
            // **والليلُ يُحوَّل هنا — بعد الربط لا قبله**
            // ══════════════════════════════════════════════════════
            //
            // (طلبُ المالك ٢٠٢٦-٠٨-٢٥: «مشكلة الثيم الغامق… طبّقها
            //  كلَّها».)
            //
            // **وخريطةٌ بيضاءُ ساطعةٌ بين شريطين غامقين تُتعب العين
            // ليلاً** — ورآها المالكُ في شاشة اختيار العنوان.
            //
            // **والتحويلُ بعد الربط**: العناوينُ حُقنت فلا تُمسّ،
            // **والألوانُ وحدَها تُبدَّل.** انظر `MapNight`.
            val styled = if (night) MapNight.apply(ready.bound.json) else ready.bound.json
            // ══════════════════════════════════════════════════════════
            // **ولا تدويرَ ولا إمالةَ في ملتقط النقطة**
            // ══════════════════════════════════════════════════════════
            //
            // **(بلاغُ المالك ٢٠٢٦-٠٨-٣١:** «تصغيرُ وتكبيرُ الخريطة
            // وتحريكُها صعبٌ جدّاً · يجب أن تكون سهلةً سلسلةً لنقل
            // الدبّوس من مكانٍ لمكانٍ آخر».)
            //
            // **والإصبعان يفعلان أربعةَ أشياءَ في وقتٍ واحد**: يكبّران
            // ويدوّران ويميلان ويحرّكان. **فمن أراد أن يكبّر دوّر
            // الخريطةَ قليلاً**، ومن أراد أن يحرّك أمالها — **فيقاتلها
            // ليصل بالدبّوس إلى بابه.**
            //
            // **ولا حاجةَ إليهما هنا أصلاً**: يضع نقطةً على خريطةٍ
            // مسطّحة، **وشمالُها شمالٌ دائماً.** (والملاحةُ تدوّر
            // بنفسها — وتلك شاشةٌ أخرى.)
            //
            // **والنقرتان تكبّران** — والضغطُ المطوّل بإصبعٍ ثمّ السحب
            // يكبّر بيدٍ واحدة، **ومن يمسك هاتفَه بيدٍ ويفتح البابَ
            // بالأخرى لا يملك إصبعين.**
            map.uiSettings.isRotateGesturesEnabled = false
            map.uiSettings.isTiltGesturesEnabled = false
            map.uiSettings.isZoomGesturesEnabled = true
            map.uiSettings.isScrollGesturesEnabled = true
            map.uiSettings.isDoubleTapGesturesEnabled = true
            map.uiSettings.isQuickZoomGesturesEnabled = true
            map.setStyle(Style.Builder().fromJson(styled)) {
                // **وهنا يصير المطلوبُ محمَّلاً** (البندان ٩ و١٢).
                MapStyleRepository.onStyleLoaded(now)
                surface.overlays.restoreAll()
                map.moveCamera(CameraUpdateFactory.newLatLngZoom(start, 15.0))
                onSettle(start)
            }
            map.addOnCameraIdleListener { onSettle(map.cameraPosition.target ?: start) }
        }
    }

    LaunchedEffect(zoomTick) {
        if (zoomTick == 0 || zoomStep == 0.0) return@LaunchedEffect
        surface.view.getMapAsync { map ->
            map.animateCamera(CameraUpdateFactory.zoomBy(zoomStep), 220)
        }
    }
    LaunchedEffect(jumpTo) {
        val to = jumpTo ?: return@LaunchedEffect
        surface.view.getMapAsync { map ->
            map.animateCamera(CameraUpdateFactory.newLatLngZoom(to, 16.0))
        }
        onJumped()
    }

    Box(modifier) {
        AndroidView(factory = { surface.view }, modifier = Modifier.fillMaxSize())
        // ══════════════════════════════════════════════════════════════
        // **ولا تُترك بيضاءَ صامتة**
        // ══════════════════════════════════════════════════════════════
        //
        // **والبياضُ يُقرأ عطباً في التطبيق** — والسببُ شبكةٌ في الغالب.
        // **فتُقال العلّةُ ويُقال ما يفعله صاحبُها.**
        val why = bound as? MapRuntime.Binding.Unavailable
        if (why != null) {
            Box(
                Modifier
                    .fillMaxSize()
                    .background(MaterialTheme.colorScheme.surface),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = stringResource(R.string.map_unreachable),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(24.dp),
                )
            }
        }
    }
}
