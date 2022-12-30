package rain

// Status 运行状态
type Status int

const (
	STATUS_NOTSTART = Status(iota - 1)
	// STATUS_BEGIN 准备中
	STATUS_BEGIN
	// STATUS_RUNNING 运行中
	STATUS_RUNNING
	// STATUS_CLOSE 关闭
	STATUS_CLOSE
	// STATUS_ERROR 错误
	STATUS_ERROR
	// STATUS_FINISH 完成
	STATUS_FINISH
)

// Is 列表中是否有相同值
func (s Status) Is(ss ...Status) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}
