package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rock-rabbit/rain"
)

var (
	cookie    = ""
	referer   = "https://www.bilibili.com/"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 Edg/106.0.1370.42"
)

func main() {
	rain.SetTimeout(time.Second * 5)
	byteCookie, _ := os.ReadFile("./tmp/cookie.txt")
	cookie = string(byteCookie)

	i := 50
	for {
		fmt.Println("开始第", i, "次")
		data, err := getRecommendList(i)
		if err != nil {
			panic(err)
		}
		err = data.downloadPic("./tmp/downpic")
		if err != nil {
			panic(err)
		}
		i++
		// time.Sleep(time.Second * 3)
	}
}

type recommendList struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Item []struct {
			Bvid  string `json:"bvid"`
			Pic   string `json:"pic"`
			Title string `json:"title"`
			Owner struct {
				Name string `json:"name"`
			} `json:"owner"`
		} `json:"item"`
	} `json:"data"`
}

func (r *recommendList) downloadPic(outdir string) error {
	for idx, item := range r.Data.Item {
		name := item.Owner.Name + "_" + item.Bvid + "_" + item.Title + ".jpg"
		uri := item.Pic
		if uri == "" {
			continue
		}
		fmt.Println("下载", idx, ":", item.Bvid, uri)
		ctl := rain.New(
			uri,
			rain.WithOutdir(outdir),
			rain.WithOutname(name),
			rain.WithHeader(func(h http.Header) {
				h.Set("referer", referer)
				h.Set("cookie", cookie)
			}),
			rain.WithBar(),
		).Run()
		if ctl.Error() != nil {
			return ctl.Error()
		}
	}
	return nil
}

// 获得视频推荐列表
func getRecommendList(idx int) (*recommendList, error) {
	url := fmt.Sprintf("https://api.bilibili.com/x/web-interface/index/top/feed/rcmd?y_num=5&fressh_type=3&feed_version=V7&fresh_idx_1h=%d&fetch_row=1&fresh_idx=%d&brush=%d&homepage_ver=1&ps=10", idx, idx+2, idx)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("referer", referer)
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("cookie", cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var jsondata recommendList
	err = json.Unmarshal(data, &jsondata)
	if err != nil {
		return nil, err
	}
	return &jsondata, nil
}
