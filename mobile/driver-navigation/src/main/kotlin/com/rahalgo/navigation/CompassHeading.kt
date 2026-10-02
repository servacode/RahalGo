package com.rahalgo.navigation

import android.hardware.GeomagneticField
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext

/**
 * ══════════════════════════════════════════════════════════════════════
 * **البوصلةُ — اتّجاهُ الجهاز حين لا يقول السيرُ شيئاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-١٠-٠٢: «المغناطيس والاتّجاهات حسب دوران وحركة الجهاز، بحيث دائماً
 *  نرى الطريقَ بشكلٍ صحيح».)
 *
 * **اتّجاهُ السير يُقرأ من الحركة** — وهو صحيحٌ ما دام يسير. **والواقفُ والبطيءُ لا اتّجاهَ
 * لسيرهما** (قِيس: ٢٨٠° ثابتةٌ داخلَ بيتٍ واقف)، فتدور الخريطةُ حينها مع الجهاز كما في غوغل.
 *
 * # ولا يُعاد تركيبُ الشاشة مع كلّ قراءة
 *
 * **المستشعرُ يقرأ عشرين مرّةً في الثانية** — ولو كان حالةَ Compose لأُعيد بناءُ شاشة الرحلة
 * كلُّها بها. **فالقيمةُ في مصفوفةٍ تقرؤها الخريطةُ في حلقتها**: `[0]` الدرجةُ الملساءُ من
 * الشمال الحقيقيّ، و`NaN` لا قراءةَ (أو مُطفأة).
 */
@Composable
fun rememberCompassHeading(enabled: Boolean): FloatArray {
    val holder = remember { floatArrayOf(Float.NaN) }
    val context = LocalContext.current
    DisposableEffect(enabled) {
        if (!enabled) {
            holder[0] = Float.NaN
            return@DisposableEffect onDispose { }
        }
        val sm = context.getSystemService(SensorManager::class.java)
        val sensor = sm?.getDefaultSensor(Sensor.TYPE_ROTATION_VECTOR)
        if (sm == null || sensor == null) {
            holder[0] = Float.NaN
            return@DisposableEffect onDispose { }
        }
        // **والمستشعرُ يقيس الشمالَ المغناطيسيّ** — والخريطةُ بالجغرافيّ. والانحرافُ في سوريا
        // نحو ٥° شرقاً، يُحسب مرّةً لوسط البلاد (يتغيّر أقلَّ من درجةٍ بين المحافظات).
        val declination = GeomagneticField(35.0f, 38.5f, 0f, System.currentTimeMillis()).declination
        val rot = FloatArray(9)
        val ori = FloatArray(3)
        val listener = object : SensorEventListener {
            override fun onSensorChanged(e: SensorEvent) {
                SensorManager.getRotationMatrixFromVector(rot, e.values)
                SensorManager.getOrientation(rot, ori)
                val raw = GpsQuality.normalize(Math.toDegrees(ori[0].toDouble()).toFloat() + declination)
                holder[0] = smooth(holder[0], raw)
            }

            override fun onAccuracyChanged(s: Sensor?, accuracy: Int) = Unit
        }
        sm.registerListener(listener, sensor, SensorManager.SENSOR_DELAY_UI)
        onDispose {
            sm.unregisterListener(listener)
            holder[0] = Float.NaN
        }
    }
    return holder
}

/** **تمهيدٌ دائريٌّ** — ٣٥٩° و١° جاران لا طرفان. */
internal fun smooth(prev: Float, now: Float, alpha: Float = COMPASS_ALPHA): Float {
    if (prev.isNaN()) return now
    val diff = ((now - prev + 540f) % 360f) - 180f
    return GpsQuality.normalize(prev + alpha * diff)
}

/** **وزنُ القراءة الجديدة** — رجفةُ اليد لا تُدير الخريطة. */
internal const val COMPASS_ALPHA = 0.15f

/** **دونَ هذه السرعة تقود البوصلةُ** — ٩ كم/س، كالمشي السريع. */
const val COMPASS_BELOW_MPS = 2.5
