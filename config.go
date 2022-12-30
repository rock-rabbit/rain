package rain

import (
	"time"
)

// Config 配置
type Config struct {
	// RoutineCount 多协程下载时最多同时下载一个文件的最大协程，默认为 1
	RoutineCount int
	// RoutineSize 多协程下载时每个协程下载的大小，默认为 10M
	RoutineSize int64
	// diskCache 磁盘缓冲区大小，默认为 1M
	DiskCache int
	// speedLimit 下载速度限制，默认为 0 无限制
	SpeedLimit int
	// createDir 当需要创建目录时，是否创建目录，默认为 true
	CreateDir bool
	// allowOverwrite 是否允许覆盖文件，默认为 false
	AllowOverwrite bool
	// autoFileRenaming 文件自动重命名，新文件名在名称之后扩展名之前加上一个点和一个数字（1..9999）。默认:true
	AutoFileRenaming bool
	// AutoFilterFilename 自动过滤掉文件名称中的非法字符
	AutoFilterFilename bool
	// breakpointResume 是否启用断点续传，默认为 true
	BreakpointResume bool
	// timeout 下载总超时时间，默认为 10 分钟
	Timeout time.Duration
	// retryNumber 最多重试次数，默认为 5
	RetryNumber int
	// retryTime 重试时的间隔时间，默认为 0
	RetryTime time.Duration
	// BreakpointExt 断点文件扩展, 默认为 .temp.rain
	BreakpointExt string
}

// NewConfig 创建默认配置
func NewConfig() *Config {
	return &Config{
		RoutineCount:       1,
		RoutineSize:        1048576 * 10,
		DiskCache:          1048576 * 1,
		SpeedLimit:         0,
		CreateDir:          true,
		AllowOverwrite:     false,
		BreakpointResume:   true,
		AutoFileRenaming:   true,
		AutoFilterFilename: true,
		Timeout:            time.Minute * 10,
		RetryNumber:        5,
		RetryTime:          0,
		BreakpointExt:      ".temp.rain",
	}
}

// Clone 拷贝数据
func (cfg *Config) Clone() *Config {
	tmp := *cfg
	return &tmp
}
