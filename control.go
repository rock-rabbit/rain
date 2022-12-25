package rain

import (
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
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
	// basic 资源基本信息
	basic *basic
	// outfile 文件指针
	outfile *os.File
	// bpfile 断点文件
	bpfile *os.File
	// breakpoint 断点
	breakpoint *Breakpoint
	// event 事件
	event []ProgressEvent
	// sendEvent 事件发送
	sendEvent func()
	// rate 限速器
	rate *Limiter

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

// setStatus 设置下载状态
func (ctl *control) setStatus(d Status) {
	ctl.status = d
}
