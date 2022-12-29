package rain

import (
	"sync/atomic"
	"time"
)

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
	// Progress 进度
	Progress int
	// Outpath 文件输出路径
	Outpath string
	// Error 下载错误信息
	Error error
}

// loadEvent 加载事件
func (ctl *control) loadEvent() {
	// 发送事件
	if len(ctl.eventExend) > 0 {
		ctl.addEvent(NewEventExtend(ctl.eventExend...))
		ctl.eventExend = make([]ProgressEventExtend, 0)
	}
	if len(ctl.event) > 0 {
		ctl.sendEvent = ctl.sendEventFunc()
	}
}

// addEvent 新增事件
func (ctl *control) addEvent(e ...ProgressEvent) {
	ctl.event = append(ctl.event, e...)
}

// addEvent 新增事件
func (ctl *control) addEventExtend(e ...ProgressEventExtend) {
	ctl.eventExend = append(ctl.eventExend, e...)
}

// autoSendEvent 自动发送事件
func (ctl *control) autoSendEvent() {
	for {
		select {
		case <-time.After(time.Millisecond * 200):
			ctl.sendEvent()
		case <-ctl.ctx.Done():
			return
		}
	}
}

// sendEventFunc 发送事件信息
func (ctl *control) sendEventFunc() func() {
	var (
		stat = &Stat{
			Status:          ctl.status,
			TotalLength:     ctl.totalSize,
			CompletedLength: atomic.LoadInt64(ctl.completedSize),
			Progress:        0,
			Outpath:         ctl.outpath,
			Error:           ctl.getError(),
		}
		nowCompletedLength = int64(0)
		remainingLength    = int64(0)
	)
	return func() {
		nowCompletedLength = atomic.LoadInt64(ctl.completedSize)
		remainingLength = ctl.totalSize - nowCompletedLength
		if remainingLength < 0 {
			remainingLength = 0
		}
		stat.Status = ctl.status
		stat.CompletedLength = nowCompletedLength
		if nowCompletedLength > 0 && ctl.totalSize > 0 {
			stat.Progress = int(float64(nowCompletedLength) / float64(ctl.totalSize) * float64(100))
		}
		stat.Error = ctl.getError()
		for _, e := range ctl.event {
			e.Change(stat)
		}
	}
}
