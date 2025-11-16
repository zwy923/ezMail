package util

import (
	"go.uber.org/zap"
)

var Log *zap.Logger

func NewLogger() *zap.Logger {
	l, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	Log = l
	return l
}
