/*
Package security provides OpenBSD-inspired security primitives for capability-based
security in the webos project.

This package implements two key security mechanisms inspired by OpenBSD:

# Pledge System

The pledge system restricts what operations a component can perform by requiring
it to declare its capabilities upfront. Components can only use system resources
and operations that match their declared promises. This follows the principle of
least privilege - components should have only the minimum permissions needed.

Example:

	cap := &security.Capability{
		Promises: security.PromiseStdio | security.PromiseRpath,
		UnveilPaths: []security.UnveilPath{{Path: "/tmp", Permissions: "r"}},
		Timeout: time.Hour,
	}
	err := securityManager.RegisterComponent("my-component", cap)

# Unveil System

The unveil system restricts which filesystem paths a component can access. After
unveiling, components can only access the specified paths with the specified
permissions. This provides filesystem sandboxing.

Example:

	err := securityManager.AddUnveilPath("my-component", "/data", "r")

# Available Promises

The following promises are available:
  - PromiseStdio: Standard input/output/error access
  - PromiseRpath: Read path access
  - PromiseWpath: Write path access
  - PromiseInet: Internet access (TCP/IP)
  - PromiseUnix: Unix domain socket access
  - PromiseFork: Process forking
  - PromiseExec: Program execution
  - PromiseSignal: Signal handling
  - PromiseTimer: Timer access
  - PromiseAudio: Audio access
  - PromiseVideo: Video access
  - PromiseSocket: Generic socket access
  - PromiseResolve: DNS resolution

# Thread Safety

All SecurityManager operations are thread-safe using sync.Map for component
registration and capability storage.
*/
package security
