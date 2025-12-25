package users

import (
	"context"
	"net/http"
	"rate-limiter/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	RequestTime time.Time
}

func NewUser() *User {
	return &User{}
}

func (user *User) GetUser(ctx *gin.Context) {
	id, name := ctx.Query("id"), ctx.Query("name")
	if id == "" || name == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id and name must be provided"})
		return
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requestBody := &User{RequestTime: time.Now(), ID: userID, Name: name}

	svcErr := validateRequest(requestBody)
	if svcErr != nil {
		ctx.JSON(svcErr.GetStatusCode(), gin.H{"error": svcErr.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"user": "Hello, " + requestBody.Name})
}

func validateRequest(user *User) models.ServiceError {
	count, expireAt := models.GetRequestCount(user.ID)
	// new entry or reset the count
	if count == 0 || expireAt.Before(user.RequestTime) {
		models.SetRequestCount(user.ID, 1, user.RequestTime.Add(1*time.Minute))
		models.AddProcessedRequest(models.UserRequest{UserID: user.ID,
			RequestTime: user.RequestTime.Add(1 * time.Minute)})
		return nil
	}

	if user.RequestTime.Before(expireAt) && count >= models.Threshold {
		return models.TooManyRequests{StatusCode: http.StatusTooManyRequests,
			Message: "too many requests"}
	}

	models.SetRequestCount(user.ID, count+1, expireAt)
	models.AddProcessedRequest(models.UserRequest{UserID: user.ID, RequestTime: expireAt})
	return nil
}

func GarbageCollector(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			models.ClearExpiredRecords()
		case <-ctx.Done():
			return
		}
	}
}
