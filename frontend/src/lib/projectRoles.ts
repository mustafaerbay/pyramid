// Single source of truth for the project_role enum on the frontend —
// consolidates what used to be 3 near-identical local `roleOptions` arrays
// (ProjectMemberFormModal, AddTeamMembersModal, ProjectWizardModal) plus the
// project detail page's local `roleTone` switch. Mirrors
// db/schema.sql's project_role enum / openapi.yaml's ProjectRole schema.
import { badgeToneClass, type BadgeTone } from './badgeTones';
import type { components } from './api/schema';

export type ProjectRole = components['schemas']['ProjectRole'];

// Ordered for select/checkbox lists: leadership-ish roles first, then the
// engineering-discipline roles (added 2026-08-09), then reviewer/stakeholder.
export const PROJECT_ROLE_OPTIONS: ProjectRole[] = [
	'lead',
	'project_manager',
	'maintainer',
	'backend_engineer',
	'frontend_engineer',
	'devops_engineer',
	'cie',
	'contributor',
	'reviewer',
	'stakeholder'
];

// Display labels — most roles are just Title Case, but "CIE" is an acronym
// so it stays fully capitalized rather than becoming "Cie".
export const projectRoleLabel: Record<ProjectRole, string> = {
	lead: 'Lead',
	maintainer: 'Maintainer',
	contributor: 'Contributor',
	reviewer: 'Reviewer',
	stakeholder: 'Stakeholder',
	project_manager: 'Project Manager',
	cie: 'CIE',
	backend_engineer: 'Backend Engineer',
	frontend_engineer: 'Frontend Engineer',
	devops_engineer: 'DevOps Engineer'
};

// One badge tone per role — there happen to be exactly 10 tones in
// badgeTones.ts and 10 roles, so every role gets its own distinct color.
export const projectRoleTone: Record<ProjectRole, BadgeTone> = {
	lead: 'dark',
	maintainer: 'violet',
	contributor: 'neutral',
	reviewer: 'info',
	stakeholder: 'warning',
	project_manager: 'success',
	cie: 'orange',
	backend_engineer: 'neutralMuted',
	frontend_engineer: 'neutralFaint',
	devops_engineer: 'danger'
};

export function projectRoleBadgeClass(role: ProjectRole): string {
	return badgeToneClass[projectRoleTone[role]];
}
