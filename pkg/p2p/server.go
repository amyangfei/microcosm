package p2p

import (
	"context"
	"net"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tiflow/dm/pkg/log"
	p2pImpl "github.com/pingcap/tiflow/pkg/p2p"
	"github.com/pingcap/tiflow/pkg/security"
	"github.com/pingcap/tiflow/proto/p2p"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// Re-export some types
type (
	Topic  = p2pImpl.Topic
	NodeID = p2pImpl.NodeID
	Config = p2pImpl.MessageServerConfig
)

type (
	TypeInformation = interface{}
	MessageValue    = interface{}
	HandlerFunc     = func(sender NodeID, value MessageValue) error
)

type (
	MessageServerOpt = func(*Config)
)

// read only
var defaultServerConfig = Config{
	MaxPendingMessageCountPerTopic:       256,
	MaxPendingTaskCount:                  102400,
	SendChannelSize:                      16,
	AckInterval:                          time.Millisecond * 200,
	WorkerPoolSize:                       4,
	MaxPeerCount:                         1024,
	WaitUnregisterHandleTimeoutThreshold: time.Millisecond * 100,
}

// MessageRPCService is a background service wrapping a MessageServer instance.
type MessageRPCService struct {
	messageServer *p2pImpl.MessageServer
	grpcServer    *grpc.Server

	noNeedToRunGRPCServer bool
}

// NewMessageServer creates a new message server from given configs
func NewMessageServer(
	selfID NodeID,
	_credential *security.Credential,
	opts ...MessageServerOpt,
) *p2pImpl.MessageServer {
	// Deep copy
	config := defaultServerConfig
	// Apply opts
	for _, opt := range opts {
		opt(&config)
	}
	return p2pImpl.NewMessageServer(selfID, &config)
}

// NewMessageRPCServiceWithRPCServer creates a new MessageRPCService with an
// existing gRPC server.
func NewMessageRPCServiceWithRPCServer(
	selfID NodeID,
	_credential *security.Credential,
	grpcSvr *grpc.Server,
	opts ...MessageServerOpt,
) *MessageRPCService {
	// Deep copy
	config := defaultServerConfig
	// Apply opts
	for _, opt := range opts {
		opt(&config)
	}
	messageServer := p2pImpl.NewMessageServer(selfID, &config)
	// TODO: support accepting TLS connections.
	return &MessageRPCService{
		messageServer: messageServer,
		grpcServer:    grpcSvr,
	}
}

// NewMessageRPCService creates a new MessageRPCService.
// Note: TLS is not supported for now.
func NewMessageRPCService(
	selfID NodeID,
	_credential *security.Credential,
	opts ...MessageServerOpt,
) (*MessageRPCService, error) {
	grpcSvr := grpc.NewServer()
	service := NewMessageRPCServiceWithRPCServer(selfID, _credential, grpcSvr, opts...)
	p2p.RegisterCDCPeerToPeerServer(grpcSvr, service.messageServer)
	return service, nil
}

// NewDependentMessageRPCService creates a new MessageRPCService
// that DOES NOT own a `grpc.Server`.
// TODO refactor the design.
func NewDependentMessageRPCService(
	selfID NodeID,
	_credential *security.Credential,
	grpcSvr *grpc.Server,
	opts ...MessageServerOpt,
) (*MessageRPCService, error) {
	service := NewMessageRPCServiceWithRPCServer(selfID, _credential, grpcSvr, opts...)
	p2p.RegisterCDCPeerToPeerServer(grpcSvr, service.messageServer)
	service.noNeedToRunGRPCServer = true
	return service, nil
}

// Serve listens on `l` and creates the background goroutine for the message server.
func (s *MessageRPCService) Serve(ctx context.Context, l net.Listener) error {
	defer func() {
		if l == nil {
			return
		}
		err := l.Close()
		if err != nil {
			log.L().Warn("failed to close Listener", zap.Error(err))
		}
	}()

	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() (err error) {
		defer log.L().ErrorFilterContextCanceled("message server exited", zap.Error(err))
		return errors.Trace(s.messageServer.Run(ctx))
	})

	// TODO redesign MessageRPCService to avoid this branch
	if s.noNeedToRunGRPCServer {
		return wg.Wait()
	}

	wg.Go(func() (err error) {
		defer func() {
			// TODO (zixiong) filter out expected harmless errors.
			log.L().Debug("grpc server exited", zap.Error(err))
		}()
		return errors.Trace(s.grpcServer.Serve(l))
	})

	// We need a separate goroutine for canceling the gRPC server
	// because the `Serve` method provides by the library does not
	// support canceling by contexts, which is a more idiomatic way.
	wg.Go(func() error {
		<-ctx.Done()
		log.L().Debug("context canceled, stopping the gRPC server")

		s.grpcServer.Stop()
		return nil
	})

	return wg.Wait()
}

func (s *MessageRPCService) GetMessageServer() *p2pImpl.MessageServer {
	return s.messageServer
}

// MakeHandlerManager returns a MessageHandlerManager
func (s *MessageRPCService) MakeHandlerManager() MessageHandlerManager {
	return newMessageHandlerManager(s.messageServer)
}
