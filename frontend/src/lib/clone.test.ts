import { describe, expect, it } from 'vitest';
import { cloneBody, descendants, templateFields } from './clone';

const task = {
	id: 't1',
	projectId: 'p',
	title: 'Login',
	description: 'desc',
	type: 'story' as const,
	customType: 'spike',
	priority: 'high' as const,
	state: 'doing',
	tags: ['ui'],
	storyPoints: 3,
	estimateManDays: 2,
	assignments: [{ personId: 'a', manDays: 2 }],
	sprintId: 's',
	areaId: 'ar',
	milestoneId: 'm',
	parentId: 'pp',
	startDate: '2026-12-01',
	dueDate: '2026-12-05',
	customValues: { risk: 'high' }
};

describe('cloneBody', () => {
	it('copies the fields and starts the copy fresh', () => {
		expect(cloneBody(task, { parentId: 'pp' })).toEqual({
			projectId: 'p',
			title: 'Copy of Login',
			description: 'desc',
			type: 'story',
			customType: 'spike',
			priority: 'high',
			tags: ['ui'],
			storyPoints: 3,
			estimateManDays: 2,
			assignments: [{ personId: 'a', manDays: 2 }],
			sprintId: 's',
			areaId: 'ar',
			milestoneId: 'm',
			parentId: 'pp',
			startDate: '2026-12-01',
			dueDate: '2026-12-05',
			customValues: { risk: 'high' }
		});
	});
	it('keeps the title for children of a copy', () => {
		expect(cloneBody(task, { parentId: 'new', keepTitle: true }).title).toBe('Login');
	});
});

describe('descendants', () => {
	it('lists children depth first in order', () => {
		const all = [
			{ id: 'a', parentId: null, rank: 1, number: 1 },
			{ id: 'b', parentId: 'a', rank: 3, number: 2 },
			{ id: 'c', parentId: 'a', rank: 2, number: 3 },
			{ id: 'd', parentId: 'c', rank: 1, number: 4 }
		];
		expect(descendants('a', all).map((x) => x.id)).toEqual(['c', 'd', 'b']);
	});
});

describe('templateFields', () => {
	it('keeps only what a template holds', () => {
		expect(templateFields(task)).toEqual({
			type: 'story',
			customType: 'spike',
			title: 'Login',
			description: 'desc',
			priority: 'high',
			tags: ['ui'],
			storyPoints: 3,
			estimateManDays: 2,
			customValues: { risk: 'high' }
		});
	});
});
