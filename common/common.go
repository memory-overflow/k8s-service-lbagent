package common

import (
	"math/rand"
	"runtime/debug"
	"sort"
	"time"

	"github.com/memory-overflow/k8s-service-lbagent/common/config"
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

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UTC().UnixNano())
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return string(result)
}
