package rain

import "time"

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
