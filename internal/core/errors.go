package core

import "fmt"

// Severity classifies a document problem. MUST violations drop the document
// from the index; SHOULD violations keep it with a warning.
type Severity string

const (
	SevError   Severity = "error"   // MUST violation, document dropped
	SevWarning Severity = "warning" // SHOULD violation, document indexed
)

// DocError is one problem found in one document.
type DocError struct {
	Path     string
	Field    string
	Severity Severity
	Message  string
}

func (e DocError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s [%s] %s: %s", e.Path, e.Severity, e.Field, e.Message)
	}
	return fmt.Sprintf("%s [%s]: %s", e.Path, e.Severity, e.Message)
}

// fatal reports whether this error drops the document from the index.
func (e DocError) fatal() bool { return e.Severity == SevError }

// errbag accumulates DocErrors for one document during parsing/validation.
type errbag struct {
	path string
	errs []DocError
}

func (b *errbag) errf(field, format string, a ...any) {
	b.errs = append(b.errs, DocError{Path: b.path, Field: field, Severity: SevError, Message: fmt.Sprintf(format, a...)})
}

func (b *errbag) warnf(field, format string, a ...any) {
	b.errs = append(b.errs, DocError{Path: b.path, Field: field, Severity: SevWarning, Message: fmt.Sprintf(format, a...)})
}

// hasFatal reports whether any accumulated error is a MUST violation.
func (b *errbag) hasFatal() bool {
	for _, e := range b.errs {
		if e.fatal() {
			return true
		}
	}
	return false
}
