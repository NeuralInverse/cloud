import { XIcon } from "lucide-react";
import { useState } from "react";

const DISMISSED_KEY = "ni_free_models_banner_dismissed";

export const FreeModelsBanner: React.FC = () => {
	const [dismissed, setDismissed] = useState(() => {
		try {
			return localStorage.getItem(DISMISSED_KEY) === "1";
		} catch {
			return false;
		}
	});

	if (dismissed) {
		return null;
	}

	const dismiss = () => {
		try {
			localStorage.setItem(DISMISSED_KEY, "1");
		} catch {
			// Non-fatal
		}
		setDismissed(true);
	};

	return (
		<div
			role="status"
			className="flex items-center justify-between gap-3 bg-surface-invert-secondary px-4 py-2"
		>
			<div className="flex min-w-0 flex-1 items-center gap-2 text-xs text-content-invert">
				<span className="font-semibold text-content-invert">
					Free AI models. Forever.
				</span>
				<span className="hidden text-content-invert/80 sm:inline">
					Llama 3.3 70B, DeepSeek R1, DeepSeek V3/V4, Mistral Large 3, and
					more — use them in your workspaces at no cost. No credit card for
					models.
				</span>
				<a
					href="/templates/neuralinverse/NeuralInverse/workspace?mode=manual&param.cpu=2&param.home_disk_size=10&param.memory=2&param.region=eastus"
					className="shrink-0 font-medium text-content-link underline-offset-2 hover:underline"
				>
					Try it free
				</a>
			</div>
			<button
				type="button"
				onClick={dismiss}
				aria-label="Dismiss banner"
				className="shrink-0 rounded p-0.5 text-content-invert/60 hover:text-content-invert"
			>
				<XIcon className="size-3.5" />
			</button>
		</div>
	);
};
