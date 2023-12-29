package k8s

import (
	"context"
	"fmt"
	"sync"

	"github.com/memory-overflow/k8s-service-lbagent/common/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var clientSet *kubernetes.Clientset
var kubeonce sync.Once

var registeredService *sync.Map
var mtx sync.Mutex

func RegisterMonitorService(name string, ch chan struct{}) {
	mtx.Lock()
	defer mtx.Unlock()

	if v, ok := registeredService.LoadOrStore(name, []chan struct{}{ch}); ok {
		chs := v.([]chan struct{})
		chs = append(chs, ch)
		registeredService.Store(name, chs)
	}
}

func GetKubeClient() *kubernetes.Clientset {
	if clientSet == nil {
		kubeonce.Do(func() {
			registeredService = &sync.Map{}
			// 创建 Kubernetes 客户端
			k8sconfig, err := clientcmd.BuildConfigFromFlags("", config.Get().KubeConfigFile)
			if err != nil {
				config.GetLogger().Sugar().Fatalf("k8s BuildConfigFromFlags error: %v", err)
			}
			client, err := kubernetes.NewForConfig(k8sconfig)
			if err != nil {
				config.GetLogger().Sugar().Fatalf("kubernetes.NewForConfig error: %v", err)
			}
			clientSet = client

			go func() {
				// 利用 informers 机制监视 endpoints 的变更
				// 创建 Service Informer
				factory := informers.NewSharedInformerFactory(clientSet, 0)
				// 创建 Endpoints Informer
				endpointsInformer := factory.Core().V1().Endpoints().Informer()

				// 设置事件处理器
				endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj interface{}) {
						// 处理添加事件
						if endpoint, ok := obj.(*corev1.Endpoints); ok {
							if v, ok := registeredService.Load(endpoint.Name); ok {
								config.GetLogger().Sugar().Infof("add endpoint.Name: %v", endpoint.Name)
								if chs, chok := v.([]chan struct{}); chok {
									for _, ch := range chs {
										ch <- struct{}{}
									}
								}
							}
						}
					},
					UpdateFunc: func(oldObj, newObj interface{}) {
						if endpoint, ok := newObj.(*corev1.Endpoints); ok {
							if v, ok := registeredService.Load(endpoint.Name); ok {
								config.GetLogger().Sugar().Infof("update endpoint.Name: %v", endpoint.Name)
								if chs, chok := v.([]chan struct{}); chok {
									for _, ch := range chs {
										ch <- struct{}{}
									}
								}
							}
						}
					},
					DeleteFunc: func(obj interface{}) {
						if endpoint, ok := obj.(*corev1.Endpoints); ok {
							if v, ok := registeredService.Load(endpoint.Name); ok {
								config.GetLogger().Sugar().Infof("delete endpoint.Name: %v", endpoint.Name)
								if chs, chok := v.([]chan struct{}); chok {
									for _, ch := range chs {
										ch <- struct{}{}
									}
								}
							}
						}
					},
				})

				// 启动 Informer
				stop := make(chan struct{})
				defer close(stop)
				endpointsInformer.Run(stop)
			}()

		})
	}

	return clientSet
}

func GetAvailableEndpoints(service, namespace string) ([]string, error) {
	client := GetKubeClient()
	endpoints, err := client.CoreV1().Endpoints(namespace).
		Get(context.Background(), service, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("k8s Endpoints error: %v", err)
	}
	ips := []string{}
	for _, endpoint := range endpoints.Subsets {
		for _, add := range endpoint.Addresses {
			ips = append(ips, add.IP)
		}
	}
	return ips, nil
}
