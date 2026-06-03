// Package sdk2db provides common conversion routines from nicloudsdk types to database types
package sdk2db

import (
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func ProvisionerDaemonStatus(status nicloudsdk.ProvisionerDaemonStatus) database.ProvisionerDaemonStatus {
	return database.ProvisionerDaemonStatus(status)
}

func ProvisionerDaemonStatuses(params []nicloudsdk.ProvisionerDaemonStatus) []database.ProvisionerDaemonStatus {
	return slice.List(params, ProvisionerDaemonStatus)
}
