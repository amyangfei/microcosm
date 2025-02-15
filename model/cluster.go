package model

import (
	"encoding/json"

	"github.com/hanfei1991/microcosm/pkg/adapter"
)

// NodeType is the type of a server instance, could be either server master or executor
type NodeType int

// All node types
const (
	NodeTypeServerMaster NodeType = iota + 1
	NodeTypeExecutor
)

// RescUnit is the min unit of resource that we count.
type RescUnit int

// DeployNodeID means the identify of a node
type DeployNodeID string

type ExecutorID = DeployNodeID

// NodeInfo describes the information of server instance, contains node type, node
// uuid, advertise address and capability(executor node only)
type NodeInfo struct {
	Type NodeType     `json:"type"`
	ID   DeployNodeID `json:"id"`
	Addr string       `json:"addr"`

	// The capability of executor, including
	// 1. cpu (goroutines)
	// 2. memory
	// 3. disk cap
	// TODO: So we should enrich the cap dimensions in the future.
	Capability int `json:"cap"`
}

func (e *NodeInfo) EtcdKey() string {
	return adapter.NodeInfoKeyAdapter.Encode(string(e.ID))
}

type ExecutorStatus int32

const (
	Initing ExecutorStatus = iota
	Running
	Disconnected
	Tombstone
)

var ExecutorStatusNameMapping = map[ExecutorStatus]string{
	Initing:      "initializing",
	Running:      "running",
	Disconnected: "disconnected",
	Tombstone:    "tombstone",
}

// String implements fmt.Stringer
func (s ExecutorStatus) String() string {
	val, ok := ExecutorStatusNameMapping[s]
	if !ok {
		return "unknown"
	}
	return val
}

func (e *NodeInfo) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
