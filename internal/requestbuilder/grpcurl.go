package requestbuilder

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TLSConfig holds TLS-related settings for grpcurl command generation.
type TLSConfig struct {
	Plaintext  bool
	CACert     string
	Cert       string
	Key        string
	ServerName string
}

// BuildGrpcurlCommand constructs a grpcurl command string from the
// current form state, metadata, and connection settings.
func BuildGrpcurlCommand(
	host string,
	plaintext bool,
	serviceName string,
	methodName string,
	fields []FormField,
	metadata map[string]string,
	tlsConfig TLSConfig,
) string {
	var parts []string
	parts = append(parts, "grpcurl")

	if plaintext || tlsConfig.Plaintext {
		parts = append(parts, "-plaintext")
	}
	if tlsConfig.CACert != "" {
		parts = append(parts, fmt.Sprintf("-cacert %s", tlsConfig.CACert))
	}
	if tlsConfig.Cert != "" {
		parts = append(parts, fmt.Sprintf("-cert %s", tlsConfig.Cert))
	}
	if tlsConfig.Key != "" {
		parts = append(parts, fmt.Sprintf("-key %s", tlsConfig.Key))
	}
	if tlsConfig.ServerName != "" {
		parts = append(parts, fmt.Sprintf("-servername %s", tlsConfig.ServerName))
	}

	// Build JSON data from fields.
	data := collectFieldValues(fields)
	if len(data) > 0 {
		jsonBytes, err := json.Marshal(data)
		if err == nil {
			parts = append(parts, fmt.Sprintf("-d '%s'", string(jsonBytes)))
		}
	}

	// Add metadata headers (sorted for deterministic output).
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("-H '%s: %s'", k, metadata[k]))
	}

	// Add host and method.
	parts = append(parts, host)
	parts = append(parts, fmt.Sprintf("%s/%s", serviceName, methodName))

	// Join with line continuations for readability.
	if len(parts) <= 3 {
		return strings.Join(parts, " ")
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " \\\n  " + parts[i]
	}
	return result
}

// collectFieldValues builds a map of field name → value for JSON serialization.
func collectFieldValues(fields []FormField) map[string]any {
	result := make(map[string]any)
	for _, f := range fields {
		if f.OneofGroup != "" && !f.OneofActive {
			continue
		}
		if f.Widget == nil {
			continue
		}
		v := f.Widget.Value()
		if v == "" {
			continue
		}
		result[f.Info.Name] = v
	}
	return result
}
