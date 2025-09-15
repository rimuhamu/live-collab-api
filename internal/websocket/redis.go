package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
	hub    *Hub
	ctx    context.Context
}

func newRedisService(redisURL string, hub *Hub) (*RedisService, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis url parse error: %v", err)
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping error: %v", err)
	}

	return &RedisService{
		client: client,
		hub:    hub,
		ctx:    ctx,
	}, nil
}

func (r *RedisService) StartSubscription() {
	pubsub := r.client.PSubscribe(r.ctx, "doc:*")
	defer pubsub.Close()

	log.Println("Redis subscription started")

	ch := pubsub.Channel()
	for msg := range ch {
		r.handleRedisMessage(msg)
	}
}

func (r *RedisService) PublishMessage(message *Message) error {
	channel := fmt.Sprintf("doc:%d", message.DocumentId)

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	return r.client.Publish(r.ctx, channel, data).Err()
}

func (r *RedisService) handleRedisMessage(msg *redis.Message) {
	var message Message
	if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
		log.Printf("Error unmarshaling Redis message: %v", err)
		return
	}

	if r.hub.GetDocumentClientCount(message.DocumentId) > 0 {
		r.hub.BroadcastMessage(&message)
	}
}

func (r *RedisService) Close() error {
	return r.client.Close()
}
