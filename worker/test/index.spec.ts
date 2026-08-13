import {
	fetchMock,
} from "cloudflare:test";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";
import worker from "../src/index";

const IncomingRequest = Request<unknown, IncomingRequestCfProperties>;
type WorkerRequest = Request<unknown, IncomingRequestCfProperties>;

let keyPair: CryptoKeyPair;
let publicKey: string;

function base64(bytes: ArrayBuffer): string {
	return btoa(
		Array.from(new Uint8Array(bytes), (byte) => String.fromCharCode(byte)).join(
			"",
		),
	);
}

async function signedRequest(
	invocation: unknown,
	options: { tamperSignature?: boolean } = {},
): Promise<WorkerRequest> {
	const body = JSON.stringify(invocation);
	const signature = new Uint8Array(
		await crypto.subtle.sign(
			"Ed25519",
			keyPair.privateKey,
			new TextEncoder().encode(body),
		),
	);
	if (options.tamperSignature) {
		signature[0] ^= 0xff;
	}

	return new IncomingRequest("https://gateway.example/webhook", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
			"X-Server-Signature": base64(signature.buffer),
		},
		body,
	});
}

async function invoke(request: WorkerRequest): Promise<Response> {
	return worker.fetch(request, { SERVER_PUBLIC_KEY: publicKey });
}

describe("webhook gateway", () => {
	beforeAll(async () => {
		keyPair = (await crypto.subtle.generateKey("Ed25519", true, [
			"sign",
			"verify",
		])) as CryptoKeyPair;
		const exportedPublicKey = (await crypto.subtle.exportKey(
			"spki",
			keyPair.publicKey,
		)) as ArrayBuffer;
		publicKey = base64(exportedPublicKey);
	});

	beforeEach(() => {
		fetchMock.activate();
		fetchMock.disableNetConnect();
	});

	it("verifies the server and forwards the webhook and client signature", async () => {
		const webhookBody = JSON.stringify({ imageID: 42, tags: ["cloudflare"] });
		const clientSignature = "a1".repeat(32);
		fetchMock
			.get("https://client.example")
			.intercept({
				path: "/hooks/longhub",
				method: "POST",
				body: webhookBody,
				headers: {
					"content-type": "application/json",
					"x-signature": clientSignature,
				},
			})
			.reply(202, "accepted", { headers: { "X-Client": "ok" } });

		const request = await signedRequest({
			url: "https://client.example/hooks/longhub",
			body: webhookBody,
			clientSignature,
		});
		const response = await invoke(request);

		expect(response.status).toBe(202);
		expect(response.headers.get("X-Client")).toBe("ok");
		expect(await response.text()).toBe("accepted");
		fetchMock.assertNoPendingInterceptors();
	});

	it("rejects an invalid server signature without making an outbound request", async () => {
		const request = await signedRequest(
			{
				url: "https://client.example/hooks/longhub",
				body: "{}",
				clientSignature: "a1".repeat(32),
			},
			{ tamperSignature: true },
		);

		const response = await invoke(request);

		expect(response.status).toBe(401);
		expect(await response.json()).toEqual({ error: "Invalid server signature" });
	});

	it("rejects malformed authenticated invocations", async () => {
		const request = await signedRequest({
			url: "file:///etc/passwd",
			body: "{}",
			clientSignature: "not-hex",
		});

		const response = await invoke(request);

		expect(response.status).toBe(400);
		expect(await response.json()).toEqual({
			error: "Webhook URL must use HTTP or HTTPS",
		});
	});

	it("rejects webhooks targeting private, loopback, or local hosts", async () => {
		const blocked = [
			"http://127.0.0.1/hook",
			"http://10.0.0.1/hook",
			"http://169.254.169.254/latest/meta-data",
			"http://172.16.0.1/hook",
			"http://192.168.1.1/hook",
			"http://[::1]/hook",
			"http://localhost/hook",
			"http://internal.example.internal/hook",
		];

		for (const url of blocked) {
			const request = await signedRequest({
				url,
				body: "{}",
				clientSignature: "a1".repeat(32),
			});
			const response = await invoke(request);

			expect(response.status).toBe(400);
			expect(await response.json()).toEqual({
				error: "Webhook URL must target a public host",
			});
		}
	});

	it("only accepts POST requests", async () => {
		const request = new IncomingRequest("https://gateway.example/webhook");
		const response = await invoke(request);

		expect(response.status).toBe(405);
		expect(response.headers.get("Allow")).toBe("POST");
	});
});
