// Package api holds the DTOs that form the HTTP contract.
//
// The types in types.gen.go are GENERATED from openapi/openapi.yaml and must
// not be edited by hand. openapi.yaml is the single source of truth: the same
// document also generates the TypeScript types used by the Svelte frontend, so
// the two ends of the wire cannot drift apart.
//
// Regenerate with `make generate` (or `go generate ./...`).
package api

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config ../../oapi-codegen.yaml -o types.gen.go ../../../openapi/openapi.yaml
