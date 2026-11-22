import { describe, expect, it } from 'vitest';
import {
	groupMilestonesByVersion,
	isAfterRelease,
	isOverdue,
	isVersionActive,
	sortVersions,
	taskProgress,
	versionLineDate,
	type VersionLike
} from './releasePlan';

const v = (name: string, status: VersionLike['status'], targetDate: string | null = null, releasedAt: string | null = null) => ({
	id: name,
	name,
	status,
	targetDate,
	releasedAt
});

describe('sortVersions', () => {
	it('orders by effective date (releasedAt for released), undated last, cancelled after all others', () => {
		const sorted = sortVersions([
			v('v2', 'planned', '2027-01-01'),
			v('rel', 'released', '2027-05-01', '2026-10-01'),
			v('nodate', 'planned'),
			v('can', 'cancelled', '2026-01-01'),
			v('v1', 'in_progress', '2026-12-01')
		]);
		expect(sorted.map((x) => x.name)).toEqual(['rel', 'v1', 'v2', 'nodate', 'can']);
	});

	it('breaks date ties by name and does not mutate the input', () => {
		const input = [v('b', 'planned', '2026-12-01'), v('a', 'planned', '2026-12-01')];
		expect(sortVersions(input).map((x) => x.name)).toEqual(['a', 'b']);
		expect(input.map((x) => x.name)).toEqual(['b', 'a']);
	});
});

describe('versionLineDate', () => {
	it('uses releasedAt for released versions, falling back to targetDate', () => {
		expect(versionLineDate(v('a', 'released', '2026-12-01', '2026-11-20'))).toBe('2026-11-20');
		expect(versionLineDate(v('a', 'released', '2026-12-01'))).toBe('2026-12-01');
	});
	it('uses targetDate for planned/in_progress and draws nothing for cancelled', () => {
		expect(versionLineDate(v('a', 'planned', '2026-12-01'))).toBe('2026-12-01');
		expect(versionLineDate(v('a', 'in_progress'))).toBeNull();
		expect(versionLineDate(v('a', 'cancelled', '2026-12-01'))).toBeNull();
	});
});

describe('isVersionActive / isOverdue', () => {
	it('only planned and in_progress are active', () => {
		expect(isVersionActive(v('a', 'planned'))).toBe(true);
		expect(isVersionActive(v('a', 'in_progress'))).toBe(true);
		expect(isVersionActive(v('a', 'released'))).toBe(false);
		expect(isVersionActive(v('a', 'cancelled'))).toBe(false);
	});
	it('is overdue when an active version is past its target date', () => {
		expect(isOverdue(v('a', 'planned', '2026-11-01'), '2026-11-22')).toBe(true);
		expect(isOverdue(v('a', 'planned', '2026-11-22'), '2026-11-22')).toBe(false);
		expect(isOverdue(v('a', 'released', '2026-11-01'), '2026-11-22')).toBe(false);
		expect(isOverdue(v('a', 'planned'), '2026-11-22')).toBe(false);
	});
});

describe('isAfterRelease', () => {
	const target = v('v1', 'in_progress', '2026-12-01');
	it('flags a milestone due after an active version target date', () => {
		expect(isAfterRelease({ dueDate: '2026-12-02' }, target)).toBe(true);
		expect(isAfterRelease({ dueDate: '2026-12-01' }, target)).toBe(false);
	});
	it('never flags released/cancelled versions or missing dates', () => {
		expect(isAfterRelease({ dueDate: '2027-01-01' }, v('v', 'cancelled', '2026-12-01'))).toBe(false);
		expect(isAfterRelease({ dueDate: '2027-01-01' }, v('v', 'released', '2026-12-01'))).toBe(false);
		expect(isAfterRelease({ dueDate: null }, target)).toBe(false);
		expect(isAfterRelease({ dueDate: '2027-01-01' }, v('v', 'planned'))).toBe(false);
	});
});

describe('groupMilestonesByVersion', () => {
	const m = (id: string, versionId: string | null) => ({ id, versionId });

	it('groups in sortVersions order, keeps empty versions, puts unassigned and unknown versions last', () => {
		const versions = [v('v2', 'planned', '2027-01-01'), v('v1', 'planned', '2026-12-01')];
		const groups = groupMilestonesByVersion(versions, [
			m('a', 'v2'),
			m('b', null),
			m('c', 'deleted-version'),
			m('d', 'v2')
		]);
		expect(groups.map((g) => g.version?.name ?? null)).toEqual(['v1', 'v2', null]);
		expect(groups[0].milestones).toEqual([]);
		expect(groups[1].milestones.map((x) => x.id)).toEqual(['a', 'd']);
		expect(groups[2].milestones.map((x) => x.id)).toEqual(['b', 'c']);
	});

	it('omits the "No version" group when every milestone has a version', () => {
		const groups = groupMilestonesByVersion([v('v1', 'planned')], [m('a', 'v1')]);
		expect(groups.map((g) => g.version?.name ?? null)).toEqual(['v1']);
	});

	it('with no versions returns a single unassigned group, or nothing when there are no milestones', () => {
		expect(groupMilestonesByVersion([], [m('a', null)])).toEqual([{ version: null, milestones: [m('a', null)] }]);
		expect(groupMilestonesByVersion([], [])).toEqual([]);
	});
});

describe('taskProgress', () => {
	const tasks = [
		{ milestoneId: 'm1', status: 'done' },
		{ milestoneId: 'm1', status: 'todo' },
		{ milestoneId: 'm2', status: 'done' },
		{ milestoneId: 'm3', status: 'done' },
		{ milestoneId: null, status: 'done' }
	];
	it('counts done over all tasks of the given milestones only', () => {
		expect(taskProgress(['m1', 'm2'], tasks)).toEqual({ done: 2, total: 3 });
	});
	it('is 0/0 for no milestones', () => {
		expect(taskProgress([], tasks)).toEqual({ done: 0, total: 0 });
	});
});
