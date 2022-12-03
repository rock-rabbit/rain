package rain

import (
	"io"
	"os"
)

// fileAt 实现指定写入文件位置数据的写入器
type fileAt struct {
	task *Block
	file *os.File
}

// fileWriteAt 指定位置写入数据
func fileWriteAt(file *os.File, task *Block) io.Writer {
	return &fileAt{
		task: task,
		file: file,
	}
}

// Write 文件写入
func (fa *fileAt) Write(p []byte) (n int, err error) {
	n, err = fa.file.WriteAt(p, fa.task.Start)
	// 保证记录断点时的数据为已存盘
	fa.file.Sync()
	fa.task.Start += int64(n)
	return
}
