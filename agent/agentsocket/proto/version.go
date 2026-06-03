package proto

import "github.com/NeuralInverse/cloud/v2/apiversion"

// Version history:
//
// API v1.0:
//   - Initial release
//   - Ping
//   - Sync operations: SyncStart, SyncWant, SyncComplete, SyncWait, SyncStatus
//
// API v1.1:
//   - UpdateAppStatus RPC (forwarded to nicloud)

const (
	CurrentMajor = 1
	CurrentMinor = 1
)

var CurrentVersion = apiversion.New(CurrentMajor, CurrentMinor)
