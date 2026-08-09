// Centralized Tailwind class literals for chart colors.
//
// These strings are written out in full (never built via `bg-${name}-500`
// string concatenation) because Tailwind's build-time scanner only picks up
// class names it can see as literal text in the source — a dynamically
// constructed class name is invisible to it and would silently render
// unstyled. Every color used anywhere in the dashboard must have its full
// class names appear literally somewhere below.
export type ChartColor = {
	/** SVG stroke, for donut/ring charts. */
	stroke: string;
	/** Solid fill, for bar charts and the "elapsed" portion of a timeline. */
	fill: string;
	/** Lighter fill, for chart tracks and the "remaining" portion of a timeline. */
	soft: string;
	/** Small legend swatch — same as `fill` today, kept separate in case that changes. */
	dot: string;
};

export const chartColors = {
	emerald: {
		stroke: 'stroke-emerald-500',
		fill: 'bg-emerald-500',
		soft: 'bg-emerald-100',
		dot: 'bg-emerald-500'
	},
	amber: { stroke: 'stroke-amber-500', fill: 'bg-amber-500', soft: 'bg-amber-100', dot: 'bg-amber-500' },
	red: { stroke: 'stroke-red-500', fill: 'bg-red-500', soft: 'bg-red-100', dot: 'bg-red-500' },
	sky: { stroke: 'stroke-sky-500', fill: 'bg-sky-500', soft: 'bg-sky-100', dot: 'bg-sky-500' },
	slate: { stroke: 'stroke-slate-400', fill: 'bg-slate-400', soft: 'bg-slate-100', dot: 'bg-slate-400' },
	slateDark: {
		stroke: 'stroke-slate-700',
		fill: 'bg-slate-700',
		soft: 'bg-slate-200',
		dot: 'bg-slate-700'
	},
	violet: { stroke: 'stroke-violet-500', fill: 'bg-violet-500', soft: 'bg-violet-100', dot: 'bg-violet-500' },
	orange: { stroke: 'stroke-orange-500', fill: 'bg-orange-500', soft: 'bg-orange-100', dot: 'bg-orange-500' },
	indigo: { stroke: 'stroke-indigo-500', fill: 'bg-indigo-500', soft: 'bg-indigo-100', dot: 'bg-indigo-500' },
	cyan: { stroke: 'stroke-cyan-500', fill: 'bg-cyan-500', soft: 'bg-cyan-100', dot: 'bg-cyan-500' },
	pink: { stroke: 'stroke-pink-500', fill: 'bg-pink-500', soft: 'bg-pink-100', dot: 'bg-pink-500' }
} satisfies Record<string, ChartColor>;

export type ChartColorName = keyof typeof chartColors;
