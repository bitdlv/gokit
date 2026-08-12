// Package gwgen provides utilities for parsing .proto files and generating
// gRPC-gateway route registration Go source files.
//
// It is designed to be used as a library by project-specific code generators.
package gwgen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// ──────────────────────────────── Types ──────────────────────────────────────

// RPC represents a single gRPC method with an HTTP binding.
type RPC struct {
	ServiceName string
	MethodName  string
	HTTPMethod  string
	HTTPPath    string
}

// RouteEntry is passed into the template for a single route line.
type RouteEntry struct {
	Entry   string // e.g. {Method: http.MethodPost, Path: "/sys/user/create", Handler: h},
	Comment string // e.g. // → SysService.UserCreate
}

// ServiceGroup groups routes under a service name for the auth block.
type ServiceGroup struct {
	ServiceName string
	Routes      []RouteEntry
}

// TemplateData is the top-level data passed to the template.
type TemplateData struct {
	NoAuthRoutes []RouteEntry
	AuthGroups   []ServiceGroup
}

// Config holds all parameters for a generation run.
type Config struct {
	// ProtoFile is the path to the .proto source file.
	ProtoFile string
	// OutFile is the output Go source file path.
	OutFile string
	// TplFile is the path to the Go template file. If empty, TplContent is used.
	TplFile string
	// TplContent is inline template content (used when TplFile is empty).
	TplContent string
	// NoAuthPaths is a set of HTTP paths that bypass auth middleware.
	NoAuthPaths map[string]bool
}

// ──────────────────────────────── Public API ─────────────────────────────────

// ParseProto reads a .proto file and extracts all RPC methods with HTTP bindings.
func ParseProto(filename string) ([]RPC, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer file.Close()

	serviceRe := regexp.MustCompile(`^service\s+(\w+)`)
	rpcRe := regexp.MustCompile(`^\s*rpc\s+(\w+)`)
	httpMethodRe := regexp.MustCompile(`^\s*(get|post|put|delete|patch):\s*"([^"]+)"`)

	var rpcs []RPC
	var currentService, currentRPC string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if m := serviceRe.FindStringSubmatch(line); m != nil {
			currentService = m[1]
			currentRPC = ""
			continue
		}
		if m := rpcRe.FindStringSubmatch(line); m != nil && currentService != "" {
			currentRPC = m[1]
			continue
		}
		if m := httpMethodRe.FindStringSubmatch(line); m != nil && currentService != "" && currentRPC != "" {
			rpcs = append(rpcs, RPC{
				ServiceName: currentService,
				MethodName:  currentRPC,
				HTTPMethod:  strings.ToUpper(m[1]),
				HTTPPath:    m[2],
			})
			currentRPC = ""
		}
	}
	return rpcs, scanner.Err()
}

// Generate parses the proto file and writes the routes Go file using the provided config.
func Generate(cfg Config) error {
	rpcs, err := ParseProto(cfg.ProtoFile)
	if err != nil {
		return err
	}

	noAuth := cfg.NoAuthPaths
	if noAuth == nil {
		noAuth = map[string]bool{}
	}

	var noAuthRPCs, authRPCs []RPC
	for _, r := range rpcs {
		if noAuth[r.HTTPPath] {
			noAuthRPCs = append(noAuthRPCs, r)
		} else {
			authRPCs = append(authRPCs, r)
		}
	}

	data := TemplateData{
		NoAuthRoutes: BuildRouteEntries(noAuthRPCs),
		AuthGroups:   BuildServiceGroups(authRPCs),
	}

	var tpl *template.Template
	if cfg.TplFile != "" {
		tpl, err = template.ParseFiles(cfg.TplFile)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", cfg.TplFile, err)
		}
	} else if cfg.TplContent != "" {
		tpl, err = template.New("gw_routes").Parse(cfg.TplContent)
		if err != nil {
			return fmt.Errorf("failed to parse inline template: %w", err)
		}
	} else {
		return fmt.Errorf("either TplFile or TplContent must be provided")
	}

	if dir := filepath.Dir(cfg.OutFile); dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.Create(cfg.OutFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", cfg.OutFile, err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("✅ ->->-> Generated %s with %d routes\n", cfg.OutFile, len(rpcs))
	return nil
}

// ──────────────────────────── Builders ───────────────────────────────────────

// BuildRouteEntries converts a slice of RPCs into template-ready RouteEntry items.
func BuildRouteEntries(rpcs []RPC) []RouteEntry {
	entries := make([]RouteEntry, 0, len(rpcs))
	for _, r := range rpcs {
		entries = append(entries, ToRouteEntry(r))
	}
	return entries
}

// BuildServiceGroups groups RPCs by ServiceName preserving proto order.
func BuildServiceGroups(rpcs []RPC) []ServiceGroup {
	var groups []ServiceGroup
	seen := map[string]int{}
	for _, r := range rpcs {
		idx, ok := seen[r.ServiceName]
		if !ok {
			idx = len(groups)
			seen[r.ServiceName] = idx
			groups = append(groups, ServiceGroup{ServiceName: r.ServiceName})
		}
		groups[idx].Routes = append(groups[idx].Routes, ToRouteEntry(r))
	}
	return groups
}

// ToRouteEntry formats a single RPC into the entry string and comment string.
func ToRouteEntry(r RPC) RouteEntry {
	path := ProtoPathToGoZero(r.HTTPPath)
	entry := fmt.Sprintf("{Method: %s, Path: \"%s\", Handler: h},", HTTPMethodConst(r.HTTPMethod), path)
	comment := fmt.Sprintf("// → %s.%s", r.ServiceName, r.MethodName)
	return RouteEntry{
		Entry:   fmt.Sprintf("%-72s", entry),
		Comment: comment,
	}
}

// ──────────────────────────────── Helpers ────────────────────────────────────

// ProtoPathToGoZero converts proto-style path params {foo} to go-zero-style :foo.
func ProtoPathToGoZero(path string) string {
	re := regexp.MustCompile(`\{(\w+)\}`)
	return re.ReplaceAllString(path, ":$1")
}

// HTTPMethodConst returns the Go http.MethodXxx constant name for a method string.
func HTTPMethodConst(method string) string {
	s := strings.ToLower(method)
	return "http.Method" + strings.ToUpper(s[:1]) + s[1:]
}

