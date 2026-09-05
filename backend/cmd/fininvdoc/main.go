package main

import (
	"encoding/json"
	"os"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

func main() {
	b, _ := json.MarshalIndent(fininv.Snapshot(), "", "  ")
	_ = os.WriteFile("../docs/testing/system/FINANCIAL_INVARIANTS.json", append(b, '\n'), 0o644)
}
