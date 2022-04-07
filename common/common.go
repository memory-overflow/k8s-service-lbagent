package common

import (
	"runtime/debug"
	"sort"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

// Recover ...
func Recover() {
	if p := recover(); p != nil {
		config.GetLogger().Sugar().Fatalf("panic: %v, stacktrace: %s", p, debug.Stack())
	}
}

// SliceSame 对于两个列表的值是否一样
func SliceSame(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
