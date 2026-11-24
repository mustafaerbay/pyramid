# README posters

The images at the top of the main README (`docs/readme/*.png`) are built
from real screens of Pyramid, so they stay true to the app.

- `posters.html` is the design: headlines, colours and where each crop of a screen sits.
- `shoot.mjs` signs in to a local Pyramid with the demo data and saves the screens to `shots/` (git-ignored).
- `render.mjs` turns `posters.html` into the four PNGs.

## Rebuild

Run Pyramid locally (frontend on `:3000`, API on `:8080`) with the demo data, then:

```bash
cd docs/readme/src
npm install
PY_EMAIL=you@example.com PY_PASS=... npm run build
```

Needs Google Chrome; set `CHROME=/path/to/chrome` if it isn't in the macOS default place.
To change only the wording or layout, edit `posters.html` and run `npm run render`
(it reuses the screens already in `shots/`).

A crop is `<div class="crop" data-img="sprints" data-x data-y data-w data-h data-s>`:
the screen's name, the area in CSS pixels of that screen (1440 wide, phones 390), and a scale.
