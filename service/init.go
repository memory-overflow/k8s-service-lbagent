package service

import (
	"context"
	"sync"
	"time"

	"github.com/memory-overflow/k8s-service-lbagent/common"
	"github.com/memory-overflow/k8s-service-lbagent/common/config"
	"github.com/memory-overflow/k8s-service-lbagent/k8s"
)

func (pxy *proxyService) init(ctx context.Context) {
	// 定时打印后端信息
	go func() {
		tick := time.NewTicker(10 * time.Second)
		for range tick.C {
			for _, r := range pxy.route {
				config.GetLogger().Sugar().Infof("current ips for url: %s", r.uri)
				r.lastConnections.Range(
					func(key, value interface{}) bool {
						config.GetLogger().Sugar().Infof("    %s: %d", key.(string), *value.(*int32))
						return true
					})
			}
		}
	}()

	// 通过 informar 机制监听 endpoints 的变化，动态更新 service ip 列表
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
	// 优化，不再轮询，通过 k8s informar 机制动态更新更优雅
	// tick := time.NewTicker(2 * time.Second)

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return
	// 	case <-tick.C:
	// 		wg := sync.WaitGroup{}
	// 		for _, service := range pxy.route {
	// 			wg.Add(1)
	// 			go func(service *k8sserviceInfo) {
	// 				defer wg.Done()
	// 				defer common.Recover()
	// 				syncIps(ctx, service)
	// 			}(service)
	// 		}
	// 		wg.Wait()
	// 	}
	// }
	k8s.GetKubeClient() // 先初始化 client

	for key := range pxy.route {
		service := pxy.route[key]
		// 初始化先把所有服务的 ip 都同步一遍
		syncIps(ctx, service)
		// 注册到全局的 k8s informar 监听中
		k8s.RegisterMonitorService(service.serviceName, service.endpointsNodifyCh)

		go func() {
			for range service.endpointsNodifyCh {
				// 监听通知，如果 service endpoints 出现变动，重新同步 ip
				syncIps(ctx, service)
			}
		}()
	}
}

func syncIps(ctx context.Context, service *k8sserviceInfo) {
	ips, err := k8s.GetAvailableEndpoints(service.serviceName, service.namespace)
	if err != nil {
		config.GetLogger().Sugar().Errorf("k8s.GetAvailableEndpoints error: %v", err)
		return
	}

	service.cond.L.Lock()
	defer service.cond.L.Unlock()

	lastIps := []string{}
	service.lastConnections.Range(
		func(key, value interface{}) bool {
			lastIps = append(lastIps, key.(string))
			return true
		})
	if !common.SliceSame(lastIps, ips) {
		// ip 发生变更
		var tot int32 = 0
		config.GetLogger().Sugar().Infof("new ips: %v for uri: %s", ips, service.uri)
		var newMap sync.Map
		for _, ip := range ips {
			if value, ok := service.lastConnections.Load(ip); ok {
				newMap.Store(ip, value)
				tot += *(value.(*int32))
			} else {
				x := int32(service.limitConnections)
				newMap.Store(ip, &x)
				tot += x
			}
		}
		service.totalConnections = int(tot)
		service.lastConnections = newMap
		service.cond.Broadcast()
	}

}
