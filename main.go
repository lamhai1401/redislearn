package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/lamhai1401/redislearn/client"
	"github.com/redis/go-redis/v9"
)

func main() {
	ttl := time.Duration(10) * time.Second
	cli, stop, err := client.NewRedisRepo(&redis.Options{
		Network:    "tcp",
		Addr:       "localhost:6379",
		ClientName: "redis-client",
		OnConnect: func(ctx context.Context, conn *redis.Conn) error {
			fmt.Println("Connected to Redis")
			return conn.Ping(ctx).Err()
		},
	}, log.NewStdLogger(io.Discard))
	if err != nil {
		panic(err.Error())
	}
	defer stop()

	err = cli.SaveItem(context.Background(), "key", []byte("value"), &ttl)
	if err != nil {
		panic(err.Error())
	}

	value, err := cli.FindItem(context.Background(), "key")
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(string(value))

	// err = cli.RemoveItem(context.Background(), "key")
	// if err != nil {
	// 	panic(err.Error())
	// }
}
