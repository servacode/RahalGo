package main

import (
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

func main() {
	h, err := auth.HashPassword("QaLive2026x")
	if err != nil {
		panic(err)
	}
	fmt.Println(h)
}
