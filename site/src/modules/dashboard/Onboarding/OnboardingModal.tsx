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

type Step = "welcome" | "choose" | "cloud" | "local" | "ai" | "hardware" | "openmodels";

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
	if (step === "ai") {
		return <AIPage onBack={() => setStep("choose")} onDone={finish} />;
	}
	if (step === "hardware") {
		return <HardwarePage onBack={() => setStep("choose")} onDone={finish} />;
	}
	if (step === "openmodels") {
		return <OpenModelsPage onBack={() => setStep("choose")} onDone={finish} />;
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
							onAI={() => setStep("ai")}
							onHardware={() => setStep("hardware")}
							onOpenModels={() => setStep("openmodels")}
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
	onAI: () => void;
	onHardware: () => void;
	onOpenModels: () => void;
}> = ({ onBack, onCloud, onLocal, onAI, onHardware, onOpenModels }) => (
	<div className="flex flex-col gap-6">
		<div className="flex flex-col gap-2">
			<h2 className="text-lg font-semibold text-content-primary m-0">
				How would you like to work?
			</h2>
			<p className="text-sm text-content-secondary m-0">
				You can use all of these together — pick where to start.
			</p>
		</div>
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
			<ChoiceCard
				title="AI Inference"
				desc="Access Anthropic, OpenAI and more directly from your workspace"
				onClick={onAI}
			/>
			<ChoiceCard
				title="Hardware Runner"
				desc="Run and debug firmware on real or simulated embedded hardware"
				onClick={onHardware}
			/>
			<ChoiceCard
				title="Open Models — Free Forever"
				desc="DeepSeek, Llama, Mistral — auto-connected to the IDE, zero setup"
				onClick={onOpenModels}
			/>
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

/* ── AI Inference full page ──────────────────────────────────────── */

const AIPage: FC<{ onBack: () => void; onDone: () => void }> = ({ onBack, onDone }) => (
	<FullPage title="AI Inference" onDone={onDone}>
		<p className="text-sm text-content-secondary m-0 leading-relaxed">
			Access frontier AI models — Anthropic Claude, OpenAI GPT, and more —
			directly from your workspace. Use them for code generation, architecture
			review, compliance checks, and legacy modernisation.
		</p>
		<div className="flex flex-col gap-3">
			{[
				["model.neuralinverse.com", "Manage API keys, usage and provider settings"],
				["Free Models", "DeepSeek, Llama, Mistral — free forever at free.neuralinverse.com"],
				["IDE Integration", "AI is built into Neural Inverse IDE — no extra config needed"],
			].map(([title, desc]) => (
				<div
					key={title}
					className="flex flex-col gap-1 p-3"
					style={{ background: "#181818", border: "1px solid #2b2b2b" }}
				>
					<span className="text-xs font-medium text-content-primary">{title}</span>
					<span className="text-xs text-content-secondary">{desc}</span>
				</div>
			))}
		</div>
		<div className="flex gap-3">
			<BackButton onClick={onBack} />
			<Button className="flex-1" onClick={onDone}>Done &rarr;</Button>
		</div>
	</FullPage>
);

/* ── Hardware Runner full page ───────────────────────────────────── */

const HardwarePage: FC<{ onBack: () => void; onDone: () => void }> = ({ onBack, onDone }) => (
	<FullPage title="Hardware Runner" onDone={onDone}>
		<p className="text-sm text-content-secondary m-0 leading-relaxed">
			Run and debug firmware on real or simulated embedded hardware — STM32,
			nRF52, ESP32, RP2040, and more — directly from your cloud workspace.
		</p>
		<div className="flex flex-col gap-3">
			{[
				["run.neuralinverse.com", "Launch and manage hardware sessions from the dashboard"],
				["GDB Debug", "Full GDB remote debugging over a secure tunnel"],
				["Supported Boards", "STM32F4, STM32H7, nRF52840, ESP32, RP2040 and more"],
			].map(([title, desc]) => (
				<div
					key={title}
					className="flex flex-col gap-1 p-3"
					style={{ background: "#181818", border: "1px solid #2b2b2b" }}
				>
					<span className="text-xs font-medium text-content-primary">{title}</span>
					<span className="text-xs text-content-secondary">{desc}</span>
				</div>
			))}
		</div>
		<div className="flex gap-3">
			<BackButton onClick={onBack} />
			<Button className="flex-1" onClick={onDone}>Done &rarr;</Button>
		</div>
	</FullPage>
);

/* ── Open Models full page ───────────────────────────────────────── */

const OpenModelsPage: FC<{ onBack: () => void; onDone: () => void }> = ({ onBack, onDone }) => (
	<FullPage title="Open Models — Free Forever" onDone={onDone}>
		<p className="text-sm text-content-secondary m-0 leading-relaxed">
			Neural Inverse includes a curated set of open-source models hosted at{" "}
			<span className="text-content-primary">free.neuralinverse.com</span>.
			They are auto-connected to the IDE — no API key, no configuration, no cost.
		</p>
		<div className="flex flex-col gap-2">
			{[
				["DeepSeek R1 / V3 / V4", "Reasoning and code generation"],
				["Llama 3.3 70B / Llama 4 Maverick", "General-purpose chat and code"],
				["Mistral Large 3", "Fast, instruction-following"],
				["Kimi K2.6", "Long-context tasks"],
			].map(([model, desc]) => (
				<div
					key={model}
					className="flex items-start justify-between gap-4 p-3"
					style={{ background: "#181818", border: "1px solid #2b2b2b" }}
				>
					<span className="text-xs font-medium text-content-primary">{model}</span>
					<span className="text-xs text-content-secondary shrink-0">{desc}</span>
				</div>
			))}
		</div>
		<div
			className="flex items-start gap-3 p-3"
			style={{ background: "#181818", border: "1px solid #358DF6" }}
		>
			<span className="text-xs text-content-secondary leading-relaxed">
				Already active in your IDE. Just open Neural Inverse and select any
				Open Model from the model picker — no extra setup needed.
			</span>
		</div>
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
