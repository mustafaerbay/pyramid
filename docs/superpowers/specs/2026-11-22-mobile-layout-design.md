# Mobile layout — short design

**Why:** On a phone, sideways scrolling makes Pyramid hard to use: the tab menu
runs off the screen, tables (with their action buttons) are wider than the
screen, and the milestone timeline is unreadable. The user wants no horizontal
scrolling anywhere, a different menu on phones, tables shown as card lists with
their buttons, and touch-friendly buttons that also work for data entry.

**Goals**
- No page scrolls horizontally at 375px width.
- Phone menu: ☰ button that opens a full-width list of all modules.
- Every data table has a phone card list (`md:hidden`); the table stays as is on
  `md+` (`hidden md:block`). Cards: the main fact on top, secondary facts below,
  action buttons at the bottom, full-width and at least 40px tall.
- Page header action buttons wrap/stack instead of squeezing.
- Project page: milestone timeline replaced on phones by a list (name, version,
  date range, status, progress). Desktop timeline unchanged.
- Form modals open full-screen on phones, with full-height scroll.

**Non-goals:** No redesign of desktop layouts, no new features, no API changes.

**Approaches**
- A (chosen by the user): a separate, hand-made card markup per table, shown
  only on phones. More markup, but each card puts the important facts and the
  buttons where they work for touch.
- B: CSS that restyles table rows as stacked "label: value" blocks. Less code,
  but buttons end up squeezed at the end — not what the user wants.

**Calls left to the builder:** card content per table (keep every column's
information reachable), breakpoint `md` (768px), Tailwind only, Svelte 5 runes as
in the codebase.

**Deploy:** push to `main` → GitHub webhook → Coolify auto-deploys
pyramid.kimlerdensin.com (set up 2026-11-22).
