import type { FC } from "react";
import { Link } from "react-router";
import { LunarLander } from "./LunarLander";

const NICupPage: FC = () => {
	return (
		<div className="relative w-screen h-screen bg-black overflow-hidden">
			<title>NI Nauts</title>

			<Link
				to="/workspaces"
				className="absolute top-3 left-3 z-10 opacity-60 hover:opacity-100 transition-opacity text-white text-sm font-bold tracking-widest"
			>
				NI
			</Link>

			<LunarLander />
		</div>
	);
};

export default NICupPage;
