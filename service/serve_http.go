package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

// ServeHTTP ...
func (pxy *proxyService) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	uri := req.RequestURI
	service, ok := pxy.route[uri]
	if !ok {
		rw.WriteHeader(404)
		io.Copy(rw, strings.NewReader(fmt.Sprintf("no route found %s", uri)))
		body, _ := ioutil.ReadAll(req.Body)
		header, _ := json.Marshal(req.Header.Clone())
		config.GetLogger().Sugar().Errorf("no route for request: %s, header: %s",
			string(body), string(header))
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
				body, _ := ioutil.ReadAll(job.req.Body)
				header, _ := json.Marshal(job.req.Header.Clone())
				config.GetLogger().Sugar().Errorf("header: %s, request: %s handle error: %v",
					string(header), string(body), err)
				break
			}
			go func() {
				defer atomic.AddInt32(last, 1) // 对应 ip 剩余连接 +1
				transport(fmt.Sprintf("http://%s:%d%s", ip, svc.k8sPort, svc.uri), job.rw, job.req)
				job.done <- struct{}{}
			}()
		default:
			continue
		}
	}
}

func transport(target string, rw http.ResponseWriter, req *http.Request) {
	header, _ := json.Marshal(req.Header.Clone())

	config.GetLogger().Sugar().Infof("before transport %s, header: %s", target, string(header))
	u, err := url.Parse(target)
	if err != nil {
		rw.WriteHeader(502)
		io.Copy(rw, strings.NewReader(err.Error()))
		config.GetLogger().Sugar().Errorf("transport url failed: %v", err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(rw, req)
	config.GetLogger().Sugar().Infof("success transport %s, header: %s", target, string(header))
}
