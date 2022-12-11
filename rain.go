package rain

import (
	"context"
	"crypto/tls"
	"io"
	"io/fs"
	"net/http"
	"time"
)

// Rain 下载器
type Rain struct {
	// config 配置
	config *Config

	// client 默认 http 客户端
	client *http.Client
	// method 默认请求方式
	method string
	// body 默认请求时的 Body
	body io.Reader
	// header 默认请求时的头部信息
	header http.Header
	// perm 默认新建文件的权限
	perm fs.FileMode
	// outdir 默认输出目录
	outdir string
}

// RainControl 下载控制器
type RainControl struct {
	ctl *control
}

// OptionFunc 参数
type OptionFunc func(ctl *control)

// NewRain 创建一个新的下载器
func NewRain() *Rain {
	// 设置默认头
	header := http.Header{}
	header.Add("accept", "*/*")
	header.Add("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6,*;q=0.5")
	header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.81 Safari/537.36 Edg/104.0.1293.54")
	// 设置默认 client
	client := &http.Client{
		Transport: &http.Transport{
			// 应用来自环境变量的代理
			Proxy: http.ProxyFromEnvironment,
			// 要求服务器返回非压缩的内容，前提是没有发送 accept-encoding 来接管 transport 的自动处理
			DisableCompression: true,
			// 接受服务器提供的任何证书
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// 每个服务器最大保留闲置连接数
			MaxIdleConnsPerHost: 10,
		},
		// 超时时间
		Timeout: 0,
	}
	return &Rain{
		config: NewConfig(),
		client: client,
		method: http.MethodGet,
		body:   nil,
		header: header,
		perm:   0600,
		outdir: "./",
	}
}

// New 创建下载控制器
func (rain *Rain) New(uri string, opts ...OptionFunc) *RainControl {
	// 拷贝 header 数据
	header := http.Header{}
	for k, v := range rain.header {
		header[k] = v
	}
	ctl := &control{
		uri:    uri,
		config: rain.config.Copy(),
		request: &request{
			uri:    uri,
			client: rain.client,
			method: rain.method,
			body:   rain.body,
			header: header,
		},
		perm:   rain.perm,
		outdir: rain.outdir,
	}

	for _, opt := range opts {
		opt(ctl)
	}

	return &RainControl{ctl: ctl}
}

// SetClient 设置默认请求客户端
func (rain *Rain) SetClient(d *http.Client) {
	rain.client = d
}

// SetMethod 设置默认请求方式
func (rain *Rain) SetMethod(d string) {
	rain.method = d
}

// SetBody 设置默认请求 Body
func (rain *Rain) SetBody(d io.Reader) {
	rain.body = d
}

// SetHeader 设置默认请求 Header
func (rain *Rain) SetHeader(k, v string) {
	rain.header.Set(k, v)
}

// ReplaceHeader 替换默认请求的 Header
func (rain *Rain) ReplaceHeader(h http.Header) {
	rain.header = h
}

// SetPerm 设置默认输出文件权限
func (rain *Rain) SetPerm(d fs.FileMode) {
	rain.perm = d
}

// SetOutdir 设置文件默认输出目录
func (rain *Rain) SetOutdir(d string) {
	rain.outdir = d
}

// SetRoutineSize 设置协程下载最大字节数
func (rain *Rain) SetRoutineSize(d int64) {
	rain.config.RoutineSize = d
}

// SetRoutineCount 设置协程最大数
func (rain *Rain) SetRoutineCount(d int) {
	rain.config.RoutineCount = d
}

// SetDiskCache 设置磁盘缓冲区大小
func (rain *Rain) SetDiskCache(d int) {
	rain.config.DiskCache = d
}

// SetSpeedLimit 设置下载速度限制
func (rain *Rain) SetSpeedLimit(d int) {
	rain.config.SpeedLimit = d
}

// SetCreateDir 设置是否可以创建目录
func (rain *Rain) SetCreateDir(d bool) {
	rain.config.CreateDir = d
}

// SetAllowOverwrite 设置是否允许覆盖下载文件
func (rain *Rain) SetAllowOverwrite(d bool) {
	rain.config.AllowOverwrite = d
}

// SetBreakpointResume 设置是否启用断点续传
func (rain *Rain) SetBreakpointResume(d bool) {
	rain.config.BreakpointResume = d
}

// SetAutoFileRenaming 设置是否自动重命名文件，新文件名在名称之后扩展名之前加上一个点和一个数字（1..9999）
func (rain *Rain) SetAutoFileRenaming(d bool) {
	rain.config.AutoFileRenaming = d
}

// SetAutoFilterFilename 设置是否自动过滤掉文件名称中的非法字符
func (rain *Rain) SetAutoFilterFilename(d bool) {
	rain.config.AutoFilterFilename = d
}

// SetTimeout 设置下载超时时间
func (rain *Rain) SetTimeout(d time.Duration) {
	rain.config.Timeout = d
}

// SetRetryNumber 设置请求重试次数
func (rain *Rain) SetRetryNumber(d int) {
	rain.config.RetryNumber = d
}

// SetRetryNumber 设置请求重试的间隔时间
func (rain *Rain) SetRetryTime(d time.Duration) {
	rain.config.RetryTime = d
}

// SetBreakpointExt 断点文件扩展, 默认为 .temp.rain
func (rain *Rain) SetBreakpointExt(d string) {
	rain.config.BreakpointExt = d
}

// Run 阻塞运行下载
func (rc *RainControl) Run() <-chan error {
	return rc.RunContext(context.Background())
}

// RunContext 基于 Context 阻塞运行下载
func (rc *RainControl) RunContext(ctx context.Context) <-chan error {
	ch := make(chan error, 1)
	defer close(ch)
	err := rc.StartContext(ctx)
	if err != nil {
		ch <- err
		return ch
	}
	ch <- <-rc.Wait()
	return ch
}

// Start 非阻塞运行下载
func (rc *RainControl) Start() error {
	return rc.StartContext(context.Background())
}

// StartContext 基于 Context 非阻塞运行下载
func (rc *RainControl) StartContext(ctx context.Context) error {
	return rc.ctl.start(ctx)
}

// Wait 阻塞通知
func (rc *RainControl) Wait() <-chan error {
	return rc.ctl.wait()
}

// Close 关闭下载
func (rc *RainControl) Close() {
	rc.ctl.close()
}

// Error 获取错误
func (rc *RainControl) Error() error {
	return rc.ctl.getError()
}

// Outpath 获取输出位置
func (rc *RainControl) Outpath() string {
	return rc.ctl.outpath()
}
