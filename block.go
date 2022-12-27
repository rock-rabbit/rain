package rain

// block 下载块
type Block struct {
	onstart bool

	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// newBlock 创建下载块
func newBlock(start int64, end int64) *Block {
	return &Block{
		Start: start,
		End:   end,
	}
}

// start 下载块开始执行
func (b *Block) start() {
	b.onstart = true
}

// addStart 新增下载进度
func (b *Block) addStart(n int64) {
	b.Start += n
}

// uncompletedSize 未下载的字节数
func (b *Block) uncompletedSize() int64 {
	return b.End - b.Start + 1
}

// isFinish 当前下载块是否完成
func (b *Block) isFinish() bool {
	return b.Start-1 == b.End
}

// isStart 是否开始
func (b *Block) isStart() bool {
	return b.onstart
}

// isSing 当前下载块是否为全部的下载
func (b *Block) isAll(totalSize int64) bool {
	return b.Start == 0 && b.End == totalSize-1
}
