package roadmatch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Q-R8 — **مسافة وحدة، تطبيق واحد** (C5): كل كيلومتر وكل خط بالمنصّة من `drivermonitor.Measure`/`Analyze`.
//
//   - `driver_road_chunks` و`driver_road_progress` بيكتبهن `roadmatch` وحده؛ و`drivermonitor` بيقرأ القطع بس.
//   - `driver_trail` بس بـ`drivermonitor` (الكتابة والقراءة) و`maplearn` و`roadmatch` (قراءة للمطابقة).
//
// فأي مكان تاني بدّه مسافة سائق بيمرق من `drivermonitor.Measure` — ما بيحسبها لحاله.
func TestDistance_OneImplementation(t *testing.T) {
	sqlRef := regexp.MustCompile(`(?i)\b(from|join|into|update|delete\s+from)\s+(driver_trail|driver_road_chunks|driver_road_progress)\b`)
	writeRef := regexp.MustCompile(`(?i)\b(into|update|delete\s+from)\s+(driver_road_chunks|driver_road_progress)\b`)
	allowed := map[string][]string{
		"driver_trail":         {"internal/drivermonitor/", "internal/maplearn/", "internal/roadmatch/"},
		"driver_road_chunks":   {"internal/roadmatch/", "internal/drivermonitor/"},
		"driver_road_progress": {"internal/roadmatch/"},
	}
	root := filepath.Join("..", "..")
	seen := 0
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			for _, m := range sqlRef.FindAllStringSubmatch(string(src), -1) {
				seen++
				table := strings.ToLower(m[2])
				ok := false
				for _, p := range allowed[table] {
					if strings.HasPrefix(rel, p) {
						ok = true
					}
				}
				if !ok {
					t.Errorf("%s: «%s» — %s ممنوع هون؛ المسافة من drivermonitor.Measure", rel, m[0], table)
				}
			}
			if strings.HasPrefix(rel, "internal/drivermonitor/") {
				for _, m := range writeRef.FindAllString(string(src), -1) {
					t.Errorf("%s: «%s» — drivermonitor بيقرأ قطع الطريق بس، الكتابة لـroadmatch", rel, m)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if seen < 5 {
		t.Fatalf("الحارس ما لقى مراجع SQL (%d) — المسار غلط؟", seen)
	}
}
