// Package backend is the root documentation of the imalive backend module
// (ADR-0001). Layout per docs/stack/backend-go.md:
//
//	cmd/       entry points: api (HTTP server), worker (escalation engine)
//	internal/  domain and platform packages, never imported from outside
//	  platform/  shared infrastructure: config, logging, httpserver, httpmw
//
// The module is stdlib-only during scaffolding; any new dependency must be
// listed in docs/stack/stack.md before it is added.
package backend
