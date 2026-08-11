package com.rahalgo.driver

import android.os.Bundle
import android.provider.Settings
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.core.tween
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.core.splashscreen.SplashScreen.Companion.installSplashScreen
import com.rahalgo.design.BrandCanvas
import com.rahalgo.design.RahalGoTheme
import com.rahalgo.design.intro.BrandIntro

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إقلاع تطبيق السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * ```
 * فتحٌ باردٌ  →  شاشةُ النظام  →  حركةُ العلامة  →  الوجهة
 * ```
 *
 * **وشاشةُ النظام قصيرةٌ كما يريدها أندرويد** — لا انتظارَ مصطنعٌ فيها.
 * تُسلَّم فورَ أن يبدأ Compose، ثمّ تعمل الحركةُ في شاشتنا نحن.
 *
 * # وحركةُ الافتتاح لا تتكرّر
 *
 * (طلب المالك: «لا أريد مشاهدة Intro كاملة كل مرة يرجع المستخدم من
 *  الخلفية».)
 *
 * **والحالةُ في الكائن المرافق لا في الشاشة** — فتبقى ما بقيت العمليةُ
 * حيّة. **فإعادةُ بناء الشاشة** (دوران الجهاز · تبدّل اللغة · عودةٌ من
 * الخلفية) **لا تُعيدها**، ولا يراها المستخدمُ إلّا حين يُقلع النظامُ
 * العمليةَ من جديد — وهو معنى «الفتح البارد» بالضبط.
 */
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        // **قبل `super`** — وهي شرطُ المكتبة: تُركّب على النافذة قبل أن
        // يُنشئ النظامُ محتواها.
        installSplashScreen()
        super.onCreate(savedInstanceState)
        setContent { DriverApp() }
    }

    companion object {
        /** **أعُرضت الحركةُ في عمر هذه العمليّة؟** */
        @Volatile
        var introShown: Boolean = false
    }
}

@Composable
private fun DriverApp() {
    val context = LocalContext.current
    // **وحركاتُ النظام تُقرأ من إعداداته** — من أطفأها أراد ذلك،
    // **وتطبيقٌ يتجاهله يُقرأ معطوباً لا أنيقاً.**
    val reduceMotion = remember {
        Settings.Global.getFloat(
            context.contentResolver,
            Settings.Global.ANIMATOR_DURATION_SCALE,
            1f,
        ) == 0f
    }
    var showIntro by remember { mutableStateOf(!MainActivity.introShown) }

    RahalGoTheme {
        Box(
            Modifier
                .fillMaxSize()
                // **وأرضٌ صريحةٌ تحت الاثنتين** — فلا ومضةٌ بيضاءُ ولا
                // سوداءُ في لحظة التبديل.
                .background(BrandCanvas)
        ) {
            Destination()

            AnimatedVisibility(
                visible = showIntro,
                enter = fadeIn(tween(0)),
                // **وتخرج صاعدةً لا مختفيةً فجأةً** — فيبدو أنّ الشاشةَ
                // التالية خرجت من الحركة نفسِها.
                exit = fadeOut(tween(200)) + slideOutVertically(tween(220)) { -it / 12 },
            ) {
                BrandIntro(
                    tagline = stringResource(R.string.intro_tagline),
                    reduceMotion = reduceMotion,
                    onFinished = {
                        MainActivity.introShown = true
                        showIntro = false
                    },
                )
            }
        }
    }
}

/**
 * **الوجهةُ بعد الافتتاح.**
 *
 * (طلب المالك: «يجب ألا تقرر Splash بنفسها أين يذهب المستخدم».)
 *
 * **وهي اليوم شاشةُ انتظارٍ فارغة** — طبقةُ التوجيه تُبنى مع الدخول
 * (الخطوة التالية)، وهناك تُقرَّر الوجهة: داخلٌ فلوحتُه، وخارجٌ فالدخول،
 * ونسخةٌ قديمةٌ فالتحديث.
 */
@Composable
private fun Destination() {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Text("رحّال غو", style = MaterialTheme.typography.headlineLarge)
    }
}
