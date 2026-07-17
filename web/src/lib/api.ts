const BASE = '';

export interface SystemInfo {
	ram: { total: number; free: number };
	diskFreeGB: number;
	cpu: number;
	os: string;
	arch: string;
	systemPrompt: string;
	temperature: number;
	isFirstLaunch: boolean;
	recommendedModel: ModelInfo | null;
	installedModels: string[];
	llamaServerRunning: boolean;
	activeModel: string;
}

export interface ModelInfo {
	id: string;
	name: string;
	sizeGB: number;
	minRamGB: number;
	quality: string;
	tagline: string;
	installed: boolean;
	file: string;
}

export interface ProgressEvent {
	type: 'progress' | 'done' | 'error';
	modelId?: string;
	bytesDownloaded?: number;
	totalBytes?: number;
	percent?: number;
	speed?: number;
	message?: string;
}

async function checkRes(res: Response): Promise<Response> {
	if (!res.ok) {
		const body = await res.text();
		let msg = `HTTP ${res.status}`;
		try { const j = JSON.parse(body); if (j.error) msg = j.error; } catch {}
		throw new Error(msg);
	}
	return res;
}

export async function getSystemInfo(): Promise<SystemInfo> {
	const res = await checkRes(await fetch(`${BASE}/api/system/info`));
	return res.json();
}

export async function listModels(): Promise<ModelInfo[]> {
	const res = await checkRes(await fetch(`${BASE}/api/models`));
	return res.json();
}

export function pullModel(modelId: string, onEvent: (evt: ProgressEvent) => void, opts?: { signal?: AbortSignal }): AbortController {
	const ctrl = new AbortController();
	const signal = opts?.signal ? AbortSignal.any([ctrl.signal, opts.signal]) : ctrl.signal;
	fetch(`${BASE}/api/models/pull`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ model: modelId }),
		signal
	}).then(async (res) => {
		if (!res.ok) {
			const body = await res.text();
			let msg = `HTTP ${res.status}`;
			try { const j = JSON.parse(body); if (j.error) msg = j.error; } catch {}
			throw new Error(msg);
		}
		const reader = res.body!.getReader();
		const decoder = new TextDecoder();
		let buf = '';
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			buf += decoder.decode(value, { stream: true });
			const lines = buf.split('\n');
			buf = lines.pop() || '';
			for (const line of lines) {
				if (line.startsWith('data: ')) {
					try {
						onEvent(JSON.parse(line.slice(6)));
					} catch {}
				}
			}
		}
	}).catch(err => onEvent({
		type: 'error' as const,
		modelId: modelId,
		message: err instanceof Error ? err.message : String(err)
	}));
	return ctrl;
}

export async function deleteModel(id: string): Promise<void> {
	await checkRes(await fetch(`${BASE}/api/models/${id}`, { method: 'DELETE' }));
}

export function chatCompletion(model: string, messages: { role: string; content: string }[], onToken: (text: string) => void, opts?: { signal?: AbortSignal; systemPrompt?: string; temperature?: number }): Promise<void> {
	const msgs = opts?.systemPrompt ? [{ role: 'system', content: opts.systemPrompt }, ...messages] : messages;
	return fetch(`${BASE}/v1/chat/completions`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ model, messages: msgs, stream: true, temperature: opts?.temperature ?? 0.7, max_tokens: 2048 }),
		signal: opts?.signal
	}).then(async (res) => {
		if (!res.ok) {
			const body = await res.text();
			let msg = `HTTP ${res.status}`;
			try { const j = JSON.parse(body); if (j.error) msg = j.error; } catch {}
			throw new Error(msg);
		}
		const reader = res.body!.getReader();
		const decoder = new TextDecoder();
		let buf = '';
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			buf += decoder.decode(value, { stream: true });
			const lines = buf.split('\n');
			buf = lines.pop() || '';
			for (const line of lines) {
				if (line.startsWith('data: ')) {
					const data = line.slice(6).trim();
					if (data === '[DONE]') return;
					try {
						const parsed = JSON.parse(data);
						const content = parsed.choices?.[0]?.delta?.content || parsed.choices?.[0]?.text || '';
						if (content) onToken(content);
					} catch {}
				}
			}
		}
	});
}

export interface ConversationSummary {
	id: string;
	title: string;
}

export interface Conversation {
	id: string;
	messages: { role: string; content: string }[];
	title: string;
}

export async function listConversations(): Promise<ConversationSummary[]> {
	const res = await checkRes(await fetch(`${BASE}/api/conversations`));
	return res.json();
}

export async function createConversation(): Promise<string> {
	const res = await checkRes(await fetch(`${BASE}/api/conversations`, { method: 'POST' }));
	const data = await res.json();
	return data.id;
}

export async function getConversation(id: string): Promise<Conversation> {
	const res = await checkRes(await fetch(`${BASE}/api/conversations/${id}`));
	return res.json();
}

export async function updateConversation(id: string, messages: { role: string; content: string }[], title: string): Promise<void> {
	await checkRes(await fetch(`${BASE}/api/conversations/${id}`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ messages, title })
	}));
}

export async function deleteConversation(id: string): Promise<void> {
	await checkRes(await fetch(`${BASE}/api/conversations/${id}`, { method: 'DELETE' }));
}

export async function updateConfig(updates: Record<string, unknown>): Promise<void> {
	await checkRes(await fetch(`${BASE}/api/config`, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(updates)
	}));
}

export interface ApiKeyInfo {
	id: string;
	name: string;
	prefix: string;
	createdAt: string;
	lastUsedAt: string;
	enabled: boolean;
}

export async function listApiKeys(): Promise<ApiKeyInfo[]> {
	const res = await checkRes(await fetch(`${BASE}/api/keys`));
	const data = await res.json();
	return data.keys;
}

export async function createApiKey(name: string): Promise<{ id: string; key: string }> {
	const res = await checkRes(await fetch(`${BASE}/api/keys`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ name })
	}));
	return res.json();
}

export async function deleteApiKey(id: string): Promise<void> {
	await checkRes(await fetch(`${BASE}/api/keys/${id}`, { method: 'DELETE' }));
}

export async function toggleApiKey(id: string): Promise<void> {
	await checkRes(await fetch(`${BASE}/api/keys/${id}/toggle`, { method: 'PUT' }));
}
