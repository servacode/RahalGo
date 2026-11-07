package catalog

// مؤقّت: بديل SuggestSection لحتى يندمج الملف الحقيقي section_suggest.go —
// احذف هالملف لمّا يندمج (وإلّا البناء بيسقط بتعريفين).

import "strings"

// SuggestSection يقترح اسم قسم سوق من اسم الصنف — هون قائمة صغيرة للاختبار بس.
func SuggestSection(name string) (string, bool) {
	n := normName(name)
	for word, sec := range map[string]string{
		"بيبسي": "مشروبات باردة",
		"مياه":  "مياه",
		"شامبو": "عناية شخصيّة",
	} {
		if strings.Contains(n, word) {
			return sec, true
		}
	}
	return "", false
}
