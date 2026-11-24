// Badge tones for goals and contributions, shared by the Growth and
// Contributions pages and the person page so a status looks the same everywhere.
import type { components } from '$lib/api/schema';
import type { BadgeTone } from '$lib/badgeTones';

export function goalStatusTone(status: components['schemas']['GoalStatus']): BadgeTone {
	switch (status) {
		case 'active':
			return 'success';
		case 'draft':
			return 'neutralMuted';
		case 'achieved':
			return 'info';
		case 'partially_achieved':
			return 'warning';
		case 'missed':
		case 'cancelled':
			return 'danger';
	}
}

export function contributionTypeTone(type: components['schemas']['ContributionType']): BadgeTone {
	switch (type) {
		case 'commit':
			return 'neutralMuted';
		case 'pull_request':
			return 'violet';
		case 'code_review':
			return 'info';
		case 'issue':
			return 'warning';
		case 'deployment':
			return 'success';
		default:
			return 'neutral';
	}
}
