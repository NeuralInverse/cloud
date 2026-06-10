import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import { type FC, useRef, useState } from "react";
import { Button } from "#/components/Button/Button";
import { API } from "#/api/api";
import type { DisplayApp } from "#/api/typesGenerated";
import { ChevronDownIcon } from "#/components/AnimatedIcons/ChevronDown";
import { ProductLogo } from "#/components/Icons/ProductLogo";
import { VSCodeInsidersIcon } from "#/components/Icons/VSCodeInsidersIcon";
import { getVSCodeHref } from "#/modules/apps/apps";
import { AgentButton } from "../AgentButton";
import { DisplayAppNameMap } from "../AppLink/AppLink";

interface VSCodeDesktopButtonProps {
	userName: string;
	workspaceName: string;
	agentName?: string;
	folderPath?: string;
	displayApps: readonly DisplayApp[];
}

type VSCodeVariant = "vscode" | "vscode-insiders";

const VARIANT_KEY = "vscode-variant";

export const VSCodeDesktopButton: FC<VSCodeDesktopButtonProps> = (props) => {
	const [isVariantMenuOpen, setIsVariantMenuOpen] = useState(false);
	const previousVariant = localStorage.getItem(VARIANT_KEY);
	const [variant, setVariant] = useState<VSCodeVariant>(() => {
		if (!previousVariant) {
			return "vscode";
		}
		return previousVariant as VSCodeVariant;
	});
	const menuAnchorRef = useRef<HTMLDivElement>(null);

	const selectVariant = (variant: VSCodeVariant) => {
		localStorage.setItem(VARIANT_KEY, variant);
		setVariant(variant);
		setIsVariantMenuOpen(false);
	};

	const includesVSCodeDesktop = props.displayApps.includes("vscode");
	const includesVSCodeInsiders = props.displayApps.includes("vscode_insiders");

	return includesVSCodeDesktop && includesVSCodeInsiders ? (
		<>
			<div ref={menuAnchorRef} className="flex items-center gap-1">
				{variant === "vscode" ? (
					<VSCodeButton {...props} />
				) : (
					<VSCodeInsidersButton {...props} />
				)}

				<AgentButton
					aria-controls={
						isVariantMenuOpen ? "vscode-variant-button-menu" : undefined
					}
					aria-expanded={isVariantMenuOpen ? "true" : undefined}
					aria-label="select VSCode variant"
					aria-haspopup="menu"
					onClick={() => {
						setIsVariantMenuOpen(true);
					}}
					size="icon-lg"
				>
					<ChevronDownIcon open={isVariantMenuOpen} />
				</AgentButton>
			</div>

			<Menu
				open={isVariantMenuOpen}
				anchorEl={menuAnchorRef.current}
				onClose={() => setIsVariantMenuOpen(false)}
				css={{
					"& .MuiMenu-paper": {
						width: menuAnchorRef.current?.clientWidth,
					},
				}}
			>
				<MenuItem
					className="text-sm"
					onClick={() => {
						selectVariant("vscode");
					}}
				>
					<ProductLogo className="w-3 h-3" />
					{DisplayAppNameMap.vscode}
				</MenuItem>
				<MenuItem
					className="text-sm"
					onClick={() => {
						selectVariant("vscode-insiders");
					}}
				>
					<VSCodeInsidersIcon className="w-3 h-3" />
					{DisplayAppNameMap.vscode_insiders}
				</MenuItem>
			</Menu>
		</>
	) : includesVSCodeDesktop ? (
		<VSCodeButton {...props} />
	) : (
		<VSCodeInsidersButton {...props} />
	);
};

const VSCodeButton: FC<VSCodeDesktopButtonProps> = ({
	userName,
	workspaceName,
	agentName,
	folderPath,
}) => {
	const [loading, setLoading] = useState(false);
	const [showDownload, setShowDownload] = useState(false);

	function handleClick() {
		setLoading(true);
		setShowDownload(false);
		API.getApiKey()
			.then(({ key }) => {
				const href = getVSCodeHref("neuralinverse", {
					owner: userName,
					workspace: workspaceName,
					token: key,
					agent: agentName,
					folder: folderPath,
				});
				location.href = href;
				// If the app isn't installed the browser stays on the page
				// Show download prompt after 2.5s if still here
				setTimeout(() => {
					setShowDownload(true);
					setLoading(false);
				}, 2500);
			})
			.catch(() => {
				setLoading(false);
			});
	}

	return (
		<div className="flex flex-col items-start gap-2">
			<AgentButton disabled={loading} onClick={handleClick}>
				<ProductLogo className="w-4 h-4" />
				{DisplayAppNameMap.vscode}
			</AgentButton>
			{showDownload && (
				<div
					className="flex items-center gap-3 px-3 py-2 text-xs"
					style={{ background: "#202020", border: "1px solid #2b2b2b" }}
				>
					<span className="text-content-secondary">Neural Inverse IDE not found.</span>
					<Button
						size="sm"
						variant="outline"
						asChild
					>
						<a href="https://neuralinverse.com/download" target="_blank" rel="noreferrer">
							Download
						</a>
					</Button>
				</div>
			)}
		</div>
	);
};

const VSCodeInsidersButton: FC<VSCodeDesktopButtonProps> = ({
	userName,
	workspaceName,
	agentName,
	folderPath,
}) => {
	const [loading, setLoading] = useState(false);

	return (
		<AgentButton
			disabled={loading}
			onClick={() => {
				setLoading(true);
				API.getApiKey()
					.then(({ key }) => {
						location.href = getVSCodeHref("vscode-insiders", {
							owner: userName,
							workspace: workspaceName,
							token: key,
							agent: agentName,
							folder: folderPath,
						});
					})
					.catch((ex) => {
						console.error(ex);
					})
					.finally(() => {
						setLoading(false);
					});
			}}
		>
			<VSCodeInsidersIcon />
			{DisplayAppNameMap.vscode_insiders}
		</AgentButton>
	);
};
