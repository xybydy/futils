package utils

import (
	"errors"
	"time"

	"github.com/xybydy/gdutils/logger"

	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
)

const (
	RequestBackendError = iota
	RequestRateLimitError
	RequestNotFoundError
	RequestTimeoutError
	RequestBadRequest
	RequestUnknownError
)

func IsRateLimitError(err error) bool {
	return RequestErrorType(err) == RequestRateLimitError
}

func IsBackendError(err error) bool {
	return RequestErrorType(err) == RequestBackendError
}

func RequestErrorType(err error) int {
	logger.Error("API Request error: %s", err)

	if err == nil {
		return 0
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)

	if ok {
		switch {
		case ae.Code >= 500 && ae.Code <= 599:
			return RequestBackendError
		case ae.Code == 400:
			return RequestBadRequest
		case ae.Code == 403:
			return RequestRateLimitError
		case ae.Code == 404:
			return RequestNotFoundError
		}
	}

	if errors.Is(err, context.Canceled) {
		return RequestTimeoutError
	}
	return RequestUnknownError
}

func ExponentialBackoffSleep(try int) {
	seconds := Pow(2, try)
	time.Sleep(time.Duration(seconds) * time.Second)
}
