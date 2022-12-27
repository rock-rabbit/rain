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
	ctl.setStatus(STATUS_RUNNING)

	// 加载下载块
	blocks := ctl.loadBlocks()
	if ctl.threadCount > len(blocks) {
		ctl.threadCount = len(blocks)
	}

	taskchan := make(chan *Block)
	done := make(chan error, ctl.threadCount)
	defer close(done)

	// 执行任务
	for i := 0; i < ctl.threadCount; i++ {
		go ctl.execute(taskchan, done)
	}

	// 发送事件
	if len(ctl.event) > 0 {
		go ctl.autoSendEvent()
	}

	// 分配任务
Allot:
	for _, block := range blocks {
		select {
		case taskchan <- block:
			block.start()
		case <-ctl.ctx.Done():
			break Allot
		}
	}

	// 分配完成
	close(taskchan)

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
	// 断点续传
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

// execute 执行任务
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

// download 执行下载任务
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
		ctl.outfile.Sync()
		ctl.breakpoint.export(ctl.bpfilepath, ctl.perm)
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
	if ctl.config.SpeedLimit > 0 && ctl.config.SpeedLimit < bufsize {
		bufsize = ctl.config.SpeedLimit
	}

	// 数据拷贝
	_, err = ctl.iocopy(dest, res.Body, bufsize)
	if err != nil {
		return err
	}
	return nil
}

// iocopy 拷贝数据
func (ctl *control) iocopy(dst io.Writer, src io.Reader, bufsize int) (written int64, err error) {
	// 创建 buffer 缓冲区
	dstbuf := bufio.NewWriterSize(dst, bufsize)
	defer dstbuf.Flush()
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return written, err
		}
		if ctl.rate != nil {
			ctl.rate.WaitN(ctl.ctx, n)
		}
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

// finish 完成任务
func (ctl *control) finish(err error) {
	// 手动 Close
	canceled := ctl.status == STATUS_CLOSE
	if canceled {
		err = nil
	}
	// 上下文超时
	if errors.Is(err, context.DeadlineExceeded) {
		err = fmt.Errorf("timeout: %w", err)
	}
	ctl.err = err
	ctl.close()
	// 断点文件
	if fileExist(ctl.bpfilepath) {
		if err == nil && !canceled {
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
