package rain

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// request 资源请求器
type request struct {
	// debug 调试开关
	debug bool
	// ctx 上下文
	ctx context.Context
	// uri 请求资源链接
	uri string
	// client http 客户端
	client *http.Client
	// method 请求方式，默认为 GET
	method string
	// body 请求时的 Body，默认为 nil
	body io.Reader
	// header 请求时的头部信息
	header http.Header
	// retryNumber 重试次数
	retryNumber int
	// retryTime 重试间隔时间
	retryTime time.Duration
}

// resourceInfo 资源信息
type resourceInfo struct {
	// uri 资源链接
	uri string
	// filesize 资源大小
	filesize int64
	// multithread 是否支持断点续传和多协程
	multithread bool
	// contentType 资源类型
	contentType string
	// contentDisposition 资源描述
	contentDisposition string
	// etag 资源唯一标识
	etag string
}

// getFilename 获取文件名
func (b *resourceInfo) getFilename() string {
	// 从附加信息中获取文件名
	name := getMimeFilename(b.contentDisposition)
	if name != "" {
		return name
	}
	// 从资源链接中获取文件名
	name = getUriFilename(b.uri)
	if name != "" {
		return name
	}
	// 随机生成名称
	return fmt.Sprintf("file_%s%d", randomString(5, 1), time.Now().UnixNano())
}

// getResourceInfo 获取资源的基础信息
func (r *request) getResourceInfo() (*resourceInfo, error) {
	res, err := r.rangeDo(0, 9)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	contentRange := res.Header.Get("content-range")
	contentLength := res.Header.Get("content-length")
	acceptRanges := res.Header.Get("accept-ranges")

	b := &resourceInfo{}
	b.etag = res.Header.Get("etag")
	b.contentType = res.Header.Get("content-type")
	b.contentDisposition = res.Header.Get("content-disposition")
	b.uri = r.uri

	// 获取文件总大小
	rangeList := strings.Split(contentRange, "/")
	if len(rangeList) > 1 {
		b.filesize, _ = strconv.ParseInt(rangeList[1], 10, 64)
	}

	// 是否可以使用多协程
	if acceptRanges != "" || strings.Contains(contentRange, "bytes") || contentLength == "10" {
		b.multithread = true
	} else {
		// 不支持多协程重新获取文件总大小
		if b.filesize == 0 {
			b.filesize, _ = strconv.ParseInt(contentLength, 10, 64)
		}
	}
	return b, nil
}

// rangeDo 根据参数发送带有 range 头信息的请求
func (r *request) rangeDo(start, end int64) (*http.Response, error) {
	req, err := r.request()
	if err != nil {
		return nil, err
	}
	req.Header.Set("range", fmt.Sprintf("bytes=%d-%d", start, end))
	res, err := r.do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// defaultDo 根据参数发送请求
func (r *request) defaultDo() (*http.Response, error) {
	req, err := r.request()
	if err != nil {
		return nil, err
	}
	res, err := r.do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// request 根据参数生产请求，拷贝 header 信息
func (r *request) request() (*http.Request, error) {
	// body reader 重复读
	if r.body != nil {
		v, ok := r.body.(*MultiReadable)
		if ok {
			v.Reset()
		}
	}
	req, err := http.NewRequestWithContext(r.ctx, r.method, r.uri, r.body)
	if err != nil {
		return nil, err
	}

	req.Header = r.header.Clone()

	return req, nil
}

// do 对于 client.Do 的包装，主要实现重试机制
func (r *request) do(rsequest *http.Request) (*http.Response, error) {
	var (
		res          *http.Response
		requestError error
		retryNum     = 0
	)

	r.logf("request header:")
	if r.debug {
		for key := range rsequest.Header {
			r.logf("\t%s: %s", key, rsequest.Header.Get(key))
		}
	}
	for ; ; retryNum++ {
		res, requestError = r.client.Do(rsequest)
		r.log("request: do retry num", retryNum)
		r.log("request: status code", res.StatusCode)
		if requestError == nil && res.StatusCode < 400 {
			break
		} else if retryNum+1 >= r.retryNumber {
			var err error
			if requestError != nil {
				err = requestError
			} else {
				err = fmt.Errorf("%s HTTP Status Code %d", r.uri, res.StatusCode)
			}
			return nil, err
		}
		time.Sleep(r.retryTime)
	}
	return res, nil
}

// log 打印调试信息
func (r *request) log(v ...interface{}) {
	if r.debug {
		log.Println(v...)
	}
}

// logf 打印调试信息
func (r *request) logf(format string, v ...any) {
	if r.debug {
		log.Printf(format, v...)
	}
}
