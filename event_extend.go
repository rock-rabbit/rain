package rain

import (
	"time"
)

// ProgressEventExtend 进度事件扩展
type ProgressEventExtend interface {
	// Change 进度变化
	Change(stat *EventExtend)
	// Close 中途暂停
	Close(stat *EventExtend)
	// Error 出现错误
	Error(stat *EventExtend)
	// Finish 下载成功
	Finish(stat *EventExtend)
}

type EventExtend struct {
	// Stat 信息
	*Stat
	// DownloadSpeed 每秒下载字节数
	DownloadSpeed int64
	// EstimatedTime 预计下载完成还需要的时间
	EstimatedTime time.Duration
	// events 事件列表
	events []ProgressEventExtend
	// record 记录下载速度
	record []int64
	// oldCompletedLength 记录上次进度
	oldCompletedLength int64
}

var _ ProgressEvent = &EventExtend{}

func NewEventExtend(e ...ProgressEventExtend) ProgressEvent {
	return &EventExtend{
		events: e,
		record: make([]int64, 0, 2),
	}
}

// AddEvent 新增事件
func (se *EventExtend) AddEvent(e ...ProgressEventExtend) {
	se.events = append(se.events, e...)
}

// sendEvent 发送事件
func (se *EventExtend) sendEvent(s string) {
	for _, event := range se.events {
		switch s {
		case "change":
			event.Change(se)
		case "close":
			event.Close(se)
		case "error":
			event.Error(se)
		case "finish":
			event.Finish(se)
		}
	}
}

// getRecord 获取每秒下载速度
func (se *EventExtend) getRecord(n int64) int64 {
	if len(se.record) >= 5 {
		se.record = se.record[1:5:5]
	}
	se.record = append(se.record, n)
	var speed int64
	for _, num := range se.record {
		speed += num
	}
	return speed
}

// Change 检查更新
func (se *EventExtend) Change(stat *Stat) {
	if se.Stat != stat {
		se.Stat = stat
		se.oldCompletedLength = se.CompletedLength
	}
	differCompletedLength := stat.CompletedLength - se.oldCompletedLength
	remainingLength := se.Stat.TotalLength - stat.CompletedLength
	se.DownloadSpeed = se.getRecord(differCompletedLength)
	if remainingLength > 0 && se.DownloadSpeed > 0 {
		se.EstimatedTime = time.Duration((remainingLength / se.DownloadSpeed) * int64(time.Second))
	}
	se.oldCompletedLength = stat.CompletedLength

	if se.Status.Is(STATUS_ERROR) {
		se.sendEvent("error")
		return
	}

	if se.Status.Is(STATUS_CLOSE) {
		se.sendEvent("close")
		return
	}

	if se.Status.Is(STATUS_FINISH) {
		se.sendEvent("finish")
		return
	}

	se.sendEvent("change")
}
