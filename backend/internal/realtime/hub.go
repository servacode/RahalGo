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
