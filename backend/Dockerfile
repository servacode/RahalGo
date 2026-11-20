# ══════════════════════════════════════════════════════════════════════
#  محرّكُ المنصّة — بناءٌ على مرحلتين
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-٠٨-١٠: «نجهّز الموقع نرفعه على Render للتجربة».)
#
# **ومرحلتان لا واحدة**: صورةُ Go كاملةً نحوَ ٨٠٠ ميغا، **والثنائيُّ وحدَه
# نحوَ ٣٠** — والفرقُ وقتُ رفعٍ في كلّ نشرةٍ وذاكرةٌ في خطّةٍ مجّانيّة.

FROM golang:1.25-alpine AS build
WORKDIR /src

# **والاعتماديّاتُ في طبقةٍ وحدَها** — تتبدّل نادراً، **فبناءٌ بعد تعديل
# سطرٍ في الشيفرة لا يُعيد تنزيلَها.**
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ══════════════════════════════════════════════════════════════════════
# **وهويّةُ البناء تُحقَن في الثنائيّ — لا تُقرأ من البيئة**
# ══════════════════════════════════════════════════════════════════════
#
# (`P-0` البندان ٥ و١٠ · و`P-10` البند ٣٣.)
#
# **متغيّرُ بيئةٍ يُبدَّل بعد النشر** — **فيقول الخادمُ إنّه يشغّل التزاماً
# وهو يشغّل غيرَه**، وذلك أسوأُ من ألّا يقول شيئاً.
#
# **وفارغتان افتراضاً** — **فبناءٌ بلا هويّةٍ يُعلن ذلك ولا يكذب.**
# **وبيئةُ التجهيز تشترطهما** (`compose.staging.yml` يرفض بلاهما).
ARG SOURCE_COMMIT=""
ARG BUILD_ID=""

# **و`CGO_ENABLED=0` يُخرج ثنائيّاً بلا اعتماديّاتِ نظام** — يعمل في صورةٍ
# فارغةٍ تقريباً. **ومن نسيها يجد «not found» عند التشغيل** والملفُّ موجود.
RUN CGO_ENABLED=0 GOOS=linux go build 	-ldflags="-s -w 	  -X github.com/servacode/rahalgo/backend/internal/envguard.buildCommit=${SOURCE_COMMIT} 	  -X github.com/servacode/rahalgo/backend/internal/envguard.buildID=${BUILD_ID}" 	-o /out/api ./cmd/api

# ══════════════════════════════════════════════════════════════════════
# ── ساعي الأمين لكمبيوتر المول (ويندوز) — يُبنى مع كلّ نشرٍ فيحمل الخادمُ أحدثَه (٢٠٢٦-١٠-١٩) ──
FROM golang:1.25-alpine AS agent
WORKDIR /agent
COPY agent/ameen-sync/go.mod agent/ameen-sync/go.sum ./
RUN go mod download
COPY agent/ameen-sync/ ./
RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H=windowsgui" -o /out/RahalGoSync.exe .  && grep -o 'var Version = "[^"]*"' main.go | cut -d'"' -f2 > /out/agent-version

# ══════════════════════════════════════════════════════════════════════
# ── قراءة صور وتسجيلات الدردشة (قرار المالك ٢٠٢٦-١١-٢٠، internal/mediascan) ──
# whisper.cpp بيحوّل التسجيل لكلام عربي على سيرفرنا — بلا GGML_NATIVE حتى يشتغل على أي معالج.
FROM alpine:3.21 AS whisper
RUN apk add --no-cache build-base cmake git curl
RUN git clone --depth 1 --branch v1.7.4 https://github.com/ggml-org/whisper.cpp /w  && cmake -S /w -B /w/build -DBUILD_SHARED_LIBS=OFF -DGGML_NATIVE=OFF -DWHISPER_BUILD_TESTS=OFF       -DCMAKE_BUILD_TYPE=Release  && cmake --build /w/build -j --config Release --target whisper-cli  && curl -fL -o /model.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small-q5_1.bin

FROM alpine:3.21
# **وشهاداتُ الجذر لازمة**: المحرّكُ ينادي واتساب وخدمةَ العناوين عبر HTTPS،
# **وصورةٌ بلاها تردّ «x509: certificate signed by unknown authority».**
# **وقراءة الصور (tesseract) وتحويل الصوت (ffmpeg + whisper)** — للدردشة (٢٠٢٦-١١-٢٠).
RUN apk add --no-cache ca-certificates tzdata tesseract-ocr tesseract-ocr-data-ara ffmpeg libstdc++ libgomp  && adduser -D -u 10001 rahal
COPY --from=whisper /w/build/bin/whisper-cli /usr/local/bin/whisper-cli
COPY --from=whisper /model.bin /opt/whisper/model.bin
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=agent /out/ /app/agent/

# **ومجلّدُ الوسائط يُنشأ ويُملَك** — والمستخدمُ غيرُ الجذر لا يكتب في مجلّدٍ
# ليس له. (وهو زائلٌ بلا قرصٍ دائم — انظر `docs/DEPLOY.md`.)
RUN mkdir -p /app/uploads && chown -R rahal:rahal /app
USER rahal

ENV UPLOADS_DIR=/app/uploads
ENV AGENT_DIR=/app/agent
EXPOSE 8080
CMD ["/app/api"]
