import type React from "react";
import type { FC } from "react";
import type { Workspace } from "#/api/typesGenerated";
import {
	StatusIndicator,
	StatusIndicatorDot,
	type StatusIndicatorProps,
} from "#/components/StatusIndicator/StatusIndicator";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "#/components/Tooltip/Tooltip";
import {
	type DisplayWorkspaceStatusType,
	getDisplayWorkspaceStatus,
} from "#/utils/workspace";

const variantByStatusType: Record<
	DisplayWorkspaceStatusType,
	StatusIndicatorProps["variant"]
> = {
	active: "pending",
	inactive: "inactive",
	success: "success",
	error: "failed",
	danger: "warning",
	warning: "warning",
};

type WorkspaceStatusIndicatorProps = {
	workspace: Workspace;
	children?: React.ReactNode;
};

export const WorkspaceStatusIndicator: FC<WorkspaceStatusIndicatorProps> = ({
	workspace,
	children,
}) => {
	let { text, type } = getDisplayWorkspaceStatus(
		workspace.latest_build.status,
		workspace.latest_build.job,
	);

	// Don't show the warning while workspace is still starting or agents are still connecting
	const buildStatus = workspace.latest_build.status;
	const agentsConnecting = workspace.latest_build.resources?.some((r) =>
		r.agents?.some((a) => a.status === "connecting"),
	);
	const stillStarting = buildStatus === "starting" || agentsConnecting;

	if (!workspace.health.healthy && !stillStarting) {
		type = "warning";
	}

	const statusIndicator = (
		<StatusIndicator variant={variantByStatusType[type]}>
			<StatusIndicatorDot />
			<span className="sr-only">Workspace status:</span> {text}
			{children}
		</StatusIndicator>
	);

	if (workspace.health.healthy || stillStarting) {
		return statusIndicator;
	}

	return (
		<Tooltip>
			<TooltipTrigger asChild>
				<StatusIndicator variant={variantByStatusType[type]}>
					<StatusIndicatorDot />
					<span className="sr-only">Workspace status:</span> {text}
					{children}
				</StatusIndicator>
			</TooltipTrigger>
			<TooltipContent>
				One or more workspace agents need attention. Expand an agent's logs for
				details.
			</TooltipContent>
		</Tooltip>
	);
};
