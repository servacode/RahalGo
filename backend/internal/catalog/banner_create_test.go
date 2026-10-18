package catalog

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestCreateBanner_NoPlacementIsHome **إنشاءُ لافتةٍ بلا موضعٍ أو بموضعٍ قديمٍ لا يسقط** — كان
// الافتراضُ `shop` والقيدُ منذ 0115 لا يقبل إلّا `home`.
func TestCreateBanner_NoPlacementIsHome(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := NewService(pool, identity.NewService(identity.NewRepo(pool), nil, nil, nil, "test-secret", slog.New(slog.NewTextHandler(io.Discard, nil))))
	actor := testdb.NewUser(t, pool, "admin")
	shop := "shop"
	for _, in := range []BannerInput{{}, {Placement: &shop}} {
		b, err := svc.CreateBanner(ctx, actor, in, "127.0.0.1")
		if err != nil {
			t.Fatalf("إنشاءُ اللافتة سقط: %v", err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM banners WHERE id = $1`, b.ID) })
	}
}
