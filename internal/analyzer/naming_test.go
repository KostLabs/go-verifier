package analyzer

import (
	"testing"
)

func TestNaming(t *testing.T) {
	a := Naming{}

	tests := []struct {
		name      string
		given     string
		pkgName   string
		src       string
		wantRules []string
	}{
		{
			name:    "interface prefixed with I",
			given:   "an interface named IUserRepository",
			pkgName: "repository",
			src: `package repository
type IUserRepository interface {
	Find(id int) error
}`,
			wantRules: []string{"naming"},
		},
		{
			name:    "single-method interface prefixed with I",
			given:   "a single-method interface named IReader",
			pkgName: "io2",
			src: `package io2
type IReader interface {
	Read(p []byte) (int, error)
}`,
			wantRules: []string{"naming"},
		},
		{
			name:    "constant using UPPER_SNAKE_CASE",
			given:   "a constant named MAX_RETRIES",
			pkgName: "config",
			src: `package config
const MAX_RETRIES = 3`,
			wantRules: []string{"naming"},
		},
		{
			name:    "multiple UPPER_SNAKE_CASE constants",
			given:   "three constants all using UPPER_SNAKE_CASE",
			pkgName: "config",
			src: `package config
const (
	MAX_RETRIES     = 3
	DEFAULT_TIMEOUT = 30
	MIN_LENGTH      = 8
)`,
			wantRules: []string{"naming", "naming", "naming"},
		},
		{
			name:    "generic package name utils",
			given:   "a package named utils",
			pkgName: "utils",
			src:     `package utils`,
			wantRules: []string{"naming"},
		},
		{
			name:    "generic package name helpers",
			given:   "a package named helpers",
			pkgName: "helpers",
			src:     `package helpers`,
			wantRules: []string{"naming"},
		},
		{
			name:    "valid interface name without I prefix",
			given:   "an interface named UserRepository (no I prefix)",
			pkgName: "repository",
			src: `package repository
type UserRepository interface {
	Find(id int) error
}`,
			wantRules: nil,
		},
		{
			name:    "valid MixedCaps constant",
			given:   "a constant named MaxRetries using MixedCaps",
			pkgName: "config",
			src: `package config
const MaxRetries = 3`,
			wantRules: nil,
		},
		{
			name:    "descriptive package name",
			given:   "a package named userservice",
			pkgName: "userservice",
			src:     `package userservice`,
			wantRules: nil,
		},
		{
			name:    "ignore directive suppresses interface naming finding",
			given:   "an I-prefixed interface annotated with //goverifier:ignore:naming",
			pkgName: "repository",
			src: `package repository
//goverifier:ignore:naming
type IUserRepository interface {
	Find(id int) error
}`,
			wantRules: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// When: the analyzer runs on the given source
			got := runAnalyzer(t, a, tc.pkgName, tc.src)

			// Then: diagnostics match expectations
			assertDiags(t, got, tc.wantRules...)
		})
	}
}
