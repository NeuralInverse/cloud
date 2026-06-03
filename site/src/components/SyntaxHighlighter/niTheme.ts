import { useTheme } from "@emotion/react";
import { useMonaco } from "@monaco-editor/react";
import { useEffect, useState } from "react";
export const useNITheme = (): { isLoading: boolean; name: string } => {
	const [isLoading, setIsLoading] = useState(true);
	const monaco = useMonaco();
	const theme = useTheme();
	const name = "neuralinverse";

	useEffect(() => {
		if (monaco) {
			monaco.editor.defineTheme(name, theme.monaco);
			setIsLoading(false);
		}
	}, [monaco, theme]);

	return {
		isLoading,
		name,
	};
};
