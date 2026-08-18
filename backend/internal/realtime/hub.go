// Package realtime مركز البث الحي (WebSocket): مواضيع → مشتركون.
// تنفيذ داخل العملية (خادم واحد — معمارية النشر المعتمدة)؛ عند التوسع لعدة
// خوادم يُستبدل النشر بـ Redis Pub/Sub خلف الواجهة نفسها.
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

const (
	// TopicOps بث غرفة العمليات: كل تحديثات الطلبات (أدمن/عمليات/مالية)
	TopicOps = "ops"

	// TopicDriverQueue إشارةُ «الطابور تغيّر» — يشترك فيها كلُّ سائق.
	//
	// **إشارةٌ لا حمولة.** ما يُبثّ هنا يصل كلَّ سائقي المنصة، وطلبُ الطابور
	// يحمل اسمَ الزبون وهاتفَه وعنوانَه ومبلغَه. **فبثُّ الطلب هنا يُسلّم بيانات
	// زبونٍ إلى عشرين سائقاً لا يخصّ الطلبُ تسعةَ عشرَ منهم.**
	//
	// فالإشارة تقول «تغيّر شيء» ولا تقول ماذا، **ويبقى القرارُ في نقطة الطابور**
	// التي تحكم ما يُرى وما لا يُرى. وهي مذهبُ المشروع نفسه: الحجبُ في الخادم
	// لا في الشاشة.
	TopicDriverQueue = "drivers:queue"

	// TopicCatalog إشارةُ «لوحةُ الإدارة كتبت شيئاً» — يشترك فيها كلُّ
	// متّصلٍ مهما كان دورُه.
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٨: «في مشكلةُ التحديث اللحظيّ بالتطبيق…
	//  أيُّ تعديلٍ من لوحة الأدمن فوراً يُطبَّق حتّى ولو الزبونُ فاتحٌ
	//  التطبيق، ما يلزم يحدّث أو يعيد تشغيل التطبيق».)
	//
	// # ولماذا موضوعٌ خامسٌ لا `ops`
	//
	// **`ops` تُبثّ للإدارة وحدَها** — والزبونُ لا يشترك فيها ولا يجوز:
	// حمولتُها طلباتٌ بأسماءِ زبائنَ وهواتفِهم.
	//
	// # وإشارةٌ لا حمولة
	//
	// **تقول «تغيّر شيء» ولا تقول ماذا** — كما `TopicDriverQueue` حرفاً.
	// ولو حملت الجديدَ لَبنى كلُّ تطبيقٍ شاشتَه من الحمولة، **فصارت
	// نسخةً ثانيةً من قواعدَ لا يملكها** — ومن حُجب عنه صنفٌ في نقطته
	// رآه في بثّه.
	TopicCatalog = "catalog"
)

type subscriber struct {
	topics map[string]bool
	ch     chan []byte
}

type Hub struct {
	mu     sync.RWMutex
	subs   map[*subscriber]bool
	logger *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{subs: map[*subscriber]bool{}, logger: logger}
}

// Publish يبث حدثاً لكل المشتركين في الموضوع — لا يحجب أبداً:
// المشترك البطيء تُسقط رسائله بدل تعطيل البقية.
func (h *Hub) Publish(topic string, event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("realtime: marshal event", "error", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subs {
		if !sub.topics[topic] {
			continue
		}
		select {
		case sub.ch <- payload:
		default: // مشترك ممتلئ — أسقط الرسالة
		}
	}
}

// Subscribe يسجل مشتركاً بمواضيعه ويعيد قناة الرسائل ودالة إلغاء.
func (h *Hub) Subscribe(topics []string) (<-chan []byte, func()) {
	sub := &subscriber{topics: map[string]bool{}, ch: make(chan []byte, 64)}
	for _, t := range topics {
		sub.topics[t] = true
	}
	h.mu.Lock()
	h.subs[sub] = true
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		delete(h.subs, sub)
		h.mu.Unlock()
	}
	return sub.ch, cancel
}

// Run لا شيء حالياً — موجودة لاتساق دورة الحياة مستقبلاً.
func (h *Hub) Run(ctx context.Context) { <-ctx.Done() }
