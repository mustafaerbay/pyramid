// Centralized Tailwind class literals for status/type pills, mirroring the
// same "write the full class names out, never build one via `bg-${x}-100`
// string interpolation" rule chartColors.ts already established for charts —
// Tailwind's build-time scanner only sees literal text, so a computed class
// name silently renders unstyled.
//
// Every page that used to hand-roll its own `xBadgeClass(value): string`
// switch statement (projects, people, goals, contributions, time off, tasks,
// project members, ProjectCard) now maps its own domain value to one of
// these shared tones instead — the domain mapping stays local to each page
// (a task's "blocked" and a project's "on_hold" both happen to be `warning`,
// but that's a coincidence each page is still free to decide for itself),
// only the tone's actual rendering is shared.
export type BadgeTone =
	| 'success'
	| 'info'
	| 'warning'
	| 'danger'
	| 'neutral'
	| 'neutralMuted'
	| 'neutralFaint'
	| 'violet'
	| 'orange'
	| 'dark';

export const badgeToneClass: Record<BadgeTone, string> = {
	success: 'bg-emerald-100 text-emerald-700',
	info: 'bg-sky-100 text-sky-700',
	warning: 'bg-amber-100 text-amber-700',
	danger: 'bg-red-100 text-red-700',
	// Three neutral weights, preserved from what each page already used
	// rather than collapsed to one — "backlog"/"low priority" reads lighter
	// than "cancelled", which reads faintest of all.
	neutral: 'bg-slate-100 text-slate-600',
	neutralMuted: 'bg-slate-200 text-slate-600',
	neutralFaint: 'bg-slate-200 text-slate-400',
	violet: 'bg-violet-100 text-violet-700',
	orange: 'bg-orange-100 text-orange-700',
	// Inverted/solid treatment — used for the project-member "lead" role only.
	dark: 'bg-slate-900 text-white'
};

// The small colored dot used for project health (list row, project detail
// header, ProjectCard) — identical in all three places before this
// extraction, just copy-pasted.
export type HealthTone = 'on_track' | 'at_risk' | 'off_track';

export const healthDotClass: Record<HealthTone, string> = {
	on_track: 'bg-emerald-500',
	at_risk: 'bg-amber-500',
	off_track: 'bg-red-500'
};
