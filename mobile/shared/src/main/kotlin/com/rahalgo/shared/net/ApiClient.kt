package com.rahalgo.shared.net

import com.rahalgo.shared.model.ApiErrorBody
import com.rahalgo.shared.model.Envelope
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.header
import io.ktor.client.request.request
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.http.ContentType
import io.ktor.http.HttpMethod
import io.ktor.http.contentType
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عميل المحرّك — واحد لكل التطبيقات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`GROUND-RULES.md` §7.1: الشبكة في `shared` لا في التطبيق.)
 *
 * # ما يفعله في كل نداء
 *
 * **يضع ترويسة نوع العميل** (`X-RahalGo-Client`) — وهي التي تفصل جلسة
 * التطبيق عن جلسة المتصفّح في المحرّك (هجرة `0099`). **وبلاها يُقرأ
 * التطبيق متصفّحا فيُخرج صاحبه من الويب** كلما فتحه.
 *
 * **ويفكّ الغلاف**: المحرّك يلفّ كل رد في `{"data": …}` والخطأ في
 * `{"error": …}`. **فمن قرأ الجسم مباشرة قرأ الغلاف لا المحتوى.**
 *
 * **ويجدّد التوكن مرّة عند ٤٠١ ثمّ يعيد النداء** — كما يفعل عميل الويب
 * منذ زمن. **وبلاه يخرج السائق كل ساعة** حين تنتهي مهلة توكن الوصول.
 */
class ApiClient(
    // ══════════════════════════════════════════════════════════════════
    // **`@PublishedApi internal` لا `private`**
    // ══════════════════════════════════════════════════════════════════
    //
    // **الدالّة المضمَّنة تُنسخ في موضع ندائها** — أي في وحدة أخرى، فلا
    // تصل ما هو خاصّ. **والتضمين شرط هنا**: `reified` هي التي تجعل
    // `call<User>(…)` تعرف نوعها، **وبدونها يُمرَّر محوّل JSON يدويا في
    // كل نداء.**
    //
    // **و`internal` لا `public`**: تُرى داخل الوحدة وحدها، **فلا تصير
    // جزءا من واجهة يعتمد عليها غيرها.**
    @PublishedApi internal val baseUrl: String,
    @PublishedApi internal val client: String,
    @PublishedApi internal val session: SessionStore,
) {
    @PublishedApi internal val json = Json {
        // **وحقل جديد في المحرّك لا يُسقط التطبيق القديم** — يُتجاهل.
        // **وبلا هذا ينكسر كل تطبيق منشور مع أول حقل يُضاف.**
        ignoreUnknownKeys = true
        // **وحقل ناقص يأخذ قيمة النموذج الافتراضية** لا يُسقط الفكّ.
        explicitNulls = false
    }

    @PublishedApi internal val http = HttpClient {
        install(ContentNegotiation) { json(json) }
    }

    /** خطأ من المحرّك — **يحمل رمزه ومفتاح رسالته كما قالهما.** */
    class ApiException(
        val status: Int,
        val body: ApiErrorBody,
    ) : Exception("api ${body.code} ($status)")

    /**
     * نداء مصادَق — **يجدّد ويعيد مرّة واحدة عند ٤٠١.**
     *
     * **ومرّة واحدة لا حلقة**: توكن تجديد ميّت يعيد ٤٠١ إلى الأبد،
     * **فيدور التطبيق في مكانه ويستنزف البطارية** بدل أن يطلب دخولا.
     */
    suspend inline fun <reified T> call(
        path: String,
        method: HttpMethod = HttpMethod.Get,
        body: Any? = null,
        idempotencyKey: String? = null,
    ): T {
        try {
            return raw(path, method, body, idempotencyKey, session.accessToken())
        } catch (e: ApiException) {
            if (e.status != 401 || session.refreshToken().isEmpty()) throw e
            refresh()
            return raw(path, method, body, idempotencyKey, session.accessToken())
        }
    }

    /** نداء بلا تجديد — للدخول وما لا توكن له. */
    suspend inline fun <reified T> raw(
        path: String,
        method: HttpMethod = HttpMethod.Get,
        body: Any? = null,
        idempotencyKey: String? = null,
        token: String = "",
    ): T {
        val res: HttpResponse = http.request(baseUrl + path) {
            this.method = method
            header(CLIENT_HEADER, client)
            if (token.isNotEmpty()) header("Authorization", "Bearer $token")
            // **ومفتاح منع التكرار حيث يُطلب** — النقاط التي تكتب مالا
            // (انظر `server/idempotency.go` و`api/contract.json`).
            if (idempotencyKey != null) header("Idempotency-Key", idempotencyKey)
            if (body != null) {
                contentType(ContentType.Application.Json)
                setBody(body)
            }
        }
        val env: Envelope<T> = res.body()
        val err = env.error
        if (err != null || env.data == null) {
            throw ApiException(res.status.value, err ?: ApiErrorBody(code = "internal"))
        }
        return env.data
    }

    /** يدوّر التوكن ويحفظ الجديد. */
    suspend fun refresh() {
        val result: com.rahalgo.shared.model.AuthResult = raw(
            "/api/v1/auth/refresh",
            HttpMethod.Post,
            mapOf("refresh_token" to session.refreshToken()),
        )
        session.save(result.tokens.accessToken, result.tokens.refreshToken)
    }

    companion object {
        const val CLIENT_HEADER = "X-RahalGo-Client"
    }
}

/**
 * **مخزن الجلسة — واجهة هنا وتنفيذ عند المنصة.**
 *
 * **و`shared` لا تعرف أندرويد** (القاعدة ٧.٢): حفظ التوكن يحتاج
 * `Context` و`EncryptedSharedPreferences` — **وكلاهما أندرويد خالص.**
 * فتُعرَّف الحاجة هنا ويُنفَّذ التخزين هناك.
 */
interface SessionStore {
    fun accessToken(): String
    fun refreshToken(): String
    fun save(access: String, refresh: String)
    fun clear()
}
