/**
 * @Author: guomumin <aaron8573@gmail.com>
 * @file:  log.go
 * @version: 1.0.0
 * @Date: 2020/6/30 下午5:22
 * @Description:
 */

package asynclog

import (
    "errors"
    "fmt"
    "os"
    "runtime"
    "sync"
    "syscall"
    "time"
)

/**
 * 异步高效写日志
 * 通过极大的降低了磁盘的io
 *
 * logLevel: 0-Debug,1-Info,2-Warn,3-Error,4-Fatal,5-Panic
 * log: [time][level][file][log data]
 */

// config
type LogConfig struct {
    Type         int    // 写日志方式 1-同步写文件，2-异步写文件
    FileFullPath string // 日志文件全路径
    QueueSize    int    // 队列大小
    BufferSize   int    // buffer大小
    SplitLogType int    // 切割日志方式 0-不切割，1-按天，2-按小时
    Level        int    // 日志级别
    Flag         int
    KafkaConfig  KafkaConfig
}

// kafka config
type KafkaConfig struct {
    Brokers         []string
    Topic           string
    Version         string
    Compression     int
    RequiredAcks    int
    MaxMessageBytes int
}

// loggers
type Logger struct {
    sync.Mutex
    logType     int            // 写日志方式 1-同步写文件，2-异步写文件，3-异步写kafka
    logLevel    int            // 日志级别
    levelMap    map[int]string // 日志级别
    splitLog    int            // 切割日志方式 0-不切割，1-按天，2-按小时
    file        *os.File       // file
    asyncLogger *asyncFile
    asyncKafka  *asyncKafka
    flag        int
    queueSize   int
}

const (
    L_Time                        = 1 << iota             // log time e.g: 2020-07-13 17:02:42.274391 +0800 CST
    L_LEVEL                                               // log level [INFO]
    L_LONG_FILE                                           // long log file
    L_SHORT_FILE                                          // short log file
    L_PID                                                 // pid
    DEFAULT_LOG                   string      = "log.log" //
    WRITE_LOG_TYPE_FILE           int         = 1         // write log file
    WRITE_LOG_TYPE_AFILE          int         = 2         // async write log file
    WRITE_LOG_TYPE_KAFKA          int         = 3         // async write kafka
    WRITE_LOG_TYPE_FILE_AND_KAFKA int         = 4         // kafka and file
)

var (
    logQueue  chan []byte // log queue
    isQuit    bool
    queueQuit chan bool
    pid       int
)

func New(s LogConfig) *Logger {
    var err error
    logger := defaultLoggerConfig()
    logger.logLevel = s.Level
    logger.logType = s.Type
    logger.flag = s.Flag
    logger.queueSize = s.QueueSize

    if logger.logType != WRITE_LOG_TYPE_KAFKA {
        if s.FileFullPath == "" {
            s.FileFullPath = DEFAULT_LOG
        }
    }

    if logger.logType == WRITE_LOG_TYPE_FILE || logger.logType == WRITE_LOG_TYPE_FILE_AND_KAFKA {

        if logger.file, err = os.OpenFile(s.FileFullPath, os.O_RDWR|os.O_SYNC|os.O_CREATE|os.O_APPEND, 0644);
            err != nil {
            panic("open log file:" + s.FileFullPath + " error: " + err.Error())
        }
    }

    if logger.logType == WRITE_LOG_TYPE_AFILE {
        if logger.queueSize == 0 {
            logger.queueSize = 10000
        }

        logQueue = make(chan []byte, s.QueueSize)
        logger.asyncLogger = newAsyncFile(s.FileFullPath, s.SplitLogType, s.BufferSize)

    }

    if logger.logType == WRITE_LOG_TYPE_KAFKA || logger.logType == WRITE_LOG_TYPE_FILE_AND_KAFKA {
        if logger.queueSize == 0 {
            logger.queueSize = 10000
        }

        logQueue = make(chan []byte, s.QueueSize)
        queueQuit = make(chan bool)
        logger.asyncKafka = newAsyncKafka(s.KafkaConfig.Brokers, s.KafkaConfig.Topic, s.KafkaConfig.Version,
            s.KafkaConfig.Compression, s.KafkaConfig.RequiredAcks, s.KafkaConfig.MaxMessageBytes)

    }

    pid = syscall.Getpid()

    return logger
}

//
func defaultLoggerConfig() *Logger {
    return &Logger{
        file:     nil,
        logLevel: 0,
        levelMap: map[int]string{
            0: "PANIC",
            1: "FATAL",
            2: "ERROR",
            3: "WARN",
            4: "INFO",
            5: "DEBUG",
        },
    }
}

// format log header
func (c *Logger) formatHeader(t time.Time, lvl int) (header string) {

    if c.flag&L_Time != 0 {
        header = fmt.Sprintf("%v ", t.Local())
    }

    if c.flag&L_PID != 0 {
        header += fmt.Sprintf("[%d] ", pid)
    }

    if c.flag&L_LEVEL != 0 {
        header += "[" + c.levelMap[lvl] + "] "
    }

    if c.flag&(L_LONG_FILE|L_SHORT_FILE) != 0 {
        var (
            ok        bool
            callDepth = 2
            file      string
            line      int
        )

        c.Lock()
        _, file, line, ok = runtime.Caller(callDepth)
        if !ok {
            file = "???"
            line = 0
        }
        c.Unlock()

        if c.flag&L_SHORT_FILE != 0 {
            short := file
            for i := len(file) - 1; i > 0; i-- {
                if file[i] == '/' {
                    short = file[i+1:]
                    break
                }
            }
            file = short
        }
        header += fmt.Sprintf("%s:%d ", file, line)
    }

    return header
}

func (c *Logger) Panic(args ...interface{}) {
    s := fmt.Sprint(args...)
    c.Write(0, s)
    c.AsyncQuite()
    panic(s)
}

func (c *Logger) Panicf(format string, args ...interface{}) {
    s := fmt.Sprintf(format, args...)
    c.Write(0, s)
    c.AsyncQuite()
    panic(s)
}

func (c *Logger) Fatal(args ...interface{}) {
    s := fmt.Sprint(args...)
    c.Write(1, s)
    c.AsyncQuite()
    os.Exit(1)
}

func (c *Logger) Fatalf(format string, args ...interface{}) {
    s := fmt.Sprintf(format, args...)
    c.Write(1, s)
    c.AsyncQuite()
    os.Exit(1)
}

func (c *Logger) Error(args ...interface{}) {
    s := fmt.Sprint(args...)
    c.Write(2, s)
}

func (c *Logger) Errorf(format string, args ...interface{}) {
    s := fmt.Sprintf(format, args...)
    c.Write(2, s)
}

func (c *Logger) Warn(args ...interface{}) {
    s := fmt.Sprint(args...)
    c.Write(3, s)
}

func (c *Logger) Warnf(format string, args ...interface{}) {
    s := fmt.Sprintf(format, args...)
    c.Write(3, s)
}

func (c *Logger) Info(args ...interface{}) {
    s := fmt.Sprint(args...)
    c.Write(4, s)
}

func (c *Logger) Infof(format string, args ...interface{}) {
    s := fmt.Sprintf(format, args...)
    c.Write(4, s)
}

func (c *Logger) Debug(args ...interface{}) {
    s := fmt.Sprint(args...)
    c.Write(5, s)
}

func (c *Logger) Debugf(format string, args ...interface{}) {
    s := fmt.Sprintf(format, args...)
    c.Write(5, s)
}

func (c *Logger) Write(level int, s string) (n int, err error) {
    if c.logLevel <= level {
        header := c.formatHeader(time.Now(), level)
        data := []byte(header + s)
        if c.logType > WRITE_LOG_TYPE_FILE {
            err := c.WriteQueue(data)
            n = len(s)
            return n, err
        }

        if c.logType == WRITE_LOG_TYPE_FILE_AND_KAFKA || c.logType == WRITE_LOG_TYPE_FILE {
            br := []byte("\n")
            for i := 0; i < len(br); i++ {
                data = append(data, br[i])
            }

            return c.file.Write(data)
        }
    }

    return 0, nil
}

// write queue
func (c *Logger) WriteQueue(data []byte) error {
    if len(logQueue) >= c.queueSize {
        return errors.New("log queue has reaches maximum")
    }

    logQueue <- data

    return nil
}

// quite write log
func (c *Logger) AsyncQuite() bool {
    isQuit = true

    if c.logType == WRITE_LOG_TYPE_KAFKA {
        return <-queueQuit
    } else {
        return c.asyncLogger.SignQuite()
    }
}

func (c *Logger) Close() error {
    return c.file.Close()
}
