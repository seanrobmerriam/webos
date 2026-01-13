// Package security provides OpenBSD-inspired security primitives including
// pledge (capability promises) and unveil (filesystem path restrictions).
// This package implements capability-based security for the webos project.
package security

import (
	"testing"
	"time"
)

// TestPromiseConstants tests that all promise constants are defined correctly
func TestPromiseConstants(t *testing.T) {
	// Verify promise values are powers of two (bit flags)
	tests := []struct {
		name     string
		promise  Promise
		expected uint64
	}{
		{"PromiseStdio", PromiseStdio, 1},
		{"PromiseRpath", PromiseRpath, 2},
		{"PromiseWpath", PromiseWpath, 4},
		{"PromiseInet", PromiseInet, 8},
		{"PromiseUnix", PromiseUnix, 16},
		{"PromiseFork", PromiseFork, 32},
		{"PromiseExec", PromiseExec, 64},
		{"PromiseSignal", PromiseSignal, 128},
		{"PromiseTimer", PromiseTimer, 256},
		{"PromiseAudio", PromiseAudio, 512},
		{"PromiseVideo", PromiseVideo, 1024},
		{"PromiseSocket", PromiseSocket, 2048},
		{"PromiseResolve", PromiseResolve, 4096},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint64(tt.promise) != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, uint64(tt.promise), tt.expected)
			}
		})
	}
}

// TestPromiseHasCapability tests the HasCapability method
func TestPromiseHasCapability(t *testing.T) {
	t.Run("single capability", func(t *testing.T) {
		p := PromiseStdio | PromiseRpath
		if !p.HasCapability(PromiseStdio) {
			t.Error("Expected to have PromiseStdio capability")
		}
		if !p.HasCapability(PromiseRpath) {
			t.Error("Expected to have PromiseRpath capability")
		}
		if p.HasCapability(PromiseWpath) {
			t.Error("Did not expect to have PromiseWpath capability")
		}
	})

	t.Run("no capabilities", func(t *testing.T) {
		var p Promise
		if p.HasCapability(PromiseStdio) {
			t.Error("Expected no capabilities")
		}
	})

	t.Run("all capabilities", func(t *testing.T) {
		var all Promise = PromiseStdio | PromiseRpath | PromiseWpath |
			PromiseInet | PromiseUnix | PromiseFork | PromiseExec |
			PromiseSignal | PromiseTimer | PromiseAudio | PromiseVideo |
			PromiseSocket | PromiseResolve
		for _, cap := range []Promise{
			PromiseStdio, PromiseRpath, PromiseWpath, PromiseInet,
			PromiseUnix, PromiseFork, PromiseExec, PromiseSignal,
			PromiseTimer, PromiseAudio, PromiseVideo, PromiseSocket,
			PromiseResolve,
		} {
			if !all.HasCapability(cap) {
				t.Errorf("Expected to have capability %v", cap)
			}
		}
	})
}

// TestPromiseAddCapability tests the AddCapability method
func TestPromiseAddCapability(t *testing.T) {
	p := PromiseStdio
	p = p.AddCapability(PromiseRpath)

	if !p.HasCapability(PromiseStdio) {
		t.Error("Expected PromiseStdio after adding")
	}
	if !p.HasCapability(PromiseRpath) {
		t.Error("Expected PromiseRpath after adding")
	}
}

// TestPromiseRemoveCapability tests the RemoveCapability method
func TestPromiseRemoveCapability(t *testing.T) {
	p := PromiseStdio | PromiseRpath | PromiseWpath
	p = p.RemoveCapability(PromiseRpath)

	if !p.HasCapability(PromiseStdio) {
		t.Error("Expected PromiseStdio to remain")
	}
	if p.HasCapability(PromiseRpath) {
		t.Error("PromiseRpath should have been removed")
	}
	if !p.HasCapability(PromiseWpath) {
		t.Error("Expected PromiseWpath to remain")
	}
}

// TestUnveilPath tests the UnveilPath struct
func TestUnveilPath(t *testing.T) {
	t.Run("create unveil path", func(t *testing.T) {
		up := UnveilPath{
			Path:        "/tmp/test",
			Permissions: "rw",
		}
		if up.Path != "/tmp/test" {
			t.Errorf("Path = %s, want /tmp/test", up.Path)
		}
		if up.Permissions != "rw" {
			t.Errorf("Permissions = %s, want rw", up.Permissions)
		}
	})

	t.Run("valid permissions", func(t *testing.T) {
		validPerms := []string{"r", "w", "x", "rw", "rx", "wx", "rwx"}
		for _, perm := range validPerms {
			up := UnveilPath{Path: "/test", Permissions: perm}
			if up.Permissions != perm {
				t.Errorf("Permission %s not set correctly", perm)
			}
		}
	})
}

// TestCapability tests the Capability struct
func TestCapability(t *testing.T) {
	t.Run("create capability", func(t *testing.T) {
		cap := Capability{
			Promises:    PromiseStdio | PromiseRpath,
			UnveilPaths: []UnveilPath{{Path: "/tmp", Permissions: "r"}},
			Timeout:     5 * time.Minute,
		}
		if !cap.Promises.HasCapability(PromiseStdio) {
			t.Error("Expected PromiseStdio")
		}
		if !cap.Promises.HasCapability(PromiseRpath) {
			t.Error("Expected PromiseRpath")
		}
		if len(cap.UnveilPaths) != 1 {
			t.Errorf("UnveilPaths length = %d, want 1", len(cap.UnveilPaths))
		}
		if cap.Timeout != 5*time.Minute {
			t.Errorf("Timeout = %v, want 5m", cap.Timeout)
		}
	})

	t.Run("empty capability", func(t *testing.T) {
		var cap Capability
		if cap.Promises != 0 {
			t.Error("Expected empty promises")
		}
		if cap.UnveilPaths != nil {
			t.Error("Expected nil unveil paths")
		}
		if cap.Timeout != 0 {
			t.Error("Expected zero timeout")
		}
	})
}

// TestNewSecurityManager tests creating a new SecurityManager
func TestNewSecurityManager(t *testing.T) {
	sm := NewSecurityManager()
	if sm == nil {
		t.Fatal("Expected non-nil SecurityManager")
	}
	if sm.policies == nil {
		t.Error("Expected policies to be initialized")
	}
	components := sm.ListComponents()
	if len(components) != 0 {
		t.Errorf("Expected no components, got %d", len(components))
	}
}

// TestSecurityManagerRegisterComponent tests registering a component with capabilities
func TestSecurityManagerRegisterComponent(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{
		Promises:    PromiseStdio | PromiseRpath,
		UnveilPaths: []UnveilPath{{Path: "/tmp", Permissions: "r"}},
		Timeout:     time.Hour,
	}

	err := sm.RegisterComponent("test-component", cap)
	if err != nil {
		t.Errorf("RegisterComponent returned error: %v", err)
	}

	retrieved, ok := sm.GetCapability("test-component")
	if !ok {
		t.Fatal("GetCapability returned false for registered component")
	}
	if retrieved.Promises != cap.Promises {
		t.Errorf("Promises = %v, want %v", retrieved.Promises, cap.Promises)
	}
}

// TestSecurityManagerRegisterDuplicate tests error on duplicate registration
func TestSecurityManagerRegisterDuplicate(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{Promises: PromiseStdio}

	err := sm.RegisterComponent("test-component", cap)
	if err != nil {
		t.Errorf("First registration returned error: %v", err)
	}

	err = sm.RegisterComponent("test-component", cap)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

// TestSecurityManagerGetNonExistent tests getting capability for non-existent component
func TestSecurityManagerGetNonExistent(t *testing.T) {
	sm := NewSecurityManager()
	_, ok := sm.GetCapability("non-existent")
	if ok {
		t.Error("Expected false for non-existent component")
	}
}

// TestSecurityManagerRevokeCapability tests revoking capabilities
func TestSecurityManagerRevokeCapability(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{Promises: PromiseStdio | PromiseRpath}

	sm.RegisterComponent("test-component", cap)
	err := sm.RevokeCapability("test-component")
	if err != nil {
		t.Errorf("RevokeCapability returned error: %v", err)
	}

	_, ok := sm.GetCapability("test-component")
	if ok {
		t.Error("Expected capability to be revoked")
	}
}

// TestSecurityManagerRevokeNonExistent tests revoking non-existent capability
func TestSecurityManagerRevokeNonExistent(t *testing.T) {
	sm := NewSecurityManager()
	err := sm.RevokeCapability("non-existent")
	if err == nil {
		t.Error("Expected error for revoking non-existent component")
	}
}

// TestSecurityManagerCheckPermission tests permission checking
func TestSecurityManagerCheckPermission(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{Promises: PromiseStdio | PromiseRpath}
	sm.RegisterComponent("test-component", cap)

	t.Run("has permission", func(t *testing.T) {
		if !sm.CheckPermission("test-component", PromiseStdio) {
			t.Error("Expected to have PromiseStdio permission")
		}
	})

	t.Run("missing permission", func(t *testing.T) {
		if sm.CheckPermission("test-component", PromiseWpath) {
			t.Error("Did not expect to have PromiseWpath permission")
		}
	})

	t.Run("non-existent component", func(t *testing.T) {
		if sm.CheckPermission("non-existent", PromiseStdio) {
			t.Error("Non-existent component should not have permissions")
		}
	})
}

// TestSecurityManagerRegisterPolicy tests registering policies
func TestSecurityManagerRegisterPolicy(t *testing.T) {
	sm := NewSecurityManager()
	err := sm.RegisterPolicy("http-server", PromiseInet|PromiseResolve)
	if err != nil {
		t.Errorf("RegisterPolicy returned error: %v", err)
	}

	promise, ok := sm.GetPolicy("http-server")
	if !ok {
		t.Fatal("GetPolicy returned false for registered policy")
	}
	if !promise.HasCapability(PromiseInet) {
		t.Error("Expected PromiseInet in policy")
	}
	if !promise.HasCapability(PromiseResolve) {
		t.Error("Expected PromiseResolve in policy")
	}
}

// TestSecurityManagerApplyPolicy tests applying a policy to a component
func TestSecurityManagerApplyPolicy(t *testing.T) {
	sm := NewSecurityManager()
	sm.RegisterPolicy("web-server", PromiseInet|PromiseResolve)

	// Register component first
	cap := &Capability{Promises: PromiseStdio}
	sm.RegisterComponent("test-component", cap)

	err := sm.ApplyPolicy("test-component", "web-server")
	if err != nil {
		t.Errorf("ApplyPolicy returned error: %v", err)
	}

	updatedCap, ok := sm.GetCapability("test-component")
	if !ok {
		t.Fatal("GetCapability returned false after ApplyPolicy")
	}
	if !updatedCap.Promises.HasCapability(PromiseInet) {
		t.Error("Expected PromiseInet after applying policy")
	}
	if !updatedCap.Promises.HasCapability(PromiseResolve) {
		t.Error("Expected PromiseResolve after applying policy")
	}
}

// TestSecurityManagerApplyNonExistentPolicy tests applying non-existent policy
func TestSecurityManagerApplyNonExistentPolicy(t *testing.T) {
	sm := NewSecurityManager()
	err := sm.ApplyPolicy("test-component", "non-existent-policy")
	if err == nil {
		t.Error("Expected error for non-existent policy")
	}
}

// TestSecurityManagerAddUnveilPath tests adding unveil paths
func TestSecurityManagerAddUnveilPath(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{Promises: PromiseStdio}
	sm.RegisterComponent("test-component", cap)

	err := sm.AddUnveilPath("test-component", "/tmp", "rw")
	if err != nil {
		t.Errorf("AddUnveilPath returned error: %v", err)
	}

	cap, ok := sm.GetCapability("test-component")
	if !ok {
		t.Fatal("GetCapability returned false")
	}
	if len(cap.UnveilPaths) != 1 {
		t.Errorf("UnveilPaths length = %d, want 1", len(cap.UnveilPaths))
	}
	if cap.UnveilPaths[0].Path != "/tmp" {
		t.Errorf("Path = %s, want /tmp", cap.UnveilPaths[0].Path)
	}
	if cap.UnveilPaths[0].Permissions != "rw" {
		t.Errorf("Permissions = %s, want rw", cap.UnveilPaths[0].Permissions)
	}
}

// TestStringMethods tests string representations
func TestStringMethods(t *testing.T) {
	t.Run("promise string", func(t *testing.T) {
		if PromiseStdio.String() != "PromiseStdio" {
			t.Errorf("PromiseStdio.String() = %s, want PromiseStdio", PromiseStdio.String())
		}
	})

	t.Run("combined promise string", func(t *testing.T) {
		p := PromiseStdio | PromiseRpath
		str := p.String()
		if str == "" {
			t.Error("Expected non-empty string for combined promises")
		}
	})

	t.Run("empty promise string", func(t *testing.T) {
		var p Promise
		if p.String() != "" {
			t.Errorf("Empty promise string = %s, want empty", p.String())
		}
	})

	t.Run("all promises string", func(t *testing.T) {
		var all Promise = PromiseStdio | PromiseRpath | PromiseWpath |
			PromiseInet | PromiseUnix | PromiseFork | PromiseExec |
			PromiseSignal | PromiseTimer | PromiseAudio | PromiseVideo |
			PromiseSocket | PromiseResolve
		str := all.String()
		if str == "" {
			t.Error("Expected non-empty string for all promises")
		}
	})
}

// TestUnveilPathMethods tests the UnveilPath CanRead, CanWrite, CanExecute methods
func TestUnveilPathMethods(t *testing.T) {
	t.Run("read permission", func(t *testing.T) {
		up := UnveilPath{Path: "/tmp", Permissions: "r"}
		if !up.CanRead() {
			t.Error("Expected CanRead to return true for 'r'")
		}
		if up.CanWrite() {
			t.Error("Expected CanWrite to return false for 'r'")
		}
		if up.CanExecute() {
			t.Error("Expected CanExecute to return false for 'r'")
		}
	})

	t.Run("write permission", func(t *testing.T) {
		up := UnveilPath{Path: "/tmp", Permissions: "w"}
		if up.CanRead() {
			t.Error("Expected CanRead to return false for 'w'")
		}
		if !up.CanWrite() {
			t.Error("Expected CanWrite to return true for 'w'")
		}
		if up.CanExecute() {
			t.Error("Expected CanExecute to return false for 'w'")
		}
	})

	t.Run("execute permission", func(t *testing.T) {
		up := UnveilPath{Path: "/tmp", Permissions: "x"}
		if up.CanRead() {
			t.Error("Expected CanRead to return false for 'x'")
		}
		if up.CanWrite() {
			t.Error("Expected CanWrite to return false for 'x'")
		}
		if !up.CanExecute() {
			t.Error("Expected CanExecute to return true for 'x'")
		}
	})

	t.Run("full permissions", func(t *testing.T) {
		up := UnveilPath{Path: "/tmp", Permissions: "rwx"}
		if !up.CanRead() {
			t.Error("Expected CanRead to return true for 'rwx'")
		}
		if !up.CanWrite() {
			t.Error("Expected CanWrite to return true for 'rwx'")
		}
		if !up.CanExecute() {
			t.Error("Expected CanExecute to return true for 'rwx'")
		}
	})
}

// TestNewUnveilPath tests creating UnveilPath with validation
func TestNewUnveilPath(t *testing.T) {
	t.Run("valid path", func(t *testing.T) {
		up, err := NewUnveilPath("/tmp", "rw")
		if err != nil {
			t.Errorf("NewUnveilPath returned error: %v", err)
		}
		if up.Path != "/tmp" {
			t.Errorf("Path = %s, want /tmp", up.Path)
		}
		if up.Permissions != "rw" {
			t.Errorf("Permissions = %s, want rw", up.Permissions)
		}
	})

	t.Run("invalid permissions", func(t *testing.T) {
		_, err := NewUnveilPath("/tmp", "invalid")
		if err == nil {
			t.Error("Expected error for invalid permissions")
		}
	})
}

// TestValidatePermissions tests permission validation
func TestValidatePermissions(t *testing.T) {
	validPerms := []string{"r", "w", "x", "rw", "rx", "wx", "rwx"}
	for _, perm := range validPerms {
		if !ValidatePermissions(perm) {
			t.Errorf("ValidatePermissions(%s) = false, want true", perm)
		}
	}

	invalidPerms := []string{"", "rxwx", "abc", "读写"}
	for _, perm := range invalidPerms {
		if ValidatePermissions(perm) {
			t.Errorf("ValidatePermissions(%s) = true, want false", perm)
		}
	}
}

// TestUnveilRegistry tests the UnveilRegistry
func TestUnveilRegistry(t *testing.T) {
	registry := NewUnveilRegistry()

	t.Run("add and get paths", func(t *testing.T) {
		registry.AddPath("component1", UnveilPath{Path: "/tmp", Permissions: "r"})
		registry.AddPath("component1", UnveilPath{Path: "/data", Permissions: "rw"})

		paths := registry.GetPaths("component1")
		if len(paths) != 2 {
			t.Errorf("GetPaths length = %d, want 2", len(paths))
		}
	})

	t.Run("has path", func(t *testing.T) {
		registry.AddPath("component2", UnveilPath{Path: "/tmp", Permissions: "r"})
		if !registry.HasPath("component2", "/tmp") {
			t.Error("Expected HasPath to return true")
		}
		if registry.HasPath("component2", "/nonexistent") {
			t.Error("Expected HasPath to return false for nonexistent path")
		}
	})

	t.Run("can access path", func(t *testing.T) {
		registry.AddPath("component3", UnveilPath{Path: "/tmp", Permissions: "rw"})
		if !registry.CanAccessPath("component3", "/tmp", true, true, false) {
			t.Error("Expected CanAccessPath to return true for read/write")
		}
		if registry.CanAccessPath("component3", "/tmp", true, false, true) {
			t.Error("Expected CanAccessPath to return false for execute on rw path")
		}
	})

	t.Run("remove paths", func(t *testing.T) {
		registry.AddPath("component4", UnveilPath{Path: "/tmp", Permissions: "r"})
		registry.RemovePaths("component4")
		paths := registry.GetPaths("component4")
		if len(paths) != 0 {
			t.Errorf("GetPaths length after remove = %d, want 0", len(paths))
		}
	})
}

// TestSecurityManagerListComponents tests listing registered components
func TestSecurityManagerListComponents(t *testing.T) {
	sm := NewSecurityManager()

	components := sm.ListComponents()
	if len(components) != 0 {
		t.Errorf("Expected empty list, got %d components", len(components))
	}

	sm.RegisterComponent("comp1", &Capability{Promises: PromiseStdio})
	sm.RegisterComponent("comp2", &Capability{Promises: PromiseRpath})

	components = sm.ListComponents()
	if len(components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(components))
	}
}

// TestSecurityManagerListPolicies tests listing registered policies
func TestSecurityManagerListPolicies(t *testing.T) {
	sm := NewSecurityManager()

	policies := sm.ListPolicies()
	if len(policies) != 0 {
		t.Errorf("Expected empty list, got %d policies", len(policies))
	}

	sm.RegisterPolicy("policy1", PromiseInet)
	sm.RegisterPolicy("policy2", PromiseResolve)

	policies = sm.ListPolicies()
	if len(policies) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(policies))
	}
}

// TestSecurityManagerGetUnveilPaths tests getting unveil paths for a component
func TestSecurityManagerGetUnveilPaths(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{
		Promises:    PromiseStdio,
		UnveilPaths: []UnveilPath{{Path: "/tmp", Permissions: "r"}},
	}
	sm.RegisterComponent("test-component", cap)

	paths := sm.GetUnveilPaths("test-component")
	if len(paths) != 1 {
		t.Errorf("GetUnveilPaths length = %d, want 1", len(paths))
	}
	if paths[0].Path != "/tmp" {
		t.Errorf("Path = %s, want /tmp", paths[0].Path)
	}
}

// TestSecurityManagerCanAccessPath tests path access checking
func TestSecurityManagerCanAccessPath(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{
		Promises:    PromiseStdio,
		UnveilPaths: []UnveilPath{{Path: "/tmp", Permissions: "rw"}},
	}
	sm.RegisterComponent("test-component", cap)

	t.Run("has read access", func(t *testing.T) {
		if !sm.CanAccessPath("test-component", "/tmp", true, false, false) {
			t.Error("Expected read access")
		}
	})

	t.Run("has write access", func(t *testing.T) {
		if !sm.CanAccessPath("test-component", "/tmp", false, true, false) {
			t.Error("Expected write access")
		}
	})

	t.Run("no execute access", func(t *testing.T) {
		if sm.CanAccessPath("test-component", "/tmp", false, false, true) {
			t.Error("Did not expect execute access")
		}
	})
}

// TestSecurityManagerUpdateTimeout tests updating component timeout
func TestSecurityManagerUpdateTimeout(t *testing.T) {
	sm := NewSecurityManager()
	cap := &Capability{Promises: PromiseStdio, Timeout: time.Hour}
	sm.RegisterComponent("test-component", cap)

	err := sm.UpdateTimeout("test-component", 2*time.Hour)
	if err != nil {
		t.Errorf("UpdateTimeout returned error: %v", err)
	}

	updatedCap, ok := sm.GetCapability("test-component")
	if !ok {
		t.Fatal("GetCapability returned false")
	}
	if updatedCap.Timeout != 2*time.Hour {
		t.Errorf("Timeout = %v, want 2h", updatedCap.Timeout)
	}

	t.Run("update non-existent component", func(t *testing.T) {
		err := sm.UpdateTimeout("non-existent", time.Hour)
		if err == nil {
			t.Error("Expected error for non-existent component")
		}
	})
}

// TestSecurityManagerAddUnveilPathToNonExistent tests adding path to non-existent component
func TestSecurityManagerAddUnveilPathToNonExistent(t *testing.T) {
	sm := NewSecurityManager()
	err := sm.AddUnveilPath("non-existent", "/tmp", "r")
	if err == nil {
		t.Error("Expected error for non-existent component")
	}
}
