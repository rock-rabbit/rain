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

运行以上例子的输出结果：

``` bash
1.01 Mib / 1.01 Mib [=====================================] 100% 630.99 Kib/s 0s
下载完成：/Users/rockrabbit/projects/rain-service/big_buck_bunny_720p_1mb.mp4
```

对于以上示例进行解读：

首先，使用 `rain.New` 方法创建了一个资源下载控制器，有两个参数、分别是：`资源链接`和`进度事件注册`，其中`进度事件注册`注册的是 rain 实现的`命令行进度条`。

之后，我们执行了控制器的 `Run` 方法，`Run` 方法会执行下载并返回带有错误信息的通道，阻塞监听通道内的错误。

最后，没有报错就表示下载完成，可以使用控制器的 `Outpath` 方法获取下载文件的绝对路径。


我们通过上面的例子可以发现，可以使用 `rain.WithXXX` 配置下载控制器，下面我们列出当前支持的配置项以及默认值：

``` golang
// WithOutdir 文件输出目录，默认值：./
rain.WithOutdir(outdir string)

// WithOutname 文件输出名称，默认值为空，为空表示自动获取文件名
rain.WithOutname(outname string)

// WithEvent 进度事件监听，默认注册的事件为空
// ProgressEvent 为接口，实现此接口就能注册事件
rain.WithEvent(e ...ProgressEvent) 

// WithClient 设置请求客户端，默认值为：
/*&http.Client{
	Transport: &http.Transport{
		// 应用来自环境变量的代理
		Proxy: http.ProxyFromEnvironment,
		// 要求服务器返回非压缩的内容，前提是没有发送 accept-encoding 来接管 transport 的自动处理
		DisableCompression: true,
		// 接受服务器提供的任何证书
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		// 每个服务器最大保留闲置连接数
		MaxIdleConnsPerHost: 10,
	},
	// 超时时间
	Timeout: 0,
}*/
rain.WithClient(d *http.Client) 

// WithMethod 设置请求方式，默认值为：GET
rain.WithMethod(d string) 

// WithBody 设置默认请求 Body，默认值为：null
rain.WithBody(d io.Reader)

// WithHeader 设置请求 Header，默认值为：
// accept: */*
// accept-language: zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6,*;q=0.5
// user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.81 Safari/537.36 Edg/104.0.1293.54
rain.WithHeader(d func(h http.Header))

// WithPerm 设置默认输出文件权限，默认为：0600
rain.WithPerm(d fs.FileMode)

// WithRoutineSize 设置协程下载最大字节数，默认为：1048576 * 20，即 20M
rain.WithRoutineSize(d int64)

// WithRoutineCount 设置协程最大数，默认值为：1
rain.WithRoutineCount(d int)

// WithDiskCache 设置磁盘缓冲区大小，默认值为：1048576 * 16，即 16M
rain.WithDiskCache(d int)

// WithSpeedLimit 设置下载速度限制，默认值为：0，即不限速
rain.WithSpeedLimit(d int)

// WithCreateDir 设置是否可以创建目录，默认值为：true
rain.WithCreateDir(d bool)

// WithAllowOverwrite 设置是否允许覆盖下载文件，默认值为：true
rain.WithAllowOverwrite(d bool)

// WithBreakpointResume 设置是否启用断点续传，默认值为：true
rain.WithBreakpointResume(d bool)

// WithAutoFileRenaming 设置是否自动重命名文件，默认值为：true
// 触发条件：AllowOverwrite 为 false，并且文件输出路径已存在相同文件
// 具体为：新文件名在名称之后扩展名之前加上一个点和一个数字（1..9999）
rain.WithAutoFileRenaming(d bool)

// WithAutoFilterFilename 设置是否自动过滤掉文件名称中的非法字符，默认值为：true
// 不管是自行设置或者自动获取的文件名，都会执行过滤
rain.WithAutoFilterFilename(d bool)

// WithTimeout 设置下载超时时间，默认为：10分钟
rain.WithTimeout(d time.Duration)

// WithRetryNumber 设置请求重试次数，默认为：5
rain.WithRetryNumber(d int)

// WithRetryNumber 设置请求重试的间隔时间，默认值为：0
rain.WithRetryTime(d time.Duration)

// WithBreakpointExt 断点文件扩展, 默认值为：.temp.rain
rain.WithBreakpointExt(d string)
```

上面这些是在创建下载控制器时，修改配置的方法，但是每次创建都要写这些配置会非常麻烦，所以我们可以提前修改配置的默认值，修改完成默认值以后 `rain.New` 时会自动应用这些修改。

具体操作如下：

``` golang
// 修改默认配置的输出目录
rain.SetOutdir("./temp")

// 因为上面已经修改了默认的配置，再次使用 rain.New 时就会应用配置
uri := "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
ctl := rain.New(uri, rain.WithEvent(rain.NewBar()))
err := <-ctl.Run()
if err != nil {
	panic(err)
}
fmt.Printf("文件位置：%s\n", ctl.Outpath())
```

运行以上例子的输出结果：

``` bash
1.01 Mib / 1.01 Mib [=====================================] 100% 694.99 Kib/s 0s
文件位置：/Users/rockrabbit/projects/rain-service/temp/big_buck_bunny_720p_1mb.mp4
```
