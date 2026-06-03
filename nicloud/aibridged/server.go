package aibridged

import "github.com/NeuralInverse/cloud/v2/nicloud/aibridged/proto"

type DRPCServer interface {
	proto.DRPCRecorderServer
	proto.DRPCMCPConfiguratorServer
	proto.DRPCAuthorizerServer
}
