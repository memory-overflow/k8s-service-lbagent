package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var failedCount, successCount int = 0, 0

func callTransCodingService(requestBody map[string]interface{}) (needTrans bool, err error) {
	codecURL := "http://ip:8080/VideoCodecService/VideoCodec"
	data, _ := json.Marshal(requestBody)
	maxTry := 5
	for i := 0; i < maxTry; i++ {
		err = nil
		httpClient := http.Client{} // 每次新创建 http client, 防止连接复用，所有请求全部打到一个转码 pod
		resp, e := httpClient.Post(codecURL, "application/json", strings.NewReader(string(data)))
		if e == nil && resp.StatusCode == 200 {
			bys, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			rspData := gjson.ParseBytes(bys)
			code := int(rspData.Get("errorcode").Int())
			if code == 0 {
				needTrans = true
				if f, e := os.Stat(requestBody["input_video_path"].(string)); e != nil {
					err = errors.New("没有生成转码视频: " + e.Error())
				} else {
					if f.Size() == 0 {
						err = errors.New("没有生成转码大小为 0, filepath: " + requestBody["input_video_path"].(string))
					} else {
						break
					}
				}
			}
			if code == 4352 {
				needTrans = false
				break
			}
			err = errors.New(rspData.String())
		} else {
			if e != nil {
				err = e
			} else {
				err = errors.New(resp.Status)
			}
		}
	}
	return needTrans, err
}

func doTest(index int, inputvideo string) {
	fmt.Printf("start request %d\n", index)
	requestBody := map[string]interface{}{
		"request_id":        "test",
		"app_id":            "test",
		"out_max_size":      1280,
		"input_video_path":  inputvideo,
		"output_video_path": fmt.Sprintf("/data/ti-platform-fs/ai-media/test/%d.mp4", index),
	}
	for {
		if _, err := callTransCodingService(requestBody); err != nil {
			fmt.Printf("request %d failed: %v\n", index, err)
			failedCount++
		} else {
			fmt.Printf("request %d success\n", index)
			successCount++
		}
	}
}

func main() {
	videos := []string{}
	go func() {
		t := time.NewTicker(1 * time.Minute)
		for range t.C {
			fmt.Printf("successCount %d, failedCount: %d\n", successCount, failedCount)
		}
	}()

	for i, vid := range videos {
		go doTest(i, vid)
	}
	for {

	}
}
