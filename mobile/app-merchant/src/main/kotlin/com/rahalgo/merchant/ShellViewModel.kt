package com.rahalgo.merchant

import android.app.Application
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.rahalgo.shared.model.MeSummary
import com.rahalgo.shared.model.Notice
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Refresh
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يحمله شريطا الزبون — رصيدُه وإشعاراتُه وصورتُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٤: «نضيف الجرسَ والمحفظةَ بالأعلى وزرَّ الثيم،
 *  ومن الأسفل حسابي على اليسار».)
 *
 * # ولماذا نموذجٌ للغلاف لا للشاشة
 *
 * **الشريطان يظهران فوق كلّ شاشة** — ولو حملت كلُّ شاشةٍ رصيدَها
 * **لَنودي المحرّكُ مرّةً عند كلّ تنقّل**، ولاختلف الرقمُ بين تبويبين.
 */
class ShellViewModel(app: Application) : AndroidViewModel(app) {

    private val backend = AppCore.get()

    var me by mutableStateOf<MeSummary?>(null)
        private set

    var balance by mutableStateOf(0L)
        private set

    /**
     * **عددُ ما لم يُقرأ — من `unread` لا من عدّ العناصر.**
     *
     * **والصندوقُ يجلب ثلاثين آخرَها** — فمن له أربعون تُعدّ ثلاثين،
     * **والشارةُ تكذب.**
     */
    var unread by mutableStateOf(0)
        private set

    var inbox by mutableStateOf<List<Notice>?>(null)
        private set

    /**
     * **أله جلسةٌ محفوظة؟**
     *
     * **ولا يُنادى المحرّكُ لمن لا توكنَ له** — ثلاثةُ ردودٍ ٤٠١
     * ومقبسُ ويبٍ يُرفض ويُعاد فتحُه: **قِيس على المحاكي فتجمّد
     * التطبيق.**
     */
    private val signedIn: Boolean get() = backend.session.refreshToken().isNotEmpty()

    init {
        if (signedIn) wake()
        // ══════════════════════════════════════════════════════════════
        // **والوصلةُ الحيّةُ تُفتح مرّةً للتطبيق كلّه**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٤: «خطأٌ بالتحديث اللحظيّ وبقسم
        //  طلباتي» — **وسببُه أنّها لم تُفتح قطّ.**)
        //
        // **والإشارةُ بلا حمولة**: يقول المحرّك «تغيّر شيء»،
        // **فيُنعَش كلُّ من يستمع** — الطلباتُ والرصيدُ والشارة.
        //
        // **ولا تُفتح في نموذج الطلبات**: يُهدَم بتبديل تبويبٍ فتُفتح
        // وتُغلق مع كلّ ضغطة.
    }

    /**
     * **يبدأ عملَ الغلاف** — يُنادى عند الإقلاع بجلسةٍ محفوظة، **وعند
     * أوّل دخولٍ ناجح.**
     *
     * **ولا يُفتح مقبسان**: `LiveSocket.start` تعود صامتةً إن كانت
     * تعمل.
     */
    fun wake() {
        refresh()
        // ══════════════════════════════════════════════════════════════
        // **والوصلةُ الحيّةُ تُفتح مرّةً للتطبيق كلّه**
        // ══════════════════════════════════════════════════════════════
        //
        // (شكوى المالك ٢٠٢٦-٠٨-١٤: «خطأٌ بالتحديث اللحظيّ وبقسم
        //  طلباتي» — **وسببُه أنّها لم تُفتح قطّ.**)
        //
        // **والإشارةُ بلا حمولة**: يقول المحرّك «تغيّر شيء»،
        // **فيُنعَش كلُّ من يستمع** — الطلباتُ والرصيدُ والشارة.
        backend.live.start(
            scope = viewModelScope,
            onState = { up ->
                Log.i("RahalGo/live", if (up) "الوصلة قامت" else "الوصلة انقطعت")
            },
            onEvent = {
                refresh()
                Refresh.bump()
            },
        )
    }

    fun refresh() {
        if (!signedIn) return
        viewModelScope.launch {
            // **وكلُّ نداءٍ على حدة** — فسقوطُ الرصيد لا يمنع الشارة:
            // **ثلاثةٌ في `try` واحدةٍ يُسقطها أوّلُ عاطل.**
            runCatching { me = backend.account.summary() }
            runCatching { balance = backend.me.wallet().balance }
            runCatching { unread = backend.me.inbox(limit = 1).unread }
        }
    }

    /** **يفتح الصندوقَ ويقرؤه** — والشارةُ تنطفئ بقراءته لا بفتحه. */
    fun openInbox() {
        viewModelScope.launch {
            runCatching {
                val box = backend.me.inbox()
                inbox = box.items
                unread = box.unread
            }
        }
    }

    fun markAllRead() {
        viewModelScope.launch {
            runCatching {
                backend.me.markRead()
                unread = 0
                inbox = inbox?.map { it.copy(read = true) }
            }
        }
    }

    fun closeInbox() {
        inbox = null
    }
}
