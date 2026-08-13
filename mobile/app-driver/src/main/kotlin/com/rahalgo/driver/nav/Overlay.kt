package com.rahalgo.driver.nav

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import com.rahalgo.driver.menu.MenuItem

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يغطّي التبويبات — رايةٌ واحدةٌ لا أربع**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «نعم افعله الآن بشكلٍ مركزيٍّ مرّةً واحدة،
 *  لاحقاً بباقي التطبيقات سوف نحتاجه أيضاً».)
 *
 * # العطبُ الذي وقع ثلاثَ مرّات
 *
 * **كانت أربعُ راياتٍ منفصلة** (`account` · `rating` · `wallet` ·
 * `picked`)، وكلُّ شاشةٍ تغطّي **تحتاج أن تُطفأ في كلّ مدخلٍ إلى
 * غيرها**: أربعةُ تبويباتٍ وثلاثُ رقاقاتٍ في الشريط — **سبعةُ مواضع.**
 *
 * **وشرطٌ يُكتب في سبعة مواضع يُنسى في أحدها**، ثمّ في الثاني حين
 * يُضاف قسمٌ ثامن. **وقد نُسي ثلاثَ مرّات**: في الحساب، ثمّ في
 * التقييمات والمحفظة، ثمّ في القائمة.
 *
 * **وأثرُه أنّ الشريطَ يُضغط ولا يستجيب**: يتبدّل التبويبُ تحت الغطاء
 * **ولا يتغيّر ما يراه** — فيُقرأ الزرُّ معطّلا.
 *
 * # والعلاجُ حالٌ واحدةٌ تُطفأ بسطر
 *
 * **`show(...)` تُظهر واحدةً وتُخفي ما سواها بحكم النوع** — ولا يبقى
 * شيءٌ يُنسى. **و`clear()` تعود إلى التبويبات.**
 *
 * # ولماذا هنا لا في الشاشة
 *
 * **تطبيقاتٌ أربعةٌ ستحتاجها** (السائق · المتجر · المندوب · الزبون)،
 * **وكلُّها شريطٌ سفليٌّ وشاشاتٌ تغطّيه.** فتُكتب مرّةً وتُنسخ لا
 * تُعاد.
 */
sealed interface Overlay {
    /** **لا شيءَ يغطّي** — التبويباتُ ظاهرة. */
    data object None : Overlay

    /** **حسابُه** — يُفتح من صورته. */
    data object Account : Overlay

    /** **تقييماتُه** — تُفتح من نجمته. */
    data object Rating : Overlay

    /** **محفظتُه** — تُفتح من رقاقة رصيده. */
    data object Wallet : Overlay

    /**
     * **صندوقُ إشعاراته** — يُفتح من الجرس.
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٣: «والمشكلة نفسُها بالإشعارات، لقد
     *  نسيتَها أيضاً».)
     *
     * **وكان يُعرض براية منفصلة** (`home.inbox != null`) خارج هذه
     * الحال — **فيُضغط التبويبُ ولا يستجيب.** وهو عينُ العطب الذي
     * وُجدت هذه الملفّة لتمنعه، **ونُسيت الإشعاراتُ لأنّها كانت
     * مبنيّةً قبلها.**
     */
    data object Inbox : Overlay

    /** **بندٌ من القائمة الجانبيّة.** */
    data class Menu(val item: MenuItem) : Overlay
}

/**
 * **حالُ ما يغطّي** — ويبقى بعد دوران الشاشة.
 *
 * **ومن كتب اسمَه نصفاً ثمّ أمال هاتفَه** لا يجد نفسَه في شاشةٍ أخرى.
 */
class OverlayState(initial: Overlay = Overlay.None) {
    var current by mutableStateOf(initial)
        private set

    /** **يُظهر واحدةً ويُخفي ما سواها** — بلا أن يُنسى شيء. */
    fun show(o: Overlay) {
        current = o
    }

    /** **يعود إلى التبويبات** — يُنادى من كلّ مدخلٍ إليها. */
    fun clear() {
        current = Overlay.None
    }

    /** **أفارغٌ هو** — فالتبويبُ المختارُ هو ما يُضاء. */
    val isClear: Boolean get() = current == Overlay.None
}

/**
 * **يُنشئها ويحفظها** — والبندُ المفتوحُ يبقى بعد الدوران.
 *
 * **ولا يُحفظ إلّا اسمُ البند**: `Overlay` واجهةٌ لا تُسلسَل، **وحفظُ
 * الكائن يحتاج مُسلسِلاً لا يستحقّه حالٌ من خمسة احتمالات.**
 */
@Composable
fun rememberOverlay(): OverlayState {
    var saved by rememberSaveable { mutableStateOf("") }
    val state = androidx.compose.runtime.remember {
        OverlayState(decode(saved))
    }
    // **ويُكتب مع كلّ تبدّل** — فيقرأه الإنشاءُ التالي.
    saved = encode(state.current)
    return state
}

private fun encode(o: Overlay): String = when (o) {
    Overlay.None -> ""
    Overlay.Account -> "account"
    Overlay.Rating -> "rating"
    Overlay.Wallet -> "wallet"
    Overlay.Inbox -> "inbox"
    is Overlay.Menu -> "menu:" + o.item.name
}

private fun decode(s: String): Overlay = when {
    s == "account" -> Overlay.Account
    s == "rating" -> Overlay.Rating
    s == "wallet" -> Overlay.Wallet
    s == "inbox" -> Overlay.Inbox
    s.startsWith("menu:") -> {
        // **وبندٌ لم يعد موجوداً يعود إلى لا شيء** — لا يُسقط التطبيق:
        // **إصدارٌ يُحذف منه قسمٌ وهاتفٌ يحمل اسمَه محفوظا.**
        val name = s.removePrefix("menu:")
        MenuItem.entries.firstOrNull { it.name == name }?.let(Overlay::Menu) ?: Overlay.None
    }
    else -> Overlay.None
}
