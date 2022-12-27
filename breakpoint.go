package rain

import (
	"encoding/json"
	"os"
)

// breakpoint 断点
type Breakpoint struct {
	// filesize 资源大小
	Filesize int64 `json:"filesize"`
	// etag 资源唯一标识
	Etag string `json:"etag"`
	// position 未分配任务的起始位置
	Position int64 `json:"position"`
	// tasks 已分配的未完成任务
	Tasks []*Block `json:"tasks"`
}

// loadBreakpoint 加载断点
func loadBreakpoint(path string) (*Breakpoint, error) {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bp Breakpoint
	err = json.Unmarshal(d, &bp)
	if err != nil {
		return nil, err
	}
	for _, block := range bp.Tasks {
		block.onstart = true
	}
	return &bp, nil
}

// addTask 添加任务
func (bp *Breakpoint) addTask(task *Block) {
	bp.Tasks = append(bp.Tasks, task)
}

// completedSize 已下载大小
func (bp *Breakpoint) completedSize() int64 {
	cl := bp.Position
	for _, v := range bp.Tasks {
		cl -= v.uncompletedSize()
	}
	return cl
}

// comparison 对比是否是相同资源
func (bp *Breakpoint) comparison(loadbp *Breakpoint) bool {
	if bp.Etag == loadbp.Etag && bp.Filesize == loadbp.Filesize {
		return true
	}
	return false
}

// export 导出
func (bp *Breakpoint) export(path string, perm os.FileMode) error {
	if len(bp.Tasks) < 1 {
		return nil
	}
	tasks := bp.Tasks
	tmp := &Breakpoint{
		Filesize: bp.Filesize,
		Etag:     bp.Etag,
		Position: tasks[len(tasks)-1].End + 1,
		Tasks:    make([]*Block, 0),
	}
	for _, v := range tasks {
		if v.isFinish() {
			continue
		}
		if !v.isStart() {
			continue
		}
		tmp.addTask(v)
	}
	d, err := json.Marshal(tmp)
	if err != nil {
		return nil
	}
	return os.WriteFile(path, d, perm)
}
