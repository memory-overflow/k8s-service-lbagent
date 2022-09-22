package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common"
	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
	"github.com/memory-overflow/highly-balanced-scheduling-agent/service"
)

func main() {
	// 异常处理
	defer common.Recover()
	ctx := context.Background()

	config.GetLogger().Sugar().Infof("agent serving on: %d", config.Get().Port)
	http.Handle("/", service.BuildProxy(ctx, config.Get().Routes))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Get().Port), nil); err != nil {
		config.GetLogger().Sugar().Errorf("anegt start error: %v", err)
	}
}
