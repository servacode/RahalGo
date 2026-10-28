package com.rahalgo.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import coil3.compose.AsyncImage
import com.rahalgo.design.Rahal

private const val MAX_ZOOM = 5f
private const val DOUBLE_TAP_ZOOM = 2.5f

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عارضُ الصورة بملء الشاشة** — قرارُ المالك ٢٠٢٦-١٠-٢٨/٢٩
 * ══════════════════════════════════════════════════════════════════════
 *
 * الصورةُ كلُّها على أسود، **تكبيرٌ بإصبعين وسحبٌ ونقرتان للتكبير**،
 * وزرُّ إغلاقٍ ظاهرٌ، **وإيماءةُ الرجوع تُغلقه.**
 */
@Composable
fun ImageViewer(url: String?, onClose: () -> Unit) {
    Dialog(
        onDismissRequest = onClose,
        properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
    ) {
        var scale by remember(url) { mutableFloatStateOf(1f) }
        var off by remember(url) { mutableStateOf(Offset.Zero) }
        BoxWithConstraints(
            Modifier
                .fillMaxSize()
                .background(Color.Black),
        ) {
            val w = constraints.maxWidth.toFloat()
            val h = constraints.maxHeight.toFloat()
            val center = Offset(w / 2f, h / 2f)
            fun clamp(o: Offset, s: Float): Offset {
                val mx = w * (s - 1f) / 2f
                val my = h * (s - 1f) / 2f
                return Offset(o.x.coerceIn(-mx, mx), o.y.coerceIn(-my, my))
            }
            AsyncImage(
                model = url,
                contentDescription = null,
                contentScale = ContentScale.Fit,
                modifier = Modifier
                    .fillMaxSize()
                    .pointerInput(url) {
                        detectTapGestures(
                            onDoubleTap = { p ->
                                if (scale > 1f) {
                                    scale = 1f
                                    off = Offset.Zero
                                } else {
                                    scale = DOUBLE_TAP_ZOOM
                                    off = clamp((p - center) * (1f - DOUBLE_TAP_ZOOM), DOUBLE_TAP_ZOOM)
                                }
                            },
                        )
                    }
                    .pointerInput(url) {
                        detectTransformGestures { c, pan, zoom, _ ->
                            val ns = (scale * zoom).coerceIn(1f, MAX_ZOOM)
                            val z = ns / scale
                            off = clamp((c - center) * (1f - z) + off * z + pan, ns)
                            scale = ns
                        }
                    }
                    .graphicsLayer {
                        scaleX = scale
                        scaleY = scale
                        translationX = off.x
                        translationY = off.y
                    },
            )
            Box(
                Modifier
                    .align(Alignment.TopEnd)
                    .statusBarsPadding()
                    .padding(Rahal.space.md)
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(Color.Black.copy(alpha = 0.55f))
                    .clickable(role = Role.Button, onClick = onClose),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painterResource(R.drawable.ic_close),
                    contentDescription = stringResource(R.string.image_viewer_close),
                    tint = Color.White,
                    modifier = Modifier.size(24.dp),
                )
            }
        }
    }
}

/**
 * **صورةُ الصنف كاملةً** — بأبعادها لا مقصوصة: عرضٌ كامل وارتفاعٌ يتبع نسبتها
 * بين [minHeight] و[maxHeight]، على أرضٍ محايدة. **والنقرُ يفتح [ImageViewer].**
 */
@Composable
fun FullPhoto(
    url: String?,
    name: String,
    modifier: Modifier = Modifier,
    shape: Shape = Rahal.shape.md,
    minHeight: Dp = 160.dp,
    maxHeight: Dp = 320.dp,
) {
    var ratio by remember(url) { mutableStateOf<Float?>(null) }
    var failed by remember(url) { mutableStateOf(false) }
    var open by remember { mutableStateOf(false) }
    BoxWithConstraints(modifier.fillMaxWidth()) {
        val target = ratio?.let { (maxWidth / it).coerceIn(minHeight, maxHeight) } ?: ((minHeight + maxHeight) / 2)
        Box(
            Modifier
                .fillMaxWidth()
                .height(target)
                .clip(shape)
                .background(Rahal.colors.field)
                .clickable(enabled = !url.isNullOrEmpty() && !failed, role = Role.Image) { open = true },
            contentAlignment = Alignment.Center,
        ) {
            if (url.isNullOrEmpty() || failed) {
                Text(name.trim().take(1), color = Rahal.colors.brand, style = MaterialTheme.typography.titleMedium)
            } else {
                AsyncImage(
                    model = url,
                    contentDescription = name,
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.fillMaxSize(),
                    onSuccess = { s ->
                        val img = s.result.image
                        if (img.width > 0 && img.height > 0) ratio = img.width.toFloat() / img.height
                    },
                    onError = { failed = true },
                )
            }
        }
    }
    if (open) ImageViewer(url = url, onClose = { open = false })
}
