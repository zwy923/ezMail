package mq

import (
	"context"
	"encoding/json"
	"fmt"
)

type TypedHandlerFunc func(ctx context.Context, data json.RawMessage) error

type Router struct {
	routes map[string]TypedHandlerFunc
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]TypedHandlerFunc),
	}
}

func (r *Router) Register(eventType string, h TypedHandlerFunc) {
	r.routes[eventType] = h
}

func (r *Router) Handle(ctx context.Context, evt Event) error {
	h, ok := r.routes[evt.Type]
	if !ok {
		fmt.Println("[Router] No handler for event: ", evt.Type)
		return nil
	}

	// 可选：panic 防御
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Println("[Router] panic recovered:", rec)
		}
	}()

	// 真正处理事件
	return h(ctx, evt.Data)
}
