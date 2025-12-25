package models

import (
	"time"

	"github.com/google/uuid"
)

var Threshold int = 5

type counter struct {
	count     int
	validTill time.Time
}

var requestCount map[uuid.UUID]counter

// TODO should i use sync once here? do i need thread safety here?
func init() {
	requestCount = make(map[uuid.UUID]counter)
}

func GetRequestCount(userID uuid.UUID) (int, time.Time) {
	ctr := requestCount[userID]
	return ctr.count, ctr.validTill
}

func SetRequestCount(userID uuid.UUID, count int, validTill time.Time) {
	requestCount[userID] = counter{count, validTill}
}

func RemoveUserRequest(userID uuid.UUID) {
	delete(requestCount, userID)
}

type UserRequest struct {
	UserID      uuid.UUID
	RequestTime time.Time
}

var processedRequests []UserRequest

func AddProcessedRequest(user UserRequest) {
	processedRequests = append(processedRequests, user)
}

func ClearExpiredRecords() {
	cur := time.Now().UTC()
	for _, user := range processedRequests {
		if user.RequestTime.Before(cur) {
			RemoveUserRequest(user.UserID)
			processedRequests = processedRequests[1:]
		} else {
			break
		}
	}
}
