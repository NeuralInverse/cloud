import type { WorkspaceAgent } from "#/api/typesGenerated";

/**
 * Canonical messages for startup and shutdown script issues.
 * Used by the per-agent-row tooltips in AgentStatus; the
 * start-related entries are also shared with per-agent health
 * classification in getAgentHealthIssues.
 */
export const agentScriptMessages = {
	start_error: {
		title: "Startup script failed",
		detail:
			"A startup script exited with an error. Check the agent logs for details.",
	},
	start_timeout: {
		title: "Startup script is taking longer than expected",
		detail:
			"A startup script has exceeded the expected time. Check the agent logs for details.",
	},
	shutdown_error: {
		title: "Shutdown script failed",
		detail:
			"A shutdown script exited with an error. Check the agent logs for details.",
	},
	shutdown_timeout: {
		title: "Shutdown script is taking longer than expected",
		detail:
			"A shutdown script has exceeded the expected time. Check the agent logs for details.",
	},
} as const;

/**
 * Canonical messages for agent connection issues (the agent
 * process connecting to the Coder control plane).
 */
export const agentConnectionMessages = {
	connecting: {
		title: "Setting up your workspace",
		detail:
			"Connecting to your region. This usually takes 1 to 3 minutes for the first launch. Hang tight.",
	},
	timeout: {
		title: "Still connecting, this is taking longer than usual",
		detail:
			"Your workspace is still starting up. This can happen with cold starts in distant regions. Give it another minute or try restarting if it stays stuck.",
	},
	disconnected: {
		title: "Your workspace lost its connection",
		detail:
			"The workspace disconnected unexpectedly. Try restarting it to reconnect.",
	},
} as const;

interface AgentHealthIssue {
	title: string;
	detail: string;
	severity: "info" | "warning";
	// Whether the alert should be visually prominent. Usually true for
	// warnings, but connection timeout and startup timeout are
	// exceptions (warning severity without prominent styling).
	prominent: boolean;
	kind?: "connecting";
}

/**
 * Classifies all health issues for an individual agent.
 */
export function getAgentHealthIssues(
	agent: WorkspaceAgent,
): AgentHealthIssue[] {
	const issues: AgentHealthIssue[] = [];

	if (agent.status === "disconnected") {
		issues.push({
			title: agentConnectionMessages.disconnected.title,
			detail: agentConnectionMessages.disconnected.detail,
			severity: "warning",
			prominent: false,
		});
	}

	if (agent.status === "timeout") {
		issues.push({
			title: agentConnectionMessages.timeout.title,
			detail: agentConnectionMessages.timeout.detail,
			severity: "warning",
			prominent: false,
		});
	}

	if (
		agent.lifecycle_state === "shutting_down" ||
		agent.lifecycle_state === "shutdown_error" ||
		agent.lifecycle_state === "shutdown_timeout"
	) {
		issues.push({
			title: "Workspace agent is shutting down",
			detail: "The workspace is not available while agents shut down.",
			severity: "info",
			prominent: false,
		});
	}

	// Ignore `start_error` and `start_timeout`, as these will eventually be
	// removed from agent health.  Instead, figure out if a script failed to start
	// by looking directly at the scripts.
	for (const script of agent.scripts) {
		switch (script.status) {
			case "timed_out":
				issues.push({
					title: `"${script.display_name}" is taking longer than expected`,
					detail: `"${script.display_name}" has exceeded the expected time. Check the agent logs for details.`,
					severity: "warning",
					prominent: false,
				});
				break;
			case "exit_failure":
				if (script.exit_code) {
					issues.push({
						title: `"${script.display_name}" failed`,
						detail: `"${script.display_name}" exited with ${script.exit_code}. Check the agent logs for details.`,
						severity: "warning",
						prominent: false,
					});
				} else {
					issues.push({
						title: `"${script.display_name}" failed`,
						detail: `"${script.display_name}" has exited with an error. Check the agent logs for details.`,
						severity: "warning",
						prominent: false,
					});
				}
				break;
			case "pipes_left_open":
				issues.push({
					title: `"${script.display_name}" left pipes open`,
					detail: "Check the agent logs for details.",
					severity: "warning",
					prominent: false,
				});
				break;
		}
	}

	if (agent.status === "connecting") {
		issues.push({
			title: agentConnectionMessages.connecting.title,
			detail: agentConnectionMessages.connecting.detail,
			severity: "info",
			prominent: false,
			kind: "connecting",
		});
	}

	return issues;
}
