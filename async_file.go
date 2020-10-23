/**
 * @Author: guomumin <aaron8573@gmail.com>
 * @File:  async_file.go
 * @version: 1.0.0
 * @Date: 2020/7/13 上午11:16
 * @Description:
 */

package asynclog

import (
    "bufio"
    "fmt"
    "os"
    "sync"
    "time"
)

type asyncFile struct {
    FileDir    string     // file full path
    SplitType  int        // split log type: 0-no 1-split by day 2-split by hour
    buffer     *BufferLog // log buffer
    BufferSize int        // log buffer size
    file       *os.File   // *os.file
    logTime    int        // last flush log success time
    logQueue   chan []byte
}

type BufferLog struct {
    B *bufio.Writer
    sync.Mutex
}

const (
    SPLIT_LOG_TYPE_NORMAL int = 0 // no split file
    SPLIT_LOG_TYPE_DAY    int = 1 // split by day
    SPLIT_LOG_TYPE_HOUR   int = 2 // split by hour
)

func newAsyncFile(fileDir string, splitType, bufferSize int, q chan []byte) *asyncFile {
    al := new(asyncFile)
    al.FileDir = fileDir
    al.SplitType = splitType
    al.BufferSize = bufferSize
    al.logQueue = q

    al.check()

    fileFullPath, _ := al.SplitFileFullPath()
    if err := al.OpenFile(fileFullPath); err != nil {
        panic("open file:" + fileFullPath + " error: " + err.Error())
    }

    al.NewBuffer()

    go al.TickerWriteBuffer()
    go al.TickerFlushBuffer()

    return al
}

func (c *asyncFile) check() {
    if c.FileDir == "" {
        c.FileDir = "log.log"
    }

    if c.BufferSize == 0 {
        c.BufferSize = 1 * 1024 * 1024
    }
}

// ticker write buffer
func (c *asyncFile) TickerWriteBuffer() {
    var (
        err error
    )

    for {
        select {
        case data := <-c.logQueue:
            var tryTimes = 1
            for {
                fileDir, needSplit := c.SplitFileFullPath()
                if needSplit {
                    c.buffer.Lock()
                    if c.buffer.B.Buffered() > 0 {
                        c.buffer.B.Flush()
                    }
                    c.buffer.Unlock()

                    c.OpenFile(fileDir)
                    c.NewBuffer()
                }

                br := []byte("\n")
                for i := 0; i < len(br); i++ {
                    data = append(data, br[i])
                }

                c.buffer.Lock()
                if c.buffer.B.Buffered()+len(data) > c.BufferSize {
                    err = c.buffer.B.Flush()
                    if err != nil {
                        // retry 10 times reopen file
                        if tryTimes == 10 {
                            c.OpenFile(fileDir)
                            c.NewBuffer()
                        }
                        tryTimes++
                        c.buffer.Unlock()
                        continue
                    }
                }

                c.buffer.B.Write(data)
                c.buffer.Unlock()

                break
            }
        }
    }

}

// ticker flush buffer
func (c *asyncFile) TickerFlushBuffer() {
    ticker := time.Tick(1 * time.Second)

    for {
        select {
        case <-ticker:
            c.FlushBuffer()
        }
    }
}

// create newAsyncFile buffer
func (c *asyncFile) NewBuffer() {
    c.buffer = &BufferLog{
        B: bufio.NewWriterSize(c.file, c.BufferSize),
    }
}

// flush buffer
func (c *asyncFile) FlushBuffer() {
    c.buffer.Lock()
    defer c.buffer.Unlock()
    if c.buffer.B.Buffered() > 0 {
        c.buffer.B.Flush()
    }
}

// get file full path
func (c *asyncFile) SplitFileFullPath() (string, bool) {
    var (
        dir        string
        nowTime    int
        needCreate bool
        format     string
    )

    dir = c.FileDir

    switch c.SplitType {
    case SPLIT_LOG_TYPE_NORMAL:
        // not split
        return c.FileDir, false

    case SPLIT_LOG_TYPE_DAY:
        // split by day
        nowTime = time.Now().Day()
        format = "20060102"

    case SPLIT_LOG_TYPE_HOUR:
        // split by hour
        nowTime = time.Now().Hour()
        format = "2006010215"
    }

    if c.logTime != nowTime {
        dir = fmt.Sprintf("%s.%s", c.FileDir, time.Now().Format(format))
        needCreate = true
        c.logTime = nowTime
    }

    return dir, needCreate
}

// open file
func (c *asyncFile) OpenFile(fileDir string) (err error) {
    var f *os.File

    if f, err = os.OpenFile(fileDir, os.O_RDWR|os.O_SYNC|os.O_CREATE|os.O_APPEND, 0644); err != nil {
        return err
    }

    c.CloseFile()

    c.file = f

    return nil
}

// close file
func (c *asyncFile) CloseFile() error {
    if c.file != nil {
        return c.file.Close()
    }

    return nil
}

// quite
func (c *asyncFile) SignQuite() bool {
    var (
        l      int
        ticker = time.Tick(100 * time.Millisecond)
    )

    for {
        select {
        case <-ticker:
            c.FlushBuffer()

            l = len(c.logQueue)
            fmt.Printf("log queue have log: %d ,wait exit\n", l)
            if l == 0 {
                c.CloseFile()
                goto END
            }
        }
    }

END:
    fmt.Println("log queue is exit")
    return true
}
