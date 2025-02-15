package cvstask

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pingcap/tiflow/dm/pkg/log"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"

	"github.com/hanfei1991/microcosm/lib"
	"github.com/hanfei1991/microcosm/lib/registry"
	"github.com/hanfei1991/microcosm/model"
	"github.com/hanfei1991/microcosm/pb"
	dcontext "github.com/hanfei1991/microcosm/pkg/context"
	"github.com/hanfei1991/microcosm/pkg/errors"
	"github.com/hanfei1991/microcosm/pkg/p2p"
)

const (
	BUFFERSIZE = 1024
)

type strPair struct {
	firstStr  string
	secondStr string
}

type Config struct {
	Idx      int    `json:"Idx"`
	SrcHost  string `json:"SrcHost"`
	DstHost  string `json:"DstHost"`
	DstDir   string `json:"DstIdx"`
	StartLoc string `json:"StartLoc"`
}

type cvsTask struct {
	lib.BaseWorker
	srcHost   string
	srcDir    string
	dstHost   string
	dstDir    string
	counter   *atomic.Int64
	recordIdx string
	fileIdx   int
	cancelFn  func()
	buffer    chan strPair
	isEOF     bool

	statusCode struct {
		sync.RWMutex
		code lib.WorkerStatusCode
	}
	runError struct {
		sync.RWMutex
		err error
	}

	statusRateLimiter *rate.Limiter
}

func RegisterWorker() {
	constructor := func(ctx *dcontext.Context, id lib.WorkerID, masterID lib.MasterID, config lib.WorkerConfig) lib.WorkerImpl {
		return NewCvsTask(ctx, id, masterID, config)
	}
	factory := registry.NewSimpleWorkerFactory(constructor, &Config{})
	registry.GlobalWorkerRegistry().MustRegisterWorkerType(lib.CvsTask, factory)
}

func NewCvsTask(ctx *dcontext.Context, _workerID lib.WorkerID, masterID lib.MasterID, conf lib.WorkerConfig) *cvsTask {
	cfg := conf.(*Config)
	task := &cvsTask{
		srcHost:           cfg.SrcHost,
		fileIdx:           cfg.Idx,
		dstHost:           cfg.DstHost,
		dstDir:            cfg.DstDir,
		recordIdx:         cfg.StartLoc,
		buffer:            make(chan strPair, BUFFERSIZE),
		statusRateLimiter: rate.NewLimiter(rate.Every(time.Second), 1),
		counter:           atomic.NewInt64(0),
	}
	return task
}

func (task *cvsTask) InitImpl(ctx context.Context) error {
	log.L().Info("init the task  ", zap.Any("task id :", task.ID()))
	task.setStatusCode(lib.WorkerStatusNormal)
	ctx, task.cancelFn = context.WithCancel(ctx)
	go func() {
		err := task.Receive(ctx)
		if err != nil {
			log.L().Error("error happened when reading data from the upstream ", zap.String("id", task.ID()), zap.Any("message", err.Error()))
			task.setRunError(err)
			task.setStatusCode(lib.WorkerStatusError)
		}
	}()
	go func() {
		err := task.Send(ctx)
		if err != nil {
			log.L().Error("error happened when writing data to the downstream ", zap.String("id", task.ID()), zap.Any("message", err.Error()))
			task.setRunError(err)
			task.setStatusCode(lib.WorkerStatusError)
		} else {
			task.setStatusCode(lib.WorkerStatusFinished)
		}
	}()

	return nil
}

// Tick is called on a fixed interval.
func (task *cvsTask) Tick(ctx context.Context) error {
	// log.L().Info("cvs task tick", zap.Any(" task id ", string(task.ID())+" -- "+strconv.FormatInt(task.counter, 10)))
	if task.statusRateLimiter.Allow() {
		err := task.BaseWorker.UpdateStatus(ctx, task.Status())
		if errors.ErrWorkerUpdateStatusTryAgain.Equal(err) {
			log.L().Warn("update status try again later", zap.String("id", task.ID()), zap.String("error", err.Error()))
			return nil
		}
		return err
	}
	switch task.getStatusCode() {
	case lib.WorkerStatusFinished, lib.WorkerStatusError, lib.WorkerStatusStopped:
		return task.BaseWorker.Exit(ctx, task.Status(), task.getRunError())
	default:
	}
	return nil
}

// Status returns a short worker status to be periodically sent to the master.
func (task *cvsTask) Status() lib.WorkerStatus {
	return lib.WorkerStatus{
		Code: task.getStatusCode(), ErrorMessage: "",
		ExtBytes: []byte(fmt.Sprintf("%d", task.counter.Load())),
	}
}

// Workload returns the current workload of the worker.
func (task *cvsTask) Workload() model.RescUnit {
	return 1
}

// OnMasterFailover is called when the master is failed over.
func (task *cvsTask) OnMasterFailover(reason lib.MasterFailoverReason) error {
	return nil
}

func (task *cvsTask) OnMasterMessage(topic p2p.Topic, message p2p.MessageValue) error {
	switch msg := message.(type) {
	case *lib.StatusChangeRequest:
		switch msg.ExpectState {
		case lib.WorkerStatusStopped:
			task.setStatusCode(lib.WorkerStatusStopped)
		default:
			log.L().Info("FakeWorker: ignore status change state", zap.Int32("state", int32(msg.ExpectState)))
		}
	default:
		log.L().Info("unsupported message", zap.Any("message", message))
	}

	return nil
}

// CloseImpl tells the WorkerImpl to quitrunStatusWorker and release resources.
func (task *cvsTask) CloseImpl(ctx context.Context) error {
	if task.cancelFn != nil {
		task.cancelFn()
	}
	return nil
}

func (task *cvsTask) Receive(ctx context.Context) error {
	conn, err := grpc.Dial(task.srcHost, grpc.WithInsecure())
	if err != nil {
		log.L().Error("cann't connect with the source address ", zap.String("id", task.ID()), zap.Any("message", task.srcHost))
		return err
	}
	client := pb.NewDataRWServiceClient(conn)
	defer conn.Close()
	reader, err := client.ReadLines(ctx, &pb.ReadLinesRequest{FileIdx: int32(task.fileIdx), LineNo: []byte(task.recordIdx)})
	if err != nil {
		log.L().Error("read data from file failed ", zap.String("id", task.ID()), zap.Any("message", task.srcDir))
		return err
	}
	for {
		reply, err := reader.Recv()
		if err != nil {
			log.L().Error("read data failed", zap.String("id", task.ID()), zap.Error(err))
			if !task.isEOF {
				task.cancelFn()
			}
			return err
		}
		if reply.IsEof {
			log.L().Info("Reach the end of the file ", zap.String("id", task.ID()), zap.Any("fileName:", task.srcDir))
			close(task.buffer)
			break
		}
		select {
		case <-ctx.Done():
			return nil
		case task.buffer <- strPair{firstStr: string(reply.Key), secondStr: string(reply.Val)}:
		}
		// waiting longer time to read lines slowly
	}
	return nil
}

func (task *cvsTask) Send(ctx context.Context) error {
	conn, err := grpc.Dial(task.dstHost, grpc.WithInsecure())
	if err != nil {
		log.L().Error("can't connect with the destination address ", zap.Any("message", task.dstHost))
		return err
	}
	client := pb.NewDataRWServiceClient(conn)
	defer conn.Close()
	writer, err := client.WriteLines(ctx)
	if err != nil {
		log.L().Error("call write data rpc failed", zap.String("id", task.ID()), zap.Error(err))
		task.cancelFn()
		return err
	}
	for {
		select {
		case kv, more := <-task.buffer:
			if !more {
				log.L().Info("Reach the end of the file ", zap.String("id", task.ID()))
				resp, err := writer.CloseAndRecv()
				if err != nil {
					return err
				}
				if len(resp.ErrMsg) > 0 {
					log.L().Warn("close writing meet error", zap.String("id", task.ID()))
				}
				return nil
			}
			err := writer.Send(&pb.WriteLinesRequest{FileIdx: int32(task.fileIdx), Key: []byte(kv.firstStr), Value: []byte(kv.secondStr), Dir: task.dstDir})
			task.counter.Add(1)
			if err != nil {
				log.L().Error("call write data rpc failed ", zap.String("id", task.ID()), zap.Error(err))
				task.cancelFn()
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (task *cvsTask) getStatusCode() lib.WorkerStatusCode {
	task.statusCode.RLock()
	defer task.statusCode.RUnlock()
	return task.statusCode.code
}

func (task *cvsTask) setStatusCode(status lib.WorkerStatusCode) {
	task.statusCode.Lock()
	defer task.statusCode.Unlock()
	task.statusCode.code = status
}

func (task *cvsTask) getRunError() error {
	task.runError.RLock()
	defer task.runError.RUnlock()
	return task.runError.err
}

func (task *cvsTask) setRunError(err error) {
	task.runError.Lock()
	defer task.runError.Unlock()
	task.runError.err = err
}
