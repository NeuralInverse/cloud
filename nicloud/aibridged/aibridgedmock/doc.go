package aibridgedmock

//go:generate go tool mockgen -destination ./clientmock.go -package aibridgedmock github.com/NeuralInverse/cloud/v2/nicloud/aibridged DRPCClient
//go:generate go tool mockgen -destination ./poolmock.go -package aibridgedmock github.com/NeuralInverse/cloud/v2/nicloud/aibridged Pooler
