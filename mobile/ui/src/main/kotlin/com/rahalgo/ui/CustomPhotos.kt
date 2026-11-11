package com.rahalgo.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صورُ الطلب الخاصّ — شريطُ عرضٍ واحدٌ للزبون والسائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١١-١١: الزبون يرفق صورةَ قائمة مشترياته أو الغرض.)
 *
 * **الروابطُ موقَّعةٌ من المحرّك** (صنفٌ محميّ) — فتُكمَّل بعنوان الخادم عبر [media] ولا تُعدَّل.
 * **ولمسةٌ على الصورة تفتحها بملء الشاشة** (`ImageViewer`).
 */
@Composable
fun CustomPhotoStrip(
    urls: List<String>,
    media: (String?) -> String?,
    modifier: Modifier = Modifier,
) {
    if (urls.isEmpty()) return
    var open by remember { mutableStateOf<String?>(null) }
    Row(
        modifier.horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
    ) {
        urls.forEach { u ->
            val full = media(u)
            RemoteImage(
                url = full,
                name = "",
                modifier = Modifier
                    .size(Rahal.size.thumb)
                    .clip(Rahal.shape.sm)
                    .clickable { open = full },
            )
        }
    }
    open?.let { ImageViewer(url = it, onClose = { open = null }) }
}

/** **عنوانٌ صغيرٌ فوق الشريط** — «صور الطلب». */
@Composable
fun CustomPhotoLabel() {
    Text(
        text = stringResource(R.string.custom_photos_label),
        color = Rahal.colors.inkMuted,
        style = Rahal.type.label,
    )
    Spacer(Modifier.height(Rahal.space.xs))
}

@Composable
fun CustomPhotoBlock(urls: List<String>, media: (String?) -> String?) {
    if (urls.isEmpty()) return
    Column {
        CustomPhotoLabel()
        CustomPhotoStrip(urls, media)
    }
}
