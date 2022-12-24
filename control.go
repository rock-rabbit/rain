package rain

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type control struct {
	// config 配置项
	config *Config

	// request 请求器
	request *request

	// perm 新建文件的权限, 默认为 0600
	perm fs.FileMode

	// uri 资源链接
	uri string
	// outdir 输出目录
	outdir string
	// outname 输出名称
	outname string

	// status 运行状态
	status Status
	// totalSize 资源大小
	totalSize int64
	// completedSize 已下载大小
	completedSize *int64
	// done 完成下载的通道通知
	done chan error
	// threadCount 协程数量
	threadCount int
	// breakpointResume 断点续传
	breakpointResume bool
	// basic 资源基本信息
	basic *basic
	// ctx 上下文
	ctx context.Context
	// cancel 取消上下文
	cancel context.CancelFunc
	// outfile 文件指针
	outfile *os.File
	// bpfile 断点文件
	bpfile *os.File
	// breakpoint 断点
	breakpoint *Breakpoint
	// err 运行时产生的错误
	err error
	// event 事件
	event []ProgressEvent
	// sendEvent 事件发送
	sendEvent func()
	// rate 限速器
	rate *Limiter
}

// Status 运行状态
type Status int

const (
	// STATUS_BEGIN 准备中
	STATUS_BEGIN = Status(iota)
	// STATUS_RUNNING 运行中
	STATUS_RUNNING
	// STATUS_FINISH 完成
	STATUS_FINISH
)

// start 开始下载
func (ctl *control) start(ctx context.Context) error {
	var err error
	// 新建 context
	if ctl.config.Timeout > 0 {
		ctl.ctx, ctl.cancel = context.WithTimeout(ctx, ctl.config.Timeout)
	} else {
		ctl.ctx, ctl.cancel = context.WithCancel(ctx)
	}
	// 准备变量、基本参数
	err = ctl.begin()
	if err != nil {
		return err
	}
	go ctl.startTask()

	return nil
}

// wait 等待下载
func (ctl *control) wait() <-chan error {
	return ctl.done
}

// close 关闭下载
func (ctl *control) close() {
	if ctl.cancel == nil {
		return
	}
	ctl.cancel()
}

// getError 获取下载的错误信息
func (ctl *control) getError() error {
	return ctl.err
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
			os.Remove(ctl.bpfilepath())
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

// startTask 开始任务
func (ctl *control) startTask() {
	taskchan := make(chan *Block)
	done := make(chan error, ctl.threadCount)
	ctl.setStatus(STATUS_RUNNING)
	// 发送事件信息
	if len(ctl.event) > 0 {
		go ctl.autoSendEvent()
	}

	defer close(done)
	// 生产任务
	go ctl.production(taskchan)
	// 消费任务
	ctl.consumption(taskchan, done)
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
		bpfileExist := fileExist(ctl.bpfilepath())
		ctl.bpfile, err = os.OpenFile(ctl.bpfilepath(), os.O_CREATE|os.O_RDWR, ctl.perm)
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

// consumption 消费任务
func (ctl *control) consumption(taskchan chan *Block, done chan error) {
	for i := 0; i < ctl.threadCount; i++ {
		go ctl.execute(taskchan, done)
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

	// 数据拷贝
	_, err = ctl.iocopy(dest, res.Body)
	if err != nil {
		return err
	}
	return nil
}

// iocopy 拷贝数据
func (ctl *control) iocopy(dst io.Writer, src io.Reader) (written int64, err error) {
	// buffer size
	size := ctl.config.DiskCache
	if ctl.rate != nil {
		size = int(math.Ceil(float64(ctl.config.SpeedLimit) / 5.0))
		if size > ctl.config.DiskCache {
			size = ctl.config.DiskCache
		}
	}

	// 创建 buffer 缓冲区
	dstbuf := bufio.NewWriterSize(dst, size)
	defer dstbuf.Flush()

	buf := make([]byte, size)
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

// begin 准备，返回是否可以断点续传、是否可以多协程
func (ctl *control) begin() error {
	// 初始化变量
	ctl.completedSize = new(int64)
	ctl.done = make(chan error, 1)
	ctl.request.ctx = ctl.ctx
	ctl.breakpoint = &Breakpoint{}

	// 限速器
	if ctl.config.SpeedLimit > 0 {
		ctl.rate = NewLimiter(Limit(ctl.config.SpeedLimit), ctl.config.SpeedLimit)
	}

	// 资源基本信息
	basic, err := ctl.request.basic()
	if err != nil {
		return err
	}
	ctl.basic = basic
	ctl.totalSize = basic.filesize
	ctl.breakpoint.Filesize = basic.filesize
	ctl.breakpoint.Etag = basic.etag
	ctl.breakpoint.Tasks = make([]*Block, 0)

	// 任务开始参数
	ctl.threadCount = 1
	if ctl.config.RoutineCount > 1 && basic.multithread {
		// 开启最小协程数
		ctl.threadCount = ctl.config.RoutineCount
		maxCount := int(math.Ceil(float64(ctl.totalSize) / float64(ctl.config.RoutineSize)))
		if maxCount < ctl.config.RoutineCount {
			ctl.threadCount = maxCount
		}
	}
	ctl.breakpointResume = basic.multithread && ctl.config.BreakpointResume

	// 文件夹检查
	if !fileExist(ctl.outdir) {
		if ctl.config.CreateDir {
			err = os.MkdirAll(ctl.outdir, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return errors.New("dir not exist")
		}
	}

	// 文件名检查
	if ctl.outname == "" {
		ctl.outname = basic.getFilename()
	}

	// 文件名非法字符过滤
	if ctl.config.AutoFilterFilename {
		ctl.outname = filterFileName(ctl.outname)
	}

	// 文件检查
	path := ctl.outpath()
	isFileExist := fileExist(path)
	isBpfileExist := fileExist(ctl.bpfilepath())
	if isFileExist && (!ctl.breakpointResume || (!isBpfileExist && ctl.breakpointResume)) {
		if ctl.config.AllowOverwrite {
			err := os.Remove(path)
			if err != nil {
				return err
			}
		} else if ctl.config.AutoFileRenaming {
			// 文件重命名
			path, ctl.outname = autoFileRenaming(ctl.outdir, ctl.outname)
		} else {
			return errors.New("file exist")
		}
	}

	// 打开文件
	ctl.outfile, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, ctl.perm)
	if err != nil {
		return err
	}
	return nil
}

// bpfilepath 断点文件路径
func (ctl *control) bpfilepath() string {
	return filepath.Join(ctl.outdir, ctl.outname+ctl.config.BreakpointExt)
}

// outpath 导出文件路径
func (ctl *control) outpath() string {
	path, _ := filepath.Abs(filepath.Join(ctl.outdir, ctl.outname))
	return path
}

// setStatus 设置下载状态
func (ctl *control) setStatus(d Status) {
	ctl.status = d
}

// exportBreakpoint 导出断点
func (ctl *control) exportBreakpoint() {
	if ctl.bpfile == nil {
		return
	}
	ctl.bpfile.Seek(0, 0)
	ctl.bpfile.Truncate(0)
	ctl.breakpoint.export(ctl.bpfile)
	ctl.bpfile.Sync()
}

// autoExportBreakpoint 自动导出断点
func (ctl *control) autoExportBreakpoint() {
	for {
		select {
		case <-time.After(ctl.config.AutoSaveTnterval):
			ctl.exportBreakpoint()
		case <-ctl.ctx.Done():
			return
		}
	}
}

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
			Outpath:         ctl.outpath(),
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
