## rain - 超快实现 http 协议下载资源

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
- 下载进度监听

## 计划

* 自动适应配置变化
* 安全复用下载
* HTTP 代理
* 完善单元测试
* 完善性能测试
* 搭建 rain 官网
* 完善文档

## 使用方法

下载到执行目录，并且显示命令行进度条

``` golang
package main

import "github.com/rock-rabbit/rain"

func main() {
	ctl := rain.New(
        "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4",
        rain.WithEvent(rain.NewBar()),
    )
	err := <-ctl.Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
```