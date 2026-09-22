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

const readErrorBackoff = 2 * time.Second

func connected() bool {
	connectionLock.Lock()
	defer connectionLock.Unlock()
	return isConnected
}

func InitializeReader(brokers []string, groupID string, topics []string, handleMessage func(topic string, message []byte) error) {
	connectionLock.Lock()
	if reader != nil {
		connectionLock.Unlock()
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
	connectionLock.Unlock()

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
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
	}

	isConnected = true
	log.Println("Kafka writer initialized")
}

func StartListening(handleMessage func(topic string, message []byte) error) {
	for {
		if !connected() {
			time.Sleep(readErrorBackoff)
			continue
		}

		m, err := reader.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v\n", err)
			time.Sleep(readErrorBackoff)
			continue
		}

		if err := safeHandle(handleMessage, m.Topic, m.Value); err != nil {
			log.Printf("Error procesando mensaje de %s (no se confirma el offset, se reintentará): %v\n", m.Topic, err)
			time.Sleep(readErrorBackoff)
			continue
		}

		if err := reader.CommitMessages(context.Background(), m); err != nil {
			log.Printf("Error confirmando offset de %s: %v\n", m.Topic, err)
		}
	}
}

func safeHandle(handleMessage func(topic string, message []byte) error, topic string, value []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic recuperado procesando mensaje de %s (se descarta, no se reintenta): %v\n", topic, r)
			err = nil
		}
	}()
	return handleMessage(topic, value)
}

func PublishData(topic string, key, data []byte) bool {
	connectionLock.Lock()
	w := writer
	ok := isConnected
	connectionLock.Unlock()

	if w == nil {
		log.Println("Kafka writer is not initialized")
		return false
	}
	if !ok {
		log.Println("Kafka client is not connected.")
		return false
	}

	err := w.WriteMessages(context.Background(), kafka.Message{
		Topic: topic,
		Key:   key,
		Value: data,
	})
	if err != nil {
		log.Printf("Error publishing message to topic %s: %v\n", topic, err)
		return false
	}
	log.Printf("Message published to topic %s\n", topic)
	return true
}

func Close() {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	if reader != nil {
		if err := reader.Close(); err != nil {
			log.Printf("Error closing Kafka reader: %v\n", err)
		}
		reader = nil
	}

	if writer != nil {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing Kafka writer: %v\n", err)
		}
		writer = nil
	}

	isConnected = false
	log.Println("Kafka connection closed")
}
