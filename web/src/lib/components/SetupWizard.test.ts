import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import SetupWizard from './SetupWizard.svelte';

beforeEach(() => {
	vi.restoreAllMocks();
	const mockModels = [
		{ id: 'test-1', name: 'Test Model 1', sizeGB: 1.5, quality: '★★★', tagline: 'A test', installed: false },
		{ id: 'test-2', name: 'Test Model 2', sizeGB: 2.5, quality: '★★★★', tagline: 'Another test', installed: true }
	];
	globalThis.fetch = vi.fn().mockImplementation((url: string) => {
		if (url.includes('/api/models')) {
			return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(mockModels), text: () => Promise.resolve(''), headers: new Map() });
		}
		return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({}), text: () => Promise.resolve(''), headers: new Map() });
	});
});

describe('SetupWizard', () => {
	it('renders welcome heading', () => {
		render(SetupWizard, { onDone: () => {} });
		expect(screen.getByText('Welcome to LocalAI Hub')).toBeTruthy();
	});

	it('renders model list', async () => {
		render(SetupWizard, { onDone: () => {} });
		await vi.waitFor(() => {
			expect(screen.getByText('Test Model 1')).toBeTruthy();
		});
	});

	it('renders skip button', () => {
		render(SetupWizard, { onDone: () => {} });
		expect(screen.getByText(/Skip/)).toBeTruthy();
	});

	it('shows installed models as disabled', async () => {
		render(SetupWizard, { onDone: () => {} });
		await vi.waitFor(() => {
			const installedBtns = screen.getAllByText('Installed');
			expect(installedBtns.length).toBe(1);
		});
	});

	it('shows install buttons for non-installed models', async () => {
		render(SetupWizard, { onDone: () => {} });
		await vi.waitFor(() => {
			const installBtns = screen.getAllByText('Install');
			expect(installBtns.length).toBe(1);
		});
	});
});
