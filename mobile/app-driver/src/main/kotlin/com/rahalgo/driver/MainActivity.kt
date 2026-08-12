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
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import androidx.lifecycle.viewmodel.compose.viewModel
import com.rahalgo.design.intro.BrandIntro
import com.rahalgo.driver.login.LoginActions
import com.rahalgo.driver.login.LoginScreen
import com.rahalgo.driver.login.LoginViewModel

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
                    brandWord = stringResource(R.string.intro_tagline_brand),
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
 * ══════════════════════════════════════════════════════════════════════
 * **الوجهة — تُقرَّر هنا لا في شاشة الافتتاح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شرط المالك: «يجب ألّا تقرّر Splash بنفسها أين يذهب المستخدم».)
 *
 * **وثلاث حالات**: يُفحص التوكن المحفوظ · فإن صحّ فلوحته · وإلّا فالدخول.
 *
 * **وشاشة انتظار أثناء الفحص لا شاشة دخول**: من له جلسة حيّة **لا يُرى
 * شاشة دخول لحظة** ثمّ تُبدَّل — وهو وميض يُقرأ عطبا.
 */
@Composable
private fun Destination() {
    val vm: LoginViewModel = viewModel()

    when {
        vm.restoring -> Box(Modifier.fillMaxSize())

        vm.user != null -> DriverHome(
            name = vm.user!!.fullName.ifBlank { vm.user!!.phone },
            onLogout = vm::logout,
        )

        else -> LoginScreen(
            state = vm.state,
            actions = LoginActions(
                login = vm::login,
                useOtp = { /* الباب الثاني — الخطوة التالية */ },
            ),
        )
    }
}

/**
 * **لوحة السائق — شاشة إثبات لا أكثر.**
 *
 * **وغرضها اليوم واحد**: أن يرى المالك اسمه فيعلم أنّ الدخول وصل
 * المحرّك الحقيقي. **وتُبنى شاشته الحقيقية في الخطوة التالية.**
 */
@Composable
private fun DriverHome(name: String, onLogout: () -> Unit) {
    Column(
        Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.login_welcome, name),
            style = MaterialTheme.typography.headlineSmall,
        )
        TextButton(onClick = onLogout) { Text("خروج") }
    }
}
