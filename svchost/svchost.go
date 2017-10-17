// Package svchost deals with the representations of the so-called "friendly
// hostnames" that we use to represent systems that provide Terraform-native
// remote services, such as module registry, remote operations, etc.
package svchost

import (
	"fmt"
	"strings"

	"golang.org/x/net/idna"
)

// Hostname is specialized name for string that indicates that the string
// has been converted to (or was already in) the storage and comparison form.
//
// Hostname values are not suitable for display in the user-interface. Use
// the ForDisplay method to obtain a form suitable for display in the UI.
//
// Unlike user-supplied hostnames, strings of type Hostname (assuming they
// were constructed by a function within this package) can be compared for
// equality using the standard Go == operator.
type Hostname string

// acePrefix is the ASCII Compatible Encoding prefix, used to indicate that
// a domain name label is in "punycode" form.
const acePrefix = "xn--"

// displayProfile is a very liberal idna profile that we use to do
// normalization for display without imposing validation rules.
var displayProfile = idna.New(
	idna.MapForLookup(),
	idna.Transitional(true),
)

// comparisonProfile is a stricter idna profile that combines basic
// normalization with validation rules that prevent use of ambiguous or
// invalid hostnames.
var comparisonProfile = idna.New(
	idna.MapForLookup(),
	idna.Transitional(true),
	idna.VerifyDNSLength(true),
	idna.ValidateLabels(true),
)

// ForDisplay takes a user-specified hostname and returns a normalized form of
// it suitable for display in the UI.
//
// If the input is so invalid that no normalization can be performed (or,
// indeed, if the input is the empty string) then this will return the
// empty string. However, this function tolerates some invalid forms that
// ForComparison does not tolerate.
//
// For validation, use either IsValid (for explicit validation) or
// ForComparison (which implicitly validates, returning an error if invalid).
func ForDisplay(given string) string {
	ascii, err := displayProfile.ToASCII(given)
	if err != nil {
		return ""
	}
	display, err := displayProfile.ToUnicode(ascii)
	if err != nil {
		return ""
	}
	return display
}

// IsValid returns true if the given user-specified hostname is a valid
// service hostname.
//
// Validity is determined by complying with the RFC 5891 requirements for
// names that are valid for domain registration (section 4.2), with the
// additional requirement that user-supplied forms must not _already_ contain
// Punycode segments.
func IsValid(given string) bool {
	_, err := ForComparison(given)
	return err == nil
}

// ForComparison takes a user-specified hostname and returns a normalized
// form of it suitable for storage and comparison. The result is not suitable
// for display to end-users because it uses Punycode to represent non-ASCII
// characters, and this form is unreadable for non-ASCII-speaking humans.
//
// The result is typed as Hostname -- a specialized name for string -- so that
// other APIs can make it clear within the type system whether they expect a
// user-specified or display-form hostname or a value already normalized for
// comparison.
//
// The returned Hostname is not valid if the returned error is non-nil.
func ForComparison(given string) (Hostname, error) {
	if given == "" {
		return Hostname(""), fmt.Errorf("empty string is not a valid hostname")
	}

	// First we'll apply our additional constraint that Punycode must not
	// be given directly by the user. This is not an IDN specification
	// requirement, but we prohibit it to force users to use human-readable
	// hostname forms within Terraform configuration.
	labels := labelIter{orig: given}
	for ; !labels.done(); labels.next() {
		label := labels.label()
		if strings.HasPrefix(label, acePrefix) {
			return Hostname(""), fmt.Errorf(
				"hostname label %q specified in punycode format; service hostnames must be given in unicode",
				label,
			)
		}
	}

	result, err := comparisonProfile.ToASCII(given)
	if err != nil {
		return Hostname(""), err
	}
	return Hostname(result), nil
}

// ForDisplay returns a version of the receiver that is appropriate for display
// in the UI. This includes converting any punycode labels to their
// corresponding Unicode characters.
//
// A round-trip through ForComparison and this ForDisplay method does not
// guarantee the same result as calling this package's top-level ForDisplay
// function, since a round-trip through the Hostname type implies stricter
// handling than we do when doing basic display-only processing.
func (h Hostname) ForDisplay() string {
	result, err := comparisonProfile.ToUnicode(string(h))
	if err != nil {
		// Should never happen, since type Hostname indicates that a string
		// passed through our validation rules.
		panic(fmt.Errorf("ForDisplay called on invalid Hostname: %s", err))
	}
	return result
}

func (h Hostname) String() string {
	return string(h)
}

func (h Hostname) GoString() string {
	return fmt.Sprintf("svchost.Hostname(%q)", string(h))
}
