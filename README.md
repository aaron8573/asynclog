# asynclog

### 异步写日志

通过异步方式，高效写文件日志，尽可能把磁盘IO降到最低，提升系统性能。


### 支持：
- 支持同步写文件日志
- 支持异步写文件日志
- 支持异步按buffer大小落磁盘
- 日志按天、小时分割，默认不分割
- 支持日志异步发送kafka

### 流程

异步写文件日志：

- 日志内容先写到channel队列中，如果队列满了则返回错误（调整队列大小可避免这个问题）
- 然后通过goroutine把日志队列中的日志异步写到buffer中
- 最后通过goroutine把buffer异步刷到磁盘
- 当退出时调用log.AsyncQuite()通知日志队列做退出清盘操作

异步写kakfa：

- 日志内容先写到channel队列中，如果队列满了则返回错误（调整队列大小可避免这个问题）
- 然后通过goroutine把日志队列中的日志异步发送到kafka中
- 当退出时调用log.AsyncQuite()通知日志队列做退出清盘操作



### 例子：

```go
// 异步写文件日志
import "asynclog"

log = asynclog.New(LogConfig{
        Type:         WRITE_LOG_TYPE_AFILE,
        QueueSize:    1000000,
        BufferSize:   1 * 1024 * 1024,
        FileFullPath: "demo.log",
        SplitLogType: async_file.SPLIT_LOG_TYPE_NORMAL,
        Level:        1,
        Flag:         L_Time | L_LEVEL | L_SHORT_FILE,
    })

log.Info("test write log")

// 程序退出时，通知日志队列退出
log.AsyncQuite()
```
```go
// 同步写文件日志
log = New(LogConfig{
        Type:  WRITE_LOG_TYPE_FILE,
        Level: 0,
        FileFullPath: "demo.log",
        Flag:  L_Time | L_LEVEL | L_SHORT_FILE,
    })


log.Info("test write log")

// 程序退出时，关闭日志文件
log.Close()
```

```go
// 异步写kafka日志
log = asynclog.New(LogConfig{
        Type:         WRITE_LOG_TYPE_KAFKA,
        Level:        0,
        KafkaConfig:KafkaConfig{
            Brokers: []string{"localhost:9092"},
            Topic:           "test",
            Version:         "1.0.0.0",
            Compression:     0,
            RequiredAcks:    1,
            MaxMessageBytes: 2 * 1024 * 1024,
        },
    })


log.Info("test write log")
// 程序退出时，通知日志队列退出
log.AsyncQuite()
```



### LogConfig配置说明

```
Type: 写日志类型
    WRITE_LOG_TYPE_FILE —— 同步写文本日志
    WRITE_LOG_TYPE_AFILE —— 异步写文本日志
    WRITE_LOG_TYPE_KAFKA —— 异步发送kafka
    WRITE_LOG_TYPE_FILE_AND_KAFKA —— 同步写文本日志并异步发送kafka （调试场景）

QueueSize： 队列大小，默认10000。根据服务QPS设置此值

BufferSize： 缓存buffer块大小， 默认1MB （1 * 1024 * 1024）

FileFullPath：落日志文件全路径（包括文件名）

SplitLogType：分割日志方式
    async_file.SPLIT_LOG_TYPE_NORMAL —— 默认不分割
    SPLIT_LOG_TYPE_DAY —— 按天分割
    SPLIT_LOG_TYPE_HOUR —— 按小时分割

Level： 日志级别，默认Debug
    0-Debug,1-Info,2-Warn,3-Error,4-Fatal,5-Panic

Flag： 日志标记
    L_Time ——— 日志时间
    L_LEVEL ———— 日志级别
    L_SHORT_FILE ———— 短日志文件
    L_LONG_FILE ———— 长日志文件

KafkaConfig
	Brokers: kafka 集群服务器列表
	Topic: 发送kafka topic
	Version： kafka版本， 默认最低版本
	  0.8.2.0 -> 2.5.0.0
	Compression： 发送kafka消息压缩级别， 默认 0
		0 —— none
		1 —— gzip
		2 —— snappy
		3 —— lz4
		4 —— zstd
	RequiredAcks： 发送kafka ack机制，默认 -1
		0 —— 只管发送kafka，不管发送结果
		1 —— 发送kakfa主节点成功即返回成功，不管副本同步状态
		-1 —— 发送kafka主节点和副本全部同步成功后返回成功
	MaxMessageBytes： kafka最大消息大小，默认1MB （1 * 1024 * 1024）
```

