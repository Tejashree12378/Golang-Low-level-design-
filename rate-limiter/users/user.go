package users

import (
	"context"
	"net/http"
	"rate-limiter/models"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	requestCount      map[int]counter
	mu                sync.Mutex
	Threshold         = 5
	processedRequests []User
)

type counter struct {
	count     int
	validTill time.Time
}

type User struct {
	ID          int `json:"id"`
	RequestTime time.Time
}

func init() {
	requestCount = make(map[int]counter)
}

func NewUser() *User {
	return &User{}
}

func (user *User) GetUser(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Query("id"))
	if id == 0 || err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id must be provided"})
		return
	}

	requestBody := &User{RequestTime: time.Now(), ID: id}

	svcErr := validateRequest(requestBody)
	if svcErr != nil {
		ctx.JSON(svcErr.GetStatusCode(), gin.H{"error": svcErr.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"user": "Hello, Welcome"})
}

func validateRequest(user *User) models.ServiceError {
	mu.Lock()
	defer mu.Unlock()

	ctr := requestCount[user.ID]
	count, expireAt := ctr.count, ctr.validTill
	// new entry or reset the count
	if count == 0 || expireAt.Before(user.RequestTime) {
		requestCount[user.ID] = counter{1, user.RequestTime.Add(1 * time.Minute)}
		processedRequests = append(processedRequests,
			User{ID: user.ID,
				RequestTime: user.RequestTime.Add(1 * time.Minute)})

		return nil
	}

	if user.RequestTime.Before(expireAt) && count >= Threshold {
		return models.TooManyRequests{StatusCode: http.StatusTooManyRequests,
			Message: "too many requests"}
	}

	requestCount[user.ID] = counter{count + 1, expireAt}
	processedRequests = append(processedRequests, User{ID: user.ID, RequestTime: expireAt})
	return nil
}

func GarbageCollector(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ClearExpiredRecords()
		case <-ctx.Done():
			return
		}
	}
}

func ClearExpiredRecords() {
	mu.Lock()
	defer mu.Unlock()
	cur := time.Now().UTC()
	userIDs := make([]int, 0)
	idx := 0

	for _, user := range processedRequests {
		if user.RequestTime.Before(cur) {
			userIDs = append(userIDs, user.ID)
			idx++
		} else {
			break
		}
	}

	for _, userID := range userIDs {
		delete(requestCount, userID)
	}

	processedRequests = processedRequests[idx:]
}
