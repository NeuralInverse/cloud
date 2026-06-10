import { type FC, type ReactNode, useState } from "react";
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

	const stepOrder: Step[] = ["welcome", "choose", "cloud"];
	const currentIdx = stepOrder.indexOf(step);

	return (
		<div className="fixed inset-0 z-50 flex flex-col" style={{ background: "#1a1a1a" }}>
			{/* Top bar */}
			<div
				className="flex items-center justify-between px-8 py-4 shrink-0"
				style={{ borderBottom: "1px solid #2b2b2b", background: "#181818" }}
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
			<div className="flex-1 flex items-center justify-center px-4 py-12 overflow-y-auto">
				<div className="w-full max-w-lg flex flex-col gap-8">
					{step === "welcome" && (
						<WelcomeStep name={userName} onNext={() => setStep("choose")} />
					)}
					{step === "choose" && (
						<ChooseStep
							onBack={() => setStep("welcome")}
							onCloud={() => setStep("cloud")}
							onLocal={() => setStep("local")}
						/>
					)}
					{step === "cloud" && (
						<CloudStep onBack={() => setStep("choose")} onDone={finish} />
					)}
					{step === "local" && (
						<LocalStep onBack={() => setStep("choose")} onDone={finish} />
					)}
				</div>
			</div>

			{/* Step dots — only for welcome/choose/cloud */}
			{step !== "local" && (
				<div
					className="flex items-center justify-center gap-2 px-6 py-4 shrink-0"
					style={{ borderTop: "1px solid #2b2b2b" }}
				>
					{stepOrder.map((_, i) => (
						<div
							key={i}
							className="h-1 transition-all"
							style={{
								width: i === currentIdx ? 24 : 8,
								background: i === currentIdx ? "#358DF6" : "#2b2b2b",
							}}
						/>
					))}
				</div>
			)}
		</div>
	);
};

/* ── Steps ─────────────────────────────────────────────────────────── */

const WelcomeStep: FC<{ name: string; onNext: () => void }> = ({ name, onNext }) => (
	<div className="flex flex-col gap-8">
		<div className="flex flex-col gap-3">
			<h1 className="text-2xl font-semibold text-content-primary m-0">
				Welcome to Neural Inverse Cloud{name ? `, ${name.split(" ")[0]}` : ""}
			</h1>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				An AI-native IDE platform for every stage of the software lifecycle —
				firmware, embedded systems, general development, legacy modernisation,
				and regulated critical software.
			</p>
		</div>
		<ul className="flex flex-col gap-4 m-0 p-0 list-none">
			{[
				["Cloud Workspaces", "Full dev environments in your browser, zero setup"],
				["Local IDE", "Download Neural Inverse IDE for macOS, Linux, or Windows"],
				["AI Inference", "Access Anthropic, OpenAI and more from your workspace"],
				["Legacy Modernisation", "Migrate and modernise codebases with AI-guided refactoring"],
			].map(([title, desc]) => (
				<li key={title} className="flex items-start gap-3">
					<div className="mt-1.5 shrink-0 size-1.5" style={{ background: "#358DF6" }} />
					<div className="flex flex-col gap-0.5">
						<span className="text-sm font-medium text-content-primary">{title}</span>
						<span className="text-xs text-content-secondary">{desc}</span>
					</div>
				</li>
			))}
		</ul>
		<Button size="lg" className="w-full" onClick={onNext}>
			Get started &rarr;
		</Button>
	</div>
);

const ChooseStep: FC<{
	onBack: () => void;
	onCloud: () => void;
	onLocal: () => void;
}> = ({ onBack, onCloud, onLocal }) => (
	<div className="flex flex-col gap-8">
		<div className="flex flex-col gap-2">
			<h1 className="text-2xl font-semibold text-content-primary m-0">
				How would you like to work?
			</h1>
			<p className="text-sm text-content-secondary m-0">
				Pick where to start — you can use everything together.
			</p>
		</div>

		{/* Clickable */}
		<div className="flex flex-col gap-3">
			<ChoiceCard
				title="Cloud IDE"
				desc="Launch a workspace in your browser — no install needed"
				onClick={onCloud}
			/>
			<ChoiceCard
				title="Local IDE"
				desc="Download Neural Inverse and run it on your machine"
				onClick={onLocal}
			/>
		</div>

		{/* Info-only */}
		<div className="flex flex-col gap-2">
			<p className="text-xs text-content-disabled uppercase tracking-widest m-0">
				Also included
			</p>
			{[
				["AI Inference", "Anthropic, OpenAI and more — bring your own key or use ours"],
				["Bring Your Own LLM", "Connect any OpenAI-compatible model endpoint"],
				["Open Models — Free Forever", "DeepSeek, Llama, Mistral — auto-connected to IDE, zero setup"],
				["Hardware Runner", "Run & debug firmware on real or simulated embedded hardware"],
			].map(([title, desc]) => (
				<div
					key={title}
					className="flex flex-col gap-0.5 px-3 py-2"
					style={{ borderLeft: "2px solid #2b2b2b" }}
				>
					<span className="text-xs font-medium text-content-primary">{title}</span>
					<span className="text-xs text-content-disabled">{desc}</span>
				</div>
			))}
		</div>

		<BackButton onClick={onBack} />
	</div>
);

const CloudStep: FC<{ onBack: () => void; onDone: () => void }> = ({ onBack, onDone }) => (
	<div className="flex flex-col gap-8">
		<div className="flex flex-col gap-2">
			<h1 className="text-2xl font-semibold text-content-primary m-0">
				Create a Cloud Workspace
			</h1>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				A workspace is a full dev environment in the cloud. Pick a template and
				you're ready in seconds.
			</p>
		</div>
		<ol className="flex flex-col gap-4 m-0 p-0 list-none">
			{[
				"Click New workspace in the top right",
				"Choose a template (e.g. Neural Inverse o1)",
				"Click Create workspace and wait ~30s",
				"Open Cloud IDE in your browser or connect via Local IDE",
			].map((s, i) => (
				<li key={i} className="flex items-start gap-4">
					<span
						className="shrink-0 size-6 flex items-center justify-center text-xs font-bold"
						style={{ background: "#2b2b2b", color: "#358DF6" }}
					>
						{i + 1}
					</span>
					<span className="text-sm text-content-secondary leading-relaxed pt-0.5">{s}</span>
				</li>
			))}
		</ol>
		<div className="flex gap-3">
			<BackButton onClick={onBack} />
			<Button size="lg" className="flex-1" onClick={onDone}>
				Go to workspaces &rarr;
			</Button>
		</div>
	</div>
);

export const LocalStep: FC<{ onBack: () => void; onDone: () => void }> = ({ onBack, onDone }) => (
	<div className="flex flex-col gap-8">
		<div className="flex flex-col gap-2">
			<h1 className="text-2xl font-semibold text-content-primary m-0">
				Download Neural Inverse IDE
			</h1>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				Install with one command. Build firmware, embedded systems, general
				software, or modernise legacy codebases — locally or connected to your
				cloud workspaces.
			</p>
		</div>

		<div className="flex flex-col gap-6">
			<CodeBlock label="macOS / Linux" code="curl -fsSL https://neuralinverse.com/sh | bash" />
			<CodeBlock label="Windows (PowerShell)" code="irm https://neuralinverse.com/win | iex" />
		</div>

		<p className="text-xs text-content-disabled m-0">
			After installing, sign in with your GitHub account to connect to your
			cloud workspaces.
		</p>

		<div className="flex gap-3">
			<BackButton onClick={onBack} />
			<Button size="lg" className="flex-1" onClick={onDone}>
				Done &rarr;
			</Button>
		</div>
	</div>
);

/* ── Shared ──────────────────────────────────────────────────────── */

const CodeBlock: FC<{ label: string; code: string }> = ({ label, code }) => {
	const [copied, setCopied] = useState(false);

	function copy() {
		navigator.clipboard.writeText(code).then(() => {
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		});
	}

	return (
		<div className="flex flex-col gap-2">
			<div
				className="px-3 py-1 self-start text-xs font-medium"
				style={{ background: "#2b2b2b", color: "#9d9d9d" }}
			>
				{label}
			</div>
			<div className="relative group">
				<pre
					className="m-0 p-4 text-sm text-content-primary overflow-x-auto pr-20"
					style={{ background: "#202020", border: "1px solid #2b2b2b" }}
				>
					<code>{code}</code>
				</pre>
				<button
					type="button"
					onClick={copy}
					className="absolute right-3 top-1/2 -translate-y-1/2 text-xs px-2 py-1 cursor-pointer transition-colors"
					style={{
						background: "#2b2b2b",
						border: "1px solid #3b3b3b",
						color: copied ? "#4ade80" : "#9d9d9d",
					}}
				>
					{copied ? "Copied!" : "Copy"}
				</button>
			</div>
		</div>
	);
};

const ChoiceCard: FC<{ title: string; desc: string; onClick: () => void }> = ({
	title,
	desc,
	onClick,
}) => (
	<button
		type="button"
		onClick={onClick}
		className="flex flex-col gap-1 p-4 text-left cursor-pointer w-full transition-colors"
		style={{ background: "#202020", border: "1px solid #2b2b2b" }}
		onMouseEnter={e => (e.currentTarget.style.borderColor = "#358DF6")}
		onMouseLeave={e => (e.currentTarget.style.borderColor = "#2b2b2b")}
	>
		<span className="text-sm font-medium text-content-primary">{title}</span>
		<span className="text-xs text-content-secondary">{desc}</span>
	</button>
);

const BackButton: FC<{ onClick: () => void }> = ({ onClick }) => (
	<Button variant="outline" onClick={onClick}>
		&larr; Back
	</Button>
);

/* ── Hook ────────────────────────────────────────────────────────── */

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
