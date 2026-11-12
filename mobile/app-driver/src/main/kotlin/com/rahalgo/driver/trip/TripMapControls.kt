package com.rahalgo.driver.trip

import com.rahalgo.navigation.TripMap
import com.rahalgo.navigation.MarkerIcons
import androidx.compose.animation.AnimatedVisibility
import com.rahalgo.ui.Countdown
import androidx.compose.foundation.layout.IntrinsicSize
import com.rahalgo.design.Rahal
import com.rahalgo.ui.etaText
import com.rahalgo.ui.minutesShort
import com.rahalgo.ui.dist
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.material3.Surface
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.gestures.detectVerticalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.layout.width
import androidx.compose.ui.layout.layout
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import com.rahalgo.driver.BuildConfig
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import com.rahalgo.driver.R
import com.rahalgo.ui.money
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.FailReasonItem
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.RahalTextButton
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أزرارُ الخريطة** — ما يطفو فوقها ويُضغط بإبهامٍ واحدٍ على مقود.
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أُخرج من `TripScreen.kt`** (٢٠٢٦-٠٨-٢٣، فحصُ التطبيق):
 * **ألفان ومئتا سطرٍ في ملفٍّ واحدٍ يصعب تعديلُه بلا كسر.**
 */

/**
 * **أزرارُ الخريطة** — فوق البطاقة وفي جهة اليمين.
 *
 * (تصحيح المالك ٢٠٢٦-٠٨-١٢: «الأزرار بالعربيّ يجب أن تكون على اليمين».)
 *
 * **واليمينُ جهةُ الإبهام في شاشةٍ عربيّة** — كما تقع كلُّ أزرار
 * التطبيق: **ومن وضعها يسارا** جعل صاحبَها يعبر الشاشةَ بيده وهو يقود.
 *
 * **والملاحةُ مكتوبةٌ لا أيقونةً وحدَها**: هي الوحيدةُ التي تُخرجه من
 * التطبيق، **وبابٌ يَخرج منه بلا اسم** يُضغط بالخطأ فيجد نفسَه في
 * تطبيقٍ آخر ولا يعرف لماذا.
 */
@Composable
internal fun MapButtons(
    onRecenter: () -> Unit,
    onChat: () -> Unit,
    /** **ولا حديثَ في «لدي توصيلة»** (٢٠٢٦-١٠-٠٣) — المستلمُ ليس على التطبيق. */
    showChat: Boolean = true,
    chatting: Boolean,
    chatUnread: Int,
    /** **أالصوتُ مكتوم؟** — (طلبُ المالك ٢٠٢٦-٠٨-٢٤). */
    voiceMuted: Boolean = false,
    /** **زرٌّ واحدٌ يقلبه** — «إمّا الصوتُ يعمل أو لا يعمل». */
    onVoice: () -> Unit = {},
    /** **رأسُ العمود** — «مشكلة» وموقعُ الزبون (`TripFloatActions`). */
    start: (@Composable () -> Unit)? = null,
) {
    // ══════════════════════════════════════════════════════════════════
    // **عمودٌ واحدٌ على اليمين، يطفو فوق الخريطة** (قرارُ المالك ٢٠٢٦-١١-١٢: «الازرار ع يمين بلاهن
    // نخليهن كلهن فوق بعض ع يسار… ولدي مشكلة خليها فوق بس بدون كتابة»)
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا ارتفاعَ له في التخطيط**: يُرسم صاعداً من حافّة اللوح، **فلا يأخذ من مكان الطلب شيئاً** —
    // كان عمودان بارتفاع أربعة أقراص يرفعان اللوحَ فيُقصّ الطلبُ ويُمرَّر.
    Row(
        Modifier
            .fillMaxWidth()
            .layout { measurable, constraints ->
                val p = measurable.measure(constraints.copy(minHeight = 0, maxHeight = androidx.compose.ui.unit.Constraints.Infinity))
                layout(p.width, 0) { p.place(0, -p.height) }
            }
            .padding(horizontal = Rahal.space.md, vertical = Rahal.space.xs),
        verticalAlignment = Alignment.Bottom,
    ) {
        // **على اليمين** (المالك ٢٠٢٦-١١-١٢: «قائمة الازرار على اليمين مو على اليسار»).
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(Rahal.space.xs),
        ) {
            start?.invoke()
            MapButton(R.drawable.ic_my_location, R.string.map_recenter, onRecenter)
            // **وحديثُ الزبون قرصٌ عائمٌ** (قرار المالك ٢٠٢٦-٠٨-١٢) — في مرمى الإبهام دائماً.
            if (showChat) {
                MapButton(
                    icon = R.drawable.ic_chat,
                    label = R.string.trip_chat,
                    onClick = onChat,
                    on = chatting,
                    badge = chatUnread,
                )
            }
            // **وزرُّ الصوت يقول حالَه بشكله** (طلبُ المالك ٢٠٢٦-٠٨-٢٤).
            MapButton(
                icon = if (voiceMuted) R.drawable.ic_volume_off else R.drawable.ic_volume_on,
                label = if (voiceMuted) R.string.map_voice_off else R.string.map_voice_on,
                onClick = onVoice,
                on = !voiceMuted,
            )
        }
        Spacer(Modifier.weight(1f))
    }
}

/** **قرصٌ واحد** — أيقونةٌ في دائرةٍ ترتفع عن الخريطة بظلّها. */
@Composable
internal fun MapButton(
    icon: Int,
    label: Int,
    onClick: () -> Unit,
    on: Boolean = false,
    /** **كم ينتظره خلف هذا الزرّ** — وصفرٌ يعني لا شارة. */
    badge: Int = 0,
    /** **قرصُ الطوارئ** — أرضٌ حمراءُ وأيقونةٌ بيضاء. */
    danger: Boolean = false,
) {
    Box(contentAlignment = Alignment.TopEnd) {
        // ══════════════════════════════════════════════════════════════
        // **وأرضُ القرص سطحٌ لا أرضُ الصفحة**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٣ بلقطةٍ من جهازه: «وزنُ الأيقونات
        //  مزعج».)
        //
        // **وقيس التباينُ في الغامقة: ١٫١١ إلى ١** — أرضُ القرص
        // (`canvas`) والأيقونةُ (`panel`) كحليّان متجاوران، **فتبدو
        // أقراصاً داكنةً صمّاءَ لا أزراراً.**
        //
        // **وسببُه أنّ اللونين كانا صحيحين في الفاتحة**: أبيضُ وكحليّ
        // — **١٥٫٢٩ إلى ١.** ثمّ جاءت الغامقةُ فصارا واحدا.
        //
        // **والسطحُ يرتفع عن الأرض درجةً** فيُرى القرص، **والحبرُ يقع
        // عليه**: ١١٫٧١ في الغامقة و٥٫٨٥ في الفاتحة.
        //
        // # وأخفُّ وزناً
        //
        // **أربعةُ أقراصٍ فوق بعضها على خريطةٍ ثقيلة** — فقُصّ قطرُها
        // إلى ثمانيةٍ وأربعين (وهو أدنى ما يُلمس بإبهام)، **وخفّ ظلُّها
        // من ستٍّ إلى ثلاث**، وضاق ما بينها.
        Box(
            Modifier
                .size(Rahal.size.touch)
                .shadow(Rahal.elevation.card, CircleShape)
                .clip(CircleShape)
                .background(
                    when {
                        danger -> Rahal.colors.danger
                        on -> Rahal.colors.brand
                        else -> Rahal.colors.surface
                    },
                )
                .clickable(onClick = onClick),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(icon),
                contentDescription = stringResource(label),
                tint = when {
                    danger -> Rahal.colors.onPanel
                    on -> Rahal.colors.onBrand
                    else -> Rahal.colors.ink
                },
                modifier = Modifier.size(Rahal.icon.md),
            )
        }
        // **والشارةُ تطفو على حافّته** — كما في كلّ تطبيق: **ومن وضعها
        // بجانبه** جعلها تُقرأ رقما آخر لا عدّ رسائل.
        if (badge > 0) {
            Box(
                Modifier
                    .clip(CircleShape)
                    .background(Rahal.colors.danger)
                    .padding(horizontal = Rahal.space.sm, vertical = Rahal.space.xxs),
            ) {
                Text(
                    // **«9+» لا «+9»** — هكذا تُقرأ بالعربيّة.
                    text = if (badge > 9) "9+" else badge.toString(),
                    color = Rahal.colors.onPanel,
                    fontWeight = FontWeight.Bold,
                    style = Rahal.type.meta,
                )
            }
        }
    }
}
/**
 * **زرُّ الرحلة المصنوعة** — أعلى يسارَ الخريطة، بعيداً عن الثلاثة.
 *
 * **ولونُه ليس لونَ العلامة** — **وزرُّ فحصٍ يشبه أزرارَ العمل يُضغط
 * سهواً**، ولو في بناء تطوير.
 *
 * **ويقول «تجريبيّة» بلفظه** — **ومن رأى سهمَه يمشي وهو جالسٌ ولم
 * يعرف أنّها محاكاةٌ ظنّ الموقعَ عطبان.**
 */
@Composable
internal fun ReplayButton(
    running: Boolean,
    onStart: () -> Unit,
    onStop: () -> Unit,
) {
    // **ورماديٌّ مكتوبٌ بيدٍ لا يعرف الغامقَ من الفاتح** — كان
    // `Color(0xFF444C56)` هنا، **وهو اللونُ الوحيدُ خارجَ التوكنز في
    // التطبيقات الأربعة** (قِيس ٢٠٢٦-٠٨-٢٦).
    Surface(
        color = if (running) Rahal.colors.danger else Rahal.colors.ink,
        contentColor = Rahal.colors.canvas,
        shape = Rahal.shape.sm,
        modifier = Modifier
            .padding(Rahal.space.md)
            .clickable { if (running) onStop() else onStart() },
    ) {
        Text(
            text = if (running) com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.driver.R.string.replay_stop) else com.rahalgo.ui.AppCore.get().app.getString(com.rahalgo.driver.R.string.replay_start),
            style = Rahal.type.label,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
        )
    }
}
