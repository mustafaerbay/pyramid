import { baseUrl } from '$lib/api/client';
import { attachmentImage } from '$lib/markdown';

/** The page of an item: by number (KEY-12 style) or, from older links, by id. */
export function itemHref(projectId: string, ref: number | string): string {
	return `/projects/${projectId}/items/${ref}`;
}

/** Uploads an image to an item's files; returns Markdown that shows it, or null. */
export async function uploadImage(taskId: string, file: File): Promise<string | null> {
	const fd = new FormData();
	fd.append('file', file, imageName(file));
	const res = await fetch(`${baseUrl}/api/v1/tasks/${taskId}/attachments`, { method: 'POST', body: fd, credentials: 'include' });
	if (!res.ok) return null;
	const a = (await res.json()) as { id: string; filename: string };
	return attachmentImage(a.id, a.filename);
}

/** Pasted screenshots arrive as "image.png"; give them a findable name. */
export function imageName(file: File): string {
	if (file.name && file.name !== 'image.png') return file.name;
	const d = new Date();
	const p = (n: number) => String(n).padStart(2, '0');
	return `screenshot-${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}.${file.type.split('/')[1] || 'png'}`;
}
