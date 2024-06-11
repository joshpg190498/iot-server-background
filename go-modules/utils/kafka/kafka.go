package kafka

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	reader         *kafka.Reader
	writer         *kafka.Writer
	isConnected    bool
	connectionLock sync.Mutex
)

func ConnectClient(brokers []string, groupID string, topics []string, handleMessage func(topic string, message []byte)) {
	if topics == nil {
		topics = []string{}
	}

	connectionLock.Lock()
	defer connectionLock.Unlock()

	if reader != nil {
		log.Println("Already connected to Kafka")
		return
	}

	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		GroupTopics: topics,
		MinBytes:    10e2, // 1KB
		MaxBytes:    10e6, // 10MB
	})

	isConnected = true
	log.Println("Connected to Kafka brokers:", brokers)

	go func() {
		for {
			if !isConnected {
				time.Sleep(2 * time.Second)
				continue
			}

			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Error reading message: %v\n", err)
				continue
			}
			handleMessage(m.Topic, m.Value)
		}
	}()
}

func PublishData(brokers []string, topic string, key, data []byte) {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if writer == nil {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	}

	if !isConnected {
		log.Println("Kafka client is not connected.")
		return
	}

	err := writer.WriteMessages(context.Background(), kafka.Message{
		Key:   key,
		Value: data,
	})
	if err != nil {
		log.Printf("Error publishing to topic %s: %v\n", topic, err)
	} else {
		log.Printf("Message published to topic %s: %s\n", topic, data)
	}
}

func Close() {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if reader != nil {
		reader.Close()
		reader = nil
	}

	if writer != nil {
		writer.Close()
		writer = nil
	}

	isConnected = false
	log.Println("Kafka connection closed")
}
