package com.rahalgo.ui

import android.graphics.Bitmap
import android.graphics.Matrix
import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Icon
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.ClipOp
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.rahalgo.design.Rahal
import kotlin.math.max
import kotlin.math.min
import kotlin.math.roundToInt

/**
 * **إطارُ القصّ** — نسبةُ العرض إلى الارتفاع، و`Full` بلا قصٍّ أصلاً.
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٢٨/٢٩: «صاحب المتجر يقدر يخلّي الصورة كاملة».)
 * صورةُ الصنف: كاملة (الافتراض) · مربّعة · طوليّة ٣:٤. **والشعارُ مربّعٌ وحدَه.**
 */
enum class CropFrame(val ratio: Float, @DrawableRes val icon: Int, @StringRes val label: Int) {
    Full(0f, R.drawable.ic_frame_full, R.string.crop_frame_full),
    Square(1f, R.drawable.ic_frame_square, R.string.crop_frame_square),
    Portrait(0.75f, R.drawable.ic_frame_portrait, R.string.crop_frame_portrait),
}

/** إطاراتُ صورة الصنف — أوّلُها الافتراض. */
val ItemCropFrames = listOf(CropFrame.Full, CropFrame.Square, CropFrame.Portrait)

/** إطارُ الشعار. */
val LogoCropFrames = listOf(CropFrame.Square)

/**
 * معاينة الصورة وقصّها قبل الرفع (طلب المالك ٢٠٢٦-١٠-٢٧).
 *
 * اللي بيبين جوّا الإطار هو بالضبط اللي بيوصل للسوق. صاحب الصورة بيكبّر
 * بإصبعين، وبيحرّكها، وبيدوّرها، وما بتنرفع إلا لما يكبس «اعتمد».
 * **وبـ«كاملة» ما في قصّ** — بتنرفع الصورة كلّها بأبعادها، والتدوير وحده مسموح.
 */
@Composable
internal fun ImageCropDialog(
    source: Bitmap,
    frames: List<CropFrame> = LogoCropFrames,
    onDone: (Bitmap) -> Unit,
    onAgain: () -> Unit,
    onCancel: () -> Unit,
) {
    var bmp by remember(source) { mutableStateOf(source) }
    var mode by remember(frames) { mutableStateOf(frames.firstOrNull() ?: CropFrame.Square) }
    var zoom by remember(bmp, mode) { mutableFloatStateOf(1f) }
    var off by remember(bmp, mode) { mutableStateOf(Offset.Zero) }
    var fw by remember { mutableFloatStateOf(0f) }
    var fh by remember { mutableFloatStateOf(0f) }

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
                .padding(Rahal.space.lg),
        ) {
            Text(
                stringResource(R.string.crop_title),
                color = Color.White,
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(Rahal.space.md))

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
                    val (frameW, frameH) = frameSize(bmp, mode, w, h)
                    fw = frameW
                    fh = frameH
                    val k = coverScale(bmp, frameW, frameH) * zoom
                    val movable = mode != CropFrame.Full
                    // **مقصوصةٌ على حدودها** (تدقيقُ ٢٠٢٦-١٠-٢٨) — صورةٌ مكبَّرةٌ أو طويلة
                    // كانت تُرسم فوق العنوان والأزرار.
                    Canvas(
                        Modifier
                            .fillMaxSize()
                            .clipToBounds()
                            .pointerInput(bmp, mode, frameW, frameH) {
                                if (!movable) return@pointerInput
                                detectTransformGestures { _, pan, z, _ ->
                                    zoom = (zoom * z).coerceIn(1f, 6f)
                                    off = clampOffset(bmp, frameW, frameH, zoom, off + pan)
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
                            Offset((size.width - frameW) / 2f, (size.height - frameH) / 2f),
                            Size(frameW, frameH),
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

            Spacer(Modifier.height(Rahal.space.md))
            // **شريطُ الأدوات** — أيقونةٌ وكلمةٌ لكلّ أداة.
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (frames.size > 1) {
                    frames.forEach { f ->
                        CropTool(f.icon, f.label, selected = mode == f) { mode = f }
                    }
                    Box(
                        Modifier
                            .width(Rahal.stroke.hair)
                            .height(36.dp)
                            .background(Color.White.copy(alpha = 0.25f)),
                    )
                }
                CropTool(R.drawable.ic_rotate, R.string.crop_rotate) { bmp = rotate90(bmp) }
                CropTool(R.drawable.ic_reset, R.string.crop_reset) {
                    bmp = source
                    mode = frames.firstOrNull() ?: CropFrame.Square
                    zoom = 1f
                    off = Offset.Zero
                }
                CropTool(R.drawable.ic_photo, R.string.crop_again, onClick = onAgain)
            }
            Spacer(Modifier.height(Rahal.space.md))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(Rahal.space.sm),
            ) {
                CropAction(
                    R.drawable.ic_close,
                    R.string.act_cancel,
                    primary = false,
                    onClick = onCancel,
                )
                CropAction(R.drawable.ic_check_mark, R.string.crop_use, primary = true) {
                    onDone(if (mode == CropFrame.Full) bmp else cropFrame(bmp, fw, fh, zoom, off))
                }
            }
        }
    }
}

@Composable
private fun CropTool(
    @DrawableRes icon: Int,
    @StringRes label: Int,
    selected: Boolean = false,
    onClick: () -> Unit,
) {
    val tint = if (selected) Rahal.colors.accent else Color.White
    Column(
        Modifier
            .clip(Rahal.shape.sm)
            .background(if (selected) Color.White.copy(alpha = 0.12f) else Color.Transparent)
            .clickable(role = Role.Button, onClick = onClick)
            .heightIn(min = 56.dp)
            .padding(horizontal = Rahal.space.sm, vertical = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(24.dp))
        Spacer(Modifier.height(2.dp))
        Text(stringResource(label), color = tint, style = MaterialTheme.typography.labelSmall)
    }
}

@Composable
private fun RowScope.CropAction(
    @DrawableRes icon: Int,
    @StringRes label: Int,
    primary: Boolean,
    onClick: () -> Unit,
) {
    val content: @Composable RowScope.() -> Unit = {
        Icon(painterResource(icon), contentDescription = null, modifier = Modifier.size(20.dp))
        Spacer(Modifier.width(6.dp))
        Text(stringResource(label))
    }
    if (primary) {
        RahalButton(onClick = onClick, modifier = Modifier.weight(1f), content = content)
    } else {
        // **محدَّدٌ أبيضُ على الأسود** — حدُّ العلامة الداكن يذوب في الخلفيّة.
        Row(
            Modifier
                .weight(1f)
                .heightIn(min = 48.dp)
                .clip(Rahal.shape.md)
                .border(Rahal.stroke.hair, Color.White.copy(alpha = 0.6f), Rahal.shape.md)
                .clickable(role = Role.Button, onClick = onClick)
                .padding(horizontal = 20.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            CompositionLocalProvider(androidx.compose.material3.LocalContentColor provides Color.White) {
                content()
            }
        }
    }
}

/** أبعادُ الإطار داخل مساحة العرض. «كاملة» = الصورةُ كلُّها محتواةً فيها. */
private fun frameSize(b: Bitmap, mode: CropFrame, w: Float, h: Float): Pair<Float, Float> {
    if (w <= 0f || h <= 0f) return 0f to 0f
    if (mode == CropFrame.Full) {
        val s = min(w / b.width, h / b.height)
        return b.width * s to b.height * s
    }
    val fw = min(w, h * mode.ratio)
    return fw to fw / mode.ratio
}

/** المقياس اللي بيخلّي الصورة تغطّي الإطار كلّه. */
private fun coverScale(b: Bitmap, fw: Float, fh: Float): Float =
    if (fw <= 0f || fh <= 0f) 1f else max(fw / b.width, fh / b.height)

/** ما بيسمح بفراغ جوّا الإطار — الصورة دايماً مغطّيته. */
private fun clampOffset(b: Bitmap, fw: Float, fh: Float, zoom: Float, o: Offset): Offset {
    val k = coverScale(b, fw, fh) * zoom
    val mx = max(0f, (b.width * k - fw) / 2f)
    val my = max(0f, (b.height * k - fh) / 2f)
    return Offset(o.x.coerceIn(-mx, mx), o.y.coerceIn(-my, my))
}

/** الجزء اللي جوّا الإطار، بإحداثيات الصورة نفسها. */
internal fun cropFrame(b: Bitmap, fw: Float, fh: Float, zoom: Float, o: Offset): Bitmap {
    if (fw <= 0f || fh <= 0f) return b
    val k = coverScale(b, fw, fh) * zoom
    val c = clampOffset(b, fw, fh, zoom, o)
    val cw = (fw / k).roundToInt().coerceIn(1, b.width)
    val ch = (fh / k).roundToInt().coerceIn(1, b.height)
    val left = ((b.width * k - fw) / 2f - c.x) / k
    val top = ((b.height * k - fh) / 2f - c.y) / k
    val x = left.roundToInt().coerceIn(0, b.width - cw)
    val y = top.roundToInt().coerceIn(0, b.height - ch)
    return Bitmap.createBitmap(b, x, y, cw, ch)
}

private fun rotate90(b: Bitmap): Bitmap {
    val m = Matrix().apply { postRotate(90f) }
    return Bitmap.createBitmap(b, 0, 0, b.width, b.height, m, true)
}
