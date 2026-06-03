package nicloudtest

import (
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbmock"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
)

func MockedDatabaseWithAuthz(t testing.TB, logger slog.Logger) (*gomock.Controller, *dbmock.MockStore, database.Store, rbac.Authorizer) {
	ctrl := gomock.NewController(t)
	mDB := dbmock.NewMockStore(ctrl)
	auth := rbac.NewStrictCachingAuthorizer(prometheus.NewRegistry())
	accessControlStore := &atomic.Pointer[dbauthz.AccessControlStore]{}
	var acs dbauthz.AccessControlStore = dbauthz.AGPLTemplateAccessControlStore{}
	accessControlStore.Store(&acs)
	// dbauthz will call Wrappers() to check for wrapped databases
	mDB.EXPECT().Wrappers().Return([]string{}).AnyTimes()
	authDB := dbauthz.New(mDB, auth, logger, accessControlStore)
	return ctrl, mDB, authDB, auth
}
