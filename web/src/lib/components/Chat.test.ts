import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Chat from './Chat.svelte';

beforeEach(() => {
	vi.restoreAllMocks();
	globalThis.fetch = vi.fn().mockResolvedValue({
		ok: true,
		status: 200,
		text: () => Promise.resolve(''),
		json: () => Promise.resolve({}),
		body: { getReader: () => ({ read: () => Promise.resolve({ done: true, value: undefined }) }) },
		headers: new Map()
	});
});

describe('Chat', () => {
	it('renders empty state', () => {
		render(Chat, { conversationId: null });
		expect(screen.getByText('LocalAI Hub')).toBeTruthy();
		expect(screen.getByText(/Your private, local AI assistant/)).toBeTruthy();
	});

	it('renders chat header', () => {
		render(Chat, { conversationId: null });
		expect(screen.getByText('Chat')).toBeTruthy();
	});

	it('renders Clear button', () => {
		render(Chat, { conversationId: null });
		expect(screen.getByText('Clear')).toBeTruthy();
	});

	it('renders Send button', () => {
		render(Chat, { conversationId: null });
		expect(screen.getByText('Send')).toBeTruthy();
	});

	it('renders textarea', () => {
		render(Chat, { conversationId: null });
		expect(screen.getByPlaceholderText('Type a message...')).toBeTruthy();
	});

	it('disables send when input is empty', () => {
		render(Chat, { conversationId: null });
		const btn = screen.getByText('Send') as HTMLButtonElement;
		expect(btn.disabled).toBe(true);
	});
});
