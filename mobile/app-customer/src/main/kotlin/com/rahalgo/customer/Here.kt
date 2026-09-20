package com.rahalgo.customer

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import com.rahalgo.ui.Accuracy
import com.rahalgo.ui.Locating

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أين هو الآن — قراءةٌ واحدةٌ عند الحاجة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا لا خدمةَ كخدمة السائق
 *
 * **السائقُ يُتتبَّع ما دامت ورديّتُه مفتوحة** — الزبونُ يراه يتحرّك.
 * **والزبونُ لا يُتتبَّع أبدا**: موقعُه لا يلزم إلّا لحظةَ يحفظ عنواناً،
 * **ومرّةً في العمر أو مرّتين.**
 *
 * **وتطبيقٌ يتتبّع من لا حاجةَ لتتبّعه يستنزف بطّاريّتَه** — ويُسأل عنه
 * في متجر غوغل، **ويُرفض إن لم يكن له سببٌ ظاهر.**
 *
 * # والقراءةُ نفسُها في وحدة الخرائط
 *
 * **وكان هذا الملفُّ نسخةً ثالثةً من `map/Here`** — **حرفاً بحرف**:
 * **فأُصلحت إحداها (مرحلتان وأسبابُ تعذّرٍ ومهلة) وبقيت هذه كما
 * كانت تنادي وتمضي.**
 *
 * **فلم يبقَ هنا إلّا ما يخصُّ الزبونَ وحدَه**: **نقطةُ الاستكشاف.**
 *
 * # وفشلُها لا يُسقط شيئا
 *
 * **من رفض الإذنَ أو أطفأ موقعَه يكتب عنوانَه نصّاً** — والحقلُ يبقى
 * فارغاً ويعرف أنّه فارغ. **وشاشةٌ تنتظر نقطةً لا تجيء أسوأُ من عنوانٍ
 * بلا نقطة.**
 */
object Here {

    /**
     * **نقطةُ الاستكشاف** — **من الجهاز، تُخبِر ولا تحكم** (`DL`).
     *
     * **ولا تصير عنوانَ توصيلٍ إلّا بفعلٍ صريحٍ من صاحبها.**
     */
    var discovery by androidx.compose.runtime.mutableStateOf<Discovery?>(null)
        private set

    fun granted(context: Context): Boolean = com.rahalgo.map.Here.granted(context)

    /** **رفضٌ عاديٌّ أم نهائيّ** — بالمحرّك المركزيّ (`CUST-DEF-006`). */
    fun deniedProblem(activity: android.app.Activity): Locating.Problem =
        com.rahalgo.map.Here.deniedProblem(activity)

    /**
     * **يقرأ الموضعَ مرّةً** — **بالمحرّك المركزيّ.**
     *
     * **وحدُّ الدقّة `CONFIRM_M`** — **فالعنوانُ يُسلَّم إليه طلبٌ**:
     * **ونقطةٌ بخطإِ كيلومترٍ بابُ حيٍّ آخر.**
     *
     * **ونقطةُ الاستكشاف تُحفَظ بدقّتها** — **ونقطةٌ لا تُعرَف دقّتُها
     * لا يُحكَم بها** (`DL-11`).
     */
    fun refresh(context: Context, state: Locating? = null) {
        com.rahalgo.map.Here.refresh(
            context = context,
            state = state,
            limit = Accuracy.CONFIRM_M,
        ) { lat, lng, acc ->
            discovery = Discovery(lat, lng, acc)
        }
    }
}
