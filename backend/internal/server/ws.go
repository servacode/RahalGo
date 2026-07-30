package server

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/coder/websocket"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// handleWS اتصال البث الحي. المتصفح لا يرسل ترويسات مع WebSocket،
// فالتوكن يصل عبر معامل الاستعلام ويُتحقق منه كأي طلب.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	claims, err := s.tokens.VerifyAccess(r.URL.Query().Get("token"))
	if err != nil {
		httpx.Error(w, errUnauthorized)
		return
	}

	topics := []string{}
	if slices.ContainsFunc(claims.Roles, func(role string) bool {
		return role == "admin" || role == "ops" || role == "finance"
	}) {
		topics = append(topics, realtime.TopicOps)
	}
	// صاحب متجر: يشترك بمواضيع متاجره (بوابة المتجر)
	if slices.Contains(claims.Roles, "merchant") {
		rows, err := s.pg.Query(r.Context(),
			`SELECT id FROM merchants WHERE owner_user_id = $1`, claims.Subject)
		if err == nil {
			for rows.Next() {
				var id string
				if rows.Scan(&id) == nil {
					topics = append(topics, "merchant:"+id)
				}
			}
			rows.Close()
		}
	}
	// الزبون: موضوعه الشخصي (تتبع طلباته حياً من الموقع/التطبيق)
	if slices.Contains(claims.Roles, "customer") {
		topics = append(topics, "customer:"+claims.Subject)
	}
	// موضوع السائق يُضاف مع تطبيقه
	if len(topics) == 0 {
		httpx.Error(w, errForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*", "*.rahalgo.com"},
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	ch, cancel := s.hub.Subscribe(topics)
	defer cancel()

	ctx := r.Context()
	// قارئ لاكتشاف الإغلاق من الطرف الآخر
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case <-ping.C:
			pingCtx, pcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			pcancel()
			if err != nil {
				return
			}
		case msg := <-ch:
			writeCtx, wcancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			wcancel()
			if err != nil {
				return
			}
		}
	}
}
