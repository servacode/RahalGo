package com.rahalgo.ui

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.res.stringResource
import androidx.core.content.FileProvider
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
 * ══════════════════════════════════════════════════════════════════════
 * **منتقي صورة — من الكاميرا أو من المعرض**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «إضافةُ صورةٍ على التطبيق لازم نسمح للكاميرا
 *  بتصويرٍ مباشرٍ أيضاً وليس فقط صورةً محفوظة».)
 *
 * # ولماذا الكاميرا شرطٌ لا زينة
 *
 * **المندوبُ واقفٌ عند الصنف** — يصوّره ويرفعه في دقيقة. **ومن طُلب منه
 * أن يصوّر أوّلاً ثمّ يفتح التطبيق ثمّ يبحث عن صورته في المعرض ترك
 * الصنفَ بلا صورة.**
 *
 * **والزبونُ كذلك** حين يضع صورةَ حسابه.
 *
 * # ولماذا يُسأل ولا يُفترض
 *
 * **من يرفع صورةَ منتجٍ يصوّرها، ومن يرفع صورةَ حسابه يختارها غالبا** —
 * **وفتحُ الكاميرا مباشرةً لمن أراد المعرضَ يُخرجه من التطبيق بلا سبب.**
 *
 * # و`TakePicture` لا `TakePicturePreview`
 *
 * **`TakePicturePreview` تعيد صورةً مصغَّرةً في الذاكرة** — تكفي بيّنةَ
 * تسليمٍ عند السائق، **ولا تكفي صورةَ صنفٍ يراها الزبونُ في السوق.**
 *
 * **فتُكتب في ملفٍّ مؤقّتٍ ويُعطى تطبيقُ الكاميرا عنوانَ مزوّدٍ** —
 * ومسارُ ملفٍّ خامٍّ يُسقط التطبيقَ منذ أندرويد السابع.
 *
 * # والقراءةُ هنا لا في نموذج الشاشة
 *
 * `Uri` صالحٌ لحظةَ اختياره، **ومن مرّره ليُقرأ لاحقاً قرأ عنواناً
 * انتهت صلاحيّتُه.**
 */
@Composable
fun rememberImagePicker(onBytes: (ByteArray) -> Unit): () -> Unit {
    val context = LocalContext.current
    var asking by remember { mutableStateOf(false) }
    // **وعنوانُ الصورة الملتقَطة يُحفظ** — الردُّ يقول «نجحت» ولا يقول
    // أين، **فمن لم يحتفظ به فقد الصورةَ التي التُقطت للتوّ.**
    var shot by remember { mutableStateOf<Uri?>(null) }

    fun deliver(uri: Uri?) {
        // **وفشلُ القراءة لا يُسقط الشاشة** — يُترك بلا صورة.
        uri?.let { runCatching { readScaledImage(context, it) }.getOrNull()?.let(onBytes) }
    }

    val gallery = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia(),
    ) { uri -> deliver(uri) }

    val camera = rememberLauncherForActivityResult(
        ActivityResultContracts.TakePicture(),
    ) { ok -> if (ok) deliver(shot) }

    if (asking) {
        AlertDialog(
            onDismissRequest = { asking = false },
            title = { Text(stringResource(R.string.photo_source)) },
            confirmButton = {
                RahalTextButton(onClick = {
                    asking = false
                    runCatching { newPhotoUri(context) }.getOrNull()?.let {
                        shot = it
                        camera.launch(it)
                    }
                }) { Text(stringResource(R.string.photo_camera)) }
            },
            dismissButton = {
                RahalTextButton(onClick = {
                    asking = false
                    gallery.launch(
                        PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly),
                    )
                }) { Text(stringResource(R.string.photo_gallery)) }
            },
        )
    }

    return { asking = true }
}

/**
 * **ملفٌّ مؤقّتٌ تكتب فيه الكاميرا** — وعنوانُه من المزوّد.
 *
 * **واسمُه يحمل الوقتَ** — **ومن كتب باسمٍ ثابتٍ وجد صورةَ الأمس** إن
 * أخفقت الكاميرا ولم تكتب شيئاً.
 */
private fun newPhotoUri(context: Context): Uri {
    val dir = java.io.File(context.cacheDir, "photos").apply { mkdirs() }
    val file = java.io.File(dir, "shot_" + System.currentTimeMillis() + ".jpg")
    return FileProvider.getUriForFile(context, context.packageName + ".photos", file)
}
