import type { FC } from "react";
import { getApplicationName, getLogoURL } from "#/utils/appearance";
import { cn } from "#/utils/cn";
import { ExternalImage } from "../ExternalImage/ExternalImage";

/**
 * Enterprise customers can set a custom logo for their Neural Inverse Cloud application. Use
 * the custom logo wherever the Neural Inverse Cloud logo is used, if a custom one is provided.
 */
export const ProductLogo: FC<{ className?: string }> = ({ className }) => {
	const applicationName = getApplicationName();
	const logoURL = getLogoURL();

	return logoURL ? (
		<ExternalImage
			alt={applicationName}
			src={logoURL}
			// This prevent browser to display the ugly error icon if the
			// image path is wrong or user didn't finish typing the url
			onError={(e) => {
				e.currentTarget.style.display = "none";
			}}
			onLoad={(e) => {
				e.currentTarget.style.display = "inline";
			}}
			className={cn("h-12 max-w-[200px] application-logo", className)}
		/>
	) : (
		<NILogo className={cn("h-12", className)} />
	);
};

const NILogo: FC<{ className?: string }> = ({ className }) => (
	<img
		src="http://cdn.neuralinverse.io/logo.png"
		alt="Neural Inverse Cloud"
		className={cn("h-7", className)}
	/>
);
