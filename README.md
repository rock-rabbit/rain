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

下面演示一下 rain 简单的使用方法：

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
对于以上示例进行解读：

首先，使用 `rain.New` 方法创建了一个资源下载控制器，有两个参数、分别是：`资源链接`和`进度事件注册`，其中`进度事件注册`注册的是 rain 实现的`命令行进度条`。

之后，我们执行了控制器的 `Run` 方法，`Run` 方法会执行下载并返回带有错误信息的通道，阻塞监听通道内的错误。

最后，没有报错就表示下载完成，可以使用控制器的 `Outpath` 方法获取下载文件的绝对路径。
