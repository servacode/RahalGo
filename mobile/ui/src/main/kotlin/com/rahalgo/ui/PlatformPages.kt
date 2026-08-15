package com.rahalgo.ui

import android.app.Application
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.model.Platform
import com.rahalgo.shared.model.SiteContact
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفحاتُ المنصّة والقانونيّة — بنودٌ واحدةٌ لكلّ التطبيقات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «المنصّةُ مركزيٌّ أساساً · والقانونيّةُ
 *  مركزيٌّ أيضا».)
 *
 * **ونصوصُها في المحرّك أصلاً** (`settings/pagetext.go`) — **فلا يحمل
 * تطبيقٌ منها حرفا.** وما يُرفع هنا البنودُ والشاشة، **لا النصّ.**
 */
object PlatformPages {

    const val HELP = "Help"
    const val ABOUT = "About"
    const val CONTACT = "Contact"
    const val TERMS = "Terms"
    const val PRIVACY = "Privacy"

    /**
     * **مجموعتان: المنصّةُ ثمّ القانونيّة** — بهذا الترتيب.
     *
     * **والقانونيُّ آخرا** — يُقرأ مرّةً عند التسجيل ولا يُفتح بعدها،
     * **وما يُفتح كثيراً يسبق ما يُفتح مرّة.**
     */
    val items: List<DrawerItem> = listOf(
        DrawerItem(HELP, R.string.menu_platform, R.string.menu_help, R.drawable.ic_info),
        DrawerItem(ABOUT, R.string.menu_platform, R.string.menu_about, R.drawable.ic_info),
        DrawerItem(CONTACT, R.string.menu_platform, R.string.menu_contact, R.drawable.ic_phone),
        DrawerItem(TERMS, R.string.menu_legal, R.string.menu_terms, R.drawable.ic_info),
        DrawerItem(PRIVACY, R.string.menu_legal, R.string.menu_privacy, R.drawable.ic_lock),
    )

    /** **أهذا البندُ منها؟** — فيُوجَّه إلى شاشتها لا إلى شاشة التطبيق. */
    fun has(key: String): Boolean = items.any { it.key == key }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقلُ صفحات المنصّة — نداءان لا خمسة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والخمسُ صفحاتٍ تقرأ نداءين اثنين** (`public/contact` و
 * `public/platform`) — **فيُجلبان مرّةً ويبقيان.**
 *
 * **وشروطُ الاستخدام لا تتبدّل بين فتحتين** — ونداءٌ عند كلّ فتحةٍ
 * لوثيقةٍ ثابتةٍ عملٌ بلا فائدة، **يستنزف حزمةَ من في الشارع.**
 */
class PagesViewModel(app: Application) : AndroidViewModel(app) {

    private val backend = AppCore.get()

    var busy by mutableStateOf(false)
        private set

    /** **خطأُ آخر نداء** — بعربيّةٍ تُقرأ لا برمزٍ إنكليزيّ. */
    var error by mutableStateOf("")
        private set

    var contact by mutableStateOf<SiteContact?>(null)
        private set

    var platform by mutableStateOf<Platform?>(null)
        private set

    /** **ولا يُعاد ما وصل** — إلّا أن يُطلب صراحةً من زرّ «أعد المحاولة». */
    fun load(force: Boolean = false) {
        error = ""
        if (!force && contact != null && platform != null) return
        busy = true
        viewModelScope.launch {
            try {
                contact = backend.auth.contact()
                platform = backend.auth.platform()
                error = ""
            } catch (e: Exception) {
                error = apiError(getApplication(), e)
            }
            busy = false
        }
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شاشةُ بندٍ من المنصّة — واحدةٌ لكلّ التطبيقات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * التعليماتُ ومن نحن وتواصل والشروطُ والخصوصيّة.
 *
 * **ولا نصَّ فيها من عندنا** — كلُّه من المحرّك، **فيُبدَّل من اللوحة
 * بلا نشرِ إصدارٍ في غوغل بلاي** ينتظر مراجعتَه أيّاما.
 *
 * **وكانت نسختين متطابقتين** في التطبيقين (قِيس ٢٠٢٦-٠٨-١٤) —
 * **ونسختان لا تفترقان اليومَ تفترقان غدا.**
 */
@Composable
fun PlatformScreen(vm: PagesViewModel, key: String, role: HelpRole) {
    androidx.compose.runtime.LaunchedEffect(key) { vm.load() }
    val item = PlatformPages.items.first { it.key == key }
    if (key == PlatformPages.CONTACT) {
        ContactPage(vm)
    } else {
        PlatformPage(vm, item, role = role)
    }
}

/**
 * **أيعرف التطبيقُ هذا المفتاح؟** — يحرسه `rememberOverlay`.
 *
 * **وبندٌ محفوظٌ حُذف من إصدارٍ لاحقٍ يفتح شاشةً بيضاءَ لا مخرجَ منها.**
 */
fun knowsKey(own: List<DrawerItem>, key: String): Boolean =
    PlatformPages.has(key) || own.any { it.key == key }
