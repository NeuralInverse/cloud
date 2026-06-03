// package psmock contains a mocked implementation of the pubsub.Pubsub interface for use in tests
package psmock

//go:generate go tool mockgen -destination ./psmock.go -package psmock github.com/NeuralInverse/cloud/v2/nicloud/database/pubsub Pubsub
