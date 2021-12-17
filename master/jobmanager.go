package master

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/hanfei1991/microcosm/client"
	"github.com/hanfei1991/microcosm/master/jobmaster/system"
	"github.com/hanfei1991/microcosm/model"
	"github.com/hanfei1991/microcosm/pb"
	"github.com/hanfei1991/microcosm/pkg/autoid"
	"github.com/hanfei1991/microcosm/pkg/errors"
	"github.com/pingcap/ticdc/dm/pkg/log"
	"go.etcd.io/etcd/clientv3"
	"go.uber.org/zap"
)

// JobMasterID is special and has no use.
const JobMasterID model.ID = -1

// JobManager is a special job master that manages all the job masters, and notify the offline executor to them.
type JobManager struct {
	*system.Master

	mu          sync.Mutex
	jobMasters  map[model.ID]*model.Task
	idAllocater autoid.JobIDAllocator
	clients     *client.Manager
	masterAddrs []string

	etcdClient *clientv3.Client
}

func (j *JobManager) Start(ctx context.Context) error {
	err := j.clients.AddMasterClient(ctx, j.masterAddrs)
	if err != nil {
		return err
	}
	j.Master = system.New(ctx, JobMasterID, j.clients)
	j.Master.StartInternal()
	return nil
}

func (j *JobManager) CancelJob(ctx context.Context, req *pb.CancelJobRequest) *pb.CancelJobResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	task, ok := j.jobMasters[model.ID(req.JobId)]
	if !ok {
		return &pb.CancelJobResponse{Err: &pb.Error{Message: "No such job"}}
	}
	err := j.Master.StopTasks(ctx, []*model.Task{task})
	if err != nil {
		return &pb.CancelJobResponse{Err: &pb.Error{Message: err.Error()}}
	}
	err = j.removeJobMeta(ctx, &model.JobMaster{ID: model.ID(req.JobId)})
	if err != nil {
		log.L().Warn("failed to remove job master meta", zap.Error(err))
	}
	delete(j.jobMasters, model.ID(req.JobId))
	return &pb.CancelJobResponse{}
}

// SubmitJob processes "SubmitJobRequest".
func (j *JobManager) SubmitJob(ctx context.Context, req *pb.SubmitJobRequest) *pb.SubmitJobResponse {
	log.L().Logger.Info("submit job", zap.String("config", string(req.Config)))
	var (
		jobTask *model.Task
		jm      *model.JobMaster
		jmBytes []byte
		resp    = &pb.SubmitJobResponse{}
		err     error
	)
	switch req.Tp {
	case pb.SubmitJobRequest_Benchmark:
		id := j.idAllocater.AllocJobID()
		// TODO: supposing job master will be running independently, then the
		// addresses of server can change because of failover, the job master
		// should have ways to detect and adapt automatically.
		jm = &model.JobMaster{
			ID:          model.ID(id),
			Tp:          model.Benchmark,
			Config:      req.Config,
			MasterAddrs: j.masterAddrs,
		}
		jmBytes, err = json.Marshal(jm)
		if err != nil {
			resp.Err = errors.ToPBError(err)
			return resp
		}
		jobTask = &model.Task{
			ID:   model.ID(id),
			OpTp: model.JobMasterType,
			Op:   jmBytes,
			Cost: 1,
		}
	default:
		err := errors.ErrBuildJobFailed.GenWithStack("unknown job type", req.Tp)
		resp.Err = errors.ToPBError(err)
		return resp
	}

	err = j.storeJobMeta(ctx, jm, jmBytes)
	if err != nil {
		log.L().Error("store job meta failed", zap.Error(err))
		err := errors.ErrBuildJobFailed.GenWithStack("failed to store job meta")
		resp.Err = errors.ToPBError(err)
		return resp
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	j.jobMasters[jobTask.ID] = jobTask
	err = j.Master.DispatchTasks(ctx, []*model.Task{jobTask})
	if err != nil {
		// remove job metadata if dispatch tasks failed
		err2 := j.removeJobMeta(ctx, jm)
		if err2 != nil {
			log.L().Warn("failed to remove job meta", zap.Error(err2))
		}
		log.L().Error("failed to dispatch tasks", zap.Error(err))
		resp.Err = errors.ToPBError(err)
		return resp
	}
	log.L().Logger.Info("finished dispatch job")
	resp.JobId = int32(jobTask.ID)
	return resp
}

func (j *JobManager) storeJobMeta(ctx context.Context, jm *model.JobMaster, jmBytes []byte) error {
	_, err := j.etcdClient.Put(ctx, jm.EtcdKey(), string(jmBytes))
	return err
}

func (j *JobManager) removeJobMeta(ctx context.Context, jm *model.JobMaster) error {
	_, err := j.etcdClient.Delete(ctx, jm.EtcdKey())
	return err
}

func NewJobManager(
	masterAddrs []string,
) *JobManager {
	return &JobManager{
		jobMasters:  make(map[model.ID]*model.Task),
		idAllocater: autoid.NewJobIDAllocator(),
		clients:     client.NewClientManager(),
		masterAddrs: masterAddrs,
	}
}
