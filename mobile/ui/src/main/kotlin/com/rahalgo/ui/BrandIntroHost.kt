package com.rahalgo.ui

import android.provider.Settings
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideOutVertically
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import com.rahalgo.design.intro.BrandIntro

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حاملُ الافتتاح — واحدٌ للتطبيقات الثلاثة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ملاحظةُ المالك ٢٠٢٦-٠٨-١٩: «تطبيقُ الزبون لا يحوي واجهةَ الدخول
 *
 *	التي اتّفقنا عليها وقمنا ببنائها… واتّفقنا يجب أن تكون مركزيّة،
 *	نستخدم نفس واجهة البداية بكلّ التطبيقات».)
 *
 * # وقِيس أنّها في السائق وحدَه
 *
 * **بُنيت `BrandIntro` في ٢٠٢٦-٠٨-١١ ورُكّبت في `DriverApp` بيدها** —
 * **فوُلدت في تطبيقٍ وبقيت فيه.** والزبونُ والمندوبُ يُقلعان على بياض.
 *
 * # ولماذا حاملٌ لا نداءٌ مباشر
 *
 * **ثلاثةُ أشياءَ تُنسى إن نُسخت**: قراءةُ إعداد الحركة من النظام،
 * ورايةُ «رُئيت»، وحركةُ الخروج. **وثلاثُ نسخٍ تفترق يوماً** — وقد
 * افترقت: نسختان لم تُكتبا أصلا.
 *
 * # ولا تُرى إلّا مرّةً في عمر العمليّة
 *
 * **ورايةٌ في حال الشاشة تعود بدورانِ الجهاز** — **فمن أمال هاتفَه
 * فرأى الافتتاحَ من جديدٍ يظنّ التطبيقَ أُقلع.** فالرايةُ ساكنةٌ
 * تعيش مع العمليّة.
 *
 * # ولا تقرّر الوجهة
 *
 * (شرطُ المالك: «يجب ألّا تقرّر Splash بنفسها أين يذهب المستخدم».)
 * **وهي تطفو فوق ما تحتها** — والوجهةُ تُقرَّر تحتها وتظهر حين تنصرف.
 */
@Composable
fun BrandIntroHost() {
    val context = LocalContext.current

    // **وحركاتُ النظام تُقرأ من إعداداته** — من أطفأها أراد ذلك،
    // **وتطبيقٌ يتجاهله يُقرأ معطوباً لا أنيقا.**
    val reduceMotion = remember {
        Settings.Global.getFloat(
            context.contentResolver,
            Settings.Global.ANIMATOR_DURATION_SCALE,
            1f,
        ) == 0f
    }

    var showing by remember { mutableStateOf(!shown) }

    AnimatedVisibility(
        visible = showing,
        enter = fadeIn(tween(0)),
        // **وتخرج صاعدةً لا مختفيةً فجأةً** — فيبدو أنّ الشاشةَ التالية
        // خرجت من الحركة نفسِها.
        exit = fadeOut(tween(200)) + slideOutVertically(tween(220)) { -it / 12 },
    ) {
        BrandIntro(
            tagline = stringResource(R.string.intro_tagline),
            brandWord = stringResource(R.string.intro_tagline_brand),
            reduceMotion = reduceMotion,
            onFinished = {
                shown = true
                showing = false
            },
        )
    }
}

/**
 * **أرُئي الافتتاحُ في هذه العمليّة؟**
 *
 * **وساكنةٌ لا حالُ شاشة** — انظر الشرحَ أعلاه: الدورانُ يُعيد بناءَ
 * الشجرة، **والحالُ تعود فيُعاد الافتتاح.**
 */
@Volatile private var shown = false
