package com.rahalgo.ui

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.rahalgo.design.Rahal

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موضعُ التوصيل — خريطةٌ تُفتح لا إحداثيّاتٌ تُقرأ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «تحديد الموقع على الخريطة لازم يفتح خريطة،
 *  لأنّ أغلب الناس ما تفهم شو يعني «نقطتك على الخريطة» أو هذه الأرقام.
 *  تفهم تحدّد الموقع على خريطةٍ واضحة وخلص».)
 *
 * # وما كان قبلَه
 *
 * **حقلٌ يعرض `35.95012، 39.00778` وزرُّ «حدّد موقعي»** — **ورقمان بست
 * منازلَ لا يقرؤهما أحد**: لا يعرف أصحيحان أم لا، ولا يعرف أين هما.
 *
 * **وقرارٌ سابقٌ يقول «مو ضروريّ تفتح خريطة»** (٢٠٢٦-٠٨-١٣) — **وكان في
 * شاشة الحساب**: هناك يحفظ عنواناً وهو في بيته، **وهنا يطلب توصيلاً
 * إلى مكانٍ قد لا يكون فيه.**
 *
 * # وما صار
 *
 * **الاسمُ لا الرقم**: الخريطةُ تقرأ اسمَ الموضع وتُكتب هنا — **ومن رأى
 * اسمَ حيّه عرف أنّه أصاب**، ومن رأى رقمين لا يعرف شيئا.
 *
 * **وزرٌّ واحدٌ يفتح خريطةً بدبّوسٍ في وسطها** — يحرّكها تحته ويؤكّد.
 * **وفيها بحثٌ وزرُّ «موقعي»** فلا يُفقَد ما كان يعطيه الزرُّ القديم.
 *
 * # ولماذا هنا
 *
 * **السلّةُ والطلبُ الخاصّ وشاشةُ الحساب تسأل السؤالَ نفسَه** —
 * **وثلاثُ نسخٍ منه تعني ثلاثةَ أشكالٍ لموضعٍ واحد**، وهو ما وقع:
 * الحسابُ يفتح خريطةً والسلّةُ تعرض رقمين.
 */
@Composable
fun PointField(
    picker: PointPicker?,
    label: String = stringResource(R.string.point_label),
    modifier: Modifier = Modifier,
) {
    var onMap by rememberSaveable { mutableStateOf(false) }
    val p = LastPoint.value

    OutlinedTextField(
        // **واسمُ الموضع إن عُرف** — **وإحداثيّاه إن لم تُجب الخريطة**:
        // رقمان يُقرآن بصعوبةٍ خيرٌ من حقلٍ فارغٍ يقول «لم تحدّد».
        value = when {
            p == null -> ""
            p.name.isNotEmpty() -> p.name
            else -> "%.5f، %.5f".format(p.lat, p.lng)
        },
        onValueChange = {},
        readOnly = true,
        label = { Text(label) },
        placeholder = { Text(stringResource(R.string.point_none)) },
        modifier = modifier.fillMaxWidth(),
    )

    Spacer(Modifier.height(6.dp))
    OutlinedButton(onClick = { onMap = true }, modifier = Modifier.fillMaxWidth()) {
        Icon(
            painter = painterResource(R.drawable.ic_pin),
            contentDescription = null,
            tint = Rahal.colors.brand,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.size(8.dp))
        Text(
            stringResource(
                if (p == null) R.string.point_pick else R.string.point_change,
            ),
        )
    }

    // **والخريطةُ تُعطى من التطبيق** — `:ui` لا تعرفها، **ولو عرفتها
    // لَحملها كلُّ تطبيقٍ معه** وفيهم من لا خريطةَ له.
    if (onMap && picker != null) {
        picker(
            { lat, lng, name ->
                LastPoint.set(lat, lng, name)
                onMap = false
            },
            { onMap = false },
        )
    }
}
