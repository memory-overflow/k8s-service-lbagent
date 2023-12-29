# k8s load balancing proxy fro service 

基于 k8s 部署的微服务，在多 pod 部署的时候，大流量情况下，流量可以做到均衡，但是在一些低并发、单次请求处理时间长的计算型微服务场景下，k8s的调度不是很均衡，会导致资源使用率不高。本库设计了一种基于连接的高度均衡调度的 k8s service 的代理。

## 代理原理
在需要负载均衡调度的 k8s service 前面再加上一层 agent，如下图

![图1](https://github.com/memory-overflow/strict-load-balancing-k8s/blob/master/images/6e51437089e94a298cf077534359455a.webp)


## Usage
### 修改配置
修改 conf/config.yaml 文件配置
```yaml
# 该配置文件仅在本地环境生效，在容器内会被configMap的内容覆盖
port: 8080 # 代理服务对外提供的 http service 的端口
log_file: ./logs/agent.log # 日志文件

# kube conifg 文件，需要在 agent 容器中挂载主机的 kube config，这样 agent 才有权限查询 k8s pod 的信息。
kube_config_file: /root/kube/config 


# 路由信息
routes:
  - uri: /VideoCodecService/VideoCodec # 需要转发的 uri
    limit: 2 # 单个 pod 的最大连接数限制
    service_name: yt-server-video-codec # name of service 
    namespace: ai-media # namespace of service 
    http_port: 20109 # 被代理的服务的端口(http 协议)
```

当然也可以通过 configmap 挂载配置文件到 agent 服务的 /usr/local/services/ai-media/conf/config.yaml 文件，这样不需要修改源码，可以直接用后文中提到的编译打包好的镜像。


## 打包镜像
### 使用打包好的镜像
镜像地址：jisuanke/k8s-service-lbagent:latest

### 手动打包
配置好 golang 环境后，修改 `scripts/build_docker.sh` 脚本中的 `dockername` 镜像地址，然后执行 `sh scripts/build_docker.sh` 完成镜像打包。


## 镜像部署
推荐使用 configmap 挂载配置。
1. 把 k8s-resource 下的资源 copy 到 k8s 环境机器上。
2. 按照自身需求修改 [`k8s-resource/configmap.yaml`](https://github.com/memory-overflow/k8s-service-lbagent/blob/master/k8s_resourse/configmap.yaml) 文件的一些配置，修改 namespace。
3. 修改 [`k8s-resource/deployment_agent.yaml`](https://github.com/memory-overflow/k8s-service-lbagent/blob/master/k8s_resourse/deployment_agent.yaml) 文件中的 namespace。
4. 修改 [`k8s-resource/service_agent.yaml`](https://github.com/memory-overflow/k8s-service-lbagent/blob/master/k8s_resourse/service_agent.yaml) 文件中的 namespace。
5. `kubectl apply -f k8s-resource` 部署服务。

## 调用
在其他 pod 内部，通过原服务 service 调度的地方，替换成 `http://lb-agent.[namespace].svc.cluster.local:8080`，比如原来调用是 `http://yt-server-video-codec.ai-media.svc.cluster.local:8080/VideoCodecService/VideoCodec`，现在替换成 `http://la-agent.ai-media.svc.cluster.local:8080/VideoCodecService/VideoCodec` 即可。
