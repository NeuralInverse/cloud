import * as path from "node:path";
import { defineConfig } from "@playwright/test";
import {
	nicloudPProfPort,
	coderPort,
	e2eFakeExperiment1,
	e2eFakeExperiment2,
	gitAuth,
	requireTerraformTests,
} from "./constants";

export const wsEndpoint = process.env.NEURALINVERSE_E2E_WS_ENDPOINT;
export const retries = (() => {
	if (process.env.NEURALINVERSE_E2E_TEST_RETRIES === undefined) {
		return undefined;
	}
	const count = Number.parseInt(process.env.NEURALINVERSE_E2E_TEST_RETRIES, 10);
	if (Number.isNaN(count)) {
		throw new Error(
			`NEURALINVERSE_E2E_TEST_RETRIES is not a number: ${process.env.NEURALINVERSE_E2E_TEST_RETRIES}`,
		);
	}
	if (count < 0) {
		throw new Error(
			`NEURALINVERSE_E2E_TEST_RETRIES is less than 0: ${process.env.NEURALINVERSE_E2E_TEST_RETRIES}`,
		);
	}
	return count;
})();

const localURL = (port: number, path: string): string => {
	return `http://localhost:${port}${path}`;
};

export default defineConfig({
	retries,
	globalSetup: require.resolve("./setup/preflight"),
	outputDir: "../test-results",
	projects: [
		{
			name: "testsSetup",
			testMatch: /setup\/.*\.spec\.ts/,
		},
		{
			name: "tests",
			testMatch: /tests\/.*\.spec\.ts/,
			dependencies: ["testsSetup"],
			timeout: 30_000,
		},
	],
	reporter: [
		["list"],
		["html", { open: "never" }],
		[
			"json",
			{ outputFile: path.join(__dirname, "../test-results/results.json") },
		],
		["./reporter.ts"],
	],
	use: {
		actionTimeout: 5000,
		baseURL: `http://localhost:${coderPort}`,
		screenshot: "only-on-failure",
		trace: "retain-on-failure",
		video: "retain-on-failure",
		...(wsEndpoint
			? {
					connectOptions: {
						wsEndpoint: wsEndpoint,
					},
				}
			: {
					launchOptions: {
						args: ["--disable-webgl"],
					},
				}),
	},
	webServer: {
		url: `http://localhost:${coderPort}/api/v2/deployment/config`,
		// The default timeout is 60s, but `go run` compilation with the
		// embed tag can take longer on CI.
		timeout: 120_000,
		command: [
			`go run -tags embed ${path.join(__dirname, "../../enterprise/cmd/neuralinverse")}`,
			"server",
			"--global-config $(mktemp -d -t e2e-XXXXXXXXXX)",
			`--access-url=http://localhost:${coderPort}`,
			`--http-address=0.0.0.0:${coderPort}`,
			"--ephemeral",
			"--telemetry=false",
			"--dangerous-disable-rate-limits",
			"--provisioner-daemons 10",
			// TODO: Enable some terraform provisioners
			`--provisioner-types=echo${requireTerraformTests ? ",terraform" : ""}`,
			"--provisioner-daemons=10",
			"--web-terminal-renderer=dom",
			"--pprof-enable",
			"--log-filter=.*",
			`--log-human=${path.join(__dirname, "test-results/debug.log")}`,
		]
			.filter(Boolean)
			.join(" "),
		stdout: "pipe",
		env: {
			...process.env,
			// Otherwise, the runner fails on Mac with: could not determine kind of name for C.uuid_string_t
			CGO_ENABLED: "0",

			// This is the test provider for git auth with devices!
			NEURALINVERSE_GITAUTH_0_ID: gitAuth.deviceProvider,
			NEURALINVERSE_GITAUTH_0_TYPE: "github",
			NEURALINVERSE_GITAUTH_0_CLIENT_ID: "client",
			NEURALINVERSE_GITAUTH_0_CLIENT_SECRET: "secret",
			NEURALINVERSE_GITAUTH_0_DEVICE_FLOW: "true",
			NEURALINVERSE_GITAUTH_0_APP_INSTALL_URL:
				"https://github.com/apps/coder/installations/new",
			NEURALINVERSE_GITAUTH_0_APP_INSTALLATIONS_URL: localURL(
				gitAuth.devicePort,
				gitAuth.installationsPath,
			),
			NEURALINVERSE_GITAUTH_0_TOKEN_URL: localURL(
				gitAuth.devicePort,
				gitAuth.tokenPath,
			),
			NEURALINVERSE_GITAUTH_0_DEVICE_CODE_URL: localURL(
				gitAuth.devicePort,
				gitAuth.codePath,
			),
			NEURALINVERSE_GITAUTH_0_VALIDATE_URL: localURL(
				gitAuth.devicePort,
				gitAuth.validatePath,
			),

			NEURALINVERSE_GITAUTH_1_ID: gitAuth.webProvider,
			NEURALINVERSE_GITAUTH_1_TYPE: "github",
			NEURALINVERSE_GITAUTH_1_CLIENT_ID: "client",
			NEURALINVERSE_GITAUTH_1_CLIENT_SECRET: "secret",
			NEURALINVERSE_GITAUTH_1_AUTH_URL: localURL(gitAuth.webPort, gitAuth.authPath),
			NEURALINVERSE_GITAUTH_1_TOKEN_URL: localURL(gitAuth.webPort, gitAuth.tokenPath),
			NEURALINVERSE_GITAUTH_1_DEVICE_CODE_URL: localURL(
				gitAuth.webPort,
				gitAuth.codePath,
			),
			NEURALINVERSE_GITAUTH_1_VALIDATE_URL: localURL(
				gitAuth.webPort,
				gitAuth.validatePath,
			),
			NEURALINVERSE_PPROF_ADDRESS: `127.0.0.1:${nicloudPProfPort}`,
			NEURALINVERSE_EXPERIMENTS: `${e2eFakeExperiment1},${e2eFakeExperiment2}`,

			// Tests for Deployment / User Authentication / OIDC
			NEURALINVERSE_OIDC_ISSUER_URL: "https://accounts.google.com",
			NEURALINVERSE_OIDC_EMAIL_DOMAIN: "cloud.neuralinverse.com",
			NEURALINVERSE_OIDC_CLIENT_ID: "1234567890",
			NEURALINVERSE_OIDC_CLIENT_SECRET: "1234567890Secret",
			NEURALINVERSE_OIDC_ALLOW_SIGNUPS: "false",
			NEURALINVERSE_OIDC_SIGN_IN_TEXT: "Hello",
			NEURALINVERSE_OIDC_ICON_URL: "/icon/google.svg",
		},
		reuseExistingServer: false,
	},
});
