package service

import (
	"context"
	"sync"
	"time"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common"
)

func (pxy *proxyService) init(ctx context.Context) {
	// 开启动态同步服务 ip 列表的后台线程
	go func() {
		defer common.Recover()
		pxy.syncRoutes(ctx)
	}()

	// 对于每个 uri 开启一个调度队列
	for _, service := range pxy.route {
		go handle(ctx, service)
	}
}

func (pxy *proxyService) syncRoutes(ctx context.Context) {
	tick := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			wg := sync.WaitGroup{}
			for _, service := range pxy.route {
				wg.Add(1)
				go func(service *k8sserviceInfo) {
					defer wg.Done()
					defer common.Recover()
					syncIps(ctx, service)
				}(service)
			}
			wg.Wait()
		}
	}

}

func syncIps(ctx context.Context, service *k8sserviceInfo) {
	ips, err := common.GetDns(ctx, service.k8sHost)
	if err != nil {
		return
	}
	var newMap sync.Map
	for _, ip := range ips {
		if value, ok := service.lastConnections.Load(ip); ok {
			newMap.Store(ip, value)
		} else {
			x := int32(service.limitConnections)
			newMap.Store(ip, &x)
		}
	}
	service.lastConnections = newMap
}
