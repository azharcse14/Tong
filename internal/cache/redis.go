package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis client — একটাই থাকবে পুরো app এ
type Cache struct {
	client *redis.Client
}

func New(addr string) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr: addr, // যেমন "localhost:6379"
	})
	return &Cache{client: client}
}

// Get — Redis থেকে data নাও
// না পেলে ("" , nil) ফেরত দেবে — err না
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// key নেই — এটা error না, স্বাভাবিক
		return "", nil
	}
	return val, err
}

// Set — Redis এ data রাখো, ttl পরে নিজেই মুছে যাবে
func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete — Redis থেকে data মুছো
// একাধিক key একসাথে মুছতে পারবে
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// DeletePattern — pattern দিয়ে সব key মুছো
// যেমন "menu:*" দিলে menu র সব cache মুছবে
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}