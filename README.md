# strict-load-balancing-k8s

基于 k8s 部署的微服务，可以通过 k8s 提供的 DNS 域名解析服务直接通过 k8s 内部域名访问。对于多 pod 的服务，k8s proxy 按照一定的规则进行转发。在大流量情况下，流量可以做到均衡，但是在一些低并发、单次请求处理时间长的计算型微服务场景下，k8s的调度不是很均衡，会导致资源使用率不高。本库设计了一种基于连接的高度均衡调度的微服务代理。

# k8s 微服务简介
## 微服务常用形式
如下图是一种 k8s 微服务最常用的构建形式。一个微服务对应一个 service、一个 deployment 以及若干个 pod 实例。

service 是掉用服务的入口，在同一集群里面的任意一个 pod 内部，通过 [svcname].[namespace]. svc.cluster.local 域名就可以访问微服务（宿主机不适用），service 通过 k8s 的负载均衡器转发到 pod。

deployment 是 pod 的管理工具，通过 ReplicaSet 维护 pod 数量和版本管理。

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/e54f5e9e0da946a18a1fcebc8a655dc2.webp)



# service 路由和负载均衡
## 路由
每个 service 都会绑定一个由集群生成的内网 ip。在宿主机上可以直接通过该 ip 访问服务。

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/d2f24143135e419faaddba9a4d4ba94b.webp)

那么 k8s 的域名是如何绑定到该 CLUSTER-IP 呢。答案就是 k8s 系统自带的 dns 服务器。
![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/c74080adf15647b8a52a550e374b401e.webp)



 进入到任意一个 pod 内部，执行 nslookup ai-media-backend.ai-media.svc.cluster.local，可以看到每个 pod 的 dns 服务器都是 kube-dns 的 CLUSTER-IP。而查到的域名 ai-media-backend.ai-media.svc.cluster.local 可转发的 ip 正好是 ai-media-backend 这个服务的 CLUSTER-IP。

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/5a813487241947bd930f6794c5094065.webp)

## 负载均衡
    kube-dns 只是解决了了域名的路由问题，而从 service 转发到 pod 还要解决负载均衡的问题。负载均衡是通过 kube-proxy 实现的，kube-proxy 是一种 DaemonSet，DaemonSet 是一种在每个 node 上正好有一个 pod 实例的组件。

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/dfd052a4fbae46b9a3d9118a3d5db74f.webp)

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/64dce4c13f82481982cbdea6a35172cb.webp)


kube-proxy 是每个 node 节点上所有 service 的代理，service ip 实际上是 vip(虚拟 ip 地址)，kube-proxy 用来维护所有 service ip 到 pod id 的映射关系，最常用的模式是通过 iptables 实现。kube-proxy 接受来自 api-server 的 pod 和 service 的变更信息，对 iptables 做相应的修改。基于 iptables 的 k8s 的整体路由如下图

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/8650b269d5ed4330bdffe7d5c39e07a5.webp)


iptables 模式下，负载均衡策略为随机选择 pod，并且连接失败也不会重试。



# 均衡负载问题
实际使用中， kube-proxy 的 iptables 模式的负载均衡效果不是很理想。尤其是对转码这种计算型服务，如果负载均衡不好，资源利用率会不高。比如后端有 2 个转码 pod，同一时间发送 4 个请求，而大多数情况下，这 4 个请求会转发到同一个 pod，导致一个 pod 很忙，另外一个 pod 却空闲着。kube-proxy 还有另外一种 ipvs 模式，在 ipvs 模式下，负载均衡可以选择[更多的策略](https://kubernetes.io/zh/docs/concepts/services-networking/service/#proxy-mode-ipvs)。但是 kube-proxy 是针对整个集群的，如果修改，对于所有业务都用同样的策略，这样会给其他业务和服务带来风险。所以需要一种可以针对特定的服务采用独立的负载均衡的方案。

## 方案1
对微服务进行改造，使用共享队列的方式存取任务。这种方式首先要对微服务进行改造，并且在微服务中引入共享队列，使得微服务耦合度变高，违背了“微”原则，所以不提倡这种方案。

## 方案2（推荐方案）
另外一种方案类似于 kube-proxy，在转码服务前面再做一层负载均衡 agent，这个 agent 不侵入 k8s 整个系统，本质是一个 pod，可以参考下图



可以看到，在 agent pod 里面，是直接掉用到转码的 pod，那么 agent pod 有两个必要工作。

1. 维护所有转码 pods 的 ip 列表。

2. 实现选取后端 pod ip 的负载均衡策略。

### 维护 pods ip 列表
如果不依赖 k8s 维护 pods ip 列表还是比较麻烦的，但是依赖 k8s 有一个简单的方法。k8s 有一种无头服务（Headless Services），无头服务不会分配 Cluster IP，但是在 k8s-dns 服务里面会有所有 pods id 的记录，所以可以通过 k8s-dns 服务获取到无头服务的 pods ip 列表。


![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/6e51437089e94a298cf077534359455a.webp)

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/6bee6d6f232a47ac8fe6bac06cba2ad2.webp)


考虑到 pod 有重启或者增删都会导致 pods ip 列表发生变化，所以我们定期轮询 kube-dns 获取到 pods id 列表更新。但是当pod重启的时候，dns 返回的 ip 列表不是很稳定。这里会重复查询5次，每次间隔 0.5s，5次结果都相同才信任。否则过10s后再轮询。

### 负载均衡策略
对于每个 pod ip，记录对应的连接数，请求进来的时候连接数+1，请求结束的时候连接数-1。每次新的请求进来，会选择连接数最少的 pod 转发。

为了实现单 pod 限流，对于每个 pod 可以配置 URL 转发规则以及对应的最大连接数，agend pod 内部有一个队列，如果所有 pod 连接都满了，那么队列中的请求等待资源释放。

该方案整体流程图如下
![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/86aa772a870e412f8272d1977ab6b9a7.webp)



### 代码
目前已经用go语言实现了该均衡调度方案，不过目前的实现方法仅支持同步任务，还不支持异步任务的均衡调度。

代码仓库 https://github.com/memory-overflow/strict-load-balancing-k8s。
