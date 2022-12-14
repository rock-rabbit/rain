package rain

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"time"
)

// BarTemplate 进度条显示内容的参数
type BarTemplate struct {
	// Template 模版
	Template *template.Template
	// NoSizeTemplate 获取不到文件大小时的模板
	NoSizeTemplate *template.Template
	// Saucer 进度字符, 默认为 =
	Saucer string
	// SaucerHead 进度条头, 使用场景 =====> , 其中的 > 就是进度条头, 默认为 >
	SaucerHead string
	// SaucerPadding 进度空白字符, 默认为 -
	SaucerPadding string
	// BarStart 进度前缀, 默认为 [
	BarStart string
	// BarEnd 进度后缀, 默认为 ]
	BarEnd string
	// BarWidth 进度条宽度, 默认为 80
	BarWidth int
}

// BarStatString 注入到模版中的字符串结构
// {{.CompletedLength}} / {{.TotalLength}} {{.Saucer}} {{.Progress}}% {{.DownloadSpeed}} {{.EstimatedTime}}
type BarStatString struct {
	// TotalLength 文件总大小
	TotalLength string
	// CompletedLength 已下载大小
	CompletedLength string
	// DownloadSpeed 文件每秒下载速度
	DownloadSpeed string
	// EstimatedTime 预计下载完成还需要的时间
	EstimatedTime string
	// Progress 下载进度, 长度为 100
	Progress string
	// Saucer 进度条
	Saucer string
}

// Bar 提供一个简单的进度条
type Bar struct {
	// Template 进度条样式
	Template *BarTemplate
	// FriendlyFormat 使用人类友好的单位
	FriendlyFormat bool
	// FinishHide 完成后隐藏进度条,下载完成后清除掉进度条
	FinishHide bool
	// Hide 是否隐藏进度条
	Hide bool
	// Stdout 进度条输出, 默认为 os.Stdout
	Stdout io.Writer
}

var _ ProgressEventExtend = &Bar{}

func NewBar() *Bar {
	t, _ := template.New("RainBarTemplate").Parse(`{{.CompletedLength}} / {{.TotalLength}} {{.Saucer}} {{.Progress}} {{.DownloadSpeed}} {{.EstimatedTime}}`)
	notsizeT, _ := template.New("RainBarNotSizeTemplate").Parse(`{{.CompletedLength}} {{.DownloadSpeed}} {{.ConsumingTime}}`)
	return &Bar{
		Template: &BarTemplate{
			Template:       t,
			NoSizeTemplate: notsizeT,
			Saucer:         "=",
			SaucerHead:     ">",
			SaucerPadding:  "-",
			BarStart:       "[",
			BarEnd:         "]",
			BarWidth:       80,
		},
		FriendlyFormat: true,
		Hide:           false,
		Stdout:         os.Stdout,
		FinishHide:     false,
	}
}

// Change 进度变化
func (bar *Bar) Change(stat *EventExtend) {
	bar.change(stat)
}

// Close 中途暂停
func (bar *Bar) Close(stat *EventExtend) {
	bar.change(stat)
}

// Error 出现错误
func (bar *Bar) Error(stat *EventExtend) {
	bar.change(stat)
}

// Finish 下载成功
func (bar *Bar) Finish(stat *EventExtend) {
	bar.change(stat)
}

// Change 检查更新
func (bar *Bar) change(stat *EventExtend) {
	if stat.Status.Is(STATUS_BEGIN, STATUS_NOTSTART) || stat == nil {
		return
	}
	var templateEntity *template.Template
	if stat.TotalLength == 0 {
		templateEntity = bar.Template.NoSizeTemplate
	} else {
		templateEntity = bar.Template.Template
	}
	if stat.Status.Is(STATUS_FINISH, STATUS_CLOSE, STATUS_ERROR) {
		// 下载完成后清除进度条
		if bar.FinishHide {
			fmt.Printf("\r%s\r", strings.Repeat(" ", bar.Template.BarWidth))
			return
		}
		// 渲染
		err := barRender(bar, stat, templateEntity, true)
		if err != nil {
			return
		}
		fmt.Println()
		return
	}
	err := barRender(bar, stat, templateEntity, false)
	if err != nil {
		return
	}
}

// barRender 渲染
func barRender(bar *Bar, stat *EventExtend, template *template.Template, finish bool) error {
	// 是否使用人性化格式
	formatFileSizeFunc := func(fileSize int64, minsize int, suffix string) (r string) {
		if bar.FriendlyFormat {
			r = formatFileSize(fileSize)
		} else {
			r = fmt.Sprintf("%d B", fileSize)
		}
		if suffix != "" {
			r = r + suffix
		}
		r = barStringSize(r, minsize)
		return
	}
	formatTimeFunc := func(t time.Duration, minsize int) (r string) {
		if bar.FriendlyFormat {
			r = fmt.Sprintf("%v", t)
		} else {
			r = fmt.Sprintf("%ds", int(t.Seconds()))
		}
		r = barStringSize(r, minsize)
		return
	}

	// 将数据转为字符串结构
	statString := BarStatString{
		// TotalLength 文件总大小
		TotalLength: formatFileSizeFunc(stat.TotalLength, 9, ""),

		// CompletedLength 已下载大小
		CompletedLength: formatFileSizeFunc(stat.CompletedLength, 9, ""),

		// DownloadSpeed 文件每秒下载速度
		DownloadSpeed: formatFileSizeFunc(stat.DownloadSpeed, 12, "/s"),

		// EstimatedTime 预计下载完成还需要的时间
		EstimatedTime: formatTimeFunc(stat.EstimatedTime, 4),

		// Progress 下载进度, 长度为 100
		Progress: barStringSize(fmt.Sprint(stat.Progress, "%"), 4),

		// Saucer 这里使用 _____Saucer_____ 占位置, 长度 16
		Saucer: "_____Saucer_____",
	}
	// 模版渲染
	barTemplate := bytes.NewBuffer(make([]byte, 0))
	err := template.Execute(barTemplate, statString)
	if err != nil {
		return err
	}
	barTemplateString := barTemplate.String()
	// 模版中是否存在占位置的 Saucer
	saucerlength := 0
	if strings.Contains(barTemplateString, statString.Saucer) {
		saucerlength = 16
	}
	// 计算进度条需要占用的长度
	barStart := bar.Template.BarStart
	barEnd := bar.Template.BarEnd
	width := bar.Template.BarWidth - len(barTemplateString) - len(barStart) - len(barEnd) + saucerlength
	saucerCount := int(float64(stat.Progress) / 100.0 * float64(width))
	// 组装进度条
	saucerBuffer := bytes.NewBuffer(make([]byte, 0))
	if saucerCount > 0 {
		saucerBuffer.WriteString(barStart)
		saucerBuffer.WriteString(strings.Repeat(bar.Template.Saucer, saucerCount-1))

		saucerHead := bar.Template.SaucerHead
		if saucerHead == "" || finish {
			saucerHead = bar.Template.Saucer
		}
		saucerBuffer.WriteString(saucerHead)
		if width-saucerCount >= 0 {
			saucerBuffer.WriteString(strings.Repeat(bar.Template.SaucerPadding, width-saucerCount))
		}

		saucerBuffer.WriteString(barEnd)
	} else {
		saucerBuffer.WriteString(barStart)
		saucerBuffer.WriteString(strings.Repeat(bar.Template.SaucerPadding, width))
		saucerBuffer.WriteString(barEnd)
	}
	// 替换占位的进度条并打印
	fmt.Fprintf(bar.Stdout, "\r%s", strings.ReplaceAll(barTemplateString, statString.Saucer, saucerBuffer.String()))
	return nil
}

// barStringSize 最小化字符串
func barStringSize(s string, minsize int) string {
	if len(s) < minsize {
		return s + strings.Repeat(" ", minsize-len(s))
	}
	return s
}
