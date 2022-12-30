package rain

import (
	"io"
	"io/fs"
	"net/http"
	"time"
)

// WithDebug 设置 debug 调试信息开关
func WithDebug(d bool) OptionFunc {
	return func(ctl *control) {
		ctl.setDebug(d)
	}
}

// WithOutdir 文件输出目录
func WithOutdir(outdir string) OptionFunc {
	return func(ctl *control) {
		ctl.outdir = outdir
	}
}

// WithOutname 文件输出名称
func WithOutname(outname string) OptionFunc {
	return func(ctl *control) {
		ctl.outname = outname
	}
}

// WithEvent 事件监听
func WithEvent(e ...ProgressEvent) OptionFunc {
	return func(ctl *control) {
		ctl.addEvent(e...)
	}
}

// WithEventExtend 事件监听
func WithEventExtend(e ...ProgressEventExtend) OptionFunc {
	return func(ctl *control) {
		ctl.addEventExtend(e...)
	}
}

// WithBar 进度条
func WithBar(bars ...*Bar) OptionFunc {
	return func(ctl *control) {
		var bar *Bar
		if len(bars) > 0 {
			bar = bars[0]
		} else {
			bar = NewBar()
		}
		ctl.addEventExtend(bar)
	}
}

// WithClient 设置默认请求客户端
func WithClient(d *http.Client) OptionFunc {
	return func(ctl *control) {
		ctl.request.client = d
	}
}

// WithMethod 设置默认请求方式
func WithMethod(d string) OptionFunc {
	return func(ctl *control) {
		ctl.request.method = d
	}
}

// WithBody 设置默认请求 Body
func WithBody(d io.Reader) OptionFunc {
	return func(ctl *control) {
		if d != nil {
			d = NewMultiReadable(d)
		}
		ctl.request.body = d
	}
}

// WithHeader 设置默认请求 Header
func WithHeader(d func(h http.Header)) OptionFunc {
	return func(ctl *control) {
		d(ctl.request.header)
	}
}

// WithPerm 设置默认输出文件权限
func WithPerm(d fs.FileMode) OptionFunc {
	return func(ctl *control) {
		ctl.perm = d
	}
}

// WithRoutineSize 设置协程下载最大字节数
func WithRoutineSize(d int64) OptionFunc {
	return func(ctl *control) {
		ctl.config.RoutineSize = d
	}
}

// WithRoutineCount 设置协程最大数
func WithRoutineCount(d int) OptionFunc {
	return func(ctl *control) {
		ctl.config.RoutineCount = d
	}
}

// WithDiskCache 设置磁盘缓冲区大小
func WithDiskCache(d int) OptionFunc {
	return func(ctl *control) {
		ctl.config.DiskCache = d
	}
}

// WithSpeedLimit 设置下载速度限制
func WithSpeedLimit(d int) OptionFunc {
	return func(ctl *control) {
		ctl.config.SpeedLimit = d
	}
}

// WithCreateDir 设置是否可以创建目录
func WithCreateDir(d bool) OptionFunc {
	return func(ctl *control) {
		ctl.config.CreateDir = d
	}
}

// WithAllowOverwrite 设置是否允许覆盖下载文件
func WithAllowOverwrite(d bool) OptionFunc {
	return func(ctl *control) {
		ctl.config.AllowOverwrite = d
	}
}

// WithBreakpointResume 设置是否启用断点续传
func WithBreakpointResume(d bool) OptionFunc {
	return func(ctl *control) {
		ctl.config.BreakpointResume = d
	}
}

// WithAutoFileRenaming 设置是否自动重命名文件，新文件名在名称之后扩展名之前加上一个点和一个数字（1..9999）
func WithAutoFileRenaming(d bool) OptionFunc {
	return func(ctl *control) {
		ctl.config.AutoFileRenaming = d
	}
}

// WithAutoFilterFilename 设置是否自动过滤掉文件名称中的非法字符
func WithAutoFilterFilename(d bool) OptionFunc {
	return func(ctl *control) {
		ctl.config.AutoFilterFilename = d
	}
}

// WithTimeout 设置下载超时时间
func WithTimeout(d time.Duration) OptionFunc {
	return func(ctl *control) {
		ctl.config.Timeout = d
	}
}

// WithRetryNumber 设置请求重试次数
func WithRetryNumber(d int) OptionFunc {
	return func(ctl *control) {
		ctl.config.RetryNumber = d
	}
}

// WithRetryNumber 设置请求重试的间隔时间
func WithRetryTime(d time.Duration) OptionFunc {
	return func(ctl *control) {
		ctl.config.RetryTime = d
	}
}

// WithBreakpointExt 断点文件扩展, 默认为 .temp.rain
func WithBreakpointExt(d string) OptionFunc {
	return func(ctl *control) {
		ctl.config.BreakpointExt = d
	}
}
