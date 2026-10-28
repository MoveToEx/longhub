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
	headers: Record<string, string>;
}

const RESERVED_HEADERS = new Set([
	"x-signature", "x-server-signature", "host", "content-length", "connection",
	"keep-alive", "te", "trailer", "transfer-encoding", "upgrade",
]);

function parseHeaders(value: unknown): Record<string, string> | null {
	if (value === undefined) return {};
	if (typeof value !== "object" || value === null || Array.isArray(value)) return null;
	const entries = Object.entries(value);
	if (entries.length > 32) return null;
	const names = new Set<string>();
	let size = 0;
	for (const [name, headerValue] of entries) {
		const normalized = name.toLowerCase();
		if (
			!/^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/.test(name) ||
			RESERVED_HEADERS.has(normalized) || normalized.startsWith("proxy-") ||
			names.has(normalized) || typeof headerValue !== "string" ||
			/[^\t\x20-\x7e]/.test(headerValue) || headerValue.length > 8 * 1024
		) return null;
		names.add(normalized);
		size += name.length + headerValue.length;
		if (size > 64 * 1024) return null;
	}
	return Object.fromEntries(entries) as Record<string, string>;
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
	const headers = parseHeaders(invocation.headers);
	if (headers === null) return null;

	return {
		url: invocation.url,
		body: invocation.body,
		clientSignature: invocation.clientSignature,
		headers,
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

const BLOCKED_HOSTNAME_SUFFIXES = [
	".localhost",
	".local",
	".internal",
	".lan",
	".home",
	".test",
	".invalid",
];

function isBlockedHostname(hostname: string): boolean {
	const host = hostname.toLowerCase();
	if (host === "localhost") {
		return true;
	}
	return BLOCKED_HOSTNAME_SUFFIXES.some((suffix) => host.endsWith(suffix));
}

function isIpv4Literal(hostname: string): boolean {
	return /^\d{1,3}(\.\d{1,3}){3}$/.test(hostname);
}

function isPrivateIpv4(hostname: string): boolean {
	const parts = hostname.split(".").map((part) => Number.parseInt(part, 10));
	if (
		parts.length !== 4 ||
		parts.some((part) => Number.isNaN(part) || part < 0 || part > 255)
	) {
		return true;
	}
	const [a, b, c] = parts;
	if (a === 0 || a === 10 || a === 127) return true;
	if (a === 100 && b >= 64 && b <= 127) return true; // 100.64.0.0/10
	if (a === 169 && b === 254) return true; // 169.254.0.0/16
	if (a === 172 && b >= 16 && b <= 31) return true; // 172.16.0.0/12
	if (a === 192 && b === 168) return true; // 192.168.0.0/16
	if (a === 192 && b === 0 && (c === 0 || c === 2)) return true; // 192.0.0.0/24, 192.0.2.0/24
	if (a === 198 && (b === 18 || b === 19)) return true; // 198.18.0.0/15
	if (a === 198 && b === 51 && c === 100) return true; // 198.51.100.0/24
	if (a === 203 && b === 0 && c === 113) return true; // 203.0.113.0/24
	if (a >= 224) return true; // multicast / reserved / broadcast
	return false;
}

function isIpv6Literal(hostname: string): boolean {
	return hostname.includes(":");
}

function isPrivateIpv6(hostname: string): boolean {
	let addr = hostname;
	if (addr.startsWith("[") && addr.endsWith("]")) {
		addr = addr.slice(1, -1);
	}
	const lower = addr.toLowerCase();

	const mapped = lower.match(/^::ffff:(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$/);
	if (mapped) {
		return isPrivateIpv4(mapped[1]);
	}
	// URL parsing canonicalizes IPv4-mapped literals such as ::ffff:127.0.0.1
	// into hexadecimal groups. Apply the same IPv4 range rules to that form.
	const mappedHex = lower.match(/^::ffff:([0-9a-f]{1,4}):([0-9a-f]{1,4})$/);
	if (mappedHex) {
		const high = Number.parseInt(mappedHex[1], 16);
		const low = Number.parseInt(mappedHex[2], 16);
		return isPrivateIpv4(`${high >>> 8}.${high & 255}.${low >>> 8}.${low & 255}`);
	}

	if (lower === "::" || lower === "::1") return true;
	const firstGroup = lower.split(":")[0];
	if (firstGroup.startsWith("fc") || firstGroup.startsWith("fd")) return true; // ULA fc00::/7
	if (lower.startsWith("ff")) return true; // multicast ff00::/8
	if (/^fe[89ab]/.test(firstGroup)) return true; // link-local fe80::/10
	return false;
}

function isPrivateOrBlockedHost(hostname: string): boolean {
	if (isBlockedHostname(hostname)) {
		return true;
	}
	if (isIpv4Literal(hostname)) {
		return isPrivateIpv4(hostname);
	}
	if (isIpv6Literal(hostname)) {
		return isPrivateIpv6(hostname);
	}
	return false;
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
		if (isPrivateOrBlockedHost(targetUrl.hostname)) {
			return jsonResponse(400, "Webhook URL must target a public host");
		}
		if (!isHex(invocation.clientSignature)) {
			return jsonResponse(400, "Client signature must be a hex string");
		}

		try {
			const headers = new Headers({ "Content-Type": "application/json" });
			for (const [name, value] of Object.entries(invocation.headers)) {
				headers.set(name, value);
			}
			headers.set(CLIENT_SIGNATURE_HEADER, invocation.clientSignature);
			return await fetch(targetUrl, {
				method: "POST",
				headers,
				body: invocation.body,
				redirect: "manual",
			});
		} catch {
			return jsonResponse(502, "Webhook destination could not be reached");
		}
	},
} satisfies ExportedHandler<WorkerEnv>;
