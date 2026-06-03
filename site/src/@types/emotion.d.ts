import type { Theme as NITheme } from "#/theme";

declare module "@emotion/react" {
	interface Theme extends NITheme {}
}
