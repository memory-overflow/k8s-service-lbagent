package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

type job struct {
	rw   http.ResponseWriter
	req  *http.Request
	done chan struct{}
}

type k8sserviceInfo struct {
	uri              string
	k8sPort          int
	k8sHost          string
	limitConnections int
	// limitMap the concurrency limit for every uri
	lastConnections sync.Map
	jobs            chan job
}

func (service *k8sserviceInfo) getIp(ctx context.Context) (ip string, last *int32, err error) {
	timeout := time.NewTimer(2 * time.Hour)
	for {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-timeout.C:
			// 超时，一直没有空资源使用
			return "", nil, errors.New("no resources available")
		default:
			var maxLast *int32
			service.lastConnections.Range(
				func(key, value interface{}) bool {
					if maxLast == nil || *value.(*int32) > *maxLast {
						maxLast = value.(*int32)
						ip = key.(string)
					}
					return true
				})
			if maxLast != nil && *maxLast > 0 {
				atomic.AddInt32(maxLast, -1)
				return ip, maxLast, nil
			}
			time.Sleep(2 * time.Second)
		}
	}
}

// proxyService 代理服务
type proxyService struct {
	// route 路由, uri map to k8sserviceInfo
	route map[string]*k8sserviceInfo
}

// BuildProxy 构建新的服务
func BuildProxy(ctx context.Context, routes []config.Route) *proxyService {
	pxy := &proxyService{
		route: map[string]*k8sserviceInfo{},
	}
	for _, route := range routes {
		serviceInfo := k8sserviceInfo{
			uri:              route.URI,
			k8sHost:          route.K8sHost,
			k8sPort:          route.K8sPort,
			limitConnections: route.Limit,
			jobs:             make(chan job, 200),
		}
		pxy.route[route.URI] = &serviceInfo
	}
	pxy.init(ctx)
	return pxy
}
