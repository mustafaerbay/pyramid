// Release-plan rules shared by the project page's Releases section and the
// milestone timeline. Pure functions over plain data, so both views order
// and judge versions identically (see
// docs/superpowers/specs/2026-11-22-project-versions-design.md).
// Dates are ISO `YYYY-MM-DD` strings, which compare correctly as strings.

export type VersionStatus = 'planned' | 'in_progress' | 'released' | 'cancelled';

export type VersionLike = {
	id: string;
	name: string;
	status: VersionStatus;
	targetDate?: string | null;
	releasedAt?: string | null;
};

export type MilestoneLike = { id: string; versionId?: string | null };

export type VersionGroup<V, M> = { version: V | null; milestones: M[] };

export function isVersionActive(v: VersionLike): boolean {
	return v.status === 'planned' || v.status === 'in_progress';
}

/** The date a version is ordered by: when it shipped, else when it's due. */
function effectiveDate(v: VersionLike): string | null {
	if (v.status === 'released') return v.releasedAt ?? v.targetDate ?? null;
	return v.targetDate ?? null;
}

/**
 * Non-cancelled versions by effective date (undated last), then name;
 * cancelled versions after them in the same sub-order. Returns a new array.
 */
export function sortVersions<V extends VersionLike>(versions: V[]): V[] {
	const key = (v: V) => [v.status === 'cancelled' ? 1 : 0, effectiveDate(v) === null ? 1 : 0] as const;
	return [...versions].sort((a, b) => {
		const [ac, an] = key(a);
		const [bc, bn] = key(b);
		if (ac !== bc) return ac - bc;
		if (an !== bn) return an - bn;
		const ad = effectiveDate(a) ?? '';
		const bd = effectiveDate(b) ?? '';
		if (ad !== bd) return ad < bd ? -1 : 1;
		return a.name.localeCompare(b.name);
	});
}

/** Where the timeline draws the version's release line; null = no line. */
export function versionLineDate(v: VersionLike): string | null {
	if (v.status === 'cancelled') return null;
	return effectiveDate(v);
}

export function isOverdue(v: VersionLike, today: string): boolean {
	return isVersionActive(v) && !!v.targetDate && v.targetDate < today;
}

export function isAfterRelease(m: { dueDate?: string | null }, v: VersionLike): boolean {
	return isVersionActive(v) && !!m.dueDate && !!v.targetDate && m.dueDate > v.targetDate;
}

/**
 * One group per version in sortVersions order (empty versions included),
 * then a trailing `version: null` group for unassigned milestones — and for
 * milestones pointing at a version that isn't in the list — only when that
 * group is non-empty. Milestones keep their input order within a group.
 */
export function groupMilestonesByVersion<V extends VersionLike, M extends MilestoneLike>(
	versions: V[],
	milestones: M[]
): VersionGroup<V, M>[] {
	const groups = sortVersions(versions).map((version) => ({ version, milestones: [] as M[] }));
	const byId = new Map(groups.map((g) => [g.version.id, g]));
	const unassigned: M[] = [];
	for (const m of milestones) {
		const g = m.versionId ? byId.get(m.versionId) : undefined;
		if (g) g.milestones.push(m);
		else unassigned.push(m);
	}
	const out: VersionGroup<V, M>[] = groups;
	if (unassigned.length > 0) out.push({ version: null, milestones: unassigned });
	return out;
}

/** Done tasks over all tasks belonging to the given milestones. */
export function taskProgress(
	milestoneIds: string[],
	tasks: { milestoneId?: string | null; status: string }[]
): { done: number; total: number } {
	const ids = new Set(milestoneIds);
	let done = 0;
	let total = 0;
	for (const t of tasks) {
		if (!t.milestoneId || !ids.has(t.milestoneId)) continue;
		total++;
		if (t.status === 'done') done++;
	}
	return { done, total };
}

/**
 * Groups for the Releases section. A project without versions shows only the
 * empty state there — its milestones are already listed in the Milestones
 * section, so an all-"No version" card would just repeat them.
 */
export function releasePlanGroups<V extends VersionLike, M extends MilestoneLike>(
	versions: V[],
	milestones: M[]
): VersionGroup<V, M>[] {
	return versions.length === 0 ? [] : groupMilestonesByVersion(versions, milestones);
}
