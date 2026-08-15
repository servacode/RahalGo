package com.rahalgo.ui

import android.graphics.BitmapFactory
import com.rahalgo.design.Rahal
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import java.net.URL
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صورة الحساب — ودائرة بحرفه إن لم تكن**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرار المالك ٢٠٢٦-٠٨-١٢: «قبل أيقونة المحفظة لازم نحطّ صورة البروفايل
 *  واسم المستخدم».)
 *
 * # ولماذا حرف لا صورة رماديّة
 *
 * **أكثر السائقين لن يرفعوا صورة** — وشبح رماديّ في كلّ شريط يُقرأ عطبا
 * («لم تُحمّل الصورة»)، **والحرف يُقرأ حسابا.**
 *
 * # ولماذا لا مكتبة صور
 *
 * **صورة واحدة صغيرة في التطبيق كلّه** — ومكتبة تحميل صور تبني ذاكرة
 * وقرصا وخيوطا لهذا. **ويوم نعرض شعارات المتاجر وصور الأصناف** تُضاف
 * `coil` وتُستبدل هذه.
 *
 * **والذاكرة في `remember` لا في متغيّر ساكن**: الشريط يُعاد رسمه عشرات
 * المرّات في الدقيقة، **ونداء شبكة في كلّ رسم** يستنزف الخطّ والبطاريّة.
 */
@Composable
fun Avatar(url: String?, name: String, size: Int = 28) {
    var bitmap by remember(url) { mutableStateOf<androidx.compose.ui.graphics.ImageBitmap?>(null) }

    LaunchedEffect(url) {
        val u = url ?: return@LaunchedEffect
        bitmap = withContext(Dispatchers.IO) {
            runCatching {
                URL(u).openStream().use { BitmapFactory.decodeStream(it) }?.asImageBitmap()
            }.onFailure {
                // **وصورةٌ لا تصل تقول لماذا** — كانت تسقط إلى الحرف
                // صامتةً، **فيُقرأ «لا صورةَ له» وهو قد رفعها.**
                //
                // **وفرقٌ بين حسابٍ بلا صورةٍ ورابطٍ لا يُفتح**: الأوّلُ
                // حالٌ، **والثاني عطبٌ لا يُرى إلّا في سجلّ.**
                android.util.Log.w("RahalGo/صورة", "تعذّر جلبُ الصورة: " + u, it)
            }.getOrNull()
        }
    }

    val shot = bitmap
    Box(
        Modifier
            .size(size.dp)
            .clip(CircleShape)
            .background(Rahal.colors.brand.copy(alpha = 0.15f)),
        contentAlignment = Alignment.Center,
    ) {
        if (shot != null) {
            Image(
                bitmap = shot,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.size(size.dp),
            )
        } else {
            Text(
                // **وحرف واحد** — أوّل اسمه. **واسم فارغ يُعطي دائرة
                // فارغة** لا حرفا غريبا.
                text = name.trim().take(1),
                color = Rahal.colors.brand,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.labelLarge,
            )
        }
    }
}
