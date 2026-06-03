// package dbmock contains a mocked implementation of the database.Store interface for use in tests
package dbmock

//go:generate go tool mockgen -destination ./dbmock.go -package dbmock github.com/NeuralInverse/cloud/v2/nicloud/database Store
