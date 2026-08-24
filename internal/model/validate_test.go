package model

import (
	"testing"
	"time"
)

// TestCreateReqValidate_NoKeywords ensures the create-short-url validation does
// not panic when the keywords field is omitted (empty slice), and instead
// succeeds. This is the regression test for the
// "index out of range [0] with length 0" crash reported when creating a short
// URL without supplying a keywords field.
func TestCreateReqValidate_NoKeywords(t *testing.T) {
	t.Run("nil keywords does not panic and is valid", func(t *testing.T) {
		req := &CreateReq{RawURL: "https://example.com"}

		if err := req.Validate(); err != nil {
			t.Fatalf("expected nil error for valid request without keywords, got %v", err)
		}
	})

	t.Run("empty keywords slice does not panic and is valid", func(t *testing.T) {
		req := &CreateReq{RawURL: "https://example.com", Keywords: []string{}}

		if err := req.Validate(); err != nil {
			t.Fatalf("expected nil error for valid request with empty keywords, got %v", err)
		}
	})

	t.Run("blank keyword is rejected", func(t *testing.T) {
		req := &CreateReq{RawURL: "https://example.com", Keywords: []string{""}}

		if err := req.Validate(); err == nil {
			t.Fatal("expected error for blank keyword, got nil")
		}
	})

	t.Run("oversized keyword is rejected", func(t *testing.T) {
		kw := make([]byte, 65)
		for i := range kw {
			kw[i] = 'a'
		}
		req := &CreateReq{RawURL: "https://example.com", Keywords: []string{string(kw)}}

		if err := req.Validate(); err == nil {
			t.Fatal("expected error for oversized keyword, got nil")
		}
	})

	t.Run("valid keyword is accepted", func(t *testing.T) {
		req := &CreateReq{RawURL: "https://example.com", Keywords: []string{"promo"}}

		if err := req.Validate(); err != nil {
			t.Fatalf("expected nil error for valid keyword, got %v", err)
		}
	})
}

// TestShortURLValidate_NoTags ensures the ShortURL validation does not panic
// when Tags is nil or empty.
func TestShortURLValidate_NoTags(t *testing.T) {
	t.Run("nil tags does not panic", func(t *testing.T) {
		s := &ShortURL{Code: "abc12345", RawURL: "https://example.com"}

		if err := s.Validate(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("empty tags does not panic", func(t *testing.T) {
		s := &ShortURL{Code: "abc12345", RawURL: "https://example.com", Tags: []string{}}

		if err := s.Validate(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("oversized tag is rejected", func(t *testing.T) {
		tag := make([]byte, 129)
		for i := range tag {
			tag[i] = 't'
		}
		s := &ShortURL{Code: "abc12345", RawURL: "https://example.com", Tags: []string{string(tag)}}

		if err := s.Validate(); err == nil {
			t.Fatal("expected error for oversized tag, got nil")
		}
	})
}

// TestLogQueryValidate_EmptyFilter ensures the log query validation does not
// panic when the filter object carries empty (or nil) slice fields. This is
// the regression test for the "index out of range [0] with length 0" crash
// reported when querying logs with an empty filter object.
func TestLogQueryValidate_EmptyFilter(t *testing.T) {
	now := time.Now()

	t.Run("filter with empty slices does not panic", func(t *testing.T) {
		q := &LogQuery{
			Filter:    &LogFilter{Levels: []LogLevel{}, Sources: []string{}},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     50,
		}

		// Must not panic; should return no validation errors.
		errs := q.Validate()
		if len(errs) != 0 {
			t.Fatalf("expected no validation errors for empty filter, got %v", errs)
		}
	})

	t.Run("nil filter slice fields do not panic", func(t *testing.T) {
		q := &LogQuery{
			Filter:    &LogFilter{},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     50,
		}

		errs := q.Validate()
		if len(errs) != 0 {
			t.Fatalf("expected no validation errors, got %v", errs)
		}
	})

	t.Run("debug level is still flagged when present", func(t *testing.T) {
		q := &LogQuery{
			Filter:    &LogFilter{Levels: []LogLevel{LevelDebug}},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     50,
		}

		errs := q.Validate()
		if len(errs) == 0 {
			t.Fatal("expected validation error for debug level, got none")
		}
	})

	t.Run("empty source string is still flagged when present", func(t *testing.T) {
		q := &LogQuery{
			Filter:    &LogFilter{Sources: []string{""}},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     50,
		}

		errs := q.Validate()
		if len(errs) == 0 {
			t.Fatal("expected validation error for empty source, got none")
		}
	})

	t.Run("nil filter with large limit is flagged", func(t *testing.T) {
		q := &LogQuery{
			Filter: nil,
			Limit:  600,
		}

		errs := q.Validate()
		if len(errs) == 0 {
			t.Fatal("expected validation error for unfiltered large query, got none")
		}
	})

	_ = now
}

// TestAlertQueryValidate_EmptyFilter ensures the alert query validation does
// not panic when the filter object carries empty slice fields.
func TestAlertQueryValidate_EmptyFilter(t *testing.T) {
	t.Run("filter with empty slices does not panic", func(t *testing.T) {
		q := &AlertQuery{
			Filter: &AlertFilter{
				Statuses:   []AlertStatus{},
				Severities: []Severity{},
				RuleIDs:    []string{},
			},
			Limit: 50,
		}

		errs := q.Validate()
		if len(errs) != 0 {
			t.Fatalf("expected no validation errors for empty filter, got %v", errs)
		}
	})

	t.Run("nil filter slice fields do not panic", func(t *testing.T) {
		q := &AlertQuery{Filter: &AlertFilter{}, Limit: 50}

		errs := q.Validate()
		if len(errs) != 0 {
			t.Fatalf("expected no validation errors, got %v", errs)
		}
	})

	t.Run("open status is still flagged when present", func(t *testing.T) {
		q := &AlertQuery{
			Filter: &AlertFilter{Statuses: []AlertStatus{AlertOpen}},
			Limit:  50,
		}

		errs := q.Validate()
		if len(errs) == 0 {
			t.Fatal("expected validation error for open status, got none")
		}
	})
}
