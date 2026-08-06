package core

import (
	"fmt"
	"regexp"
)

// Identifier patterns, frozen by SCHEMA.md schema version 1.
var (
	lineageRe = regexp.MustCompile(`^\d{4}-\d{2}-[a-z0-9]+(-[a-z0-9]+)*$`)
	projectRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$`)
	adrRe     = regexp.MustCompile(`^adr-\d{4}-[a-z0-9]+(-[a-z0-9]+)*$`)
	cycleRe   = regexp.MustCompile(`^\d{4}-C\d+$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ValidLineageID reports whether s is a well-formed lineage id.
func ValidLineageID(s string) bool { return lineageRe.MatchString(s) }

// ValidProjectID reports whether s is a well-formed project id (owner/name).
func ValidProjectID(s string) bool { return projectRe.MatchString(s) }

// ValidADRID reports whether s is a well-formed ADR id.
func ValidADRID(s string) bool { return adrRe.MatchString(s) }

// ValidCycleID reports whether s is a well-formed cycle id.
func ValidCycleID(s string) bool { return cycleRe.MatchString(s) }

// ValidDate reports whether s is an ISO 8601 date (YYYY-MM-DD, shape only).
func ValidDate(s string) bool { return dateRe.MatchString(s) }

// QualifiedID joins a project id and a local id into the aggregated index key.
// It is never stored in files; the core computes it from the owning manifest.
func QualifiedID(projectID, localID string) string {
	return fmt.Sprintf("%s:%s", projectID, localID)
}
