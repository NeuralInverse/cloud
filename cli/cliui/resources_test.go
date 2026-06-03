package cliui_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/pty/ptytest"
)

func TestWorkspaceResources(t *testing.T) {
	t.Parallel()
	t.Run("SingleAgentSSH", func(t *testing.T) {
		t.Parallel()
		ptty := ptytest.New(t)
		done := make(chan struct{})
		go func() {
			err := cliui.WorkspaceResources(ptty.Output(), []nicloudsdk.WorkspaceResource{{
				Type:       "google_compute_instance",
				Name:       "dev",
				Transition: nicloudsdk.WorkspaceTransitionStart,
				Agents: []nicloudsdk.WorkspaceAgent{{
					Name:            "dev",
					Status:          nicloudsdk.WorkspaceAgentConnected,
					LifecycleState:  nicloudsdk.WorkspaceAgentLifecycleCreated,
					Architecture:    "amd64",
					OperatingSystem: "linux",
					Health:          nicloudsdk.WorkspaceAgentHealth{Healthy: true},
				}},
			}}, cliui.WorkspaceResourcesOptions{
				WorkspaceName: "example",
			})
			assert.NoError(t, err)
			close(done)
		}()
		ptty.ExpectMatch("neuralinverse ssh example")
		<-done
	})

	t.Run("MultipleStates", func(t *testing.T) {
		t.Parallel()
		ptty := ptytest.New(t)
		disconnected := dbtime.Now().Add(-4 * time.Second)
		done := make(chan struct{})
		go func() {
			err := cliui.WorkspaceResources(ptty.Output(), []nicloudsdk.WorkspaceResource{{
				Transition: nicloudsdk.WorkspaceTransitionStart,
				Type:       "google_compute_disk",
				Name:       "root",
			}, {
				Transition: nicloudsdk.WorkspaceTransitionStop,
				Type:       "google_compute_disk",
				Name:       "root",
			}, {
				Transition: nicloudsdk.WorkspaceTransitionStart,
				Type:       "google_compute_instance",
				Name:       "dev",
				Agents: []nicloudsdk.WorkspaceAgent{{
					CreatedAt:       dbtime.Now().Add(-10 * time.Second),
					Status:          nicloudsdk.WorkspaceAgentConnecting,
					LifecycleState:  nicloudsdk.WorkspaceAgentLifecycleCreated,
					Name:            "dev",
					OperatingSystem: "linux",
					Architecture:    "amd64",
					Health:          nicloudsdk.WorkspaceAgentHealth{Healthy: true},
				}},
			}, {
				Transition: nicloudsdk.WorkspaceTransitionStart,
				Type:       "kubernetes_pod",
				Name:       "dev",
				Agents: []nicloudsdk.WorkspaceAgent{{
					Status:          nicloudsdk.WorkspaceAgentConnected,
					LifecycleState:  nicloudsdk.WorkspaceAgentLifecycleReady,
					Name:            "go",
					Architecture:    "amd64",
					OperatingSystem: "linux",
					Health:          nicloudsdk.WorkspaceAgentHealth{Healthy: true},
				}, {
					DisconnectedAt:  &disconnected,
					Status:          nicloudsdk.WorkspaceAgentDisconnected,
					LifecycleState:  nicloudsdk.WorkspaceAgentLifecycleReady,
					Name:            "postgres",
					Architecture:    "amd64",
					OperatingSystem: "linux",
					Health: nicloudsdk.WorkspaceAgentHealth{
						Healthy: false,
						Reason:  "agent has lost connection",
					},
				}},
			}}, cliui.WorkspaceResourcesOptions{
				WorkspaceName:  "dev",
				HideAgentState: false,
				HideAccess:     false,
			})
			assert.NoError(t, err)
			close(done)
		}()
		ptty.ExpectMatch("google_compute_disk.root")
		ptty.ExpectMatch("google_compute_instance.dev")
		ptty.ExpectMatch("healthy")
		ptty.ExpectMatch("neuralinverse ssh dev.dev")
		ptty.ExpectMatch("kubernetes_pod.dev")
		ptty.ExpectMatch("healthy")
		ptty.ExpectMatch("neuralinverse ssh dev.go")
		ptty.ExpectMatch("agent has lost connection")
		ptty.ExpectMatch("neuralinverse ssh dev.postgres")
		<-done
	})
}
