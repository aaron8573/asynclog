/**
 * @Author: guomumin <aaron8573@gmail.com>
 * @Software: GoLand
 * @File:  async_kafka.go
 * @version: 1.0.0
 * @Date: 2020/7/15 上午11:56
 * @Description:
 */

package asynclog

import (
    "fmt"
    "github.com/Shopify/sarama"
    "time"
)

type asyncKafka struct {
    producer        sarama.AsyncProducer
    brokers         []string
    topic           string
    version         string
    compression     int
    requiredAcks    int
    MaxMessageBytes int
    logQueue        chan []byte
    isQuit          bool
    queueQuit       chan bool
}

// new kafka
func newAsyncKafka(brokers []string, topic, version string, compression, acks, maxMessageBytes int, q chan []byte) *asyncKafka {

    c := new(asyncKafka)
    c.brokers = brokers
    c.topic = topic
    c.version = version
    c.compression = compression
    c.requiredAcks = acks
    c.MaxMessageBytes = maxMessageBytes
    c.logQueue = q

    c.check()

    c.client()

    go c.flushKafka()

    return c
}

// check param
func (c *asyncKafka) check() {
    if len(c.brokers) == 0 {
        panic("brokers is empty")
    }

    if c.topic == "" {
        panic("topic is empty")
    }

    if c.MaxMessageBytes == 0 {
        c.MaxMessageBytes = 1 * 1024 * 1024
    }
}

// kafka client
func (c *asyncKafka) client() {
    var err error
    config := sarama.NewConfig()
    config.Producer.RequiredAcks = kafkaRequiredAcks(c.requiredAcks)
    config.Producer.Partitioner = sarama.NewRandomPartitioner
    config.Producer.Return.Successes = true
    config.Producer.Return.Errors = true
    config.Version = kafkaVersion(c.version)
    config.Metadata.RefreshFrequency = 60 * time.Second
    config.Producer.MaxMessageBytes = c.MaxMessageBytes
    config.Producer.Compression = kafkaCompression(c.compression)

    fmt.Printf("kafka init: kafka verison:%s host:%+v requiredAcks:%d",
        c.version, c.brokers, c.requiredAcks)

    c.producer, err = sarama.NewAsyncProducer(c.brokers, config)
    if err != nil {
        panic("client kafka error: " + err.Error())
    }
}

// flush kafka
func (c *asyncKafka) flushKafka() {
    defer c.producer.AsyncClose()

    var logBody []byte
    for {
        select {
        case logBody = <-c.logQueue:
            msg := &sarama.ProducerMessage{
                Topic: c.topic,
                Value: sarama.ByteEncoder(string(logBody)),
            }

            c.producer.Input() <- msg

        case msg := <-c.producer.Successes():
            data, err := msg.Value.Encode()
            // send success
            fmt.Printf("send kafka success | data: %v | topic: %s | offset: %d | partition: %d | error: %v", string(data), msg.Topic, msg.Offset, msg.Partition, err)
            if c.isQuit && len(c.logQueue) == 0 {
                goto END
            }

        case kafkaErr := <-c.producer.Errors():
            msg, _ := kafkaErr.Msg.Value.Encode()
            // send failed
            //fmt.Printf("send kafka error: %s | data: %s ", kafkaErr.Error(), string(msg))
            if c.isQuit {
                // send failed, write screen when sign quite
                fmt.Println("log queue is exit, send kafka error: ", kafkaErr.Error(), " message: ", string(msg))

                if len(c.logQueue) == 0 {
                    goto END
                }

            } else {
                // send failed, retry
                c.logQueue <- msg
            }
        }
    }

END:
    fmt.Println("log kafka is exit")
    c.queueQuit <- true
    return
}

// quite
func (c *asyncKafka) SignQuite() bool {
    c.isQuit = true
    return <-c.queueQuit
}

// kafka compression
func kafkaCompression(k int) sarama.CompressionCodec {
    kafkaCompressionMap := map[int]sarama.CompressionCodec{
        0: sarama.CompressionNone,
        1: sarama.CompressionGZIP,
        2: sarama.CompressionSnappy,
        3: sarama.CompressionLZ4,
        4: sarama.CompressionZSTD,
    }

    if vs, ok := kafkaCompressionMap[k]; ok {
        return vs
    }

    return sarama.CompressionNone
}

// kafka required acks
func kafkaRequiredAcks(k int) sarama.RequiredAcks {
    kafkaRequiredAcksMap := map[int]sarama.RequiredAcks{
        0:  sarama.NoResponse,
        1:  sarama.WaitForLocal,
        -1: sarama.WaitForAll,
    }

    if vs, ok := kafkaRequiredAcksMap[k]; ok {
        return vs
    }

    return sarama.WaitForAll
}

// kafka version
func kafkaVersion(v string) sarama.KafkaVersion {
    var (
        kafkaVersionMap = map[string]sarama.KafkaVersion{
            "0.8.2.0":  sarama.V0_8_2_0,
            "0.8.2.1":  sarama.V0_8_2_1,
            "0.8.2.2":  sarama.V0_8_2_2,
            "0.9.0.0":  sarama.V0_9_0_0,
            "0.9.0.1":  sarama.V0_9_0_1,
            "0.10.0.0": sarama.V0_10_0_0,
            "0.10.0.1": sarama.V0_10_0_1,
            "0.10.1.0": sarama.V0_10_1_0,
            "0.10.1.1": sarama.V0_10_1_1,
            "0.10.2.0": sarama.V0_10_2_0,
            "0.10.2.1": sarama.V0_10_2_1,
            "0.11.0.0": sarama.V0_11_0_0,
            "0.11.0.1": sarama.V0_11_0_1,
            "0.11.0.2": sarama.V0_11_0_2,
            "1.0.0.0":  sarama.V1_0_0_0,
            "1.1.0.0":  sarama.V1_1_0_0,
            "1.1.1.0":  sarama.V1_1_1_0,
            "2.0.0.0":  sarama.V2_0_0_0,
            "2.0.1.0":  sarama.V2_0_1_0,
            "2.1.0.0":  sarama.V2_1_0_0,
            "2.2.0.0":  sarama.V2_2_0_0,
            "2.3.0.0":  sarama.V2_3_0_0,
            "2.4.0.0":  sarama.V2_4_0_0,
            "2.5.0.0":  sarama.V2_5_0_0,
        }
    )

    if vs, ok := kafkaVersionMap[v]; ok {
        return vs
    }

    return sarama.MaxVersion
}
