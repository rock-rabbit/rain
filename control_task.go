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

// addTask 新增任务
func (ctl *control) addTask(taskchan chan *Block, t *Block) bool {
	select {
	case taskchan <- t:
		ctl.breakpoint.addTask(t)
	case <-ctl.ctx.Done():
		return false
	}
	return true
}

// startTask 开始任务
func (ctl *control) startTask() {
	taskchan := make(chan *Block)
	done := make(chan error, ctl.threadCount)
	defer close(done)

	ctl.setStatus(STATUS_RUNNING)

	// 发送事件信息
	if len(ctl.event) > 0 {
		go ctl.autoSendEvent()
	}
	// 生产任务
	go ctl.production(taskchan)
	// 消费任务
	for i := 0; i < ctl.threadCount; i++ {
		go ctl.execute(taskchan, done)
	}
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

// recover 恢复断点
func (ctl *control) recover(taskchan chan *Block) int64 {
	bp, err := breakpointParse(ctl.bpfile)
	// 读取断点
	if err == nil && ctl.breakpoint.comparison(bp) {
		cl := bp.Position
		for _, v := range bp.Tasks {
			ctl.breakpoint.addTask(v)
			cl -= v.uncompletedSize()
		}
		atomic.AddInt64(ctl.completedSize, cl)
		// 发送断点任务
		for _, v := range bp.Tasks {
			select {
			case taskchan <- v:
			case <-ctl.ctx.Done():
				return bp.Position
			}
		}
		return bp.Position
	}
	return 0
}

// production 生产任务
func (ctl *control) production(taskchan chan *Block) {
	// 生产完毕，关闭通达
	defer close(taskchan)

	var (
		position int64
		err      error
		ok       bool
	)
	// 断点续传残留任务
	if ctl.breakpointResume {
		bpfileExist := fileExist(ctl.bpfilepath)
		ctl.bpfile, err = os.OpenFile(ctl.bpfilepath, os.O_CREATE|os.O_RDWR, ctl.perm)
		if err != nil {
			fmt.Println(err)
		}
		if bpfileExist {
			// 尝试恢复断点
			position = ctl.recover(taskchan)
		}
		// 自动存储断点
		go ctl.autoExportBreakpoint()
	}
	// 单协程任务分配
	if position < ctl.totalSize && (ctl.totalSize == 0 || !ctl.basic.multithread || ctl.config.RoutineCount == 1) {
		ctl.addTask(taskchan, newBlock(position, ctl.totalSize-1))
		return
	}
	// 多协程任务分配
Allot:
	for position < ctl.totalSize {
		start := position
		end := position + ctl.config.RoutineSize
		if end > ctl.totalSize-1 {
			end = ctl.totalSize - 1
		}
		ok = ctl.addTask(taskchan, newBlock(start, end))
		if !ok {
			break Allot
		}
		position = end + 1
	}
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
	dest = fileWriteAt(ctl.outfile, task)

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

// iocopy 拷贝数据
func (ctl *control) iocopy(dst io.Writer, src io.Reader, bufsize int) (written int64, err error) {

	// 创建 buffer 缓冲区
	dstbuf := bufio.NewWriterSize(dst, bufsize)
	defer dstbuf.Flush()

	buf := make([]byte, bufsize)
	for {
		nr, er := src.Read(buf)
		// 限速器
		if ctl.rate != nil && nr > 0 {
			ctl.rate.WaitN(ctl.ctx, nr)
		}
		if nr > 0 {
			nw, ew := dstbuf.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			nw64 := int64(nw)
			atomic.AddInt64(ctl.completedSize, nw64)
			written += nw64
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = errors.New("short write")
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// finish 完成任务
func (ctl *control) finish(err error) {
	// 取消上下文不算错误
	canceled := errors.Is(err, context.Canceled)
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
	if ctl.bpfile != nil {
		ctl.exportBreakpoint()
		ctl.bpfile.Close()
		if err == nil && !canceled {
			os.Remove(ctl.bpfilepath)
		}
	}
	// 输出文件
	if ctl.outfile != nil {
		ctl.outfile.Close()
	}
	// 设置完成状态
	ctl.setStatus(STATUS_FINISH)
	// 发送完成信息
	if ctl.sendEvent != nil {
		ctl.sendEvent()
	}
	// 发送完成信息并关闭
	ctl.done <- err
	close(ctl.done)
}
