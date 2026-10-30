package com.rahalgo.customer

import android.content.Context
import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal
import com.rahalgo.shared.customer.CustomerApi
import com.rahalgo.shared.model.Referral
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.RahalButton
import com.rahalgo.ui.Tone
import com.rahalgo.ui.money
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«ادعُ صديقاً» ظاهرةً** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **أوّلُ بندٍ في القائمة بلونٍ ومبلغ**، **وبطاقةٌ صغيرةٌ تحت البحث تُخفى بـ✕ للأبد**،
 * **وبعد أوّل طلبٍ يصله تعود مرّةً بجملة «عجبتك الخدمة؟».**
 *
 * **والمبلغُ من المحرّك** (`auth/referral` ← `next_reward`) — لا رقمٌ مكتوب. **وبلا
 * مبلغٍ (صفر) لا يُعرض شيء**: وعدٌ بلا رقمٍ لا يحرّك أحداً.
 */
object InviteNudge {
    private const val FILE = "rahalgo_invite_nudge"
    private const val HOME_HIDDEN = "home_hidden"
    private const val DELIVERED = "delivered_seen"
    private const val AFTER_HIDDEN = "after_hidden"

    var referral: Referral? by mutableStateOf(null)
        private set

    /** **أُخفيت بطاقةُ الرئيسيّة؟** — و«بعد التسليم» أُخفيت؟ — **ووصله طلبٌ؟** */
    var homeHidden by mutableStateOf(false)
        private set
    var afterHidden by mutableStateOf(false)
        private set
    var delivered by mutableStateOf(false)
        private set

    private var loaded = false

    private fun prefs(context: Context) =
        context.applicationContext.getSharedPreferences(FILE, Context.MODE_PRIVATE)

    fun load(context: Context) {
        if (loaded) return
        loaded = true
        val p = prefs(context)
        homeHidden = p.getBoolean(HOME_HIDDEN, false)
        afterHidden = p.getBoolean(AFTER_HIDDEN, false)
        delivered = p.getBoolean(DELIVERED, false)
    }

    /** **يُجلب المبلغُ والرمز** — لصاحب حسابٍ وحدَه. */
    fun refresh(scope: CoroutineScope) {
        val core = AppCore.get()
        if (core.session.refreshToken().isEmpty()) {
            referral = null
            return
        }
        scope.launch {
            runCatching { CustomerApi(core.api).referral() }.onSuccess { referral = it }
        }
    }

    fun clear() {
        referral = null
    }

    /** **وصله أوّلُ طلب** — فتعود البطاقةُ مرّةً بجملة «عجبتك الخدمة؟». */
    fun markDelivered(context: Context) {
        if (delivered) return
        delivered = true
        prefs(context).edit().putBoolean(DELIVERED, true).apply()
    }

    /** **✕ — تُخفى البطاقةُ الحاليّةُ للأبد على هذا الجهاز.** */
    fun hide(context: Context) {
        val after = afterMode()
        if (after) afterHidden = true else homeHidden = true
        prefs(context).edit().putBoolean(if (after) AFTER_HIDDEN else HOME_HIDDEN, true).apply()
    }

    /** **أيّ جملة** — «عجبتك الخدمة؟» بعد أوّل تسليمٍ ما لم تُخفَ. */
    fun afterMode(): Boolean = delivered && !afterHidden

    /** **أتُعرض البطاقة؟** */
    fun showHome(): Boolean {
        val r = referral ?: return false
        if (r.nextReward <= 0 || r.code.isBlank()) return false
        return afterMode() || !homeHidden
    }

    /** **قائمةُ المشاركة في الهاتف** — الرمزُ والرابطُ بجملةٍ واحدة. */
    fun share(context: Context, r: Referral) {
        val text = context.getString(R.string.inv_share_text, r.code, r.link)
        runCatching {
            context.startActivity(
                Intent.createChooser(
                    Intent(Intent.ACTION_SEND).apply {
                        type = "text/plain"
                        putExtra(Intent.EXTRA_TEXT, text)
                    },
                    null,
                ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
            )
        }
    }
}

/**
 * **أوّلُ سطرٍ في القائمة** — بصبغة البرتقاليّ وأيقونةِ هديّةٍ ومبلغ:
 * «ادعُ صديقاً — اربح 30 ل.س». **وقبل وصول المبلغ يُقرأ «ادعُ صديقاً» وحدَها.**
 */
@Composable
fun InviteDrawerRow(onOpen: () -> Unit) {
    val r = InviteNudge.referral
    val label = if (r != null && r.nextReward > 0) {
        stringResource(R.string.inv_drawer, money(r.nextReward))
    } else {
        stringResource(com.rahalgo.ui.R.string.menu_invite)
    }
    Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = Rahal.space.sm)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.warnTint)
            .clickable(onClick = onOpen)
            .padding(horizontal = Rahal.space.md, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_gift),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier.size(22.dp),
        )
        Spacer(Modifier.size(10.dp))
        Text(
            text = label,
            color = Rahal.colors.accent,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

/**
 * **بطاقةُ الرئيسيّة تحت البحث** — جملةٌ ومبلغ، و«شارك رمزك»، و✕ تُخفيها للأبد.
 */
@Composable
fun InviteHomeCard() {
    if (!InviteNudge.showHome()) return
    val r = InviteNudge.referral ?: return
    val context = LocalContext.current
    val after = InviteNudge.afterMode()
    Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 14.dp, vertical = 6.dp)
            .clip(Rahal.shape.md)
            .background(Rahal.colors.warnTint)
            .padding(start = Rahal.space.md, top = Rahal.space.xs, bottom = Rahal.space.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_gift),
            contentDescription = null,
            tint = Rahal.colors.accent,
            modifier = Modifier.size(22.dp),
        )
        Spacer(Modifier.size(Rahal.space.sm))
        Column(Modifier.weight(1f)) {
            Text(
                text = stringResource(
                    if (after) R.string.inv_home_after else R.string.inv_home,
                    money(r.nextReward),
                ),
                color = Rahal.colors.ink,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.bodyMedium,
            )
        }
        Spacer(Modifier.size(Rahal.space.sm))
        RahalButton(
            onClick = { InviteNudge.share(context, r) },
            tone = Tone.Accent,
        ) { Text(stringResource(R.string.inv_share_code)) }
        IconButton(onClick = { InviteNudge.hide(context) }) {
            Icon(
                painter = painterResource(com.rahalgo.ui.R.drawable.ic_close),
                contentDescription = stringResource(R.string.inv_hide),
                tint = Rahal.colors.inkMuted,
                modifier = Modifier.size(18.dp),
            )
        }
    }
}
