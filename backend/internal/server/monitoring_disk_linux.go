//go:build linux

package server

import "syscall"

// diskUsedPercent **كم في المئة امتلأ القرصُ الذي فيه هذا المجلّد** — لينكس.
//
// **والمحجوزُ للجذر يُعدّ ممتلئاً** — كما يقوله `df`: ما لا يقدر الخادمُ
// أن يكتب فيه ليس مساحةً له.
func diskUsedPercent(dir string) (float64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil || st.Blocks == 0 {
		return 0, false
	}
	total := float64(st.Blocks)
	free := float64(st.Bavail)
	return (total - free) / total * 100, true
}
