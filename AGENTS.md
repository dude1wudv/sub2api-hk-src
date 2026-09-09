Changes to the managed-protocol contract must be checked against `../heliosgen-sub2api/` on both ends.

Deployment defaults to committing and pushing to GitHub, then fetching the exact commit on the server, building there, and switching only the target application image. Building locally and transferring an image is a fallback only. Preserve shared PostgreSQL, Redis, networks, and volumes.
