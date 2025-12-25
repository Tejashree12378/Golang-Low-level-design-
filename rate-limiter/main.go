package main

import (
	"context"
	"rate-limiter/users"

	"github.com/gin-gonic/gin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// cleanup goroutine
	go users.GarbageCollector(ctx)

	router := gin.Default()
	router.GET("/greet_user", users.NewUser().GetUser)
	router.Run(":8080")
}
