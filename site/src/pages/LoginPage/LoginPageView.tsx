import { type FC, useState, useRef } from "react";
import { useLocation } from "react-router";
import type { AuthMethods, BuildInfoResponse } from "#/api/typesGenerated";
import { Button } from "#/components/Button/Button";
import { ExternalImage } from "#/components/ExternalImage/ExternalImage";
import { ProductLogo } from "#/components/Icons/ProductLogo";
import { Loader } from "#/components/Loader/Loader";
import { Spinner } from "#/components/Spinner/Spinner";
import { PasswordSignInForm } from "./PasswordSignInForm";
import { TermsOfServiceLink } from "./TermsOfServiceLink";

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
	const [emailUpdates, setEmailUpdates] = useState(false);
	const [tosAccepted, setTosAccepted] = useState(false);
	const tosAcceptanceRequired =
		authMethods?.terms_of_service_url && !tosAccepted;

	const githubEnabled = authMethods?.github.enabled ?? false;
	const passwordEnabled = authMethods?.password.enabled ?? true;

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
				) : tosAcceptanceRequired ? (
					<div className="w-full flex flex-col gap-4">
						<TermsOfServiceLink url={authMethods.terms_of_service_url} />
						<Button size="lg" className="w-full" onClick={() => setTosAccepted(true)}>
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
							<div className="flex flex-col items-center gap-3 w-full">
								<Button
									variant="outline"
									asChild
									disabled={isSigningIn}
									className="w-full"
									size="lg"
								>
									<a
										href={`/api/v2/users/oauth2/github/callback?redirect=${encodeURIComponent(redirectTo)}`}
									>
										<ExternalImage src="/icon/github.svg" className="size-4" />
										Continue with GitHub
									</a>
								</Button>
								<p className="text-xs text-content-disabled text-center m-0 leading-relaxed">
									By continuing, you agree to our{" "}
									<a
										href="https://neuralinverse.com/terms"
										target="_blank"
										rel="noreferrer"
										className="text-content-secondary hover:text-content-primary underline"
									>
										Terms of Service
									</a>
									{" "}and{" "}
									<a
										href="https://neuralinverse.com/privacy"
										target="_blank"
										rel="noreferrer"
										className="text-content-secondary hover:text-content-primary underline"
									>
										Privacy Policy
									</a>
								</p>
								<label className="flex items-start gap-2 cursor-pointer w-full">
									<input
										type="checkbox"
										checked={emailUpdates}
										onChange={(e) => setEmailUpdates(e.target.checked)}
										className="mt-0.5 shrink-0 accent-content-link"
									/>
									<span className="text-xs text-content-disabled leading-relaxed text-left">
										I agree to receive occasional product updates via email{" "}
										<span className="opacity-60">(optional)</span>
									</span>
								</label>
							</div>
						)}

						{/* Admin password form — hidden by default */}
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
					{tosAccepted && (
						<TermsOfServiceLink url={authMethods?.terms_of_service_url} />
					)}
				</footer>
			</div>
		</div>
	);
};
