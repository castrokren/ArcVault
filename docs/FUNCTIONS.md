# Moved

The HTTP endpoint inventory now lives in **[backend.md](backend.md)**, where the route list is a
**test-enforced** contract (`internal/docs` fails if the doc and the registered routes drift).

This file was hand-maintained and had already drifted from the real routes — that is exactly why the
inventory moved to a tested doc.
