export const pageTitle = (
	...crumbs: Array<string | boolean | undefined | null>
): string => {
	return [...crumbs, "Neural Inverse Cloud"].filter(Boolean).join(" - ");
};
