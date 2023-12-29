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

	"github.com/memory-overflow/k8s-service-lbagent/common"
	"github.com/memory-overflow/k8s-service-lbagent/common/config"
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
		config.GetLogger().Sugar().Errorf("no route for uri %s, request: %s, header: %s",
			uri, string(body), string(header))
		return
	}
	j := job{
		jobId: common.GenerateRandomString(8),
		rw:    rw,
		req:   req,
		done:  make(chan struct{}),
	}
	config.GetLogger().Sugar().Infof("[%s]before push job: %s", j.jobId, j.jobId)
	service.jobs <- j
	config.GetLogger().Sugar().Infof("[%s]pushed job: %s", j.jobId, j.jobId)
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
				config.GetLogger().Sugar().Errorf("[%s]uri %s, header: %s, request: %s handle error: %v",
					job.jobId, job.req.RequestURI, string(header), string(body), err)
				break
			}
			go func() {
				defer svc.IncreaseConnection(ip)
				config.GetLogger().Sugar().Infof("[%s]selected ip %s, last conn: %d", job.jobId, ip, *last)
				transport(fmt.Sprintf("http://%s:%d%s", ip, svc.httpPort, svc.uri), job.rw, job.req, job.jobId)
				job.done <- struct{}{}
			}()
		default:
			continue
		}
	}
}

func transport(target string, rw http.ResponseWriter, req *http.Request, jobId string) {
	header, _ := json.Marshal(req.Header.Clone())

	config.GetLogger().Sugar().Infof("[%s]before transport %s, header: %s", jobId, target, string(header))
	u, err := url.Parse(target)
	if err != nil {
		rw.WriteHeader(502)
		io.Copy(rw, strings.NewReader(err.Error()))
		config.GetLogger().Sugar().Errorf("[%s]transport url failed: %v", jobId, err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(rw, req)
	config.GetLogger().Sugar().Infof("[%s] success transport %s, header: %s", jobId, target, string(header))
}
