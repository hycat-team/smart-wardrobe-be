# Closy — Project Presentation

Deliverable for the course presentation. **24 slides, English, 16:9.**

```
docs/system-design/presentation/
├── README.md                        this file
├── drawio/                         editable diagram sources (hand-drawn, source of truth)
│   ├── c4-level2-container.drawio      1 page   (slide 9)
│   ├── event-backbone.drawio           1 page   (slide 11)
│   ├── ci-pipeline.drawio              1 page   (slide 22)
│   ├── cd-pipeline-aws.drawio          1 page   (slide 23)
│   └── erd-by-schema.drawio           10 pages  (slides 12-21)
└── png/                            images embedded in the deck
```

> **The deck itself is not committed.** `closy-project-overview.pptx` is a build artefact and
> lives in `docs/temp/closy-slides/` alongside the build scripts, so `.gitignore` keeps it out of
> the repository. That folder also holds the speaker notes (in Vietnamese) and the export/build
> scripts.
>
> ```
> docs/temp/closy-slides/          (git-ignored)
> ├── closy-project-overview.pptx      the deck
> ├── speaker-notes.md                 slide-by-slide content + speaker notes (Vietnamese)
> ├── export-png.mjs                   drawio -> png
> ├── build-pptx.mjs                   png -> pptx
> ├── build-drawio.mjs                 OBSOLETE - do not run
> └── build-erd.mjs                    OBSOLETE - do not run
> ```

## Slide order

| # | Slide | Kind |
|---|---|---|
| 1 | Title | text |
| 2 | Agenda | text |
| 3 | Problem | text |
| 4 | Solution | text |
| 5 | Actors and Roles | text |
| 6 | Key Use Cases — Consumer | text |
| 7 | Key Use Cases — Brand | text |
| 8 | Key Use Cases — System Admin | text |
| 9 | System Architecture — C4 Level 2 | diagram |
| 10 | How the Services Talk to Each Other | text |
| 11 | Event Backbone — Transactional Outbox | diagram |
| 12 | Conceptual ERD — Schema Map | diagram |
| 13-21 | ERD per schema: `identity` `wardrobe` `fashion` `brand` `subscription` `aiexecutor` `community` `dashboard` `shared` | diagram |
| 22 | CI Pipeline | diagram |
| 23 | CD Pipeline — AWS Axis | diagram |
| 24 | Takeaways | text |

Slides 1-8, 10 and 24 use native PowerPoint shapes — the wording is selectable and editable.
Slides 9, 11-23 embed PNGs exported from draw.io.

## Diagram style

The diagrams are deliberately sparse. They are meant to be *read*, not studied:

- **A shape carries a name and at most one short qualifier.** `Commerce Core / Go · Gin`.
  No descriptions, no paragraph labels.
- **No note boxes and no legends inside the diagrams.** Where a diagram needs context, it
  becomes the slide subtitle — one line — and the rest goes in the speaker notes under
  `docs/temp/closy-slides/`, which is a document rather than a slide.
- **One thin stroke, one accent per category.** White fill for ordinary containers; a single
  tint to mark a hub, a gate, an artefact, or a success check. The C4 diagram alone uses the
  official C4 colours, because the notation is the point there.
- **Crow's-foot arrows, no edge labels.** Cardinality is readable from the arrowhead; labelling
  every edge put text on top of the shapes.
- **Aligned shapes get a straight line; anything diagonal gets a right angle.** Every bend is an
  explicit waypoint routed through an empty corridor, so no segment ever crosses a box. Each
  gutter has three routing lanes so parallel links do not stack on one line. Both generators
  deliberately use plain segments with waypoints and **no `edgeStyle`** — draw.io's own router
  would second-guess the waypoints and re-cut the path straight through the shapes.
- **Generous spacing.** Row pitch and column gaps are sized so the corridors stay empty.

If you add a diagram to this deck, follow those rules — a fourth style will look like a mistake.

## Where the numbers come from

Every figure in the deck was verified against the **running** local database
(`postgres-closy-dev`, `pgvector/pgvector:pg17`) on 2026-09-30, not read from the docs:

| Fact | Value | Query |
|---|---|---|
| Tables (excluding `public.goose_db_version*`) | 53 | `information_schema.tables` |
| Schemas | 9 | same |
| Foreign keys | 81 | `pg_constraint` where `contype = 'f'` |
| FKs pointing at `identity.users` | 39 | same, filtered on the parent table |
| Extensions | `vector 0.8.6`, `pg_trgm 1.6`, `unaccent 1.1` | `pg_extension` |

The documentation had drifted and was corrected as part of this work — see
[architecture.md §VII](../architecture.md). Three numbers were stale:

- `45 bảng + 1 sequence` → **53**
- `aiexecutor (6)` → **(7)** — `ai_cost_policy_audit_logs` was missing
- `shared (1)` → **(2)** — `admin_audit_logs` lives in `shared`, not `identity`
- `30 FK` → **39 FK**
- `community (6)` was absent from the table list and from `search_path`

## Editing a diagram

1. Open the `.drawio` file from `drawio/` in draw.io desktop.
2. Edit freely — the files are ordinary draw.io XML and are the **hand-drawn, final source of
   truth**. They are never regenerated from a script.
3. Re-export with `node export-png.mjs` and rebuild with `node build-pptx.mjs`.

To adjust the C4 Level 2 diagram, follow the walkthrough in
`docs/temp/closy-slides/speaker-notes.md` (mục "Cách vẽ lại diagram C4").

## Regenerating the deck

The build scripts live in `docs/temp/closy-slides/` (git-ignored, per `.gitignore`).
Requirements: Node 18+, draw.io desktop on `C:\Apps\draw.io\draw.io.exe` or `drawio` on PATH.

```bash
cd docs/temp/closy-slides
npm install

node export-png.mjs       # -> ../../system-design/presentation/png/*.png (plain PNG, no embedded XML)
node build-pptx.mjs       # -> closy-project-overview.pptx (here, in docs/temp/closy-slides/)
```

> **`build-drawio.mjs` and `build-erd.mjs` are OBSOLETE — do not run them.** They were the
> original generators, but the `.drawio` files have since been redrawn by hand and those scripts
> would overwrite that work. They are kept only as a record of the earlier draft.

Notes for anyone touching the build scripts:

- **The exported PNGs carry no embedded XML.** draw.io's `-e` flag was dropped on purpose, so a
  double-click in PowerPoint will not reopen the diagram in draw.io, and no absolute path or
  external link ends up inside the `.pptx`. Re-export from draw.io desktop if you need to edit.
- **`pptxgenjs` copies the image path into the shape's alt text.** `build-pptx.mjs` always passes
  an explicit `altText`, otherwise `C:\...` build paths leak into the slide XML.
- **No Graphviz on this machine.** The drawio-skill `autolayout.py` path needs `dot`, so every
  node coordinate in the original generators was computed by hand.

## Related documents

| Topic | File |
|---|---|
| Architecture rules and layering | [../architecture.md](../architecture.md) |
| Brand permission matrix | [../brand-permission-matrix.md](../brand-permission-matrix.md) |
| AI pipeline | [../ai-pipeline.md](../ai-pipeline.md) |
| AI cost control | [../ai-cost-control.md](../ai-cost-control.md) |
| Event-driven design | [../event-driven-design.md](../event-driven-design.md) |
| The B2B2C pivot (ADR 001) | [../../decisions/001-b2b2c-pivot.md](../../decisions/001-b2b2c-pivot.md) |
| Removing resale (ADR 002) | [../../decisions/002-remove-community-resale.md](../../decisions/002-remove-community-resale.md) |