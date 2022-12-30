package rain

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
)

// startTask 开始任务
func (ctl *control) startTask() {
	ctl.log("download start")
	ctl.log("outpath: ", ctl.outpath)

	// 任务块数量不会太多，提前生产出来
	blocks := ctl.loadBlocks()

	ctl.log("blocks:")
	if ctl.debug {
		for _, v := range blocks {
			ctl.logf("\t%#v", v)
		}
	}

	// 任务块数量比设置的 goroutine 数量少，使用任务块的数量
	if ctl.threadCount > len(blocks) {
		ctl.threadCount = len(blocks)
	}

	ctl.log("goroutine count: ", ctl.threadCount)

	ctl.setStatus(STATUS_RUNNING)
	// taskchan 负责任务的发送与接收
	taskchan := make(chan *Block)
	// done 负责接收 goroutine 错误
	done := make(chan error, ctl.threadCount)
	// 任务执行完毕，关闭 done channel
	defer close(done)

	// 启动固定数量的 goroutine 消费任务
	for i := 0; i < ctl.threadCount; i++ {
		go ctl.execute(taskchan, done)
	}

	// 有发送进度事件时，启动自动发送事件 goroutine
	if len(ctl.event) > 0 {
		go ctl.autoSendEvent()
	}

	// 将任务发送到 channel 传递给消费任务的 goroutine
	go func() {
	Allot:
		for _, block := range blocks {
			select {
			case taskchan <- block:
				block.start()
			case <-ctl.ctx.Done():
				break Allot
			}
		}

		// 生产端负责关闭 channel
		close(taskchan)
	}()

	// 等待任务完成
	var errs []error
	for i := 0; i < ctl.threadCount; i++ {
		err := <-done
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		ctl.finish(nil)
	} else {
		ctl.finish(errs[0])
	}
}

// loadBlocks 加载任务块
func (ctl *control) loadBlocks() []*Block {
	// ctl.breakpoint.Tasks 为 0 时为复用下载，已经存在下载任务
	if len(ctl.breakpoint.Tasks) != 0 {
		return ctl.breakpoint.Tasks
	}
	// 可以进行断点续传时，加载断点文件
	if ctl.breakpointResume {
		bp, err := loadBreakpoint(ctl.bpfilepath)
		if err == nil && ctl.breakpoint.comparison(bp) {
			ctl.breakpoint = bp
			atomic.AddInt64(ctl.completedSize, bp.completedSize())
		}
	}
	position := ctl.breakpoint.Position
	// 单协程任务分配
	if position < ctl.totalSize && (ctl.totalSize == 0 || !ctl.multithread || ctl.config.RoutineCount == 1) {
		ctl.breakpoint.addTask(newBlock(position, ctl.totalSize-1))
		return ctl.breakpoint.Tasks
	}
	// 多协程任务分配
	for position < ctl.totalSize {
		start := position
		end := position + ctl.config.RoutineSize
		if end > ctl.totalSize-1 {
			end = ctl.totalSize - 1
		}
		ctl.breakpoint.addTask(newBlock(start, end))
		position = end + 1
	}
	return ctl.breakpoint.Tasks
}

// execute 执行任务的单个 goroutine
// 不断地消费任务，直到没有任务或者出现错误
func (ctl *control) execute(taskchan chan *Block, done chan error) {
	for task := range taskchan {
		if contextDone(ctl.ctx) {
			break
		}
		err := ctl.download(task)
		if err != nil {
			done <- err
			return
		}
	}
	done <- nil
}

// download 执行下载任务的具体实现
func (ctl *control) download(task *Block) error {
	var (
		err  error
		res  *http.Response
		dest io.Writer
	)
	// 创建文件写入器
	dest = newWriteFunc(func(b []byte) (n int, err error) {
		n, err = ctl.outfile.WriteAt(b, task.Start)
		task.addStart(int64(n))
		// 需要断点续传时，保存断点文件，输出文件强制存盘
		if ctl.breakpointResume {
			ctl.outfile.Sync()
			ctl.breakpoint.export(ctl.bpfilepath, ctl.perm)
		}
		return
	})

	// 当一次性下载完整文件时
	if task.isAll(ctl.totalSize) {
		res, err = ctl.request.defaultDo()
	} else {
		res, err = ctl.request.rangeDo(task.Start, task.End)
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// buffer size
	bufsize := ctl.config.DiskCache
	tasksize := task.uncompletedSize()
	if int64(bufsize) > tasksize {
		bufsize = int(tasksize)
	}

	// 数据拷贝
	_, err = ctl.iocopy(dest, res.Body, bufsize)
	if err != nil {
		return err
	}
	return nil
}

// COPY_BUFFER_SIZE 接收数据的 buffer 大小
const COPY_BUFFER_SIZE = 1024 * 32

// iocopy 拷贝数据
func (ctl *control) iocopy(dst io.Writer, src io.Reader, bufsize int) (written int64, err error) {
	// 创建 buffer 缓冲区
	dstbuf := bufio.NewWriterSize(dst, bufsize)
	defer dstbuf.Flush()

	buf := make([]byte, COPY_BUFFER_SIZE)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return written, err
		}
		// 消费限速器
		ctl.rateWaitN(n)
		dstbuf.Write(buf[0:n])
		nw64 := int64(n)
		atomic.AddInt64(ctl.completedSize, nw64)
		written += nw64
		if err == io.EOF {
			break
		}
	}
	return written, err
}

// finish 执行下载结束后的善后工作
func (ctl *control) finish(err error) {
	// 手动 Close
	if ctl.isclose {
		err = nil
	}
	// 上下文超时
	if errors.Is(err, context.DeadlineExceeded) {
		err = fmt.Errorf("timeout: %w", err)
	}
	ctl.err = err
	ctl.cancel()
	// 断点文件
	if fileExist(ctl.bpfilepath) {
		if err == nil && !ctl.isclose {
			os.Remove(ctl.bpfilepath)
		} else {
			ctl.breakpoint.export(ctl.bpfilepath, ctl.perm)
		}
	}
	// 输出文件
	if ctl.outfile != nil {
		ctl.outfile.Close()
	}

	// 设置完成状态
	if ctl.err != nil {
		ctl.setStatus(STATUS_ERROR)
	} else if ctl.isclose {
		ctl.setStatus(STATUS_CLOSE)
	} else {
		ctl.setStatus(STATUS_FINISH)
	}

	// 发送完成信息
	if ctl.sendEvent != nil {
		ctl.sendEvent()
	}
	// 发送完成信息并关闭
	ctl.done <- err
	close(ctl.done)
}
