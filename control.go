package rain

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/time/rate"
)

type control struct {
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
	// breakpointResume 断点续传
	breakpointResume bool
	// multithread 多线程
	multithread bool
	// outfile 文件指针
	outfile *os.File
	// breakpoint 断点
	breakpoint *Breakpoint
	// event 事件
	event []ProgressEvent
	// eventExend 事件扩展
	eventExend []ProgressEventExtend
	// sendEvent 事件发送
	sendEvent func()
	// rate 限速器
	rate *rate.Limiter

	// err 运行时产生的错误
	err error
	// done 完成下载的通道通知
	done chan error
}

// Status 运行状态
type Status int

const (
	// STATUS_BEGIN 准备中
	STATUS_BEGIN = Status(iota)
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
	err = ctl.Init(ctx)
	if err != nil {
		return err
	}
	go ctl.startTask()
	return nil
}

// Init 初始化
func (ctl *control) Init(ctx context.Context) error {
	// 新建 context
	if ctl.config.Timeout > 0 {
		ctl.ctx, ctl.cancel = context.WithTimeout(ctx, ctl.config.Timeout)
	} else {
		ctl.ctx, ctl.cancel = context.WithCancel(ctx)
	}

	// 初始化变量
	ctl.completedSize = new(int64)
	ctl.done = make(chan error, 1)
	ctl.request.ctx = ctl.ctx
	ctl.breakpoint = &Breakpoint{}

	// 限速器
	if ctl.config.SpeedLimit > 0 {
		ctl.rate = rate.NewLimiter(rate.Limit(ctl.config.SpeedLimit), ctl.config.SpeedLimit)
	}

	// 资源基本信息
	resInfo, err := ctl.request.getResourceInfo()
	if err != nil {
		return err
	}
	ctl.multithread = resInfo.multithread
	ctl.totalSize = resInfo.filesize
	ctl.breakpoint.Filesize = resInfo.filesize
	ctl.breakpoint.Etag = resInfo.etag
	ctl.breakpoint.Tasks = make([]*Block, 0)

	// 任务开始参数
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

	// 文件名检查
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
	ctl.outfile, err = os.OpenFile(ctl.outpath, os.O_CREATE|os.O_RDWR, ctl.perm)
	if err != nil {
		return err
	}

	// 发送事件
	if len(ctl.eventExend) > 0 {
		ctl.addEvent(NewEventExtend(ctl.eventExend...))
	}
	if len(ctl.event) > 0 {
		ctl.sendEvent = ctl.sendEventFunc()
	}

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
	ctl.setStatus(STATUS_CLOSE)
	ctl.cancel()
}

// getError 获取下载的错误信息
func (ctl *control) getError() error {
	return ctl.err
}

// setStatus 设置下载状态
func (ctl *control) setStatus(d Status) {
	ctl.status = d
}
