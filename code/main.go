package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-redis/redis/v8"
)

var (
	requests      = make(map[int]struct{})
	requestsMutex sync.Mutex
	redisClient   *redis.Client
	kafkaProducer sarama.SyncProducer
	ctx           = context.Background()
)

func main() {
	// Initialize Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Initialize Kafka producer
	var err error
	kafkaProducer, err = sarama.NewSyncProducer([]string{"localhost:9092"}, nil)
	if err != nil {
		log.Fatalf("Failed to start Kafka producer: %s", err)
	}
	defer kafkaProducer.Close()

	http.HandleFunc("/api/verve/accept", acceptHandler)
	go logUniqueRequests()

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}

func acceptHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	endpoint := r.URL.Query().Get("endpoint")

	// Use Redis for deduplication
	unique, err := redisClient.SetNX(ctx, fmt.Sprintf("request:%d", id), true, 1*time.Minute).Result()
	if err != nil || !unique {
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}

	requestsMutex.Lock()
	requests[id] = struct{}{}
	requestsMutex.Unlock()

	if endpoint != "" {
		go sendRequestToEndpoint(endpoint)
	}

	fmt.Fprintln(w, "ok")
}

func logUniqueRequests() {
	for {
		time.Sleep(1 * time.Minute)

		requestsMutex.Lock()
		count := len(requests)
		requests = make(map[int]struct{})
		requestsMutex.Unlock()

		log.Printf("Unique requests in the last minute: %d", count)

		// Send count to Kafka
		sendCountToKafka(count)
	}
}

func sendRequestToEndpoint(endpoint string) {
	requestsMutex.Lock()
	count := len(requests)
	requestsMutex.Unlock()

	data := map[string]int{"count": count}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal JSON: %s", err)
		return
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send request to endpoint: %s", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Endpoint response status: %s", resp.Status)
}

func sendCountToKafka(count int) {
	message := &sarama.ProducerMessage{
		Topic: "unique_requests",
		Value: sarama.StringEncoder(fmt.Sprintf("%d", count)),
	}

	partition, offset, err := kafkaProducer.SendMessage(message)
	if err != nil {
		log.Printf("Failed to send message to Kafka: %s", err)
		return
	}

	log.Printf("Message sent to Kafka partition %d at offset %d", partition, offset)
}
