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
	options: { tamperSignature?: boolean; rawBody?: string } = {},
): Promise<WorkerRequest> {
	const body = options.rawBody ?? JSON.stringify(invocation);
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

	it("forwards multiple rendered headers and a custom content type", async () => {
		const clientSignature = "a1".repeat(32);
		fetchMock.get("https://client.example").intercept({
			path: "/headers", method: "POST", body: "{}",
			headers: {
				"content-type": "application/vnd.longhub+json",
				"x-signature": clientSignature,
				"authorization": "Bearer token", "x-event": "image.delete", "x-image-id": "42",
			},
		}).reply(204);
		const response = await invoke(await signedRequest({
			url: "https://client.example/headers", body: "{}", clientSignature,
			headers: { "Authorization": "Bearer token", "X-Event": "image.delete", "X-Image-ID": "42", "Content-Type": "application/vnd.longhub+json" },
		}));
		expect(response.status).toBe(204);
		fetchMock.assertNoPendingInterceptors();
	});

	it.each([
		null, [], "headers", { "X-Test": 1 }, { "X-Test": null },
		{ "Bad Name": "value" }, { "": "value" }, { "X-Test": "bad\r\nInjected: true" },
		{ "X-Test": "\x00" }, { "X-Test": "世界" }, { "X-Test": "value", "x-test": "other" },
		{ "x-signature": "override" }, { "X-Server-Signature": "override" },
		{ "Host": "example.com" }, { "Content-Length": "1" }, { "Connection": "close" },
		{ "Transfer-Encoding": "chunked" }, { "Proxy-Authorization": "token" },
		{ "X-Test": "x".repeat(8 * 1024 + 1) },
		Object.fromEntries(Array.from({ length: 33 }, (_, i) => [`X-${i}`, "value"])),
		Object.fromEntries(Array.from({ length: 9 }, (_, i) => [`X-${i}`, "x".repeat(8 * 1024)])),
	])("rejects invalid or reserved headers: %#", async (headers) => {
		const response = await invoke(await signedRequest({
			url: "https://client.example/headers", body: "{}", clientSignature: "a1".repeat(32), headers,
		}));
		expect(response.status).toBe(400);
		expect(await response.json()).toEqual({ error: "Invalid webhook invocation" });
	});

	it("authenticates custom headers as part of the signed envelope", async () => {
		const invocation = {
			url: "https://client.example/headers", body: "{}", clientSignature: "a1".repeat(32),
			headers: { Authorization: "Bearer original" },
		};
		const signed = await signedRequest(invocation);
		invocation.headers.Authorization = "Bearer tampered";
		const response = await invoke(new IncomingRequest(signed, { body: JSON.stringify(invocation) }));
		expect(response.status).toBe(401);
	});

	it("rejects non-HTTP webhook URLs", async () => {
		const request = await signedRequest({
			url: "file:///etc/passwd",
			body: "{}",
			clientSignature: "a1".repeat(32),
		});

		const response = await invoke(request);

		expect(response.status).toBe(400);
		expect(await response.json()).toEqual({
			error: "Webhook URL must use HTTP or HTTPS",
		});
	});

	it("rejects non-hex client signatures for otherwise valid invocations", async () => {
		const request = await signedRequest({
			url: "https://client.example/hooks/longhub",
			body: "{}",
			clientSignature: "not-hex",
		});
		const response = await invoke(request);

		expect(response.status).toBe(400);
		expect(await response.json()).toEqual({
			error: "Client signature must be a hex string",
		});
	});

	it.each([null, {}, { url: "https://client.example", body: "{}" }])(
		"rejects authenticated invocations with missing fields: %j",
		async (invocation) => {
			const response = await invoke(await signedRequest(invocation));
			expect(response.status).toBe(400);
			expect(await response.json()).toEqual({ error: "Invalid webhook invocation" });
		},
	);

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
			"http://0.0.0.0/hook",
			"http://100.64.0.1/hook",
			"http://172.31.255.254/hook",
			"http://192.0.2.1/hook",
			"http://198.18.0.1/hook",
			"http://198.51.100.1/hook",
			"http://203.0.113.1/hook",
			"http://224.0.0.1/hook",
			"http://255.255.255.255/hook",
			"http://[::]/hook",
			"http://[fc00::1]/hook",
			"http://[fd00::1]/hook",
			"http://[fe80::1]/hook",
			"http://[ff02::1]/hook",
			"http://[::ffff:127.0.0.1]/hook",
			"http://[::ffff:c0a8:101]/hook",
			"http://sub.localhost/hook",
			"http://service.local/hook",
			"http://service.lan/hook",
			"http://service.home/hook",
			"http://service.test/hook",
			"http://service.invalid/hook",
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

	it.each(["spki", "pem", "raw", "alias"])("accepts supported server public keys: %s", async (format) => {
		const raw = (await crypto.subtle.exportKey("raw", keyPair.publicKey)) as ArrayBuffer;
		const encoded = format === "raw" ? base64(raw) : format === "pem"
			? `-----BEGIN PUBLIC KEY-----\n${publicKey}\n-----END PUBLIC KEY-----` : publicKey;
		const env = format === "alias" ? { PUBLIC_KEY: encoded } : { SERVER_PUBLIC_KEY: encoded };
		fetchMock.get("https://client.example").intercept({ path: "/key", method: "POST" }).reply(200, "ok");
		const request = await signedRequest({ url: "https://client.example/key", body: "{}", clientSignature: "a1".repeat(32) });
		const response = await worker.fetch(request, env);
		expect(response.status).toBe(200);
		fetchMock.assertNoPendingInterceptors();
	});

	it.each([undefined, "", "%%%", "YWJj"])("rejects missing or malformed server key configuration: %#", async (key) => {
		const request = await signedRequest({ url: "https://client.example/hook", body: "{}", clientSignature: "a1".repeat(32) });
		const response = await worker.fetch(request, { SERVER_PUBLIC_KEY: key });
		expect(response.status).toBe(500);
		expect(response.headers.get("Cache-Control")).toBe("no-store");
	});

	it.each([undefined, "%%%", "YQ==", "00".repeat(64)])("rejects missing, malformed, short, or invalid server signatures: %#", async (signature) => {
		const headers = new Headers();
		if (signature !== undefined) headers.set("X-Server-Signature", signature);
		const response = await invoke(new IncomingRequest("https://gateway.example/webhook", { method: "POST", headers, body: "{}" }));
		expect(response.status).toBe(401);
	});

	it("accepts a valid hex-encoded server signature", async () => {
		const request = await signedRequest({ url: "https://client.example/hex", body: "{}", clientSignature: "a1".repeat(32) });
		const signature = atob(request.headers.get("X-Server-Signature")!);
		const hex = Array.from(signature, char => char.charCodeAt(0).toString(16).padStart(2, "0")).join("");
		const headers = new Headers(request.headers);
		headers.set("X-Server-Signature", hex);
		fetchMock.get("https://client.example").intercept({ path: "/hex", method: "POST" }).reply(200, "ok");
		expect((await invoke(new IncomingRequest(request, { headers }))).status).toBe(200);
		fetchMock.assertNoPendingInterceptors();
	});

	it("rejects authenticated malformed JSON", async () => {
		const response = await invoke(await signedRequest(null, { rawBody: "{" }));
		expect(response.status).toBe(400);
		expect(await response.json()).toEqual({ error: "Invalid webhook invocation" });
	});

	it("rejects malformed target URLs", async () => {
		const response = await invoke(await signedRequest({ url: "://invalid", body: "{}", clientSignature: "a1".repeat(32) }));
		expect(response.status).toBe(400);
	});

	it("returns redirects without following them", async () => {
		fetchMock.get("https://client.example").intercept({ path: "/redirect", method: "POST" })
			.reply(302, "redirect", { headers: { Location: "http://127.0.0.1/private" } });
		const response = await invoke(await signedRequest({ url: "https://client.example/redirect", body: "{}", clientSignature: "a1".repeat(32) }));
		expect(response.status).toBe(302);
		expect(response.headers.get("Location")).toBe("http://127.0.0.1/private");
		fetchMock.assertNoPendingInterceptors();
	});

	it("returns a controlled gateway error for a network failure", async () => {
		fetchMock.get("https://client.example").intercept({ path: "/offline", method: "POST" }).replyWithError(new Error("network unavailable"));
		const response = await invoke(await signedRequest({ url: "https://client.example/offline", body: "{}", clientSignature: "a1".repeat(32) }));
		expect(response.status).toBe(502);
		expect(await response.json()).toEqual({ error: "Webhook destination could not be reached" });
		fetchMock.assertNoPendingInterceptors();
	});

	it.each(["http://8.8.8.8", "http://[2606:4700:4700::1111]", "http://[::ffff:808:808]"])("allows public address literals: %s", async (origin) => {
		fetchMock.get(origin).intercept({ path: "/public", method: "POST" }).reply(200, "ok");
		const response = await invoke(await signedRequest({ url: origin + "/public", body: "{}", clientSignature: "a1".repeat(32) }));
		expect(response.status).toBe(200);
		fetchMock.assertNoPendingInterceptors();
	});

	it("only accepts POST requests", async () => {
		const request = new IncomingRequest("https://gateway.example/webhook");
		const response = await invoke(request);

		expect(response.status).toBe(405);
		expect(response.headers.get("Allow")).toBe("POST");
	});
});
