package nicloud

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestDeriveTaskCurrentState_Unit(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name               string
		task               database.Task
		agentLifecycle     *nicloudsdk.WorkspaceAgentLifecycle
		appHealth          *nicloudsdk.WorkspaceAppHealth
		latestAppStatus    *nicloudsdk.WorkspaceAppStatus
		latestBuild        nicloudsdk.WorkspaceBuild
		expectCurrentState bool
		expectedTimestamp  time.Time
		expectedState      nicloudsdk.TaskState
		expectedMessage    string
	}{
		{
			name: "NoAppStatus",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusActive,
			},
			agentLifecycle:  nil,
			appHealth:       nil,
			latestAppStatus: nil,
			latestBuild: nicloudsdk.WorkspaceBuild{
				Transition: nicloudsdk.WorkspaceTransitionStart,
				CreatedAt:  now,
			},
			expectCurrentState: false,
		},
		{
			name: "BuildStartTransition_AppStatus_NewerThanBuild",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusActive,
			},
			agentLifecycle: nil,
			appHealth:      nil,
			latestAppStatus: &nicloudsdk.WorkspaceAppStatus{
				State:     nicloudsdk.WorkspaceAppStatusStateWorking,
				Message:   "Task is working",
				CreatedAt: now.Add(1 * time.Minute),
			},
			latestBuild: nicloudsdk.WorkspaceBuild{
				Transition: nicloudsdk.WorkspaceTransitionStart,
				CreatedAt:  now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now.Add(1 * time.Minute),
			expectedState:      nicloudsdk.TaskState(nicloudsdk.WorkspaceAppStatusStateWorking),
			expectedMessage:    "Task is working",
		},
		{
			name: "BuildStartTransition_StaleAppStatus_OlderThanBuild",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusActive,
			},
			agentLifecycle: nil,
			appHealth:      nil,
			latestAppStatus: &nicloudsdk.WorkspaceAppStatus{
				State:     nicloudsdk.WorkspaceAppStatusStateComplete,
				Message:   "Previous task completed",
				CreatedAt: now.Add(-1 * time.Minute),
			},
			latestBuild: nicloudsdk.WorkspaceBuild{
				Transition: nicloudsdk.WorkspaceTransitionStart,
				CreatedAt:  now,
			},
			expectCurrentState: false,
		},
		{
			name: "BuildStopTransition",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusActive,
			},
			agentLifecycle: nil,
			appHealth:      nil,
			latestAppStatus: &nicloudsdk.WorkspaceAppStatus{
				State:     nicloudsdk.WorkspaceAppStatusStateComplete,
				Message:   "Task completed before stop",
				CreatedAt: now.Add(-1 * time.Minute),
			},
			latestBuild: nicloudsdk.WorkspaceBuild{
				Transition: nicloudsdk.WorkspaceTransitionStop,
				CreatedAt:  now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now.Add(-1 * time.Minute),
			expectedState:      nicloudsdk.TaskState(nicloudsdk.WorkspaceAppStatusStateComplete),
			expectedMessage:    "Task completed before stop",
		},
		{
			name: "TaskInitializing_WorkspacePending",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusInitializing,
			},
			agentLifecycle:  nil,
			appHealth:       nil,
			latestAppStatus: nil,
			latestBuild: nicloudsdk.WorkspaceBuild{
				Status:    nicloudsdk.WorkspaceStatusPending,
				CreatedAt: now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now,
			expectedState:      nicloudsdk.TaskStateWorking,
			expectedMessage:    "Workspace is pending",
		},
		{
			name: "TaskInitializing_WorkspaceStarting",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusInitializing,
			},
			agentLifecycle:  nil,
			appHealth:       nil,
			latestAppStatus: nil,
			latestBuild: nicloudsdk.WorkspaceBuild{
				Status:    nicloudsdk.WorkspaceStatusStarting,
				CreatedAt: now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now,
			expectedState:      nicloudsdk.TaskStateWorking,
			expectedMessage:    "Workspace is starting",
		},
		{
			name: "TaskInitializing_AgentConnecting",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusInitializing,
			},
			agentLifecycle:  ptr.Ref(nicloudsdk.WorkspaceAgentLifecycleCreated),
			appHealth:       nil,
			latestAppStatus: nil,
			latestBuild: nicloudsdk.WorkspaceBuild{
				Status:    nicloudsdk.WorkspaceStatusRunning,
				CreatedAt: now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now,
			expectedState:      nicloudsdk.TaskStateWorking,
			expectedMessage:    "Agent is connecting",
		},
		{
			name: "TaskInitializing_AgentStarting",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusInitializing,
			},
			agentLifecycle:  ptr.Ref(nicloudsdk.WorkspaceAgentLifecycleStarting),
			appHealth:       nil,
			latestAppStatus: nil,
			latestBuild: nicloudsdk.WorkspaceBuild{
				Status:    nicloudsdk.WorkspaceStatusRunning,
				CreatedAt: now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now,
			expectedState:      nicloudsdk.TaskStateWorking,
			expectedMessage:    "Agent is starting",
		},
		{
			name: "TaskInitializing_AppInitializing",
			task: database.Task{
				ID:     uuid.New(),
				Status: database.TaskStatusInitializing,
			},
			agentLifecycle:  ptr.Ref(nicloudsdk.WorkspaceAgentLifecycleReady),
			appHealth:       ptr.Ref(nicloudsdk.WorkspaceAppHealthInitializing),
			latestAppStatus: nil,
			latestBuild: nicloudsdk.WorkspaceBuild{
				Status:    nicloudsdk.WorkspaceStatusRunning,
				CreatedAt: now,
			},
			expectCurrentState: true,
			expectedTimestamp:  now,
			expectedState:      nicloudsdk.TaskStateWorking,
			expectedMessage:    "App is initializing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ws := nicloudsdk.Workspace{
				LatestBuild:     tt.latestBuild,
				LatestAppStatus: tt.latestAppStatus,
			}

			currentState := deriveTaskCurrentState(tt.task, ws, tt.agentLifecycle, tt.appHealth)

			if tt.expectCurrentState {
				require.NotNil(t, currentState)
				assert.Equal(t, tt.expectedTimestamp.UTC(), currentState.Timestamp.UTC())
				assert.Equal(t, tt.expectedState, currentState.State)
				assert.Equal(t, tt.expectedMessage, currentState.Message)
			} else {
				assert.Nil(t, currentState)
			}
		})
	}
}
