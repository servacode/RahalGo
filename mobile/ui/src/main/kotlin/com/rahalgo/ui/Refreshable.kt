package com.rahalgo.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.pulltorefresh.PullToRefreshDefaults
import androidx.compose.material3.pulltorefresh.rememberPullToRefreshState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السحبُ إلى الأسفل يُنعش — شبكةُ أمانٍ للتحديث اللحظيّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «عند سحب الشاشة إلى الأسفل يتحدّث البرنامج
 *
 *	— تحسّباً لأمرِ تحديثٍ لحظيٍّ لم يصل أو أيّ خللٍ آخر».)
 *
 * # ولماذا مع البثّ لا بدلاً منه
 *
 * **البثُّ يصل حين تكون الشبكةُ قائمةً والوصلةُ حيّة** — ومن انقطع عنه
 * الإنترنتُ لحظةَ أن غُيّر سعرٌ **لا يصله شيء**، ويعود فيرى القديم ولا
 * يعرف أنّه قديم.
 *
 * **والضيفُ لا وصلةَ له أصلاً** (المقبسُ يحتاج توكناً) — **فهذه هي
 * طريقتُه الوحيدة.**
 *
 * # ولماذا مركزيّةٌ لا في كلّ شاشة
 *
 * **حركةٌ يتعلّمها المستخدمُ مرّةً ويتوقّعها في كلّ مكان** — **وشاشةٌ
 * تُنعش وأختُها لا تُنعش تُقرأ عطباً في التي لا تفعل.**
 *
 * # ودوّارةٌ بلون العلامة
 *
 * **وافتراضُ Material لونُ السمة** — وهو قريبٌ من لوننا ولا يطابقه،
 * **ولونان متجاوران يفترقان بشعرةٍ يُقرآن خطأً في الطباعة.**
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun Refreshable(
    refreshing: Boolean,
    onRefresh: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    val state = rememberPullToRefreshState()
    PullToRefreshBox(
        isRefreshing = refreshing,
        onRefresh = onRefresh,
        state = state,
        modifier = modifier.fillMaxSize(),
        indicator = {
            PullToRefreshDefaults.Indicator(
                state = state,
                isRefreshing = refreshing,
                containerColor = Rahal.colors.surface,
                color = Rahal.colors.brand,
                modifier = Modifier.align(Alignment.TopCenter),
            )
        },
    ) {
        Box(Modifier.fillMaxSize()) { content() }
    }
}
