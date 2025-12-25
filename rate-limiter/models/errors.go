package models

type error interface {
	Error() string
}

type ServiceError interface {
	error
	GetStatusCode() int
}

type TooManyRequests struct {
	StatusCode int
	Message    string
}

func (e TooManyRequests) Error() string {
	return e.Message
}

func (e TooManyRequests) GetStatusCode() int {
	return e.StatusCode
}
