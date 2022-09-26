package service

import (
	"context"
	"sync"
	"time"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common"
	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

func (pxy *proxyService) init(ctx context.Context) {
	// 定时打印后端信息
	go func() {
		tick := time.NewTicker(10 * time.Second)
		for range tick.C {
			for _, r := range pxy.route {
				config.GetLogger().Sugar().Infof("current ips for url: %s, loacked: %v", r.uri, r.locked)
				r.lastConnections.Range(
					func(key, value interface{}) bool {
						config.GetLogger().Sugar().Infof("    %s: %d", key.(string), *value.(*int32))
						return true
					})
			}
		}
	}()

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
	lastIps := []string{}
	service.lastConnections.Range(
		func(key, value interface{}) bool {
			lastIps = append(lastIps, key.(string))
			return true
		})
	if !common.SliceSame(lastIps, ips) {
		// dns 发生变化，服务有重启
		// 先锁住服务调度
		service.locked = true
		defer func() {
			service.locked = false
		}()
		time.Sleep(10 * time.Second) // 等待服务重启完成
		// 5次拉取到的都是同一个 ip 列表再继续，防止 pod 重启过程中dns ip 列表不稳定
		for i := 0; i < 4; i++ {
			time.Sleep(500 * time.Millisecond)
			dupips, err := common.GetDns(ctx, service.k8sHost)
			if err != nil {
				return
			}
			if !common.SliceSame(ips, dupips) {
				return
			}
		}
		config.GetLogger().Sugar().Infof("new ips: %v for uri: %s", ips, service.uri)
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

}
