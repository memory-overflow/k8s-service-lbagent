package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

// ServeHTTP ...
func (pxy *proxyService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	uri := req.RequestURI
	service, ok := pxy.route[uri]
	if !ok {
		rw.WriteHeader(404)
		io.Copy(rw, strings.NewReader(fmt.Sprintf("no route found %s", uri)))
		body, _ := json.Marshal(req)
		config.GetLogger().Sugar().Errorf("no route for request: %s", string(body))
		return
	}
	j := job{
		rw:   rw,
		req:  req,
		done: make(chan struct{}),
	}
	service.jobs <- j
	<-j.done // 等待完成处理
}

func handle(ctx context.Context, svc *k8sserviceInfo) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-svc.jobs:
			if !ok {
				break
			}
			ip, last, err := svc.getIp(ctx)
			if err != nil {
				job.rw.WriteHeader(502)
				io.Copy(job.rw, strings.NewReader(err.Error()))
				body, _ := json.Marshal(job.req)
				config.GetLogger().Sugar().Errorf("request %s handle error: %v", string(body), err)
				break
			}
			go func() {
				defer atomic.AddInt32(last, 1) // 对应 ip 剩余连接 +1
				forward(fmt.Sprintf("http://%s:%d%s", ip, svc.k8sPort, svc.uri), job.rw, job.req)
				job.done <- struct{}{}
			}()
		default:
			continue
		}
	}
}

func forward(url string, rw http.ResponseWriter, req *http.Request) {
	time.Sleep(5 * time.Second)
	rw.WriteHeader(200)
}
