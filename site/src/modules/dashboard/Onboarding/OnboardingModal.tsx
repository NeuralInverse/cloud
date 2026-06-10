import { type FC, useState } from "react";
import { Button } from "#/components/Button/Button";
import { ProductLogo } from "#/components/Icons/ProductLogo";

const ONBOARDING_KEY = "ni_onboarding_done";

function isDone(userId: string): boolean {
	try {
		return localStorage.getItem(`${ONBOARDING_KEY}_${userId}`) === "1";
	} catch {
		return false;
	}
}

function markDone(userId: string) {
	try {
		localStorage.setItem(`${ONBOARDING_KEY}_${userId}`, "1");
	} catch {}
}

type Step = "welcome" | "choose" | "cloud" | "local";

interface OnboardingModalProps {
	userId: string;
	userName: string;
	onDone: () => void;
}

export const OnboardingModal: FC<OnboardingModalProps> = ({
	userId,
	userName,
	onDone,
}) => {
	const [step, setStep] = useState<Step>("welcome");

	function finish() {
		markDone(userId);
		onDone();
	}

	return (
		<div
			className="fixed inset-0 z-50 flex items-center justify-center px-4"
			style={{ background: "rgba(0,0,0,0.82)" }}
		>
			<div
				className="w-full max-w-md flex flex-col"
				style={{ background: "#202020", border: "1px solid #2b2b2b" }}
			>
				{/* Header */}
				<div
					className="flex items-center justify-between px-6 py-4"
					style={{ borderBottom: "1px solid #2b2b2b" }}
				>
					<ProductLogo className="h-6" />
					<button
						type="button"
						onClick={finish}
						className="text-xs text-content-disabled hover:text-content-secondary bg-transparent border-none cursor-pointer"
					>
						Skip
					</button>
				</div>

				{/* Body */}
				<div className="px-6 py-8 flex flex-col gap-6">
					{step === "welcome" && (
						<WelcomeStep name={userName} onNext={() => setStep("choose")} />
					)}
					{step === "choose" && (
						<ChooseStep
							onCloud={() => setStep("cloud")}
							onLocal={() => setStep("local")}
						/>
					)}
					{step === "cloud" && (
						<CloudStep onDone={finish} />
					)}
					{step === "local" && (
						<LocalStep onDone={finish} />
					)}
				</div>

				{/* Step indicator */}
				<div
					className="flex items-center justify-center gap-2 px-6 py-4"
					style={{ borderTop: "1px solid #2b2b2b" }}
				>
					{(["welcome", "choose", step === "cloud" ? "cloud" : "local"] as Step[]).map((s, i) => {
						const steps: Step[] = ["welcome", "choose", step === "cloud" ? "cloud" : "local"];
						const current = steps.indexOf(step);
						return (
							<div
								key={i}
								className="h-1 rounded-none transition-all"
								style={{
									width: i === current ? 24 : 8,
									background: i === current ? "#358DF6" : "#2b2b2b",
								}}
							/>
						);
					})}
				</div>
			</div>
		</div>
	);
};

const WelcomeStep: FC<{ name: string; onNext: () => void }> = ({ name, onNext }) => (
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				Welcome to Neural Inverse Cloud{name ? `, ${name.split(" ")[0]}` : ""}
			</h2>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				Neural Inverse is an AI-native IDE platform built for regulated and
				critical software development — firmware, embedded systems, and beyond.
			</p>
		</div>
		<ul className="flex flex-col gap-3 m-0 p-0 list-none">
			{[
				["Cloud Workspaces", "Full dev environments in your browser, zero setup"],
				["Local IDE", "Download Neural Inverse IDE for macOS, Linux, or Windows"],
				["AI Inference", "Access Anthropic, OpenAI and more from your workspace"],
			].map(([title, desc]) => (
				<li key={title} className="flex items-start gap-3">
					<div
						className="mt-1 shrink-0 size-1.5 rounded-full"
						style={{ background: "#358DF6" }}
					/>
					<div className="flex flex-col gap-0.5">
						<span className="text-xs font-medium text-content-primary">{title}</span>
						<span className="text-xs text-content-secondary">{desc}</span>
					</div>
				</li>
			))}
		</ul>
		<Button className="w-full" onClick={onNext}>
			Get started &rarr;
		</Button>
	</div>
);

const ChooseStep: FC<{ onCloud: () => void; onLocal: () => void }> = ({ onCloud, onLocal }) => (
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				How would you like to work?
			</h2>
			<p className="text-sm text-content-secondary m-0">
				You can always switch between both options later.
			</p>
		</div>
		<div className="flex flex-col gap-3">
			<button
				type="button"
				onClick={onCloud}
				className="flex flex-col gap-1 p-4 text-left cursor-pointer transition-colors"
				style={{
					background: "#181818",
					border: "1px solid #2b2b2b",
				}}
				onMouseEnter={e => (e.currentTarget.style.borderColor = "#358DF6")}
				onMouseLeave={e => (e.currentTarget.style.borderColor = "#2b2b2b")}
			>
				<span className="text-sm font-medium text-content-primary">
					Cloud IDE
				</span>
				<span className="text-xs text-content-secondary">
					Launch a workspace in your browser — no install needed
				</span>
			</button>
			<button
				type="button"
				onClick={onLocal}
				className="flex flex-col gap-1 p-4 text-left cursor-pointer transition-colors"
				style={{
					background: "#181818",
					border: "1px solid #2b2b2b",
				}}
				onMouseEnter={e => (e.currentTarget.style.borderColor = "#358DF6")}
				onMouseLeave={e => (e.currentTarget.style.borderColor = "#2b2b2b")}
			>
				<span className="text-sm font-medium text-content-primary">
					Local IDE
				</span>
				<span className="text-xs text-content-secondary">
					Download and install Neural Inverse on your machine
				</span>
			</button>
		</div>
	</div>
);

const CloudStep: FC<{ onDone: () => void }> = ({ onDone }) => (
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				Create a Cloud Workspace
			</h2>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				A workspace is a full dev environment running in the cloud. Pick a
				template and you're ready to code in seconds.
			</p>
		</div>
		<ol className="flex flex-col gap-3 m-0 p-0 list-none">
			{[
				"Click New workspace in the top right",
				"Choose a template (e.g. Neural Inverse o1)",
				"Click Create workspace and wait ~30s",
				"Open VS Code or the browser terminal",
			].map((step, i) => (
				<li key={i} className="flex items-start gap-3">
					<span
						className="shrink-0 size-5 flex items-center justify-center text-xs font-bold"
						style={{
							background: "#2b2b2b",
							color: "#358DF6",
						}}
					>
						{i + 1}
					</span>
					<span className="text-xs text-content-secondary leading-relaxed pt-0.5">{step}</span>
				</li>
			))}
		</ol>
		<Button className="w-full" onClick={onDone}>
			Go to workspaces &rarr;
		</Button>
	</div>
);

const LocalStep: FC<{ onDone: () => void }> = ({ onDone }) => (
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				Download Neural Inverse IDE
			</h2>
			<p className="text-sm text-content-secondary m-0">
				Install with one command, then connect to your cloud workspaces.
			</p>
		</div>
		<div className="flex flex-col gap-3">
			<div className="flex flex-col gap-2">
				<span
					className="text-xs font-medium px-2 py-0.5 self-start"
					style={{ background: "#2b2b2b", color: "#9d9d9d" }}
				>
					macOS / Linux
				</span>
				<pre
					className="m-0 p-3 text-xs text-content-primary overflow-x-auto"
					style={{ background: "#181818", border: "1px solid #2b2b2b" }}
				>
					<code>curl -fsSL https://neuralinverse.com/sh | bash</code>
				</pre>
			</div>
			<div className="flex flex-col gap-2">
				<span
					className="text-xs font-medium px-2 py-0.5 self-start"
					style={{ background: "#2b2b2b", color: "#9d9d9d" }}
				>
					Windows (PowerShell)
				</span>
				<pre
					className="m-0 p-3 text-xs text-content-primary overflow-x-auto"
					style={{ background: "#181818", border: "1px solid #2b2b2b" }}
				>
					<code>irm https://neuralinverse.com/win | iex</code>
				</pre>
			</div>
		</div>
		<Button className="w-full" onClick={onDone}>
			Done &rarr;
		</Button>
	</div>
);

// Hook to use in DashboardLayout
export function useOnboarding(userId: string) {
	const [visible, setVisible] = useState(() => !isDone(userId));
	return {
		visible,
		dismiss: () => {
			markDone(userId);
			setVisible(false);
		},
	};
}
