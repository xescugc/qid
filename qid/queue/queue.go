package queue

import (
	"context"

	"gocloud.dev/pubsub"
)

//go:generate go tool mockgen -destination=../mock/topic.go -mock_names=Topic=Topic -package mock github.com/xescugc/qid/qid/queue Topic

// COPIED from https://pkg.go.dev/gocloud.dev/pubsub@v0.43.0#Topic as it's an interface
// and not a specific type
// Topic publishes messages to all its subscribers.
type Topic interface {
	// Send publishes a message. It only returns after the message has been sent, or failed to be sent. Send can be called from multiple goroutines at once.
	Send(ctx context.Context, m *pubsub.Message) (err error)
	// ErrorAs converts err to driver-specific types
	ErrorAs(err error, i any) bool
	// As converts i to driver-specific types.
	As(i any) bool

	// Shutdown flushes pending message sends and disconnects the Topic. It only returns after all pending messages have been sent.
	Shutdown(ctx context.Context) (err error)
}

type Subscription interface {
	// As converts i to driver-specific types
	As(i any) bool

	// ErrorAs converts err to driver-specific types.
	ErrorAs(err error, i any) bool

	// Receive receives and returns the next message from the Subscription's queue, blocking and polling if none are available. It can be called concurrently from multiple goroutines.
	Receive(ctx context.Context) (_ *pubsub.Message, err error)

	// Shutdown flushes pending ack sends and disconnects the Subscription.
	Shutdown(ctx context.Context) (err error)
}

type Body struct {
	PipelineName string `json:"pipeline_name"`
	JobName      string `json:"job_name"`
}
