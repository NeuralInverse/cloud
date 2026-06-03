package nicloud

import "github.com/NeuralInverse/cloud/v2/nicloud/x/chatd"

// ChatDaemonForTest returns the background chat processor for test harnesses.
func (api *API) ChatDaemonForTest() *chatd.Server {
	return api.chatDaemon
}
