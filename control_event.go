package rain

import (
	"sync/atomic"
	"time"
)

// addEvent 新增事件
func (ctl *control) addEvent(e ...ProgressEvent) {
	ctl.event = append(ctl.event, e...)
}

// autoSendEvent 自动发送事件信息
func (ctl *control) autoSendEvent() {
	ctl.sendEvent = ctl.sendEventFunc()
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
			DownloadSpeed:   0,
			EstimatedTime:   0,
			Progress:        0,
			Outpath:         ctl.outpath,
			Error:           ctl.getError(),
		}
		// ratio                 = 1 / 0.2
		nowCompletedLength    = int64(0)
		differCompletedLength = int64(0)
		remainingLength       = int64(0)
		// downloadRecord 记录五次下载的字节数，用于计算下载速度
		downloadRecord = make([]int64, 0, 5)
		// downloadRecordFunc 记录下载字节数并返回下载速度
		downloadRecordFunc = func(n int64) int64 {
			if len(downloadRecord) >= 5 {
				downloadRecord = downloadRecord[1:5:5]
			}

			downloadRecord = append(downloadRecord, n)

			var speed int64
			for _, num := range downloadRecord {
				speed += num
			}
			return speed
		}
	)

	return func() {
		// 计算
		nowCompletedLength = atomic.LoadInt64(ctl.completedSize)
		differCompletedLength = nowCompletedLength - stat.CompletedLength
		remainingLength = ctl.totalSize - nowCompletedLength
		if remainingLength < 0 {
			remainingLength = 0
		}

		stat.Status = ctl.status
		stat.CompletedLength = nowCompletedLength
		stat.DownloadSpeed = downloadRecordFunc(differCompletedLength)
		if remainingLength > 0 && stat.DownloadSpeed > 0 {
			stat.EstimatedTime = time.Duration((remainingLength / stat.DownloadSpeed) * int64(time.Second))
		}
		if nowCompletedLength > 0 && ctl.totalSize > 0 {
			stat.Progress = int(float64(nowCompletedLength) / float64(ctl.totalSize) * float64(100))
		}
		stat.Error = ctl.getError()
		for _, e := range ctl.event {
			e.Change(stat)
		}
	}
}
