import { type FC, useState } from "react";
import { useLocation } from "react-router";
import type { AuthMethods, BuildInfoResponse } from "#/api/typesGenerated";
import { Button } from "#/components/Button/Button";
import { ExternalImage } from "#/components/ExternalImage/ExternalImage";
import { ProductLogo } from "#/components/Icons/ProductLogo";
import { Loader } from "#/components/Loader/Loader";
import { PasswordSignInForm } from "./PasswordSignInForm";
import { TermsOfServiceLink } from "./TermsOfServiceLink";

const TOS_STORAGE_KEY = "ni_tos_agreed";

interface LoginPageViewProps {
	authMethods: AuthMethods | undefined;
	error: unknown;
	isLoading: boolean;
	buildInfo?: BuildInfoResponse;
	isSigningIn: boolean;
	onSignIn: (credentials: { email: string; password: string }) => void;
	redirectTo: string;
}

export const LoginPageView: FC<LoginPageViewProps> = ({
	authMethods,
	error,
	isLoading,
	buildInfo,
	isSigningIn,
	onSignIn,
	redirectTo,
}) => {
	const location = useLocation();
	const message = new URLSearchParams(location.search).get("message");
	const [showAdmin, setShowAdmin] = useState(false);
	const [showTosModal, setShowTosModal] = useState(false);
	const [emailUpdates, setEmailUpdates] = useState(false);
	const [serverTosAccepted, setServerTosAccepted] = useState(false);

	const serverTosRequired = authMethods?.terms_of_service_url && !serverTosAccepted;
	const githubEnabled = authMethods?.github.enabled ?? false;
	const passwordEnabled = authMethods?.password.enabled ?? true;

	const githubUrl = `/api/v2/users/oauth2/github/callback?redirect=${encodeURIComponent(redirectTo)}`;

	function handleGitHubClick() {
		const alreadyAgreed = localStorage.getItem(TOS_STORAGE_KEY);
		if (alreadyAgreed) {
			window.location.href = githubUrl;
		} else {
			setShowTosModal(true);
		}
	}

	function handleAgree() {
		localStorage.setItem(TOS_STORAGE_KEY, JSON.stringify({
			agreedAt: new Date().toISOString(),
			emailUpdates,
		}));
		window.location.href = githubUrl;
	}

	return (
		<div
			className="min-h-screen flex flex-col items-center justify-center"
			style={{ background: "#1a1a1a" }}
		>
			<div className="w-full max-w-[340px] flex flex-col items-center gap-8 px-4">
				{/* Logo + title */}
				<div className="flex flex-col items-center gap-3">
					<ProductLogo className="h-10" />
					<p className="text-sm text-content-secondary m-0">Login / Sign up</p>
				</div>

				{isLoading ? (
					<Loader />
				) : serverTosRequired ? (
					<div className="w-full flex flex-col gap-4">
						<TermsOfServiceLink url={authMethods.terms_of_service_url} />
						<Button size="lg" className="w-full" onClick={() => setServerTosAccepted(true)}>
							I agree
						</Button>
					</div>
				) : (
					<div className="w-full flex flex-col gap-3">
						{message && (
							<p className="text-xs text-content-secondary text-center">{message}</p>
						)}

						{/* GitHub — primary sign-in */}
						{githubEnabled && !showAdmin && (
							<Button
								variant="outline"
								disabled={isSigningIn}
								className="w-full"
								size="lg"
								onClick={handleGitHubClick}
							>
								<ExternalImage src="/icon/github.svg" className="size-4" />
								Continue with GitHub
							</Button>
						)}

						{/* Super user password form */}
						{showAdmin && passwordEnabled && (
							<div className="w-full flex flex-col gap-4">
								<PasswordSignInForm
									onSubmit={onSignIn}
									autoFocus
									isSigningIn={isSigningIn}
									error={error}
								/>
								<button
									type="button"
									onClick={() => setShowAdmin(false)}
									className="text-xs text-content-secondary hover:text-content-primary bg-transparent border-none cursor-pointer text-center"
								>
									Back
								</button>
							</div>
						)}
					</div>
				)}

				{/* Footer */}
				<footer className="flex flex-col items-center gap-2">
					<p className="text-xs text-content-disabled m-0">
						Neural Inverse Inc. 2026
					</p>
					{!showAdmin && passwordEnabled && (
						<button
							type="button"
							onClick={() => setShowAdmin(true)}
							className="text-xs text-content-disabled hover:text-content-secondary bg-transparent border-none cursor-pointer"
						>
							Super User
						</button>
					)}
				</footer>
			</div>

			{/* ToS Modal */}
			{showTosModal && (
				<div
					className="fixed inset-0 z-50 flex items-center justify-center px-4"
					style={{ background: "rgba(0,0,0,0.75)" }}
					onClick={(e) => { if (e.target === e.currentTarget) setShowTosModal(false); }}
				>
					<div
						className="w-full max-w-sm flex flex-col gap-5 p-6"
						style={{ background: "#202020", border: "1px solid #2b2b2b" }}
					>
						<div className="flex flex-col gap-1">
							<h2 className="text-sm font-semibold text-content-primary m-0">
								Before you continue
							</h2>
							<p className="text-xs text-content-secondary m-0">
								Please review our terms before signing in.
							</p>
						</div>

						<p className="text-xs text-content-secondary m-0 leading-relaxed">
							By clicking "I Agree", you agree to our{" "}
							<a
								href="https://neuralinverse.com/terms"
								target="_blank"
								rel="noreferrer"
								className="text-content-primary underline hover:text-content-primary"
							>
								Terms of Service
							</a>
							{" "}and{" "}
							<a
								href="https://neuralinverse.com/privacy"
								target="_blank"
								rel="noreferrer"
								className="text-content-primary underline hover:text-content-primary"
							>
								Privacy Policy
							</a>
							.
						</p>

						<label className="flex items-start gap-3 cursor-pointer">
							<input
								type="checkbox"
								checked={emailUpdates}
								onChange={(e) => setEmailUpdates(e.target.checked)}
								className="mt-0.5 shrink-0 accent-content-link"
							/>
							<span className="text-xs text-content-secondary leading-relaxed">
								I agree to receive occasional product updates via email{" "}
								<span className="text-content-disabled">(optional)</span>
							</span>
						</label>

						<div className="flex gap-2">
							<Button
								variant="subtle"
								size="sm"
								className="flex-1"
								onClick={() => setShowTosModal(false)}
							>
								Cancel
							</Button>
							<Button
								size="sm"
								className="flex-1"
								onClick={handleAgree}
							>
								I Agree &rarr;
							</Button>
						</div>
					</div>
				</div>
			)}
		</div>
	);
};
