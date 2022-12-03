package rain

import (
	"bytes"
	"encoding/json"
	"io"
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

// breakpointParse 解析断点
func breakpointParse(r io.Reader) (*Breakpoint, error) {
	d, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var data Breakpoint
	err = json.Unmarshal(d, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// addTask 添加任务
func (bp *Breakpoint) addTask(task *Block) {
	bp.Tasks = append(bp.Tasks, task)
}

// comparison 对比是否是相同资源
func (bp *Breakpoint) comparison(loadbp *Breakpoint) bool {
	if bp.Etag == loadbp.Etag && bp.Filesize == loadbp.Filesize {
		return true
	}
	return false
}

// export 导出
func (bp *Breakpoint) export(w io.Writer) error {
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
		tmp.addTask(v)
	}
	d, err := json.Marshal(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(d))
	if err != nil {
		return err
	}
	return nil
}
