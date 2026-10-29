package com.rahalgo.map

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.RahalOutlineButton
import com.rahalgo.ui.RahalTextButton
import android.annotation.SuppressLint
import android.os.SystemClock
import com.google.android.gms.location.LocationServices
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import com.rahalgo.ui.Locating
import com.rahalgo.ui.LocatingText
import com.rahalgo.ui.locationServiceEnabled
import kotlinx.coroutines.delay
import org.maplibre.android.geometry.LatLng
import kotlin.math.abs
import kotlin.math.ceil


/**
 * **الموقعُ يُختار على الخريطة الصغيرة نفسِها** (طلبُ المالك ٢٠٢٦-١٠-٢٨: «خليها فوراً
 * بالخريطة الصغيرة، أحسن ما تفتح بخريطة كبيرة مرة ثانية») — **ولا ملتقطَ بشاشةٍ كاملة.**
 *
 * - **الدبّوسُ ثابتٌ في الوسط والخريطةُ تتحرّك تحته** (كما في `PickPoint`): حين تستقرّ
 *   يصير ما تحته هو الموقعَ المختار (`onPoint`) — **ولا يُحفظ حتّى
 *   «حفظ» أو «التالي»** كما كان.
 * - **وبلا نقطةٍ تبدأ من مركز الرقّة ولا تُعتمد** حتّى يحرّكها بإصبعه أو يأتي موقعُه
 *   — **فمركزُ المدينة ليس متجراً.**
 * - **والسحبُ على الخريطة لها لا للصفحة** (`holdTouch`) — والسحبُ خارجها يمرّر الصفحة.
 * - **وزرّا «+» و«−»** لمن يمسك الهاتفَ بيدٍ واحدة.
 * - **و«حدّد موقعي بدقة» في البطاقة نفسِها**: إذنٌ ثمّ تشغيلُ الموقع ثمّ آخرُ موضعٍ
 *   حديثٍ فوراً، أو أوّلُ قراءةٍ حيّةٍ بـ٥٠ م فأقلّ (`LiveFix`)، **ثمّ يتحسّن وحدَه** حتّى
 *   ١٥ م (`CONFIRM_ACCURACY_M`) أو ٢٠ ثانية — **وسحبُ الخريطة يوقف التحسين.**
 */

/**
 * **أيُّ بطاقةٍ تملك اللوحَ الآن.** فاللوحُ مخبوءٌ للنشاط، **ومستمعو بطاقةٍ سابقةٍ يبقون
 * عليه** (`MapCanvas` يضيفهم ولا يرفعهم) — فلا يكتب في الموقع إلّا مستمعُ البطاقة الحاضرة.
 */
private object MapCardOwner {
    var live: Any? = null
}

/** **فرقٌ أصغرُ من هذا (نحو عشرة سنتيمترات) ليس حركة.** */
private const val SAME_POINT_DEG = 1e-6

/** **أوّلُ قراءةٍ بهذه الدقّة أو أفضل تضع النقطةَ فوراً** — ثمّ يتحسّن. */
private const val FIRST_FIX_M = 50f

/** **آخرُ موضعٍ يعرفه الجهاز يُقبل إن لم يتجاوز عمرُه دقيقتين.** */
private const val LAST_KNOWN_MAX_AGE_MS = 120_000L

/** **التحسينُ يقف بعد هذه المدّة** — وما وُضع يبقى. */
private const val REFINE_MS = 20_000L

/**
 * **آخرُ موضعٍ محفوظٍ في الجهاز** — يصل في اللحظة بلا انتظار قمر. **ويُقبل إن كان
 * حديثاً (دقيقتان) ودقّتُه ٥٠ م فأقلّ**، وإلّا انتُظرت القراءةُ الحيّة.
 */
@SuppressLint("MissingPermission")
private fun lastKnownFix(context: android.content.Context, onFix: (LatLng, Float) -> Unit) {
    if (!Here.granted(context)) return
    runCatching {
        LocationServices.getFusedLocationProviderClient(context.applicationContext)
            .lastLocation
            .addOnSuccessListener { loc ->
                if (loc == null || !loc.hasAccuracy()) return@addOnSuccessListener
                val ageMs = (SystemClock.elapsedRealtimeNanos() - loc.elapsedRealtimeNanos) / 1_000_000
                if (ageMs > LAST_KNOWN_MAX_AGE_MS || loc.accuracy > FIRST_FIX_M) return@addOnSuccessListener
                onFix(LatLng(loc.latitude, loc.longitude), loc.accuracy)
            }
    }
}

@Composable
fun PointMapCard(
    lat: Double?,
    lng: Double?,
    pending: Boolean,
    enabled: Boolean,
    onPoint: (Double, Double) -> Unit,
) {
    val context = LocalContext.current
    val set = lat != null && lng != null
    val setPoint by rememberUpdatedState(onPoint)
    val here = if (lat != null && lng != null) LatLng(lat, lng) else com.rahalgo.map.RAQQA
    // **وتبدأ الكاميرا مرّةً واحدة** — ثمّ يحرّكها إصبعُه، **ولا تُبنى الخريطةُ من جديدٍ
    // مع كلّ استقرار.**
    val start = remember { here }

    val me = remember { Any() }
    val touched = remember { mutableStateOf(false) }
    val gps = remember { mutableStateOf(false) }
    // **دقّةُ النقطة التي وضعها «حدّد موقعي» في هذه الجولة** — وفارغةٌ إن لم يضعها بعد
    // أو حرّك الخريطةَ بيده بعدها.
    val fixAcc = remember { mutableStateOf<Float?>(null) }
    var problem by remember { mutableStateOf<Locating.Problem?>(null) }
    var jump by remember { mutableStateOf<LatLng?>(null) }
    var zoomTick by remember { mutableIntStateOf(0) }
    var zoomStep by remember { mutableDoubleStateOf(0.0) }
    val shown by rememberUpdatedState(here.takeIf { set })
    val canEdit by rememberUpdatedState(enabled)

    DisposableEffect(me) {
        MapCardOwner.live = me
        onDispose {
            if (MapCardOwner.live === me) MapCardOwner.live = null
            if (gps.value) LiveFix.stop()
        }
    }

    // **استقرّت الخريطة** — وما تحت الدبّوس يُعتمد إن حرّكها هو، **لا عند التحميل.**
    val onSettle: (LatLng) -> Unit = remember {
        { at ->
            val now = shown
            val same = now != null &&
                abs(now.latitude - at.latitude) < SAME_POINT_DEG &&
                abs(now.longitude - at.longitude) < SAME_POINT_DEG
            if (MapCardOwner.live === me && touched.value && canEdit && !same) {
                setPoint(at.latitude, at.longitude)
            }
        }
    }
    // **حرّكها بإصبعه** — فيتوقّف التحسين، **وما يختاره بيده يغلب.**
    val onUserMove: () -> Unit = remember {
        {
            if (MapCardOwner.live === me) {
                touched.value = true
                fixAcc.value = null
                if (gps.value) {
                    gps.value = false
                    LiveFix.stop()
                }
            }
        }
    }

    // **نقطةٌ من «حدّد موقعي»** — تُوضع إن كانت أدقَّ ممّا وُضع قبلها في هذه الجولة.
    val takeFix: (LatLng, Float) -> Unit = remember {
        { at, a ->
            val best = fixAcc.value
            if (gps.value && !touched.value && canEdit && (best == null || a < best)) {
                jump = at
                setPoint(at.latitude, at.longitude)
                fixAcc.value = a
            }
        }
    }

    // ── «حدّد موقعي بدقة» في البطاقة نفسِها ──
    // (طلبُ المالك ٢٠٢٦-١٠-٢٩: «لازم تكون أسرع».) **آخرُ موضعٍ يعرفه الجهازُ يُوضع
    // في اللحظة إن كان حديثاً، ثمّ أوّلُ قراءةٍ بدقّة ٥٠ م فأقلّ، ثمّ يتحسّن وحدَه
    // حتّى ١٥ م أو ٢٠ ثانية.**
    val startGps: () -> Unit = {
        problem = null
        touched.value = false
        fixAcc.value = null
        gps.value = true
        LiveFix.start(context)
        if (!LiveFix.running) {
            gps.value = false
            problem = Locating.Problem.UNAVAILABLE
        } else {
            lastKnownFix(context, takeFix)
        }
    }
    val enableLocation = rememberLocationEnabler { on ->
        if (on) startGps() else problem = Locating.Problem.SERVICE_OFF
    }
    val askPermission = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { ok ->
        when {
            !ok -> problem = context.findActivity()?.let { Here.deniedProblem(it) }
                ?: Locating.Problem.PERMISSION_DENIED
            locationServiceEnabled(context) -> startGps()
            else -> {
                problem = Locating.Problem.SERVICE_OFF
                enableLocation()
            }
        }
    }
    val locate: () -> Unit = {
        when {
            !Here.granted(context) -> askPermission.launch(android.Manifest.permission.ACCESS_FINE_LOCATION)
            !locationServiceEnabled(context) -> {
                problem = Locating.Problem.SERVICE_OFF
                enableLocation()
            }
            else -> startGps()
        }
    }
    // **ومن شغّل الموقعَ من الشريط العلويّ يُكمَل له** — بلا ضغطةٍ ثانية.
    LocationServiceEffect { if (problem == Locating.Problem.SERVICE_OFF) startGps() }

    // **كلُّ قراءةٍ أدقّ تنقل النقطة** — من ٥٠ م فأقلّ، **والتحسينُ يقف عند ١٥ م.**
    val live = LiveFix.point
    val acc = LiveFix.accuracyM
    LaunchedEffect(gps.value, live, acc) {
        if (!gps.value || live == null || acc == null) return@LaunchedEffect
        if (acc <= FIRST_FIX_M) {
            takeFix(live, acc)
        } else if (fixAcc.value == null) {
            // **قراءةٌ تقريبيّةٌ تحرّك الخريطةَ ولا تُعتمد.**
            jump = live
        }
        if (acc <= com.rahalgo.map.CONFIRM_ACCURACY_M) {
            gps.value = false
            LiveFix.stop()
        }
    }
    // **ولا ينتظر بلا نهاية** — بعد المهلة يقف التحسين، **وبلا نقطةٍ يُقال له.**
    LaunchedEffect(gps.value) {
        if (!gps.value) return@LaunchedEffect
        delay(REFINE_MS)
        if (gps.value) {
            val hadReading = LiveFix.point != null
            gps.value = false
            LiveFix.stop()
            if (fixAcc.value == null) {
                problem = if (hadReading) Locating.Problem.WEAK_ACCURACY else Locating.Problem.TIMEOUT
            }
        }
    }

    Column(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.md)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, if (set) Rahal.colors.line else Rahal.colors.danger, Rahal.shape.md),
    ) {
        Box(
            Modifier
                .fillMaxWidth()
                .height(260.dp),
        ) {
            run {
                com.rahalgo.map.MapCanvas(
                    start = start,
                    onSettle = onSettle,
                    jumpTo = jump,
                    onJumped = { jump = null },
                    onUserMove = onUserMove,
                    zoomTick = zoomTick,
                    zoomStep = zoomStep,
                    holdTouch = true,
                    modifier = Modifier.fillMaxSize(),
                )
            }
            // **ورأسُ الدبّوس هو النقطة** — فيُرفع بنصف طوله.
            Icon(
                painter = painterResource(com.rahalgo.map.R.drawable.ic_pin_center),
                contentDescription = null,
                tint = Rahal.colors.accent,
                modifier = Modifier
                    .align(Alignment.Center)
                    .offset(y = (-18).dp)
                    .size(36.dp),
            )
            if (!set && !gps.value) {
                Text(
                    stringResource(R.string.map_drag_to_set),
                    color = Rahal.colors.canvas,
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.Bold,
                    textAlign = TextAlign.Center,
                    modifier = Modifier
                        .align(Alignment.TopCenter)
                        .padding(Rahal.space.sm)
                        .clip(Rahal.shape.pill)
                        .background(Rahal.colors.ink)
                        .padding(horizontal = Rahal.space.md, vertical = Rahal.space.sm),
                )
            }
            if (enabled) {
                Column(
                    Modifier
                        .align(Alignment.CenterEnd)
                        .padding(end = Rahal.space.sm),
                    verticalArrangement = Arrangement.spacedBy(Rahal.space.sm),
                ) {
                    MapZoomKey("+", stringResource(R.string.map_zoom_in)) { zoomStep = 1.0; zoomTick++ }
                    MapZoomKey("−", stringResource(R.string.map_zoom_out)) { zoomStep = -1.0; zoomTick++ }
                }
            } else {
                // **متجرٌ موقوفٌ يُقرأ ولا يُعدَّل** — غطاءٌ يحجب الخريطة، **والسحبُ يمرّر الصفحة.**
                Box(
                    Modifier
                        .matchParentSize()
                        .pointerInput(Unit) {
                            awaitPointerEventScope { while (true) awaitPointerEvent() }
                        },
                )
            }
        }
        Column(Modifier.padding(Rahal.space.md)) {
            MapCardStatus(
                set = set,
                pending = pending,
                gpsOn = gps.value,
                liveAcc = acc,
                fixAcc = fixAcc.value,
                problem = problem,
                onFix = { why ->
                    when (why) {
                        Locating.Problem.SERVICE_OFF -> enableLocation()
                        Locating.Problem.PERMISSION_DENIED ->
                            askPermission.launch(android.Manifest.permission.ACCESS_FINE_LOCATION)
                        else -> if (com.rahalgo.ui.fixProblem(context, why)) locate()
                    }
                },
            )
            Spacer(Modifier.height(Rahal.space.md))
            RahalOutlineButton(
                onClick = locate,
                enabled = enabled,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp),
            ) {
                Icon(
                    painterResource(com.rahalgo.map.R.drawable.ic_my_location),
                    contentDescription = null,
                    tint = Rahal.colors.brand,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.width(Rahal.space.sm))
                Text(stringResource(R.string.map_locate_exact), fontWeight = FontWeight.Bold)
            }
        }
    }
}

/**
 * **سطرُ الحال تحت الخريطة** — العطلُ وعلاجُه، أو تقدّمُ تحديد الموقع ودقّتُه، أو حالُ
 * النقطة نفسِها. **والأرقامُ إنكليزيّةٌ دائماً** (قرارُ المالك).
 */
@Composable
private fun MapCardStatus(
    set: Boolean,
    pending: Boolean,
    gpsOn: Boolean,
    liveAcc: Float?,
    fixAcc: Float?,
    problem: Locating.Problem?,
    onFix: (Locating.Problem) -> Unit,
) {
    val context = LocalContext.current
    if (problem != null) {
        Text(
            LocatingText.message(context, problem),
            color = Rahal.colors.danger,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
        )
        RahalTextButton(onClick = { onFix(problem) }) {
            Text(LocatingText.action(context, problem), fontWeight = FontWeight.Bold)
        }
        return
    }
    // **وُضعت النقطة** — بدقّتها، **و«جارٍ تحسين الدقة» ما دام يتحسّن.**
    if (fixAcc != null) {
        val meters = ceil(fixAcc).toInt().toString()
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (gpsOn) {
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    strokeWidth = Rahal.stroke.focus,
                    color = Rahal.colors.success,
                )
                Spacer(Modifier.width(Rahal.space.sm))
            }
            Text(
                stringResource(if (gpsOn) R.string.map_located_refining else R.string.map_located, meters),
                color = Rahal.colors.success,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
        if (pending) {
            Spacer(Modifier.height(Rahal.space.xs))
            Text(
                stringResource(R.string.map_point_chosen),
                color = Rahal.colors.inkMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        return
    }
    if (gpsOn) {
        val meters = liveAcc?.let { ceil(it).toInt().toString() }
        Row(verticalAlignment = Alignment.CenterVertically) {
            CircularProgressIndicator(
                modifier = Modifier.size(16.dp),
                strokeWidth = Rahal.stroke.focus,
                color = Rahal.colors.accent,
            )
            Spacer(Modifier.width(Rahal.space.sm))
            Text(
                if (meters == null) stringResource(R.string.map_locating)
                else stringResource(R.string.map_locating_acc, meters),
                color = Rahal.colors.accent,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
        return
    }
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(
            painterResource(com.rahalgo.ui.R.drawable.ic_pin),
            contentDescription = null,
            tint = if (set) Rahal.colors.success else Rahal.colors.danger,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.width(Rahal.space.sm))
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(
                    when {
                        !set -> R.string.map_point_none
                        pending -> R.string.map_point_chosen
                        else -> R.string.map_point_set
                    },
                ),
                color = if (set) Rahal.colors.success else Rahal.colors.danger,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
            // **والشرحُ مرّةً واحدة — فوق الخريطة** (تجربةُ المالك ٢٠٢٦-١٠-٢٩: كان مكرّراً تحتها).
        }
    }
}

/** **مفتاحُ تكبيرٍ** — دائرةٌ بحرفٍ واحد، يبلغها الإبهامُ وحدَه. */
@Composable
private fun MapZoomKey(sign: String, label: String, onClick: () -> Unit) {
    Box(
        Modifier
            .size(40.dp)
            .clip(Rahal.shape.pill)
            .background(Rahal.colors.canvas)
            .border(Rahal.stroke.hair, Rahal.colors.line, Rahal.shape.pill)
            .semantics { contentDescription = label }
            .clickable(role = Role.Button, onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            sign,
            color = Rahal.colors.ink,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleLarge,
        )
    }
}

private fun android.content.Context.findActivity(): android.app.Activity? {
    var c: android.content.Context? = this
    while (c is android.content.ContextWrapper) {
        if (c is android.app.Activity) return c
        c = c.baseContext
    }
    return null
}
