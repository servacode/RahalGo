package obs

import "testing"

// BenchmarkWSAuth **كلفةُ عدّ سببٍ** — على مسارِ كلّ مصافحة.
func BenchmarkWSAuth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		WSAuth(WSExpiredAccess)
	}
}

// BenchmarkClient **كلفةُ عدّ نسخةٍ** — على مسارِ كلّ نداءٍ من تطبيق.
//
// **وهي الأغلى** (قفلٌ وخريطة)، **فهي التي تُقاس.**
func BenchmarkClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Client("customer", 11)
	}
}

// BenchmarkClientParallel **وتحت تزاحمٍ حقيقيّ** — قفلٌ واحدٌ لكلّ نداء.
func BenchmarkClientParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Client("customer", 11)
		}
	})
}

// BenchmarkTake **وقراءةُ اللقطة** — مرّةً لكلّ نداءِ تشخيص، لا أكثر.
func BenchmarkTake(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Take()
	}
}
