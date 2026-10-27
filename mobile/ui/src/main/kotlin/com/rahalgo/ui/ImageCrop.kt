package com.rahalgo.ui

import android.graphics.Bitmap
import android.graphics.Matrix
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ClipOp
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import kotlin.math.max
import kotlin.math.min
import kotlin.math.roundToInt

/**
 * معاينة الصورة وقصّها قبل الرفع (طلب المالك ٢٠٢٦-١٠-٢٧).
 *
 * الزبون بيشوف صورة الصنف مربّعة، فالإطار هون مربّع: اللي بيبين جوّاته
 * هو بالضبط اللي بيوصل للسوق. صاحب الصورة بيكبّر بإصبعين، وبيحرّكها،
 * وبيدوّرها، وما بتنرفع إلا لما يكبس «اعتمد».
 */
@Composable
internal fun ImageCropDialog(
    source: Bitmap,
    onDone: (Bitmap) -> Unit,
    onAgain: () -> Unit,
    onCancel: () -> Unit,
) {
    var bmp by remember(source) { mutableStateOf(source) }
    var zoom by remember(bmp) { mutableFloatStateOf(1f) }
    var off by remember(bmp) { mutableStateOf(Offset.Zero) }
    var frame by remember { mutableFloatStateOf(0f) }

    Dialog(
        onDismissRequest = onCancel,
        properties = DialogProperties(usePlatformDefaultWidth = false, decorFitsSystemWindows = false),
    ) {
        Column(
            Modifier
                .fillMaxSize()
                .background(Color.Black)
                .statusBarsPadding()
                .navigationBarsPadding()
                .padding(16.dp),
        ) {
            Text(
                stringResource(R.string.crop_title),
                color = Color.White,
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.crop_hint),
                color = Color.White.copy(alpha = 0.75f),
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(12.dp))

            // الإحداثيات هون فيزيائية (يسار←يمين) بغضّ النظر عن اتّجاه
            // الواجهة، وإلا انعكس السحب بالعربي.
            CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
                BoxWithConstraints(
                    Modifier
                        .fillMaxWidth()
                        .weight(1f),
                ) {
                    val density = LocalDensity.current
                    val w = with(density) { maxWidth.toPx() }
                    val h = with(density) { maxHeight.toPx() }
                    val side = min(w, h)
                    frame = side
                    val k = coverScale(bmp, side) * zoom
                    // **مقصوصةٌ على حدودها** (تدقيقُ ٢٠٢٦-١٠-٢٨) — صورةٌ مكبَّرةٌ أو طويلة
                    // كانت تُرسم فوق العنوان والأزرار.
                    Canvas(
                        Modifier
                            .fillMaxSize()
                            .clipToBounds()
                            .pointerInput(bmp, side) {
                                detectTransformGestures { _, pan, z, _ ->
                                    zoom = (zoom * z).coerceIn(1f, 6f)
                                    off = clampOffset(bmp, side, zoom, off + pan)
                                }
                            },
                    ) {
                        val img = bmp.asImageBitmap()
                        val dw = bmp.width * k
                        val dh = bmp.height * k
                        val cx = size.width / 2f + off.x
                        val cy = size.height / 2f + off.y
                        drawImage(
                            img,
                            dstOffset = IntOffset((cx - dw / 2f).roundToInt(), (cy - dh / 2f).roundToInt()),
                            dstSize = IntSize(dw.roundToInt(), dh.roundToInt()),
                        )
                        val r = Rect(
                            Offset((size.width - side) / 2f, (size.height - side) / 2f),
                            Size(side, side),
                        )
                        // تعتيم خارج الإطار — بيشوف شو رح ينقصّ.
                        clipRect(r.left, r.top, r.right, r.bottom, clipOp = ClipOp.Difference) {
                            drawRect(Color.Black.copy(alpha = 0.6f))
                        }
                        drawRect(
                            Color.White,
                            topLeft = r.topLeft,
                            size = r.size,
                            style = Stroke(width = 2.dp.toPx()),
                        )
                    }
                }
            }

            Spacer(Modifier.height(12.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                RahalButton(
                    onClick = { onDone(cropSquare(bmp, frame, zoom, off)) },
                    modifier = Modifier.weight(1f),
                ) { Text(stringResource(R.string.crop_use)) }
                RahalOutlineButton(onClick = {
                    bmp = rotate90(bmp)
                }) { Text(stringResource(R.string.crop_rotate)) }
            }
            Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                RahalTextButton(onClick = onAgain) { Text(stringResource(R.string.crop_again)) }
            }
        }
    }
}

/** المقياس اللي بيخلّي الصورة تغطّي الإطار كلّه. */
private fun coverScale(b: Bitmap, side: Float): Float =
    if (side <= 0f) 1f else side / min(b.width, b.height).toFloat()

/** ما بيسمح بفراغ جوّا الإطار — الصورة دايماً مغطّيته. */
private fun clampOffset(b: Bitmap, side: Float, zoom: Float, o: Offset): Offset {
    val k = coverScale(b, side) * zoom
    val mx = max(0f, (b.width * k - side) / 2f)
    val my = max(0f, (b.height * k - side) / 2f)
    return Offset(o.x.coerceIn(-mx, mx), o.y.coerceIn(-my, my))
}

/** الجزء اللي جوّا الإطار، بإحداثيات الصورة نفسها. */
internal fun cropSquare(b: Bitmap, side: Float, zoom: Float, o: Offset): Bitmap {
    if (side <= 0f) return b
    val k = coverScale(b, side) * zoom
    val c = clampOffset(b, side, zoom, o)
    val n = (side / k).roundToInt().coerceIn(1, min(b.width, b.height))
    val left = ((b.width * k - side) / 2f - c.x) / k
    val top = ((b.height * k - side) / 2f - c.y) / k
    val x = left.roundToInt().coerceIn(0, b.width - n)
    val y = top.roundToInt().coerceIn(0, b.height - n)
    return Bitmap.createBitmap(b, x, y, n, n)
}

private fun rotate90(b: Bitmap): Bitmap {
    val m = Matrix().apply { postRotate(90f) }
    return Bitmap.createBitmap(b, 0, 0, b.width, b.height, m, true)
}
