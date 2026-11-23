import { describe, expect, it } from 'vitest';
import { describeChange, type Names } from './history';

const names: Names = {
	person: (id) => ({ p1: 'Ada Kim', p2: 'Bo Li' })[id] ?? 'Unknown person',
	milestone: (id) => ({ m1: 'M1' })[id] ?? 'Unknown milestone',
	task: (id) => ({ t1: '#1 Setup' })[id] ?? 'Unknown task'
};

describe('describeChange', () => {
	it('shows plain fields as before → after', () => {
		expect(describeChange('title', { from: 'Old', to: 'New' }, names)).toBe('Title: “Old” → “New”');
		expect(describeChange('status', { from: 'backlog', to: 'in_progress' }, names)).toBe('Status: backlog → in progress');
		expect(describeChange('dueDate', { from: null, to: '2026-12-01' }, names)).toBe('Due date: none → 2026-12-01');
		expect(describeChange('estimateManDays', { from: 2, to: 4.5 }, names)).toBe('Size: 2 → 4.5 md');
		expect(describeChange('storyPoints', { from: null, to: 5 }, names)).toBe('Story points: none → 5');
		expect(describeChange('remainingManDays', { from: 3, to: 0 }, names)).toBe('Remaining: 3 → 0 md');
	});
	it('names milestones, parents and blockers', () => {
		expect(describeChange('milestoneId', { from: null, to: 'm1' }, names)).toBe('Milestone: none → M1');
		expect(describeChange('parentId', { from: 't1', to: null }, names)).toBe('Parent task: #1 Setup → none');
		expect(describeChange('blockedByIds', { from: [], to: ['t1'] }, names)).toBe('Waiting on: added #1 Setup');
	});
	it('lists added and removed tags', () => {
		expect(describeChange('tags', { from: ['a', 'b'], to: ['b', 'c'] }, names)).toBe('Tags: added c; removed a');
	});
	it('describes assignee changes', () => {
		expect(
			describeChange('assignments', { from: { p1: 2, p2: 1 }, to: { p1: 3, p3: 0 } }, names)
		).toBe('Assignees: added Unknown person; removed Bo Li; Ada Kim 2 → 3 md');
	});
	it('names sprints and areas when it knows them', () => {
		const n = { ...names, sprint: (id: string) => ({ s1: 'Sprint 1' })[id] ?? '?', area: (id: string) => ({ a1: 'Web' })[id] ?? '?' };
		expect(describeChange('sprintId', { from: null, to: 's1' }, n)).toBe('Sprint: none → Sprint 1');
		expect(describeChange('areaId', { from: 'a1', to: null }, n)).toBe('Area: Web → none');
	});
	it('only says the description changed', () => {
		expect(describeChange('description', { from: 'x', to: 'y' }, names)).toBe('Description edited');
	});
});
