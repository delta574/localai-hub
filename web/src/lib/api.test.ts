import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	getSystemInfo,
	listModels,
	pullModel,
	deleteModel,
	chatCompletion,
	listConversations,
	createConversation,
	getConversation,
	updateConversation,
	deleteConversation,
	updateConfig,
	listApiKeys,
	createApiKey,
	deleteApiKey,
	toggleApiKey
} from './api';

beforeEach(() => {
	vi.restoreAllMocks();
});

function mockFetch(status: number, body: unknown, headers?: Record<string, string>) {
	const bodyStr = typeof body === 'string' ? body : JSON.stringify(body);
	const response = {
		ok: status >= 200 && status < 300,
		status,
		text: () => Promise.resolve(bodyStr),
		json: () => Promise.resolve(body),
		body: null as any,
		headers: new Map(Object.entries(headers || {}))
	};
	globalThis.fetch = vi.fn().mockResolvedValue(response);
	return response;
}

function mockStreamFetch(chunks: string[]) {
	const encoder = new TextEncoder();
	let idx = 0;
	const reader = {
		read: () => {
			if (idx >= chunks.length) return Promise.resolve({ done: true, value: undefined });
			const value = encoder.encode(chunks[idx]);
			idx++;
			return Promise.resolve({ done: false, value });
		}
	};
	const response = {
		ok: true,
		status: 200,
		text: () => Promise.resolve(''),
		json: () => Promise.resolve({}),
		body: { getReader: () => reader },
		headers: new Map()
	};
	globalThis.fetch = vi.fn().mockResolvedValue(response);
	return response;
}

describe('getSystemInfo', () => {
	it('returns system info on success', async () => {
		const info = { ram: { total: 16, free: 12 }, os: 'test' };
		mockFetch(200, info);
		const result = await getSystemInfo();
		expect(result.os).toBe('test');
	});

	it('throws on error', async () => {
		mockFetch(500, { error: 'server error' });
		await expect(getSystemInfo()).rejects.toThrow();
	});
});

describe('listModels', () => {
	it('returns model list', async () => {
		const models = [{ id: 'test', name: 'Test Model' }];
		mockFetch(200, models);
		const result = await listModels();
		expect(result).toHaveLength(1);
		expect(result[0].id).toBe('test');
	});
});

describe('pullModel', () => {
	it('returns an AbortController', () => {
		mockStreamFetch([]);
		const ctrl = pullModel('test', () => {});
		expect(ctrl).toBeInstanceOf(AbortController);
	});

	it('fires onEvent with progress data', async () => {
		const events: any[] = [];
		const chunk = 'data: {"type":"progress","percent":50}\n\n';
		mockStreamFetch([chunk]);

		pullModel('test', (evt) => events.push(evt));
		await vi.waitFor(() => {
			expect(events.length).toBeGreaterThan(0);
		});
	});

	it('fires onEvent with error on bad HTTP', async () => {
		const events: any[] = [];
		const response = {
			ok: false,
			status: 404,
			text: () => Promise.resolve('not found'),
			json: () => Promise.resolve({}),
			body: null,
			headers: new Map()
		};
		globalThis.fetch = vi.fn().mockResolvedValue(response);

		pullModel('test', (evt) => events.push(evt));
		await vi.waitFor(() => {
			expect(events.length).toBeGreaterThan(0);
			expect(events[0].type).toBe('error');
		});
	});
});

describe('deleteModel', () => {
	it('succeeds on 204', async () => {
		mockFetch(204, '');
		await expect(deleteModel('test')).resolves.toBeUndefined();
	});
});

describe('chatCompletion', () => {
	it('sends messages and calls onToken', async () => {
		const onToken = vi.fn();
		const chunk = 'data: {"choices":[{"delta":{"content":"Hello"}}]}\n\n';
		mockStreamFetch([chunk]);

		await chatCompletion('model', [{ role: 'user', content: 'hi' }], onToken);
		expect(onToken).toHaveBeenCalledWith('Hello');
	});

	it('handles [DONE] signal', async () => {
		const onToken = vi.fn();
		const chunk = 'data: [DONE]\n\n';
		mockStreamFetch([chunk]);

		await chatCompletion('model', [{ role: 'user', content: 'hi' }], onToken);
		expect(onToken).not.toHaveBeenCalled();
	});

	it('handles error response', async () => {
		const response = {
			ok: false,
			status: 500,
			text: () => Promise.resolve('server error'),
			json: () => Promise.resolve({}),
			body: null,
			headers: new Map()
		};
		globalThis.fetch = vi.fn().mockResolvedValue(response);
		await expect(chatCompletion('model', [{ role: 'user', content: 'hi' }], vi.fn())).rejects.toThrow();
	});
});

describe('conversations', () => {
	it('listConversations', async () => {
		mockFetch(200, [{ id: '1', title: 'Test' }]);
		const result = await listConversations();
		expect(result).toHaveLength(1);
	});

	it('createConversation', async () => {
		mockFetch(201, { id: 'new-id' });
		const id = await createConversation();
		expect(id).toBe('new-id');
	});

	it('getConversation', async () => {
		const conv = { id: '1', messages: [], title: 'Test' };
		mockFetch(200, conv);
		const result = await getConversation('1');
		expect(result.title).toBe('Test');
	});

	it('updateConversation', async () => {
		mockFetch(200, {});
		await expect(updateConversation('1', [], 'Title')).resolves.toBeUndefined();
	});

	it('deleteConversation', async () => {
		mockFetch(204, '');
		await expect(deleteConversation('1')).resolves.toBeUndefined();
	});
});

describe('updateConfig', () => {
	it('sends config update', async () => {
		mockFetch(200, { status: 'ok' });
		await expect(updateConfig({ theme: 'dark' })).resolves.toBeUndefined();
	});
});

describe('api keys', () => {
	it('listApiKeys', async () => {
		mockFetch(200, { keys: [{ id: '1', name: 'test' }] });
		const keys = await listApiKeys();
		expect(keys).toHaveLength(1);
	});

	it('createApiKey', async () => {
		mockFetch(201, { id: '1', key: 'lah_secret' });
		const result = await createApiKey('test');
		expect(result.key).toBe('lah_secret');
	});

	it('deleteApiKey', async () => {
		mockFetch(204, '');
		await expect(deleteApiKey('1')).resolves.toBeUndefined();
	});

	it('toggleApiKey', async () => {
		mockFetch(200, { status: 'ok' });
		await expect(toggleApiKey('1')).resolves.toBeUndefined();
	});
});
