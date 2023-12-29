package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/memory-overflow/k8s-service-lbagent/common/config"
)

type job struct {
	jobId string
	rw    http.ResponseWriter
	req   *http.Request
	done  chan struct{}
}

type k8sserviceInfo struct {
	uri         string
	serviceName string
	namespace   string
	httpPort    int

	limitConnections int

	// limitMap the concurrency limit for every uri
	lastConnections   sync.Map
	jobs              chan job
	endpointsNodifyCh chan struct{}
	cond              sync.Cond
	totalConnections  int
}

func (service *k8sserviceInfo) IncreaseConnection(ip string) {
	service.cond.L.Lock()
	defer service.cond.L.Unlock()
	if v, ok := service.lastConnections.Load(ip); ok {
		atomic.AddInt32(v.(*int32), 1)
		service.totalConnections++
		service.cond.Broadcast()
	}
}

func (service *k8sserviceInfo) getIp(ctx context.Context) (ip string, last *int32, err error) {
	timeout := time.NewTimer(2 * time.Hour)
	service.cond.L.Lock()
	defer service.cond.L.Unlock()
	waitCh := make(chan struct{})
	defer close(waitCh)
	// 等待条件，剩余连接数大于 0
	go func() {
		for err == nil && service.totalConnections == 0 {
			service.cond.Wait()
		}
		if err != nil {
			// 主线程可能因为超时退出了，需要解锁 cond.Wait 拿到的锁
			service.cond.L.Unlock()
			return
		}
		waitCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	case <-timeout.C:
		// 超时，一直没有空资源使用
		return "", nil, errors.New("no resources available")
	case <-waitCh:
		// 等到条件才 unlock 锁
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
			service.totalConnections--
			return ip, maxLast, nil
		} else {
			return "", nil, errors.New("no resources available")
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
			uri:               route.URI,
			serviceName:       route.ServiceName,
			namespace:         route.Namespace,
			httpPort:          route.HttpPort,
			limitConnections:  route.Limit,
			cond:              *sync.NewCond(&sync.Mutex{}),
			jobs:              make(chan job, 256),
			endpointsNodifyCh: make(chan struct{}, 256),
		}
		pxy.route[route.URI] = &serviceInfo
	}
	pxy.init(ctx)
	return pxy
}
