package rain

import "time"

// ProgressEvent 进度事件
type ProgressEvent interface {
	Change(stat *Stat)
}

// Stat 下载进行中的信息
type Stat struct {
	// Status 状态
	Status Status
	// TotalLength 文件总大小
	TotalLength int64
	// CompletedLength 已下载的文件大小
	CompletedLength int64
	// DownloadSpeed 每秒下载字节数
	DownloadSpeed int64
	// EstimatedTime 预计下载完成还需要的时间
	EstimatedTime time.Duration
	// Progress 下载进度, 长度为 100
	Progress int
	// Outpath 文件输出路径
	Outpath string
	// Error 下载错误信息
	Error error
}
