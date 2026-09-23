package com.rahalgo.map

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.geo.Place
import org.maplibre.android.geometry.LatLng
import com.rahalgo.ui.Locating
import com.rahalgo.ui.LocatingText
import com.rahalgo.ui.MapGestureLock

/**
 * ══════════════════════════════════════════════════════════════════════
 * **منتقي نقطةٍ — يضعها حيث يريد لا حيث هو**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: نبدأ بالمندوب — **ويلزمه أن يضع نقطةَ متجرٍ
 *  ليست نقطتَه.**)
 *
 * # والفرقُ عن زرّ «حدّد موقعي»
 *
 * **الزبونُ يقف في بيته فيقرأ هاتفُه موضعَه** — وذلك يكفيه.
 *
 * **والمندوبُ يضيف متجراً وهو في مكتبه**، **والمتجرُ يصحّح نقطتَه بعد
 * أن أخطأ من أدخلها**، **والزبونُ يحفظ بيتَ أمّه.** **وثلاثتُهم يضعون
 * نقطةً ليست تحت أقدامهم.**
 *
 * # والدبّوسُ ثابتٌ والخريطةُ تتحرّك
 *
 * **لا دبّوسٌ يُسحب بالإصبع**: الإصبعُ يغطّيه فلا يُرى أين يقع.
 * **والخريطةُ تتحرّك تحت دبّوسٍ في وسطها** — فيرى ما تحته دائما، **وهو
 * ما تفعله كلُّ التطبيقات** فيُعرف بلا تعليم.
 *
 * # والعنوانُ يُقرأ بعد أن تستقرّ
 *
 * **ونداءٌ مع كلّ حركةٍ يُغرق المزوّدَ ويستنزف الحزمة** — فيُنادى حين
 * يرفع إصبعَه.
 *
 * # ولا تُغلق إلّا بنقطة
 *
 * **ومن خرج بلا اختيارٍ يعود بلا شيء** — **ونقطةٌ بإحداثيٍّ صفرٍ تُرسل
 * سائقاً إلى المحيط الأطلسيّ.**
 */
@Composable
fun PickPoint(
    /** **من أين تبدأ الخريطة** — موضعُه إن عُرف، وإلّا فمركزُ المدينة. */
    start: LatLng?,
    /** **اسمُ الموضع الحاليّ** — يُملأ من البحث أو من قراءة النقطة. */
    vm: PickPointViewModel,
    onPick: (LatLng, String) -> Unit,
    onCancel: () -> Unit,
    /**
     * **يطلب من التطبيق أن يقرأ موضعَ الجهاز** — ويكتبه في `LastPoint`.
     *
     * **ووحدةُ الخرائط لا تعرف جهازَ التموضع** — ولو عرفته لَحملته كلُّ
     * شاشةٍ ترسم خريطة، **ولَاحتاجت إذنَ موقعٍ لا تستعمله.**
     */
    onLocate: () -> Unit = {},
    /**
     * **حالُ تحديد الموقع** (`MLW-02`، ٢٠٢٦-٠٩-١٥).
     *
     * **ومن ضغط ولم يتبدّل شيءٌ يُقرأ ظنّ الزرَّ معطوباً فضغطه خمساً** —
     * **فالزرُّ يدور ما دام النداءُ قائماً، والتعذّرُ يُقال بسببه.**
     *
     * **وفارغةٌ تعني شاشةً قديمةً لم تُوصَل بعد** — **والزرُّ يعمل كما
     * كان.**
     */
    locating: Locating? = null,
) {
    // ══════════════════════════════════════════════════════════════════
    // **وإيماءةُ فتحِ الدرج تُقفَل ما دام هذا السطحُ حاضراً** (`DWR`)
    // ══════════════════════════════════════════════════════════════════
    //
    // **بلاغُ المالك ٢٠٢٦-٠٩-٠١**: «عند سحب الخريطة تُفتح القائمةُ
    // الجانبيّة — بكلّ التطبيقات». **وحافّةُ الشاشة تلتقط السحبَ قبل
    // الخريطة.**
    //
    // **وهنا موضعُه لا في الشاشة**: **كلُّ من رسم خريطةً تفاعليّةً نال
    // القفلَ بلا أن يكتب سطراً** — **وشرطٌ يُكتب لكلّ شاشةٍ يُنسى في
    // الشاشة الخامسة.**
    //
    // **ويُفتَح القفلُ عند الخروج حتماً** — فلا حالَ عالقةٌ ولا إعادةُ
    // تشغيل.
    MapGestureLock()

    var query by remember { mutableStateOf("") }

    // **والرجوعُ زرُّ النظام** — حُذف زرُّ «إلغاء» من الشاشة، **فلو لم
    // يُربط الرجوعُ لَخرج صاحبُه من التطبيق كلِّه ليغلق خريطة.**
    androidx.activity.compose.BackHandler { onCancel() }

    // **وحين يجيء الموضعُ تقفز الخريطةُ إليه** — ومن ضغط «موقعي» ثمّ لم
    // تتحرّك الخريطةُ ظنّ أنّ الزرَّ لا يعمل.
    val here = com.rahalgo.ui.LastPoint.value
    LaunchedEffect(vm.wantingHere, here) {
        if (vm.wantingHere && here != null) vm.jumpToPoint(here.lat, here.lng)
    }

    Box(Modifier.fillMaxSize()) {
        var zoomTick by remember { mutableStateOf(0) }
        var zoomStep by remember { mutableStateOf(0.0) }
        MapCanvas(
            start = start ?: RAQQA,
            onSettle = vm::readAddress,
            jumpTo = vm.jumpTo,
            onJumped = vm::jumped,
            modifier = Modifier.fillMaxSize(),
            zoomTick = zoomTick,
            zoomStep = zoomStep,
        )

        // **وزرّا التكبير** — (بلاغُ المالك ٢٠٢٦-٠٨-٣١). **ويمينَ
        // الشاشة في منتصفها**: حيث يقع الإبهامُ وحدَه.
        Column(
            Modifier
                .align(Alignment.CenterEnd)
                .padding(end = 12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            ZoomKey("+") { zoomStep = 1.0; zoomTick++ }
            Spacer(Modifier.height(8.dp))
            ZoomKey("−") { zoomStep = -1.0; zoomTick++ }
        }

        // **والدبّوسُ في وسط الشاشة لا على الخريطة** — لا يتحرّك معها،
        // **فما تحته هو المختار.**
        // ══════════════════════════════════════════════════════════════
        // **وتلميحٌ فوق الدبّوس — عند ما يُفعل به**
        // ══════════════════════════════════════════════════════════════
        //
        // (صورُ المالك المرجعيّة ٢٠٢٦-٠٨-١٨.)
        //
        // **وكان سطراً في أسفل الشاشة** — **ومن قرأه لا يربطه بالدبّوس
        // في الوسط**، فيبقى ينتظر شيئاً يقع.
        Text(
            text = stringResource(R.string.pick_drag),
            color = Rahal.colors.canvas,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier
                .align(Alignment.Center)
                .offset(y = (-64).dp)
                .clip(RoundedCornerShape(22.dp))
                .background(Rahal.colors.ink)
                .padding(horizontal = 18.dp, vertical = 10.dp),
        )

        Icon(
            painter = painterResource(R.drawable.ic_pin_center),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier
                .align(Alignment.Center)
                // **ورأسُ الدبّوس هو النقطة لا وسطُه** — فيُرفع بنصف
                // طوله: **من حاذى وسطَه وضع نقطةً تحت ما يشير إليه.**
                .offset(y = (-18).dp)
                .size(36.dp),
        )

        // ══════════════════════════════════════════════════════════════
        // **والبحثُ في الأعلى** — لمن يضيف متجراً وهو بعيدٌ عنه
        // ══════════════════════════════════════════════════════════════
        Column(
            Modifier
                .align(Alignment.TopCenter)
                .fillMaxWidth()
                .padding(12.dp),
        ) {
            OutlinedTextField(
                value = query,
                onValueChange = { query = it; vm.search(it) },
                label = { Text(stringResource(R.string.pick_search)) },
                singleLine = true,
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Rahal.colors.canvas, RoundedCornerShape(12.dp)),
            )
            // **ونتائجُ البحث تحته** — تُضغط فتقفز الخريطةُ إليها.
            vm.results.take(4).forEach { p ->
                Spacer(Modifier.height(4.dp))
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(10.dp))
                        .background(Rahal.colors.canvas)
                        .padding(10.dp),
                ) {
                    TextButton(onClick = { query = p.label; vm.goTo(p) }) {
                        Text(
                            text = p.label,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            }
            // **وإن تعذّر البحثُ صُرّح به** — لا يُقرأ فراغا؛ ويُتاح تكرارُه،
            // والمسارُ اليدويُّ (سحبُ الدبّوس) باقٍ. (CUST-07-006)
            if (vm.searchFailed) {
                Spacer(Modifier.height(4.dp))
                Column(
                    Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(10.dp))
                        .background(Rahal.colors.canvas)
                        .padding(10.dp),
                ) {
                    Text(
                        text = stringResource(R.string.pick_search_failed),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.error,
                    )
                    TextButton(onClick = { vm.retrySearch() }) {
                        Text(stringResource(R.string.pick_search_retry))
                    }
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وأيقونةُ «موقعي» — كما يعرفها الناسُ من خرائط غوغل**
        // ══════════════════════════════════════════════════════════════
        //
        // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «تُفتح الخريطةُ وهناك أيقونةُ تحديد
        //  الموقع بدقّة، يضغط عليها فيُحدَّد عنوانُه ثمّ حفظ وانتهى
        //  الأمر — هذا الأمرُ متعارَفٌ عليه».)
        //
        // **وكان يُطلب منه أن يحرّك الخريطةَ بإصبعه حتّى يقع الدبّوس** —
        // وهو في بيته يعرف موضعَه ولا يعرف أين هو على خريطةٍ بلا لافتات.
        //
        // **وموضعُها فوق الشريط السفليّ لا في زاويةٍ بعيدة** — الإبهامُ
        // يبلغها وهو ممسكٌ بالهاتف بيدٍ واحدة.
        //
        // **والضغطةُ الثانيةُ لا تفتح نداءً ثانياً** — **ونداءان
        // يتسابقان يصل أقدمُهما آخراً فيُوسَّط به** (`MLW-04`).
        val busy = locating?.busy == true
        FloatingActionButton(
            onClick = { if (!busy) { vm.wantHere(); onLocate() } },
            containerColor = Rahal.colors.canvas,
            modifier = Modifier
                .align(Alignment.BottomEnd)
                .padding(end = 16.dp, bottom = 148.dp)
                .size(48.dp),
        ) {
            if (busy) {
                // **ويدور ليُعلَم أنّه عمل** — **وجوابٌ في اللحظة.**
                CircularProgressIndicator(
                    modifier = Modifier.size(22.dp),
                    strokeWidth = 2.dp,
                    color = Rahal.colors.accent,
                )
            } else {
                Icon(
                    painter = painterResource(R.drawable.ic_my_location),
                    contentDescription = stringResource(R.string.pick_my_location),
                    tint = Rahal.colors.accent,
                    modifier = Modifier.size(24.dp),
                )
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **والتعذّرُ يُقال بسببه — لا «حدث خطأ»** (`MLW-06`)
        // ══════════════════════════════════════════════════════════════
        //
        // **وعلاجُ الإذن غيرُ علاج الخدمة المطفأة** — **ومن قيل له
        // «حاول ثانية» وخدمتُه مطفأةٌ حاول عشراً ثمّ ترك التطبيق.**
        //
        // **وفوق الشريط السفليّ** — **حيث ينظر بعد أن ضغط الزرّ.**
        locating?.problem?.let { why ->
            val ctx = androidx.compose.ui.platform.LocalContext.current
            Column(
                Modifier
                    .align(Alignment.BottomCenter)
                    .fillMaxWidth()
                    .padding(horizontal = 14.dp)
                    .offset(y = (-132).dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(Rahal.colors.canvas)
                    .padding(12.dp),
            ) {
                Text(
                    text = LocatingText.message(ctx, why),
                    style = MaterialTheme.typography.bodyMedium,
                )
                TextButton(
                    onClick = {
                        // **والعلاجُ فعلٌ لا «حسناً»** — **ففاتحُ
                        // الإعدادات يفتحها، والمحاولةُ تعيد النداء.**
                        if (com.rahalgo.ui.fixProblem(ctx, why)) {
                            vm.wantHere()
                            onLocate()
                        }
                    },
                ) {
                    Text(LocatingText.action(ctx, why), color = Rahal.colors.accent)
                }
            }
        }

        // ══════════════════════════════════════════════════════════════
        // **وما تحت الدبّوس مكتوبٌ قبل أن يُثبَّت**
        // ══════════════════════════════════════════════════════════════
        //
        // **ومن ثبّت نقطةً بلا أن يقرأ اسمَها لا يعرف ماذا اختار** —
        // **والخريطةُ في حيٍّ لا لافتاتِ فيه تبدو متشابهة.**
        Column(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .background(Rahal.colors.canvas)
                .padding(14.dp),
        ) {
            // **واسمُ ما تحت الدبّوس يُقرأ قبل التأكيد** — **ومن ثبّت
            // نقطةً بلا أن يقرأ اسمَها لا يعرف ماذا اختار.**
            if (vm.label.isNotEmpty()) {
                Text(
                    text = vm.label,
                    fontWeight = FontWeight.Medium,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(Modifier.height(10.dp))
            }
            // ══════════════════════════════════════════════════════════
            // **وزرٌّ واحدٌ عريضٌ يقول حالَه**
            // ══════════════════════════════════════════════════════════
            //
            // (صورُ المالك: الزرُّ يُعطَّل ويقول «جاري تحديد الموقع…»
            //  ريثما يُقرأ العنوان.)
            //
            // **وزرّان متجاوران أحدُهما «إلغاء» يجعلان القرارَ اثنين** —
            // **والرجوعُ زرُّ النظام**، وهو ما يعرفه كلُّ من يستعمل هاتفا.
            //
            // **والتعطيلُ يُقال لا يُترك صامتا**: زرٌّ باهتٌ بلا سببٍ
            // يُقرأ عطباً، **وزرٌّ يقول «جاري…» يُقرأ انتظارا.**
            Button(
                onClick = { vm.point?.let { onPick(it, vm.label) } },
                // **ولا يُثبَّت قبل أن تُقرأ نقطة** — نقطةٌ صفريّةٌ
                // تُرسل سائقاً إلى لا مكان.
                enabled = vm.point != null && !vm.reading,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (vm.reading) {
                    CircularProgressIndicator(
                        Modifier.size(16.dp),
                        strokeWidth = 2.dp,
                        color = Rahal.colors.onBrand,
                    )
                    Spacer(Modifier.size(8.dp))
                }
                Text(
                    stringResource(
                        if (vm.reading) R.string.pick_reading else R.string.pick_confirm,
                    ),
                )
            }
        }
    }
}

/** **مركزُ الرقّة** — حيث تبدأ الخريطةُ لمن لا موضعَ له. */
val RAQQA = LatLng(35.9528, 39.0079)

/** **مفتاحُ تكبيرٍ — دائرةٌ واحدةٌ بحرفٍ واحد.** */
@Composable
private fun ZoomKey(sign: String, onClick: () -> Unit) {
    Box(
        Modifier
            .size(44.dp)
            .clip(RoundedCornerShape(22.dp))
            .background(Rahal.colors.canvas)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = sign,
            color = Rahal.colors.ink,
            style = MaterialTheme.typography.titleLarge,
        )
    }
}
