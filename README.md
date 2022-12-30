## rain - 超快下载 http 资源

rain 一款 golang 包，可以让你快速实现 http 协议的资源下载，为此 rain 拥有一些方便的特性，比如：多协程、断点续传、自动重命名、限速等。


## 安装

使用 go get 安装 rain

``` sh
go get -u github.com/rock-rabbit/rain
```
 
## 特性

- 多协程分块下载
- 断点下载
- 限速下载
- 文件自动重命名
- 文件名非法字符过滤
- 磁盘缓冲区
- 下载进度和状态监听
- 可自定义的命令行进度条
- 运行时修改配置
- 非阻塞下载

当前测试覆盖： coverage: 78.5% of statements

## 使用方法

**简单使用方法**

``` golang
package main

import (
	"fmt"
	"github.com/rock-rabbit/rain"
)

func main() {
	uri := "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
	ctl, err := rain.New(uri, rain.WithOutdir("./tmp"), rain.WithBar()).Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
```

**监听下载**

``` golang
type EventExtend struct{}

// Change 下载进度
func (ee *EventExtend) Change(stat *rain.EventExtend) {
	fmt.Println("change", stat.Progress)
}

// Error 错误
func (ee *EventExtend) Error(stat *rain.EventExtend) {
	fmt.Println("error", stat.Error)
}

// Close 执行 Close
func (ee *EventExtend) Close(stat *rain.EventExtend) {
	fmt.Println("close")
}

// Finish 成功下载
func (ee *EventExtend) Finish(stat *rain.EventExtend) {
	fmt.Println("finish", stat.Progress)
}

var _ rain.ProgressEventExtend = &EventExtend{}

func main() {
	uri := "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
	ctl, err := rain.New(uri, rain.WithEventExtend(&EventExtend{})).Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
```

**非阻塞下载**

``` golang
func main() {
	uri := "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
	ctl, err := rain.New(uri, rain.WithOutdir("./tmp"), rain.WithBar()).Start()
	if err != nil {
		panic(err)
	}

	// ... 其他逻辑

	_, err = ctl.Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
```

**暂停下载**

```golang
func main() {
	uri := "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
	ctl, err := rain.New(uri, rain.WithOutdir("./tmp"), rain.WithBar()).Start()
	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(time.Second * 2)
		// 暂停下载
		ctl.Close()
	}()

	err = ctl.Wait()
	if err != nil {
		panic(err)
	}

	// 继续下载
	_, err = ctl.Run()
	if err != nil {
		panic(err)
	}

	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
```
...
有很多参数可以自行去查看使用。

## 项目
以下项目使用到了 rain :

* [rain-service](https://github.com/rock-rabbit/rain-service): rpc 下载服务
* [rain-service-gui](https://github.com/rock-rabbit/rain-service-gui): 基于 rain-service 的跨平台图形界面

