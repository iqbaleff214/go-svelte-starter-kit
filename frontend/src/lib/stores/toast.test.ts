import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { toast } from './toast';

// Reset store between tests by removing all items
function clearToasts() {
	const current = get(toast);
	current.forEach((t) => toast.remove(t.id));
}

describe('toast store', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		clearToasts();
	});

	afterEach(() => {
		vi.useRealTimers();
		clearToasts();
	});

	it('adds a success toast', () => {
		toast.success('Done', 'Operation successful');
		const items = get(toast);
		expect(items).toHaveLength(1);
		expect(items[0].type).toBe('success');
		expect(items[0].title).toBe('Done');
		expect(items[0].message).toBe('Operation successful');
	});

	it('adds an error toast', () => {
		toast.error('Failed');
		const items = get(toast);
		expect(items).toHaveLength(1);
		expect(items[0].type).toBe('error');
	});

	it('adds an info toast', () => {
		toast.info('Info');
		expect(get(toast)[0].type).toBe('info');
	});

	it('adds a warning toast', () => {
		toast.warning('Warn');
		expect(get(toast)[0].type).toBe('warning');
	});

	it('returns a string id', () => {
		const id = toast.success('Test');
		expect(typeof id).toBe('string');
		expect(id.length).toBeGreaterThan(0);
	});

	it('removes a toast by id', () => {
		const id = toast.success('Remove me');
		expect(get(toast)).toHaveLength(1);
		toast.remove(id);
		expect(get(toast)).toHaveLength(0);
	});

	it('auto-removes after duration', () => {
		toast.success('Auto remove');
		// Default duration is 4000ms
		expect(get(toast)).toHaveLength(1);
		vi.advanceTimersByTime(4001);
		expect(get(toast)).toHaveLength(0);
	});

	it('accumulates multiple toasts', () => {
		toast.success('First');
		toast.error('Second');
		toast.info('Third');
		expect(get(toast)).toHaveLength(3);
	});

	it('removing one toast does not affect others', () => {
		toast.success('Keep');
		const idToRemove = toast.error('Remove');
		toast.remove(idToRemove);
		const remaining = get(toast);
		expect(remaining).toHaveLength(1);
		expect(remaining[0].title).toBe('Keep');
	});
});
