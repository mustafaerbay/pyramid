// Work item types: label, badge colours and the default type for a child.

export type WorkItemType = 'epic' | 'feature' | 'story' | 'task' | 'bug';

export const TYPE_OPTIONS: { value: WorkItemType; label: string; badge: string }[] = [
	{ value: 'epic', label: 'Epic', badge: 'bg-orange-100 text-orange-800 ring-orange-200' },
	{ value: 'feature', label: 'Feature', badge: 'bg-violet-100 text-violet-800 ring-violet-200' },
	{ value: 'story', label: 'Story', badge: 'bg-sky-100 text-sky-800 ring-sky-200' },
	{ value: 'bug', label: 'Bug', badge: 'bg-red-100 text-red-800 ring-red-200' },
	{ value: 'task', label: 'Task', badge: 'bg-amber-50 text-amber-800 ring-amber-200' }
];

export function typeMeta(type: string | undefined) {
	return TYPE_OPTIONS.find((t) => t.value === type) ?? TYPE_OPTIONS[4];
}

/** What a new child of `type` usually is. */
export function childType(type: string | undefined): WorkItemType {
	switch (type) {
		case 'epic':
			return 'feature';
		case 'feature':
			return 'story';
		default:
			return 'task';
	}
}
