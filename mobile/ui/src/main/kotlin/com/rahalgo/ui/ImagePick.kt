package com.rahalgo.ui

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext
import java.io.ByteArrayOutputStream
import java.io.IOException

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختيارُ صورةٍ وقراءتُها — لكلّ شاشةٍ ترفع**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت في `AccountViewModel` وحدَها** — صورةُ الحساب. **ثمّ احتاجها
 * المندوبُ لصور الأصناف** (طلبُ المالك ٢٠٢٦-٠٨-١٨).
 *
 * **ونسخةٌ ثانيةٌ من التصغير تعني موضعين يُصلَح فيهما العيبُ ويُنسى
 * ثانيهما** — **وهاتفٌ يسقط عند رفع صورةٍ كبيرة يسقط في أحدهما فقط**،
 * فيُظنّ أنّه أُصلح.
 */

/** **أطولُ ضلعٍ يُرسَل** — والمحرّكُ يصغّرها ثانيةً إلى حدّه. */
private const val MAX_EDGE = 1600

private const val JPEG_QUALITY = 85

/**
 * **يقرأ الصورةَ ويصغّرها ويضغطها.**
 *
 * **والأبعادُ تُقرأ أوّلاً بلا فكّ** (`inJustDecodeBounds`) — صورةٌ
 * بأربعة آلاف بكسلٍ تُفكّ إلى أربعةٍ وستّين ميغا في ذاكرة الهاتف،
 * **وهاتفٌ قديمٌ يسقط قبل أن يرفع.**
 */
fun readScaledImage(context: Context, uri: Uri): ByteArray {
    val cr = context.contentResolver
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    cr.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, bounds) }
    var sample = 1
    while (bounds.outWidth / sample > MAX_EDGE || bounds.outHeight / sample > MAX_EDGE) {
        sample *= 2
    }
    val bmp = cr.openInputStream(uri)?.use {
        BitmapFactory.decodeStream(
            it,
            null,
            BitmapFactory.Options().apply { inSampleSize = sample },
        )
    } ?: throw IOException(context.getString(R.string.err_photo_unreadable))

    val out = ByteArrayOutputStream()
    bmp.compress(Bitmap.CompressFormat.JPEG, JPEG_QUALITY, out)
    bmp.recycle()
    return out.toByteArray()
}

/**
 * **منتقي صورةٍ من النظام — يردّ بايتاتِها مصغَّرة.**
 *
 * **`PickVisualMedia` يعطي ملفّاً واحداً اختاره صاحبُه** — **والإذنُ
 * العامُّ يطلب أن يُفتح الألبومُ كلُّه لتطبيق عمل**، وهو ما يرفضه
 * أكثرُهم فيقف الرفع.
 *
 * **والقراءةُ هنا لا في نموذج الشاشة**: `Uri` صالحٌ لحظةَ اختياره،
 * **ومن مرّره ليُقرأ لاحقاً قرأ عنواناً انتهت صلاحيّتُه.**
 */
@Composable
fun rememberImagePicker(onBytes: (ByteArray) -> Unit): () -> Unit {
    val context = LocalContext.current
    val launcher = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia(),
    ) { uri ->
        // **وفشلُ القراءة لا يُسقط الشاشة** — يُترك بلا صورة.
        uri?.let { runCatching { readScaledImage(context, it) }.getOrNull()?.let(onBytes) }
    }
    return {
        launcher.launch(
            PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
        )
    }
}
