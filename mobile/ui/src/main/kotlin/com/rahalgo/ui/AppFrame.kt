package com.rahalgo.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.core.view.WindowCompat
import com.rahalgo.design.DarkPalette
import com.rahalgo.design.LightPalette
import com.rahalgo.design.RahalGoTheme

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إطارُ التطبيق — السمةُ وشريطا النظام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «نرى ما المركزيّ وهي الأمور بشكلٍ كامل».)
 *
 * # ولماذا هذا وحدَه لا الغلافُ كلُّه
 *
 * **قِيس (٢٠٢٦-٠٨-١٤)**: غلافُ الزبون والمندوب متطابقان **٦١٪**،
 * **وغلافُ السائق والزبون ٢٦٪** — تبويباتُه وخريطتُه ورحلتُه تختلف.
 *
 * **ورفعُ «غلافٍ واحدٍ للثلاثة» يُخرج قطعةً بوسائطَ تصف الجميعَ ولا
 * تصف أحدا** — وهو ضدُّ ما تُرفع القِطعُ لأجله.
 *
 * **فيُرفع ما يتطابق حقّا**: السمةُ وشريطا النظام.
 *
 * # وشريطا النظام يأخذان أرضَ الصفحة
 *
 * **وإلّا بقيا فاتحين فوق شاشةٍ كحليّة** — **شريطان أبيضان يحدّان شاشةً
 * داكنةً يُقرآن عطبا.** (قِيس على جهاز المالك ٢٠٢٦-٠٨-١٣.)
 *
 * **واللوحةُ تُقرأ من القرار لا من الشجرة**: هذا خارج `RahalGoTheme`،
 * **فـ`Rahal.colors` فيه يردّ الفاتحةَ دائماً** مهما كان الاختيار.
 */
@Composable
fun AppFrame(content: @Composable (theme: ThemeState, dark: Boolean) -> Unit) {
    val context = LocalContext.current
    // **والسمةُ قرارُه هو وتبقى بعد إغلاق التطبيق** — لا تتبع النظامَ
    // وحدَه: **من فضّل الفاتحةَ في هاتفٍ غامقٍ أراد ذلك.**
    val theme = rememberTheme(context)
    val dark = theme.isDark()

    val activity = context as? android.app.Activity
    val canvas = (if (dark) DarkPalette else LightPalette).canvas
    LaunchedEffect(dark, activity) {
        val window = activity?.window ?: return@LaunchedEffect
        window.statusBarColor = canvas.toArgb()
        window.navigationBarColor = canvas.toArgb()
        WindowCompat.getInsetsController(window, window.decorView).apply {
            isAppearanceLightStatusBars = !dark
            isAppearanceLightNavigationBars = !dark
        }
    }

    RahalGoTheme(dark = dark) { content(theme, dark) }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بوّابةُ الدخول — أربعُ حالاتٍ وترتيبُها واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وترتيبُها ليس ذوقا**: التسجيلُ والاستعادةُ يغطّيان الدخول،
 * **والدخولُ يغطّي ما بعده**، **وشاشةُ الانتظار تسبقهنّ جميعاً** ما دام
 * التوكنُ المحفوظُ يُفحص.
 *
 * **ومن قلب اثنتين رأى شاشةَ دخولٍ فوق حسابٍ مفتوح.**
 *
 * # وشاشةُ انتظارٍ لا شاشةُ دخول
 *
 * **من له حسابٌ لا يُطلب منه أن يدخل من جديد** لأنّ النداءَ لم يعد بعد.
 */
@Composable
fun AuthGate(
    vm: AuthViewModel,
    /** **عنوانُ شاشة الدخول** — «دخول السائق» أو «دخول». */
    title: String,
    /** **أيُعرض بابُ إنشاء الحساب؟** — الزبونُ وحدَه يُنشئ حسابَه. */
    signup: Boolean = false,
    /** **يُفتح الدخولُ صراحةً؟** — لتطبيقٍ يتصفّح ضيفاً قبل أن يدخل. */
    asking: Boolean = true,
    onSignedIn: @Composable () -> Unit,
    /** **ما يُعرض لضيفٍ لم يطلب الدخولَ بعد** — أو فارغ. */
    guest: (@Composable () -> Unit)? = null,
) {
    when {
        vm.restoring -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator()
        }

        vm.signup != null -> SignupScreen(
            state = vm.signup!!,
            actions = SignupActions(
                setPhone = vm::setSignupPhone,
                sendCode = vm::sendSignupCode,
                verifyCode = vm::verifySignupCode,
                confirm = vm::confirmSignup,
                cancel = vm::closeSignup,
            ),
        )

        vm.reset != null -> ResetScreen(
            state = vm.reset!!,
            actions = ResetActions(
                setPhone = vm::setResetPhone,
                sendCode = vm::sendResetCode,
                verifyCode = vm::verifyResetCode,
                confirm = vm::confirmReset,
                cancel = vm::closeReset,
            ),
        )

        vm.user != null -> onSignedIn()

        guest != null && !asking -> guest()

        else -> AuthScreen(
            state = vm.state,
            title = title,
            signup = signup,
            actions = LoginActions(
                login = vm::login,
                setMode = vm::setMode,
                sendCode = vm::sendLoginCode,
                verifyCode = vm::verifyLoginCode,
                resetCode = vm::clearCode,
                forgot = vm::openReset,
                signup = { vm.openSignup() },
            ),
        )
    }
}
