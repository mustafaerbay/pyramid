// Which items belong to a project team: those in its areas (and below), or
// — for a team without areas — those assigned to its members.

import { areaIdsUnder } from './areas';

type Team = { memberIds: string[]; areaIds: string[] };

export function teamFilter(team: Team, areas: readonly { id: string; path: string }[]) {
	if (team.areaIds.length) {
		const ids = new Set(team.areaIds.flatMap((a) => [...areaIdsUnder(areas, a)]));
		return (t: { areaId?: string | null }) => !!t.areaId && ids.has(t.areaId);
	}
	const members = new Set(team.memberIds);
	return (t: { assigneeIds: string[] }) => t.assigneeIds.some((p) => members.has(p));
}
