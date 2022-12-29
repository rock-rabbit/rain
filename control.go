package rain

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/time/rate"
)

type control struct {
	// debug 调试模式
	debug bool
	// ctx 上下文
	ctx context.Context
	// cancel 取消上下文
	cancel context.CancelFunc

	// config 配置项
	config *Config
	// request 请求器
	request *request

	// uri 资源链接
	uri string
	// outdir 输出目录
	outdir string
	// outname 输出名称
	outname string
	// perm 新建文件的权限, 默认为 0600
	perm fs.FileMode

	// bpfilepath 断点文件
	bpfilepath string
	// outpath 输出文件
	outpath string

	// status 运行状态
	status Status
	// totalSize 资源大小
	totalSize int64
	// completedSize 已下载大小
	completedSize *int64
	// threadCount 协程数量
	threadCount int
	// breakpointResume 是否可以断点续传
	breakpointResume bool
	// multithread 是否支持多线程
	multithread bool
	// outfile 文件指针
	outfile *os.File
	// breakpoint 断点
	breakpoint *Breakpoint
	// event 进度事件
	event []ProgressEvent
	// eventExend 进度事件扩展
	eventExend []ProgressEventExtend
	// sendEvent 事件发送
	sendEvent func()
	// rate 限速器
	rate *rate.Limiter

	// mux 锁
	mux sync.Mutex
	// err 运行时产生的错误
	err error
	// done 完成下载的通道通知
	done chan error
}

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

// start 开始下载
func (ctl *control) start(ctx context.Context) (err error) {
	// 已经下载完成
	if ctl.status == STATUS_FINISH {
		return errors.New("status is finish")
	}
	// 已经在运行中
	if ctl.status == STATUS_RUNNING || ctl.status == STATUS_BEGIN {
		return errors.New("status is rinning")
	}
	// 启动已经关闭的下载
	if ctl.status == STATUS_CLOSE || ctl.status == STATUS_ERROR {
		ctl.log("reuse download: ", ctl.uri)
		return ctl.reuse(ctx)
	}

	// 竞争首次启动
	ctl.mux.Lock()
	if ctl.status != STATUS_NOTSTART {
		ctl.mux.Unlock()
		return errors.New("status not is STATUS_NOTSTART")
	}
	ctl.setStatus(STATUS_BEGIN)
	ctl.mux.Unlock()

	ctl.log("new download: ", ctl.uri)

	err = ctl.Init(ctx)
	if err != nil {
		ctl.err = err
		return err
	}
	go ctl.startTask()
	return nil
}

// reuse 复用操作
func (ctl *control) reuse(ctx context.Context) (err error) {
	ctl.packContext(ctx)
	ctl.completedSize = new(int64)
	ctl.done = make(chan error, 1)
	ctl.err = nil

	// 打开文件
	ctl.outfile, err = os.OpenFile(ctl.outpath, os.O_CREATE|os.O_WRONLY, ctl.perm)
	if err != nil {
		return err
	}

	// 加载事件
	ctl.loadEvent()

	go ctl.startTask()

	return nil
}

// packContext 包装上下文
func (ctl *control) packContext(ctx context.Context) {
	if ctl.config.Timeout > 0 {
		ctl.ctx, ctl.cancel = context.WithTimeout(ctx, ctl.config.Timeout)
	} else {
		ctl.ctx, ctl.cancel = context.WithCancel(ctx)
	}
	ctl.request.ctx = ctl.ctx
}

// Init 初始化
func (ctl *control) Init(ctx context.Context) error {
	// 包装上下文
	ctl.packContext(ctx)

	// 资源基本信息
	resInfo, err := ctl.request.getResourceInfo()
	if err != nil {
		return err
	}

	// 断点信息
	ctl.breakpoint = &Breakpoint{
		Filesize: resInfo.filesize,
		Etag:     resInfo.etag,
		Position: 0,
		Tasks:    make([]*Block, 0),
	}

	// 设置限速器
	ctl.setSpeedLimit(ctl.config.SpeedLimit)

	ctl.multithread = resInfo.multithread
	ctl.totalSize = resInfo.filesize
	ctl.threadCount = ctl.config.RoutineCount
	ctl.breakpointResume = ctl.multithread && ctl.config.BreakpointResume

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

	// 自动获取文件名
	if ctl.outname == "" {
		ctl.outname = resInfo.getFilename()
	}

	// 文件名非法字符过滤
	if ctl.config.AutoFilterFilename {
		ctl.outname = filterFileName(ctl.outname)
	}

	// 文件检查
	ctl.outpath, _ = filepath.Abs(filepath.Join(ctl.outdir, ctl.outname))
	ctl.bpfilepath = filepath.Join(ctl.outdir, ctl.outname+ctl.config.BreakpointExt)
	isFileExist := fileExist(ctl.outpath)
	isBpfileExist := fileExist(ctl.bpfilepath)
	if isFileExist && (!ctl.breakpointResume || (!isBpfileExist && ctl.breakpointResume)) {
		if ctl.config.AllowOverwrite {
			err := os.Remove(ctl.outpath)
			if err != nil {
				return err
			}
		} else if ctl.config.AutoFileRenaming {
			// 文件重命名
			ctl.outpath, ctl.outname = autoFileRenaming(ctl.outdir, ctl.outname)
			ctl.bpfilepath = filepath.Join(ctl.outdir, ctl.outname+ctl.config.BreakpointExt)
		} else {
			return errors.New("file exist")
		}
	}

	// 打开文件
	ctl.outfile, err = os.OpenFile(ctl.outpath, os.O_CREATE|os.O_WRONLY, ctl.perm)
	if err != nil {
		return err
	}

	// 加载事件
	ctl.loadEvent()

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
	// 等待关闭
	<-ctl.done
}

// getError 获取下载的错误信息
func (ctl *control) getError() error {
	return ctl.err
}

// setStatus 设置下载状态
func (ctl *control) setStatus(d Status) {
	ctl.status = d
}

// setSpeedLimit 设置限速
func (ctl *control) setSpeedLimit(speedLimit int) {
	ctl.mux.Lock()
	defer ctl.mux.Unlock()
	if speedLimit > 0 {
		if speedLimit < COPY_BUFFER_SIZE {
			speedLimit = COPY_BUFFER_SIZE
		}
		ctl.rate = rate.NewLimiter(rate.Limit(speedLimit), speedLimit)
	} else {
		ctl.rate = nil
	}
	ctl.config.SpeedLimit = speedLimit
}

// rateWaitN 消费限速器
func (ctl *control) rateWaitN(n int) {
	if ctl.rate == nil {
		return
	}
	ctl.mux.Lock()
	defer ctl.mux.Unlock()
	ctl.rate.WaitN(ctl.ctx, n)
}

// setDebug 设置 debug
func (ctl *control) setDebug(v bool) {
	ctl.debug = v
	ctl.request.debug = v
}

// log 打印调试信息
func (ctl *control) log(v ...interface{}) {
	if ctl.debug {
		log.Println(v...)
	}
}

// logf 打印调试信息
func (ctl *control) logf(format string, v ...any) {
	if ctl.debug {
		log.Printf(format, v...)
	}
}
