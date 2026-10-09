package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedis ينشئ عميل Redis ويتحقق من الاتصال.
func NewRedis(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	TuneRedis(opts)
	client := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return client, nil
}

// TuneRedis **مهلٌ قصيرةٌ لـRedis** (اختبارُ التحمّل ٢٠٢٦-١٠-٠٩) — **قِيس بتجميده خمسَ
// عشرةَ ثانية**: بالافتراض (ثلاثُ ثوانٍ للقراءة) هبطت المنصّةُ من ٢٣٠ نداءً في الثانية
// إلى ١٨ وصار الردُّ ستَّ ثوانٍ، **لأنّ كلَّ نداءٍ ينتظره.** وكلُّ ما يُحفظ فيه ذاكرةٌ
// أو عدّادٌ له بديلٌ عند غيابه (القاعدة) — **فالانتظارُ القصيرُ ثمّ المضيُّ أسلم.**
func TuneRedis(opts *redis.Options) {
	opts.DialTimeout = time.Second
	opts.ReadTimeout = 400 * time.Millisecond
	opts.WriteTimeout = 400 * time.Millisecond
	opts.PoolTimeout = time.Second
	opts.MaxRetries = 1
}
