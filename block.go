package rain

// block 下载块
type Block struct {
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

// uncompletedSize 未下载的字节数
func (b *Block) uncompletedSize() int64 {
	return b.End - b.Start + 1
}

// isFinish 当前下载块是否完成
func (b *Block) isFinish() bool {
	return b.Start-1 == b.End
}

// isSing 当前下载块是否为全部的下载
func (b *Block) isAll(totalSize int64) bool {
	return b.Start == 0 && b.End == totalSize-1
}
