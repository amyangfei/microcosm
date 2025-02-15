package registry

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hanfei1991/microcosm/lib"
	"github.com/hanfei1991/microcosm/lib/fake"
	dcontext "github.com/hanfei1991/microcosm/pkg/context"
)

var (
	_                 WorkerFactory = (*SimpleWorkerFactory)(nil)
	fakeWorkerFactory WorkerFactory = NewSimpleWorkerFactory(fake.NewDummyWorker, &fake.WorkerConfig{})
)

const (
	fakeWorkerType = lib.WorkerType(100)
)

func TestGlobalRegistry(t *testing.T) {
	GlobalWorkerRegistry().MustRegisterWorkerType(fakeWorkerType, fakeWorkerFactory)

	worker, err := GlobalWorkerRegistry().CreateWorker(
		makeCtxWithMockDeps(t),
		fakeWorkerType,
		"worker-1",
		"master-1",
		[]byte(`{"target-tick":10}`))
	require.NoError(t, err)
	require.IsType(t, &lib.DefaultBaseWorker{}, worker)
	impl := worker.(*lib.DefaultBaseWorker).Impl
	require.IsType(t, &fake.Worker{}, impl)
	require.NotNil(t, impl.(*fake.Worker).BaseWorker)
	require.Equal(t, "worker-1", worker.ID())
}

func TestRegistryDuplicateType(t *testing.T) {
	registry := NewRegistry()
	ok := registry.RegisterWorkerType(fakeWorkerType, fakeWorkerFactory)
	require.True(t, ok)

	ok = registry.RegisterWorkerType(fakeWorkerType, fakeWorkerFactory)
	require.False(t, ok)

	require.Panics(t, func() {
		registry.MustRegisterWorkerType(fakeWorkerType, fakeWorkerFactory)
	})
}

func TestRegistryWorkerTypeNotFound(t *testing.T) {
	registry := NewRegistry()
	ctx := dcontext.Background()
	_, err := registry.CreateWorker(ctx, fakeWorkerType, "worker-1", "master-1", []byte(`{"Val"":0}`))
	require.Error(t, err)
}

func TestLoadFake(t *testing.T) {
	registry := NewRegistry()
	require.NotPanics(t, func() {
		RegisterFake(registry)
	})
}

func TestGetTypeNameOfVarPtr(t *testing.T) {
	t.Parallel()

	type myType struct{}

	var (
		a int
		b myType
	)
	require.Equal(t, "int", getTypeNameOfVarPtr(&a))
	require.Equal(t, "myType", getTypeNameOfVarPtr(&b))
}

func TestImplHasMember(t *testing.T) {
	t.Parallel()

	type myImpl struct {
		MyMember int
	}
	type myIface interface{}

	var iface myIface = &myImpl{}
	require.True(t, implHasMember(iface, "MyMember"))
	require.False(t, implHasMember(iface, "notFound"))
}

func TestSetImplMember(t *testing.T) {
	t.Parallel()

	type MyBase interface{}

	type myImpl struct {
		MyBase
	}
	type myImplIface interface{}

	var iface myImplIface = &myImpl{}

	setImplMember(iface, "MyBase", 2)
	require.Equal(t, 2, iface.(*myImpl).MyBase.(int))
}
