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

	// Full-page takeovers
	if (step === "local") {
		return <LocalPage onBack={() => setStep("choose")} onDone={finish} />;
	}

	const stepOrder: Step[] = ["welcome", "choose", "cloud"];
	const currentIdx = stepOrder.indexOf(step);

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
							onBack={() => setStep("welcome")}
							onCloud={() => setStep("cloud")}
							onLocal={() => setStep("local")}
						/>
					)}
					{step === "cloud" && (
						<CloudStep onBack={() => setStep("choose")} onDone={finish} />
					)}
				</div>

				{/* Step dots */}
				<div
					className="flex items-center justify-center gap-2 px-6 py-4"
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
			</div>
		</div>
	);
};

/* ── Steps ─────────────────────────────────────────────────────────── */

const WelcomeStep: FC<{ name: string; onNext: () => void }> = ({ name, onNext }) => (
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				Welcome to Neural Inverse Cloud{name ? `, ${name.split(" ")[0]}` : ""}
			</h2>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				An AI-native IDE platform for every stage of the software lifecycle —
				firmware, embedded systems, general development, legacy modernisation,
				and regulated critical software.
			</p>
		</div>
		<ul className="flex flex-col gap-3 m-0 p-0 list-none">
			{[
				["Cloud Workspaces", "Full dev environments in your browser, zero setup"],
				["Local IDE", "Download Neural Inverse IDE for macOS, Linux, or Windows"],
				["AI Inference", "Access Anthropic, OpenAI and more from your workspace"],
				["Legacy Modernisation", "Migrate and modernise codebases with AI-guided refactoring"],
			].map(([title, desc]) => (
				<li key={title} className="flex items-start gap-3">
					<div className="mt-1 shrink-0 size-1.5" style={{ background: "#358DF6" }} />
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

const ChooseStep: FC<{
	onBack: () => void;
	onCloud: () => void;
	onLocal: () => void;
}> = ({ onBack, onCloud, onLocal }) => (
	<div className="flex flex-col gap-5">
		<div className="flex flex-col gap-1">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				How would you like to work?
			</h2>
			<p className="text-xs text-content-secondary m-0">
				Pick where to start — you can use everything together.
			</p>
		</div>

		{/* Clickable — Cloud or Local */}
		<div className="flex flex-col gap-2">
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

		{/* Info-only — already included */}
		<div className="flex flex-col gap-1">
			<p className="text-xs text-content-disabled uppercase tracking-widest m-0 mb-1">
				Also included
			</p>
			{[
				["AI Inference", "Anthropic, OpenAI and more — bring your own key or use ours"],
				["Bring Your Own LLM", "Connect any OpenAI-compatible model endpoint"],
				["Open Models — Free Forever", "DeepSeek, Llama, Mistral — auto-connected, zero setup"],
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
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				Create a Cloud Workspace
			</h2>
			<p className="text-sm text-content-secondary m-0 leading-relaxed">
				A workspace is a full dev environment in the cloud. Pick a template and
				you're ready in seconds.
			</p>
		</div>
		<ol className="flex flex-col gap-3 m-0 p-0 list-none">
			{[
				"Click New workspace in the top right",
				"Choose a template (e.g. Neural Inverse o1)",
				"Click Create workspace and wait ~30s",
				"Open VS Code or the browser terminal",
			].map((s, i) => (
				<li key={i} className="flex items-start gap-3">
					<span
						className="shrink-0 size-5 flex items-center justify-center text-xs font-bold"
						style={{ background: "#2b2b2b", color: "#358DF6" }}
					>
						{i + 1}
					</span>
					<span className="text-xs text-content-secondary leading-relaxed pt-0.5">{s}</span>
				</li>
			))}
		</ol>
		<div className="flex gap-2">
			<BackButton onClick={onBack} />
			<Button className="flex-1" onClick={onDone}>
				Go to workspaces &rarr;
			</Button>
		</div>
	</div>
);

/* ── Local IDE full page ─────────────────────────────────────────── */

const LocalPage: FC<{ onBack: () => void; onDone: () => void }> = ({ onBack, onDone }) => (
	<FullPage title="Download Neural Inverse IDE" onDone={onDone}>
		<p className="text-sm text-content-secondary m-0 leading-relaxed">
			Install with one command. Build firmware, embedded systems, general
			software, or modernise legacy codebases — locally or connected to
			your cloud workspaces.
		</p>
		<div className="flex flex-col gap-3">
			<div
				className="px-3 py-1 self-start text-xs font-medium"
				style={{ background: "#2b2b2b", color: "#9d9d9d" }}
			>
				macOS / Linux
			</div>
			<pre
				className="m-0 p-4 text-sm text-content-primary overflow-x-auto"
				style={{ background: "#202020", border: "1px solid #2b2b2b" }}
			>
				<code>curl -fsSL https://neuralinverse.com/sh | bash</code>
			</pre>
		</div>
		<div className="flex flex-col gap-3">
			<div
				className="px-3 py-1 self-start text-xs font-medium"
				style={{ background: "#2b2b2b", color: "#9d9d9d" }}
			>
				Windows (PowerShell)
			</div>
			<pre
				className="m-0 p-4 text-sm text-content-primary overflow-x-auto"
				style={{ background: "#202020", border: "1px solid #2b2b2b" }}
			>
				<code>irm https://neuralinverse.com/win | iex</code>
			</pre>
		</div>
		<p className="text-xs text-content-disabled m-0">
			After installing, sign in with your GitHub account to connect to your
			cloud workspaces.
		</p>
		<div className="flex gap-3">
			<BackButton onClick={onBack} />
			<Button className="flex-1" onClick={onDone}>Done &rarr;</Button>
		</div>
	</FullPage>
);

/* ── Shared full-page wrapper ────────────────────────────────────── */

const FullPage: FC<{ title: string; onDone: () => void; children: ReactNode }> = ({
	title,
	onDone,
	children,
}) => (
	<div className="fixed inset-0 z-50 flex flex-col" style={{ background: "#1a1a1a" }}>
		<div
			className="flex items-center justify-between px-8 py-4 shrink-0"
			style={{ borderBottom: "1px solid #2b2b2b", background: "#181818" }}
		>
			<ProductLogo className="h-6" />
			<button
				type="button"
				onClick={onDone}
				className="text-xs text-content-disabled hover:text-content-secondary bg-transparent border-none cursor-pointer"
			>
				Skip
			</button>
		</div>
		<div className="flex-1 flex items-center justify-center px-4 py-12 overflow-y-auto">
			<div className="w-full max-w-lg flex flex-col gap-8">
				<h1 className="text-2xl font-semibold text-content-primary m-0">{title}</h1>
				{children}
			</div>
		</div>
	</div>
);

/* ── Shared ──────────────────────────────────────────────────────── */

const ChoiceCard: FC<{ title: string; desc: string; onClick: () => void }> = ({
	title,
	desc,
	onClick,
}) => (
	<button
		type="button"
		onClick={onClick}
		className="flex flex-col gap-1 p-4 text-left cursor-pointer w-full transition-colors"
		style={{ background: "#181818", border: "1px solid #2b2b2b" }}
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
