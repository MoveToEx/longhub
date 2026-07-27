const SERVER_SIGNATURE_HEADER = "X-Server-Signature";
const CLIENT_SIGNATURE_HEADER = "X-Signature";

interface WorkerEnv {
	/** Base64-encoded SPKI DER, PEM SPKI, or base64-encoded raw Ed25519 key. */
	SERVER_PUBLIC_KEY?: string;
	/** Kept as a compatibility alias for deployments that use this shorter name. */
	PUBLIC_KEY?: string;
}

interface WebhookInvocation {
	url: string;
	body: string;
	clientSignature: string;
}

function jsonResponse(status: number, error: string): Response {
	return Response.json(
		{ error },
		{
			status,
			headers: { "Cache-Control": "no-store" },
		},
	);
}

function decodeBase64(value: string): Uint8Array {
	const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
	const padded = normalized.padEnd(
		normalized.length + ((4 - (normalized.length % 4)) % 4),
		"=",
	);
	const decoded = atob(padded);
	return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

function decodeSignature(value: string): Uint8Array {
	const trimmed = value.trim();
	if (/^[0-9a-fA-F]{128}$/.test(trimmed)) {
		return Uint8Array.from(trimmed.match(/.{2}/g)!, (byte) =>
			Number.parseInt(byte, 16),
		);
	}
	return decodeBase64(trimmed);
}

async function importServerPublicKey(encodedKey: string): Promise<CryptoKey> {
	const pemBody = encodedKey
		.trim()
		.replace(/-----BEGIN PUBLIC KEY-----|-----END PUBLIC KEY-----/g, "")
		.replace(/\s/g, "");
	const keyBytes = decodeBase64(pemBody);
	const format = keyBytes.byteLength === 32 ? "raw" : "spki";

	return crypto.subtle.importKey(format, keyBytes, "Ed25519", false, [
		"verify",
	]);
}

function parseInvocation(rawBody: ArrayBuffer): WebhookInvocation | null {
	let value: unknown;
	try {
		value = JSON.parse(new TextDecoder().decode(rawBody));
	} catch {
		return null;
	}

	if (typeof value !== "object" || value === null) {
		return null;
	}

	const invocation = value as Record<string, unknown>;
	if (
		typeof invocation.url !== "string" ||
		typeof invocation.body !== "string" ||
		typeof invocation.clientSignature !== "string"
	) {
		return null;
	}

	return {
		url: invocation.url,
		body: invocation.body,
		clientSignature: invocation.clientSignature,
	};
}

function isHex(value: string): boolean {
	return value.length > 0 && value.length % 2 === 0 && /^[0-9a-f]+$/i.test(value);
}

function parseTargetUrl(value: string): URL | null {
	try {
		const url = new URL(value);
		return url.protocol === "https:" || url.protocol === "http:" ? url : null;
	} catch {
		return null;
	}
}

export default {
	async fetch(request, env): Promise<Response> {
		if (request.method !== "POST") {
			return new Response(null, {
				status: 405,
				headers: { Allow: "POST", "Cache-Control": "no-store" },
			});
		}

		const publicKeyValue = env.SERVER_PUBLIC_KEY ?? env.PUBLIC_KEY;
		if (!publicKeyValue) {
			return jsonResponse(500, "Server public key is not configured");
		}

		const encodedSignature = request.headers.get(SERVER_SIGNATURE_HEADER);
		if (!encodedSignature) {
			return jsonResponse(401, "Missing server signature");
		}

		const rawBody = await request.arrayBuffer();
		let publicKey: CryptoKey;
		try {
			publicKey = await importServerPublicKey(publicKeyValue);
		} catch {
			return jsonResponse(500, "Invalid server public key configuration");
		}

		let serverSignature: Uint8Array;
		try {
			serverSignature = decodeSignature(encodedSignature);
		} catch {
			return jsonResponse(401, "Invalid server signature");
		}

		if (serverSignature.byteLength !== 64) {
			return jsonResponse(401, "Invalid server signature");
		}

		const authenticated = await crypto.subtle.verify(
			"Ed25519",
			publicKey,
			serverSignature,
			rawBody,
		);
		if (!authenticated) {
			return jsonResponse(401, "Invalid server signature");
		}

		const invocation = parseInvocation(rawBody);
		if (!invocation) {
			return jsonResponse(400, "Invalid webhook invocation");
		}

		const targetUrl = parseTargetUrl(invocation.url);
		if (!targetUrl) {
			return jsonResponse(400, "Webhook URL must use HTTP or HTTPS");
		}
		if (!isHex(invocation.clientSignature)) {
			return jsonResponse(400, "Client signature must be a hex string");
		}

		try {
			return await fetch(targetUrl, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					[CLIENT_SIGNATURE_HEADER]: invocation.clientSignature,
				},
				body: invocation.body,
				redirect: "manual",
			});
		} catch {
			return jsonResponse(502, "Webhook destination could not be reached");
		}
	},
} satisfies ExportedHandler<WorkerEnv>;
