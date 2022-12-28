package rain

import (
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"time"
)

// std 默认下载器
var std = NewRain()

// New 创建下载控制器
func New(uri string, opts ...OptionFunc) *RainControl {
	return std.New(uri, opts...)
}

// SetProxy 设置客户端代理
func SetProxy(p func(*http.Request) (*url.URL, error), h ...http.Header) error {
	return std.SetProxy(p, h...)
}

// AddOptions 添加 New 时的 option
func AddOptions(opt ...OptionFunc) {
	std.AddOptions(opt...)
}

// SetOptions 设置 New 时的 option
func SetOptions(opts []OptionFunc) {
	std.SetOptions(opts)
}

// SetRoutineSize 设置协程下载最大字节数
func SetRoutineSize(d int64) {
	std.SetRoutineSize(d)
}

// SetRoutineCount 设置协程最大数
func SetRoutineCount(d int) {
	std.SetRoutineCount(d)
}

// SetClient 设置默认请求客户端
func SetClient(d *http.Client) {
	std.SetClient(d)
}

// SetMethod 设置默认请求方式
func SetMethod(d string) {
	std.SetMethod(d)
}

// SetBody 设置默认请求 Body
func SetBody(d io.Reader) {
	std.SetBody(d)
}

// SetHeader 设置默认请求 Header
func SetHeader(k, v string) {
	std.SetHeader(k, v)
}

// ReplaceHeader 替换默认请求的 Header
func ReplaceHeader(h http.Header) {
	std.ReplaceHeader(h)
}

// SetPerm 设置默认输出文件权限
func SetPerm(d fs.FileMode) {
	std.SetPerm(d)
}

// SetOutdir 设置文件默认输出目录
func SetOutdir(d string) {
	std.SetOutdir(d)
}

// SetDiskCache 设置磁盘缓冲区大小
func SetDiskCache(d int) {
	std.SetDiskCache(d)
}

// SetSpeedLimit 设置下载速度限制
func SetSpeedLimit(d int) {
	std.SetSpeedLimit(d)
}

// SetCreateDir 设置是否可以创建目录
func SetCreateDir(d bool) {
	std.SetCreateDir(d)
}

// SetAllowOverwrite 设置是否允许覆盖下载文件
func SetAllowOverwrite(d bool) {
	std.SetAllowOverwrite(d)
}

// SetBreakpointResume 设置是否启用断点续传
func SetBreakpointResume(d bool) {
	std.SetBreakpointResume(d)
}

// SetAutoFileRenaming 设置是否自动重命名文件，新文件名在名称之后扩展名之前加上一个点和一个数字（1..9999）
func SetAutoFileRenaming(d bool) {
	std.SetAutoFileRenaming(d)
}

// SetTimeout 设置下载超时时间
func SetTimeout(d time.Duration) {
	std.SetTimeout(d)
}

// SetRetryNumber 设置请求重试次数
func SetRetryNumber(d int) {
	std.SetRetryNumber(d)
}

// SetRetryNumber 设置请求重试的间隔时间
func SetRetryTime(d time.Duration) {
	std.SetRetryTime(d)
}

// SetTempfileExt 断点文件扩展, 默认为 .temp.rain
func SetBreakpointExt(d string) {
	std.SetBreakpointExt(d)
}
