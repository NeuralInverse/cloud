package database

import (
	"github.com/google/uuid"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// PrebuildsSystemUserID mirrors nicloudsdk.PrebuildsSystemUserID, parsed
// for use as a uuid.UUID. Both must agree; tests pin the value to the
// nicloudsdk constant so the two cannot drift.
var PrebuildsSystemUserID = uuid.MustParse(nicloudsdk.PrebuildsSystemUserID)
