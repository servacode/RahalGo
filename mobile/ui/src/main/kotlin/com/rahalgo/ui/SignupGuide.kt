package com.rahalgo.ui

import android.content.Context
import android.provider.Settings
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.State
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شرحُ إنشاء الحساب — سهمٌ يمشي مع الزبون خطوةً خطوة** (قرارُ المالك ٢٠٢٦-١١-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «يظهر له سهم يدلّه على إنشاء حساب ويمشي معه خطوة بخطوة.» — **أوّلَ مرّةٍ
 * على هذا الجهاز**: سهمٌ وإضاءةٌ خفيفةٌ على زرّ «إنشاء حساب جديد»، ثمّ داخل
 * التسجيل فقاعةٌ صغيرةٌ فوق الحقل الحاليّ وحدَه، **وتنتقل حين يتقدّم.**
 *
 * - **ومرّةً واحدة**: يُطفأ حين يتخطّاه، أو يخرج من التسجيل، أو يدخل بحساب.
 * - **ولا يغطّي شيئاً**: الفقاعةُ في سطرها فوق الحقل لا طبقةٌ فوقه، فلا تحجب
 *   الكتابةَ ولا الكيبورد.
 * - **وللزبون وحدَه**: لا يُقرأ إلّا حيث `signup = true` — والتسجيلُ للزبون فقط.
 * - **ومن أطفأ الحركةَ في نظامه** يرى سهماً ساكناً.
 *
 * **والمنطقُ هنا صافٍ** (`GuideSteps`، `SignupGuide`) — يُقاس بلا جهاز.
 */

/** **فوق أيّ حقلٍ تقف الفقاعة.** */
enum class GuideSpot { PHONE, CODE, NAME, PASSWORD, REFERRAL }

/**
 * **أرقامُ رمز التسجيل** — كما يولّده المحرّك (`identity/service.go`:
 * `randomDigits(6)`). **ولا يُرسله المحرّكُ في حال المنصّة**، فيُكتب هنا
 * والحارسُ (`SignupGuideTest`) يقرأ المحرّكَ ويُسقط البناءَ إن افترقا.
 */
const val SIGNUP_CODE_DIGITS = 6

object GuideSteps {
    private val details = setOf(GuideSpot.NAME, GuideSpot.PASSWORD, GuideSpot.REFERRAL)

    /**
     * **الفقاعةُ تتبع الخطوةَ الحقيقيّة** — الرقم ثمّ الرمز (إن طلبته المنصّة)
     * ثمّ البيانات. **وفي البيانات**: الحقلُ الذي لمسه يغلب، وإلّا فأوّلُ ما لم
     * يكتمل — الاسم ثمّ كلمة السرّ (وتأكيدُها) ثمّ رمزُ الدعوة.
     */
    fun spot(
        step: SignupStep,
        nameFilled: Boolean,
        passwordDone: Boolean,
        focused: GuideSpot?,
    ): GuideSpot = when (step) {
        SignupStep.PHONE -> GuideSpot.PHONE
        SignupStep.CODE -> GuideSpot.CODE
        SignupStep.DETAILS -> when {
            focused != null && focused in details -> focused
            !nameFilled -> GuideSpot.NAME
            !passwordDone -> GuideSpot.PASSWORD
            else -> GuideSpot.REFERRAL
        }
    }

    /** **كلمةُ السرّ تمّت** — بطولها الأدنى، وتأكيدُها مطابق. */
    fun passwordDone(password: String, password2: String, min: Int): Boolean =
        password.length >= min && password2 == password
}

/** **أين يُحفظ أنّه رآه** — تفضيلٌ بسيطٌ في التطبيق، وبديلٌ مزيّفٌ في الاختبار. */
interface GuideStore {
    fun seen(): Boolean
    fun markSeen()
}

class SignupGuide(private val store: GuideStore) {
    /** **أيُعرض الآن؟** — وتخزينٌ عاطلٌ يُظهره (لا ضررَ أن يظهر ثانية). */
    var active by mutableStateOf(runCatching { !store.seen() }.getOrDefault(true))
        private set

    /** **انتهى** — تخطّاه، أو خرج من التسجيل، أو دخل. **ولا يعود.** */
    fun finish() {
        if (!active) return
        active = false
        runCatching { store.markSeen() }
    }

    companion object {
        /** **واحدٌ للتطبيق** — ولا يلمس التخزينَ إلّا من يقرؤه (الزبونُ وحدَه). */
        val shared: SignupGuide by lazy { SignupGuide(PrefsGuideStore) }
    }
}

private object PrefsGuideStore : GuideStore {
    private const val PREFS = "rahal_signup_guide"
    private const val KEY = "seen"
    private fun prefs() = AppCore.get().app.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
    override fun seen(): Boolean = prefs().getBoolean(KEY, false)
    override fun markSeen() {
        prefs().edit().putBoolean(KEY, true).apply()
    }
}

/** **هل أطفأ صاحبُ الجهاز الحركة؟** — من إعدادات النظام (`ANIMATOR_DURATION_SCALE`). */
@Composable
fun rememberReduceMotion(): Boolean {
    val context = LocalContext.current
    return remember {
        runCatching {
            Settings.Global.getFloat(context.contentResolver, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) == 0f
        }.getOrDefault(false)
    }
}

/** **نبضةٌ بين قيمتين** — أو قيمةٌ ساكنةٌ لمن أطفأ الحركة. */
@Composable
private fun pulse(from: Float, to: Float, still: Float, millis: Int): State<Float> {
    if (rememberReduceMotion()) return remember { mutableStateOf(still) }
    return rememberInfiniteTransition(label = "guide").animateFloat(
        initialValue = from,
        targetValue = to,
        animationSpec = infiniteRepeatable(tween(millis, easing = FastOutSlowInEasing), RepeatMode.Reverse),
        label = "guide-pulse",
    )
}

/**
 * **فقاعةُ الشرح وسهمُها إلى تحت** — بلون زرّ «إنشاء حساب» نفسِه، وفيها
 * «تخطّي الشرح». **تُوضع في سطرها فوق الحقل** فلا تغطّيه.
 */
@Composable
fun GuideBubble(text: String, onSkip: () -> Unit, modifier: Modifier = Modifier) {
    // **ولا تُشبه زرَّ «إنشاء حساب»** (D15): أرضٌ فاتحةٌ بحدٍّ برتقاليٍّ وحبرٌ عاديّ — لا برتقاليٌّ مملوء.
    val fill = Rahal.colors.accentInk
    val on = Rahal.colors.ink
    val bob by pulse(from = 0f, to = 6f, still = 0f, millis = 700)
    Column(modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clip(Rahal.shape.md)
                .background(Rahal.colors.warnTint)
                .border(androidx.compose.foundation.BorderStroke(Rahal.stroke.hair, fill), Rahal.shape.md)
                .padding(start = Rahal.space.md, end = Rahal.space.xs),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = text,
                color = on,
                fontWeight = FontWeight.Bold,
                style = Rahal.type.body,
                modifier = Modifier
                    .weight(1f)
                    .padding(vertical = Rahal.space.sm),
            )
            RahalSmallButton(onClick = onSkip, ink = fill) {
                Text(stringResource(R.string.guide_skip), style = Rahal.type.label)
            }
        }
        Icon(
            painter = painterResource(R.drawable.ic_chevron_down),
            contentDescription = null,
            tint = fill,
            modifier = Modifier
                .size(22.dp)
                .graphicsLayer { translationY = bob * density },
        )
    }
}

/**
 * **إضاءةٌ خفيفةٌ حول الزرّ المقصود** — تنبض، أو تسكن لمن أطفأ الحركة.
 * **ولا تغيّر ما في داخلها**: الزرُّ يُضغط كما هو.
 */
@Composable
fun GuideSpotlight(active: Boolean, content: @Composable () -> Unit) {
    if (!active) {
        content()
        return
    }
    val glow = toneFill(Tone.AccentSolid)
    val alpha by pulse(from = 0.12f, to = 0.38f, still = 0.25f, millis = 900)
    Box(
        Modifier
            .fillMaxWidth()
            .clip(Rahal.shape.lg)
            .drawBehind { drawRect(glow.copy(alpha = alpha)) }
            .padding(Rahal.space.xs),
    ) {
        content()
    }
}

/** **نصُّ الفقاعة لكلّ حقل** — والأرقامُ من مصدرها لا مكتوبةً في النصّ. */
@Composable
fun guideText(spot: GuideSpot, passwordMin: Int): String = when (spot) {
    GuideSpot.PHONE -> stringResource(R.string.guide_phone)
    GuideSpot.CODE -> stringResource(R.string.guide_code, SIGNUP_CODE_DIGITS)
    GuideSpot.NAME -> stringResource(R.string.guide_name)
    GuideSpot.PASSWORD -> stringResource(R.string.guide_password, passwordMin)
    GuideSpot.REFERRAL -> stringResource(R.string.guide_referral)
}
