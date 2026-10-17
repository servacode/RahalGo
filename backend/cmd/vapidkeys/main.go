// Command vapidkeys **يولّد زوجَ مفاتيح VAPID لإشعارات المتصفّح** — مرّةً واحدة.
//
//	go run ./cmd/vapidkeys
//
// **والناتجُ يُلصق في متغيّرات الخادم لا في المستودع** — المستودعُ عامّ.
// وتبديلُ المفتاح بعد النشر يُبطل كلَّ اشتراكٍ قائم، فيُحفظ ولا يُعاد توليدُه.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	priv, err := key.Bytes()
	if err != nil {
		log.Fatal(err)
	}
	pub, err := key.PublicKey.Bytes()
	if err != nil {
		log.Fatal(err)
	}
	enc := base64.RawURLEncoding
	fmt.Println("WEBPUSH_VAPID_PUBLIC_KEY=" + enc.EncodeToString(pub))
	fmt.Println("WEBPUSH_VAPID_PRIVATE_KEY=" + enc.EncodeToString(priv))
	fmt.Println("WEBPUSH_SUBJECT=mailto:you@example.com")
}
