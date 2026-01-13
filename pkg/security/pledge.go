package security

// Promise represents capability permissions for a component, inspired by
// OpenBSD's pledge system. Each promise is a bit flag that can be combined
// using bitwise OR to represent multiple capabilities.
type Promise uint64

// Promise constants define the available capabilities that a component can request.
// These follow the principle of least privilege - components should request only
// the minimum capabilities they need to function.
const (
	// PromiseStdio allows standard input, output, and error stream access.
	PromiseStdio Promise = 1 << iota
	// PromiseRpath allows reading from filesystem paths.
	PromiseRpath
	// PromiseWpath allows writing to filesystem paths.
	PromiseWpath
	// PromiseInet allows internet access (TCP/IPv4 and IPv6).
	PromiseInet
	// PromiseUnix allows Unix domain socket access.
	PromiseUnix
	// PromiseFork allows forking new processes.
	PromiseFork
	// PromiseExec allows executing programs.
	PromiseExec
	// PromiseSignal allows sending and receiving signals.
	PromiseSignal
	// PromiseTimer allows setting timers and using time operations.
	PromiseTimer
	// PromiseAudio allows audio access.
	PromiseAudio
	// PromiseVideo allows video access.
	PromiseVideo
	// PromiseSocket allows generic socket operations.
	PromiseSocket
	// PromiseResolve allows DNS resolution.
	PromiseResolve
)

// HasCapability returns true if the promise includes the specified capability.
func (p Promise) HasCapability(cap Promise) bool {
	return uint64(p)&uint64(cap) != 0
}

// AddCapability returns a new promise with the additional capability included.
func (p Promise) AddCapability(cap Promise) Promise {
	return p | cap
}

// RemoveCapability returns a new promise with the specified capability removed.
func (p Promise) RemoveCapability(cap Promise) Promise {
	return p &^ cap
}

// String returns a human-readable string representation of the promise.
// Returns a comma-separated list of capability names, or "PromiseStdio" for
// a single PromiseStdio capability, or empty string for no capabilities.
func (p Promise) String() string {
	if p == 0 {
		return ""
	}

	names := make([]string, 0)
	if p.HasCapability(PromiseStdio) {
		names = append(names, "PromiseStdio")
	}
	if p.HasCapability(PromiseRpath) {
		names = append(names, "PromiseRpath")
	}
	if p.HasCapability(PromiseWpath) {
		names = append(names, "PromiseWpath")
	}
	if p.HasCapability(PromiseInet) {
		names = append(names, "PromiseInet")
	}
	if p.HasCapability(PromiseUnix) {
		names = append(names, "PromiseUnix")
	}
	if p.HasCapability(PromiseFork) {
		names = append(names, "PromiseFork")
	}
	if p.HasCapability(PromiseExec) {
		names = append(names, "PromiseExec")
	}
	if p.HasCapability(PromiseSignal) {
		names = append(names, "PromiseSignal")
	}
	if p.HasCapability(PromiseTimer) {
		names = append(names, "PromiseTimer")
	}
	if p.HasCapability(PromiseAudio) {
		names = append(names, "PromiseAudio")
	}
	if p.HasCapability(PromiseVideo) {
		names = append(names, "PromiseVideo")
	}
	if p.HasCapability(PromiseSocket) {
		names = append(names, "PromiseSocket")
	}
	if p.HasCapability(PromiseResolve) {
		names = append(names, "PromiseResolve")
	}

	result := ""
	for i, name := range names {
		if i > 0 {
			result += " | "
		}
		result += name
	}
	return result
}
