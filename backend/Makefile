.PHONY: run build test test-db fmt lint tidy seed

run:
	go run ./cmd/api

# زراعة بيانات التطوير (حسابات بكلمات مرور معروفة، متاجر، مناطق، أكواد)
seed:
	go run ./cmd/seed

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

# **صمتُ التخطّي يُقال.**
#
# اختباراتُ القاعدة تتخطّى نفسها بهدوء إن لم تُضبط TEST_DATABASE_URL — وهو
# صحيحٌ لمن لا قاعدة لديه. لكنّ `ok` بلا نجمةٍ ولا تنبيه **يُقرأ نجاحاً وهو
# تخطٍّ**، وقد قُرئ كذلك فعلاً: أُعلن «ست حزمٍ خضراء» وأخطرُ اختباراتِ المال
# لم تُشغَّل أصلاً.
test:
ifndef TEST_DATABASE_URL
	@echo "تنبيه: TEST_DATABASE_URL غير مضبوط — اختبارات القاعدة ستُتخطّى (المال والهجرات)."
	@echo "      للتشغيل الكامل: make test-db"
endif
	go test ./...

# التشغيل الكامل على قاعدة اختبارٍ منفصلة عن التطوير.
#
# **قاعدةٌ أخرى لا قاعدةُ التطوير**: الاختبارات تُنشئ متاجر وطلباتٍ وقيوداً
# مالية، وتشغيلها على قاعدة التطوير يلوّثها ببياناتٍ تبدو حقيقية. وحزمةُ
# `testdb` ترفض أيَّ قاعدة لا يقول اسمُها إنها للاختبار.
test-db:
	@docker exec rahalgo-postgres psql -U rahalgo -d postgres -c "CREATE DATABASE rahalgo_test" 2>/dev/null || true
	@docker exec rahalgo-postgres psql -U rahalgo -d rahalgo_test -c "CREATE EXTENSION IF NOT EXISTS postgis" >/dev/null
	TEST_DATABASE_URL="postgres://rahalgo:rahalgo_dev@localhost:5434/rahalgo_test?sslmode=disable" go test -count=1 -p 1 ./...

fmt:
	gofmt -l -w .

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "gofmt: files need formatting:" && gofmt -l . && exit 1)

tidy:
	go mod tidy

# ══════════════════════════════════════════════════════════════════════
# **أثرُ التغيير — `P-9` البند ٤١**
# ══════════════════════════════════════════════════════════════════════
#
# **ولا يفتح المطوّرُ خمسةَ عشرَ ملفّاً بيده ليعرف ماذا يشغّل.**

.PHONY: test-impact impact truth matrices

## ما الذي انكسر؟ — شجرةُ العمل
test-impact:
	@go run ./cmd/testimpact

## مدىً في git — make impact BASE=HEAD~1 HEAD=HEAD
impact:
	@go run ./cmd/testimpact -base $(BASE) -head $(HEAD)

## إعادةُ توليد حقيقة الاختبار
truth:
	@go run ./cmd/testtruth

## إعادةُ توليد المصفوفات الآليّة الثمان
matrices:
	@go run ./cmd/fininvdoc

# ══════════════════════════════════════════════════════════════════════
# **بوّابةُ الإطلاق — `P-10`**
# ══════════════════════════════════════════════════════════════════════
#
# **ورمزُ الخروج هو الحكم**: 0 جاهزٌ · 1 غيرُ جاهزٍ · 2 عطبُ بوّابة.
# **فمن ربطها بخطّ نشرٍ مَنَع الرفعَ بلا قراءةِ تقرير.**

.PHONY: gate gate-json

## READY FOR PRODUCTION = YES / NO
gate:
	@go run ./cmd/releasegate

## الحكمُ الآليُّ وحدَه — لخطّ النشر
gate-json:
	@go run ./cmd/releasegate -json
