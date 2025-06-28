package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
ToString converts a Redis reply to a String
*/
func ToString(data []byte) (string, error) {
	if data == nil {
		return "", nil
	}
	return string(data), nil
}

/*
ToBool converts a Redis reply to a bool
*/
func ToBool(data []byte) (bool, error) {
	if data == nil {
		return false, nil
	}
	return string(data) == "1", nil
}

// Get retrieves a key's value
func Get(key string) ([]byte, error) {
	return Client.Get(ctx, key).Bytes()
}

// GetFrom retrieves a key's value from a specific named client
func GetFrom(clientName, key string) ([]byte, error) {
	client := GetClient(clientName)
	return client.Get(ctx, key).Bytes()
}

// Set sets a key's value
func Set(key string, value interface{}) error {
	return Client.Set(ctx, key, value, 0).Err()
}

// SetOn sets a key's value on a specific named client
func SetOn(clientName, key string, value interface{}) error {
	client := GetClient(clientName)
	return client.Set(ctx, key, value, 0).Err()
}

// SetEX sets a key's value with an expiration time in seconds
func SetEX(key string, value interface{}, seconds int) error {
	return Client.Set(ctx, key, value, time.Duration(seconds)*time.Second).Err()
}

// SetEXOn sets a key's value with an expiration time on a specific client
func SetEXOn(clientName, key string, value interface{}, seconds int) error {
	client := GetClient(clientName)
	return client.Set(ctx, key, value, time.Duration(seconds)*time.Second).Err()
}

// Exists checks if a key exists
func Exists(key string) (bool, error) {
	result, err := Client.Exists(ctx, key).Result()
	return result > 0, err
}

// ExistsOn checks if a key exists on a specific client
func ExistsOn(clientName, key string) (bool, error) {
	client := GetClient(clientName)
	result, err := client.Exists(ctx, key).Result()
	return result > 0, err
}

// Delete removes a key
func Delete(keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// DeleteOn removes keys from a specific client
func DeleteOn(clientName string, keys ...string) error {
	client := GetClient(clientName)
	return client.Del(ctx, keys...).Err()
}

// Expire sets a key's expiration time in seconds
func Expire(key string, seconds int) error {
	return Client.Expire(ctx, key, time.Duration(seconds)*time.Second).Err()
}

// ExpireOn sets a key's expiration time on a specific client
func ExpireOn(clientName, key string, seconds int) error {
	client := GetClient(clientName)
	return client.Expire(ctx, key, time.Duration(seconds)*time.Second).Err()
}

// Ping checks the connection to Redis
func Ping() error {
	return Client.Ping(ctx).Err()
}

// PingClient checks the connection to a specific Redis client
func PingClient(clientName string) error {
	client := GetClient(clientName)
	return client.Ping(ctx).Err()
}

// Incr increments a key's integer value
func Incr(key string) (int64, error) {
	return Client.Incr(ctx, key).Result()
}

// IncrOn increments a key's integer value on a specific client
func IncrOn(clientName, key string) (int64, error) {
	client := GetClient(clientName)
	return client.Incr(ctx, key).Result()
}

// Decr decrements a key's integer value
func Decr(key string) (int64, error) {
	return Client.Decr(ctx, key).Result()
}

// DecrOn decrements a key's integer value on a specific client
func DecrOn(clientName, key string) (int64, error) {
	client := GetClient(clientName)
	return client.Decr(ctx, key).Result()
}

// RPush adds values to the end of a list
func RPush(key string, values ...interface{}) error {
	return Client.RPush(ctx, key, values...).Err()
}

// RPushOn adds values to the end of a list on a specific client
func RPushOn(clientName, key string, values ...interface{}) error {
	client := GetClient(clientName)
	return client.RPush(ctx, key, values...).Err()
}

// LPop removes and returns the first element of a list
func LPop(key string) ([]byte, error) {
	return Client.LPop(ctx, key).Bytes()
}

// LPopFrom removes and returns the first element of a list from a specific client
func LPopFrom(clientName, key string) ([]byte, error) {
	client := GetClient(clientName)
	return client.LPop(ctx, key).Bytes()
}

// SAdd adds members to a set
func SAdd(key string, members ...interface{}) error {
	return Client.SAdd(ctx, key, members...).Err()
}

// SAddOn adds members to a set on a specific client
func SAddOn(clientName, key string, members ...interface{}) error {
	client := GetClient(clientName)
	return client.SAdd(ctx, key, members...).Err()
}

// SIsMember checks if a value is a member of a set
func SIsMember(key string, member interface{}) (bool, error) {
	return Client.SIsMember(ctx, key, member).Result()
}

// SIsMemberOn checks if a value is a member of a set on a specific client
func SIsMemberOn(clientName, key string, member interface{}) (bool, error) {
	client := GetClient(clientName)
	return client.SIsMember(ctx, key, member).Result()
}

// SMembers returns all members of a set
func SMembers(key string) ([]string, error) {
	return Client.SMembers(ctx, key).Result()
}

// SMembersFrom returns all members of a set from a specific client
func SMembersFrom(clientName, key string) ([]string, error) {
	client := GetClient(clientName)
	return client.SMembers(ctx, key).Result()
}

// HSet sets a field in a hash
func HSet(key, field string, value interface{}) error {
	return Client.HSet(ctx, key, field, value).Err()
}

// HSetOn sets a field in a hash on a specific client
func HSetOn(clientName, key, field string, value interface{}) error {
	client := GetClient(clientName)
	return client.HSet(ctx, key, field, value).Err()
}

// HGet gets a field from a hash
func HGet(key, field string) ([]byte, error) {
	return Client.HGet(ctx, key, field).Bytes()
}

// HGetFrom gets a field from a hash from a specific client
func HGetFrom(clientName, key, field string) ([]byte, error) {
	client := GetClient(clientName)
	return client.HGet(ctx, key, field).Bytes()
}

// HDel deletes a field from a hash
func HDel(key string, fields ...string) error {
	return Client.HDel(ctx, key, fields...).Err()
}

// HDelOn deletes a field from a hash on a specific client
func HDelOn(clientName, key string, fields ...string) error {
	client := GetClient(clientName)
	return client.HDel(ctx, key, fields...).Err()
}

// HIncrBy increments a hash field by the given number
func HIncrBy(key, field string, incr int64) (int64, error) {
	return Client.HIncrBy(ctx, key, field, incr).Result()
}

// HIncrByOn increments a hash field by the given number on a specific client
func HIncrByOn(clientName, key, field string, incr int64) (int64, error) {
	client := GetClient(clientName)
	return client.HIncrBy(ctx, key, field, incr).Result()
}

// HDecrBy decrements a hash field by the given number
func HDecrBy(key, field string, decr int64) (int64, error) {
	return Client.HIncrBy(ctx, key, field, -decr).Result()
}

// HDecrByOn decrements a hash field by the given number on a specific client
func HDecrByOn(clientName, key, field string, decr int64) (int64, error) {
	client := GetClient(clientName)
	return client.HIncrBy(ctx, key, field, -decr).Result()
}

// Keys gets all keys matching a pattern
func Keys(pattern string) ([]string, error) {
	return Client.Keys(ctx, pattern).Result()
}

// KeysFrom gets all keys matching a pattern from a specific client
func KeysFrom(clientName, pattern string) ([]string, error) {
	client := GetClient(clientName)
	return client.Keys(ctx, pattern).Result()
}

// Scan iterates over keys matching a pattern
func Scan(pattern string) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = Client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// ScanFrom iterates over keys matching a pattern from a specific client
func ScanFrom(clientName, pattern string) ([]string, error) {
	client := GetClient(clientName)
	var keys []string
	var cursor uint64

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// MarshalJSON serializes an object to JSON for Redis storage
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON deserializes JSON from Redis into an object
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// SetJSON stores a struct as JSON
func SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, 0).Err()
}

// SetJSONOn stores a struct as JSON on a specific client
func SetJSONOn(clientName, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	client := GetClient(clientName)
	return client.Set(ctx, key, data, 0).Err()
}

// GetJSON retrieves a JSON value and unmarshals it
func GetJSON(key string, dest interface{}) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// GetJSONFrom retrieves a JSON value from a specific client
func GetJSONFrom(clientName, key string, dest interface{}) error {
	client := GetClient(clientName)
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// WithContext executes a function with a specific context
func WithContext(c context.Context, fn func(ctx context.Context) error) error {
	return fn(c)
}

// WithClientContext executes a function with a specific context and client
func WithClientContext(clientName string, c context.Context, fn func(client *redis.Client, ctx context.Context) error) error {
	client := GetClient(clientName)
	return fn(client, c)
}

// Pipeline creates a new Redis pipeline
func Pipeline() redis.Pipeliner {
	return Client.Pipeline()
}

// PipelineFrom creates a new Redis pipeline from a specific client
func PipelineFrom(clientName string) redis.Pipeliner {
	client := GetClient(clientName)
	return client.Pipeline()
}

// TxPipeline creates a new Redis transaction pipeline
func TxPipeline() redis.Pipeliner {
	return Client.TxPipeline()
}

// TxPipelineFrom creates a new Redis transaction pipeline from a specific client
func TxPipelineFrom(clientName string) redis.Pipeliner {
	client := GetClient(clientName)
	return client.TxPipeline()
}
