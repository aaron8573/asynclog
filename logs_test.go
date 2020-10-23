/**
 * @Author: guomumin <aaron8573@gmail.com>
 * @File:  logs_test.go
 * @version: 1.0.0
 * @Date: 2020/7/13 下午4:32
 * @Description:
 */

package asynclog

import (
    "testing"
)

var (
    log *Logger
)

func TestNew(t *testing.T) {
    // async write file
    // 1000000
    // BufferSize:1MB 2.50s
    // BufferSize:2MB 2.37s
    log = New(LogConfig{
        Type:         WRITE_LOG_TYPE_AFILE,
        QueueSize:    1000000,
        BufferSize:   1 * 1024 * 1024, // 1MB
        SplitLogType: SPLIT_LOG_TYPE_NORMAL,
        FileFullPath: "demo.log",
        Level:        0,
        Flag:         L_Time | L_LEVEL | L_LONG_FILE | L_PID,
        CallDepth:    2,
    })

    for i := 0; i < 100; i++ {
        log.Info("test write log")
    }

    log.AsyncQuite()
}

func TestNew2(t *testing.T) {
    // write file
    log = New(LogConfig{
        Type:         WRITE_LOG_TYPE_FILE,
        Level:        0,
        FileFullPath: "demo.log",
        Flag:         L_Time | L_LEVEL | L_SHORT_FILE | L_PID,
    })

    for i := 0; i < 100; i++ {
        log.Info("test write log")
    }

    log.Close()
}

func TestNew3(t *testing.T) {
    // send kafka
    log = New(LogConfig{
        Type:      WRITE_LOG_TYPE_KAFKA,
        QueueSize: 1000000,
        Level:     0,
        KafkaConfig: KafkaConfig{
            Brokers:         []string{"localhost:9092"},
            Topic:           "test",
            Version:         "1.0.0.0",
            Compression:     0,
            RequiredAcks:    1,
            MaxMessageBytes: 2 * 1024 * 1024,
        },
    })

    for i := 0; i < 100; i++ {
        log.Info("test write log")
    }

    log.AsyncQuite()
}

func TestNew4(t *testing.T){
    // async write more file
    log := New(LogConfig{
        Type:         WRITE_LOG_TYPE_AFILE,
        QueueSize:    1000000,
        BufferSize:   1 * 1024 * 1024, // 1MB
        SplitLogType: SPLIT_LOG_TYPE_NORMAL,
        FileFullPath: "demo.log",
        Level:        0,
        Flag:         L_Time | L_LEVEL | L_LONG_FILE | L_PID,
        CallDepth:    2,
    })

    log2 := New(LogConfig{
        Type:         WRITE_LOG_TYPE_AFILE,
        QueueSize:    1000000,
        BufferSize:   1 * 1024 * 1024, // 1MB
        SplitLogType: SPLIT_LOG_TYPE_NORMAL,
        FileFullPath: "demo2.log",
        Level:        0,
        Flag:         L_Time | L_LEVEL | L_SHORT_FILE | L_PID,
        CallDepth:    2,
    })

    for i := 0; i < 100; i++ {
        log.Info("test write log")
        log2.Info("test write log")
    }

    log.AsyncQuite()
    log2.AsyncQuite()
}