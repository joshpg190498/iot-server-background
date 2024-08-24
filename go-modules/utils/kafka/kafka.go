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

func InitializeReader(brokers []string, groupID string, topics []string, handleMessage func(topic string, message []byte)) {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if reader != nil {
		log.Println("Kafka reader is already initialized")
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
	log.Println("Kafka reader initialized")

	go StartListening(handleMessage)
}

func InitializeWriter(brokers []string) {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if writer != nil {
		log.Println("Kafka writer is already initialized")
		return
	}

	writer = &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	isConnected = true
	log.Println("Kafka writer initialized")
}

func StartListening(handleMessage func(topic string, message []byte)) {
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
}

func PublishData(topic string, key, data []byte) {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if writer == nil {
		log.Println("Kafka writer is not initialized")
		return
	}

	if !isConnected {
		log.Println("Kafka client is not connected.")
		return
	}

	err := writer.WriteMessages(context.Background(), kafka.Message{
		Topic: topic,
		Key:   key,
		Value: data,
	})
	if err != nil {
		log.Printf("Error publishing message to topic %s: %v\n", topic, err)
	} else {
		log.Printf("Message published to topic %s: %s\n", topic, data)
	}
}

func Close() {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if reader != nil {
		err := reader.Close()
		if err != nil {
			log.Printf("Error closing Kafka reader: %v\n", err)
		}
		reader = nil
	}

	if writer != nil {
		err := writer.Close()
		if err != nil {
			log.Printf("Error closing Kafka writer: %v\n", err)
		}
		writer = nil
	}

	isConnected = false
	log.Println("Kafka connection closed")
}
