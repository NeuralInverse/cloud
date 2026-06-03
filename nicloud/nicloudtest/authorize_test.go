package nicloudtest_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
)

func TestAuthzRecorder(t *testing.T) {
	t.Parallel()

	t.Run("Authorize", func(t *testing.T) {
		t.Parallel()

		rec := &nicloudtest.RecordingAuthorizer{
			Wrapped: &nicloudtest.FakeAuthorizer{},
		}
		sub := nicloudtest.RandomRBACSubject()
		pairs := fuzzAuthz(t, sub, rec, 10)
		rec.AssertActor(t, sub, pairs...)
		require.NoError(t, rec.AllAsserted(), "all assertions should have been made")
	})

	t.Run("Authorize2Subjects", func(t *testing.T) {
		t.Parallel()

		rec := &nicloudtest.RecordingAuthorizer{
			Wrapped: &nicloudtest.FakeAuthorizer{},
		}
		a := nicloudtest.RandomRBACSubject()
		aPairs := fuzzAuthz(t, a, rec, 10)

		b := nicloudtest.RandomRBACSubject()
		bPairs := fuzzAuthz(t, b, rec, 10)

		rec.AssertActor(t, b, bPairs...)
		rec.AssertActor(t, a, aPairs...)
		require.NoError(t, rec.AllAsserted(), "all assertions should have been made")
	})

	t.Run("Authorize_Prepared", func(t *testing.T) {
		t.Parallel()

		rec := &nicloudtest.RecordingAuthorizer{
			Wrapped: &nicloudtest.FakeAuthorizer{},
		}
		a := nicloudtest.RandomRBACSubject()
		aPairs := fuzzAuthz(t, a, rec, 10)

		b := nicloudtest.RandomRBACSubject()

		act, objTy := nicloudtest.RandomRBACAction(), nicloudtest.RandomRBACObject().Type
		prep, _ := rec.Prepare(context.Background(), b, act, objTy)
		bPairs := fuzzAuthzPrep(t, prep, 10, act, objTy)

		rec.AssertActor(t, b, bPairs...)
		rec.AssertActor(t, a, aPairs...)
		require.NoError(t, rec.AllAsserted(), "all assertions should have been made")
	})

	t.Run("AuthorizeOutOfOrder", func(t *testing.T) {
		t.Parallel()

		rec := &nicloudtest.RecordingAuthorizer{
			Wrapped: &nicloudtest.FakeAuthorizer{},
		}
		sub := nicloudtest.RandomRBACSubject()
		pairs := fuzzAuthz(t, sub, rec, 10)
		rand.Shuffle(len(pairs), func(i, j int) {
			pairs[i], pairs[j] = pairs[j], pairs[i]
		})

		rec.AssertOutOfOrder(t, sub, pairs...)
		require.NoError(t, rec.AllAsserted(), "all assertions should have been made")
	})

	t.Run("AllCalls", func(t *testing.T) {
		t.Parallel()

		rec := &nicloudtest.RecordingAuthorizer{
			Wrapped: &nicloudtest.FakeAuthorizer{},
		}
		sub := nicloudtest.RandomRBACSubject()
		calls := rec.AllCalls(&sub)
		pairs := make([]nicloudtest.ActionObjectPair, 0, len(calls))
		for _, call := range calls {
			pairs = append(pairs, nicloudtest.ActionObjectPair{
				Action: call.Action,
				Object: call.Object,
			})
		}

		rec.AssertActor(t, sub, pairs...)
		require.NoError(t, rec.AllAsserted(), "all assertions should have been made")
	})
}

// fuzzAuthzPrep has same action and object types for all calls.
func fuzzAuthzPrep(t *testing.T, prep rbac.PreparedAuthorized, n int, action policy.Action, objectType string) []nicloudtest.ActionObjectPair {
	t.Helper()
	pairs := make([]nicloudtest.ActionObjectPair, 0, n)

	for i := 0; i < n; i++ {
		obj := nicloudtest.RandomRBACObject()
		obj.Type = objectType
		p := nicloudtest.ActionObjectPair{Action: action, Object: obj}
		_ = prep.Authorize(context.Background(), p.Object)
		pairs = append(pairs, p)
	}
	return pairs
}

func fuzzAuthz(t *testing.T, sub rbac.Subject, rec rbac.Authorizer, n int) []nicloudtest.ActionObjectPair {
	t.Helper()
	pairs := make([]nicloudtest.ActionObjectPair, 0, n)

	for i := 0; i < n; i++ {
		p := nicloudtest.ActionObjectPair{Action: nicloudtest.RandomRBACAction(), Object: nicloudtest.RandomRBACObject()}
		_ = rec.Authorize(context.Background(), sub, p.Action, p.Object)
		pairs = append(pairs, p)
	}
	return pairs
}
