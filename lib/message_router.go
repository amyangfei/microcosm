package lib

import (
	"context"

	"github.com/hanfei1991/microcosm/pkg/p2p"
	"github.com/pingcap/errors"
	"github.com/pingcap/tiflow/dm/pkg/log"
	"github.com/pingcap/tiflow/pkg/workerpool"
	"go.uber.org/zap"
)

const defaultMessageRouterBufferSize = 4

type messageWrapper struct {
	topic p2p.Topic
	msg   p2p.MessageValue
}

// MessageRouter is a SPSC(single producer, single consumer) work model, since
// the message frequency is not high, we use a simple channel for message transit.
type MessageRouter struct {
	workerID WorkerID
	buffer   chan messageWrapper
	pool     workerpool.AsyncPool
	errCh    chan error
	routeFn  func(topic p2p.Topic, msg p2p.MessageValue) error
}

func NewMessageRouter(
	workerID WorkerID,
	pool workerpool.AsyncPool,
	bufferSize int,
	routeFn func(topic p2p.Topic, msg p2p.MessageValue) error,
) *MessageRouter {
	return &MessageRouter{
		workerID: workerID,
		pool:     pool,
		buffer:   make(chan messageWrapper, bufferSize),
		errCh:    make(chan error, 1),
		routeFn:  routeFn,
	}
}

func (r *MessageRouter) Tick(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.Trace(ctx.Err())
	case msg := <-r.buffer:
		return r.routeMessage(ctx, msg)
	default:
	}
	return nil
}

// AppendMessage always appends a new message into buffer, if the message buffer
// is full, it will evicits the oldest message
func (r *MessageRouter) AppendMessage(topic p2p.Topic, msg p2p.MessageValue) {
	msgWrap := messageWrapper{
		topic: topic,
		msg:   msg,
	}
	for {
		select {
		case r.buffer <- msgWrap:
			return
		default:
			select {
			case dropMsg := <-r.buffer:
				log.L().Warn("drop message because of buffer is full",
					zap.String("topic", dropMsg.topic), zap.Any("message", dropMsg.msg))
			default:
			}
		}
	}
}

func (r *MessageRouter) onError(err error) {
	select {
	case r.errCh <- err:
	default:
		log.L().Warn("error is dropped because errCh is full", zap.Error(err))
	}
}

func (r *MessageRouter) routeMessage(ctx context.Context, msg messageWrapper) error {
	err := r.pool.Go(ctx, func() {
		err := r.routeFn(msg.topic, msg.msg)
		if err != nil {
			r.onError(err)
		}
	})
	return errors.Trace(err)
}
