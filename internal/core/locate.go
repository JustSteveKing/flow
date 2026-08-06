package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func itoa(n int) string { return strconv.Itoa(n) }
func atoi(s string) int { n, _ := strconv.Atoi(s); return n }
func pad4(n int) string { return fmt.Sprintf("%04d", n) }

// FindRoot walks up from start until it finds a directory containing
// `.flow/manifest.yaml`, and returns that directory. It reports whether one was
// found.
func FindRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, ".flow", "manifest.yaml")); err == nil && !fi.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

var slugCleanRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a free-text title into the slug portion of a lineage id.
func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugCleanRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

// MintLineage builds a lineage id from a year-month prefix and a title. It checks
// the given set of existing lineages and appends a numeric suffix on collision.
func MintLineage(yearMonth, title string, existing map[string]*Pitch) string {
	base := yearMonth + "-" + Slugify(title)
	if _, clash := existing[base]; !clash {
		return base
	}
	for n := 2; ; n++ {
		cand := base + "-" + itoa(n)
		if _, clash := existing[cand]; !clash {
			return cand
		}
	}
}

// NextADRID returns the next sequential ADR id for a project given the prefix and
// the existing decisions across both layers.
func NextADRID(prefix string, existing map[string]*Decision, title string) string {
	if prefix == "" {
		prefix = "adr"
	}
	max := 0
	for id := range existing {
		if n := adrSeq(id); n > max {
			max = n
		}
	}
	return prefix + "-" + pad4(max+1) + "-" + Slugify(title)
}

var adrSeqRe = regexp.MustCompile(`^adr-(\d{4})-`)

func adrSeq(id string) int {
	m := adrSeqRe.FindStringSubmatch(id)
	if m == nil {
		return 0
	}
	return atoi(m[1])
}
