package schema

import (
	"strings"
)

type RefKind int

const (
	RefNone RefKind = iota
	RefDB
	RefSecret
)

type EnvRef struct {
	Kind       RefKind
	DBName     string
	SecretName string
	TenantDB   bool
}

func ParseEnvValue(value string) (EnvRef, bool) {
	switch {
	case strings.HasPrefix(value, "$db:"):
		rest := strings.TrimPrefix(value, "$db:")
		if rest == "" {
			return EnvRef{}, false
		}
		if strings.HasPrefix(rest, "tenant/") {
			name := strings.TrimPrefix(rest, "tenant/")
			if name == "" {
				return EnvRef{}, false
			}
			return EnvRef{Kind: RefDB, DBName: name, TenantDB: true}, true
		}
		return EnvRef{Kind: RefDB, DBName: rest, TenantDB: false}, true
	case strings.HasPrefix(value, "$secret:"):
		name := strings.TrimPrefix(value, "$secret:")
		if name == "" {
			return EnvRef{}, false
		}
		return EnvRef{Kind: RefSecret, SecretName: name}, true
	default:
		return EnvRef{}, false
	}
}

func IsDollarRef(value string) bool {
	return strings.HasPrefix(value, "$")
}
