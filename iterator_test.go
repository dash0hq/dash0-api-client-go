package dash0

import (
	"errors"
	"testing"
)

func TestIter(t *testing.T) {
	t.Run("iterates over initial items", func(t *testing.T) {
		items := []*string{ptr("a"), ptr("b"), ptr("c")}
		iter := newIter(items, false, nil, nil)

		var result []string
		for iter.Next() {
			result = append(result, *iter.Current())
		}

		if iter.Err() != nil {
			t.Fatalf("unexpected error: %v", iter.Err())
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 items, got %d", len(result))
		}
		if result[0] != "a" || result[1] != "b" || result[2] != "c" {
			t.Errorf("unexpected items: %v", result)
		}
	})

	t.Run("handles empty iterator", func(t *testing.T) {
		iter := newIter([]*string{}, false, nil, nil)

		if iter.Next() {
			t.Error("expected Next() to return false for empty iterator")
		}
		if iter.Err() != nil {
			t.Errorf("unexpected error: %v", iter.Err())
		}
		if iter.Current() != nil {
			t.Error("expected Current() to return nil for empty iterator")
		}
	})

	t.Run("fetches next page when hasMore is true", func(t *testing.T) {
		page1 := []*string{ptr("a"), ptr("b")}
		page2 := []*string{ptr("c"), ptr("d")}
		cursor := "cursor1"

		fetchCalled := 0
		fetch := func(c *string) ([]*string, *string, error) {
			fetchCalled++
			if c == nil || *c != "cursor1" {
				t.Errorf("expected cursor 'cursor1', got %v", c)
			}
			return page2, nil, nil
		}

		iter := newIter(page1, true, &cursor, fetch)

		var result []string
		for iter.Next() {
			result = append(result, *iter.Current())
		}

		if iter.Err() != nil {
			t.Fatalf("unexpected error: %v", iter.Err())
		}
		if fetchCalled != 1 {
			t.Errorf("expected fetch to be called once, got %d", fetchCalled)
		}
		if len(result) != 4 {
			t.Fatalf("expected 4 items, got %d", len(result))
		}
		expected := []string{"a", "b", "c", "d"}
		for i, v := range result {
			if v != expected[i] {
				t.Errorf("item %d: expected %s, got %s", i, expected[i], v)
			}
		}
	})

	t.Run("fetches multiple pages", func(t *testing.T) {
		cursor1 := "cursor1"
		cursor2 := "cursor2"

		fetchCalled := 0
		fetch := func(c *string) ([]*string, *string, error) {
			fetchCalled++
			switch fetchCalled {
			case 1:
				return []*string{ptr("b")}, &cursor2, nil
			case 2:
				return []*string{ptr("c")}, nil, nil
			default:
				t.Error("fetch called too many times")
				return nil, nil, nil
			}
		}

		iter := newIter([]*string{ptr("a")}, true, &cursor1, fetch)

		var result []string
		for iter.Next() {
			result = append(result, *iter.Current())
		}

		if iter.Err() != nil {
			t.Fatalf("unexpected error: %v", iter.Err())
		}
		if fetchCalled != 2 {
			t.Errorf("expected fetch to be called twice, got %d", fetchCalled)
		}
		if len(result) != 3 {
			t.Fatalf("expected 3 items, got %d", len(result))
		}
	})

	t.Run("stops iteration on fetch error", func(t *testing.T) {
		cursor := "cursor1"
		fetchErr := errors.New("fetch failed")

		fetch := func(c *string) ([]*string, *string, error) {
			return nil, nil, fetchErr
		}

		iter := newIter([]*string{ptr("a")}, true, &cursor, fetch)

		var result []string
		for iter.Next() {
			result = append(result, *iter.Current())
		}

		if iter.Err() != fetchErr {
			t.Errorf("expected fetch error, got %v", iter.Err())
		}
		if len(result) != 1 {
			t.Errorf("expected 1 item before error, got %d", len(result))
		}
	})

	t.Run("does not fetch when hasMore is false", func(t *testing.T) {
		fetchCalled := false
		fetch := func(c *string) ([]*string, *string, error) {
			fetchCalled = true
			return nil, nil, nil
		}

		iter := newIter([]*string{ptr("a")}, false, nil, fetch)

		for iter.Next() {
		}

		if fetchCalled {
			t.Error("fetch should not be called when hasMore is false")
		}
	})

	t.Run("does not fetch when fetch function is nil", func(t *testing.T) {
		cursor := "cursor1"
		iter := newIter([]*string{ptr("a")}, true, &cursor, nil)

		var result []string
		for iter.Next() {
			result = append(result, *iter.Current())
		}

		if len(result) != 1 {
			t.Errorf("expected 1 item, got %d", len(result))
		}
	})

	t.Run("handles fetch returning empty page", func(t *testing.T) {
		cursor := "cursor1"
		fetch := func(c *string) ([]*string, *string, error) {
			return []*string{}, nil, nil
		}

		iter := newIter([]*string{ptr("a")}, true, &cursor, fetch)

		var result []string
		for iter.Next() {
			result = append(result, *iter.Current())
		}

		if len(result) != 1 {
			t.Errorf("expected 1 item, got %d", len(result))
		}
	})
}

func TestNewIterWithError(t *testing.T) {
	t.Run("returns error immediately", func(t *testing.T) {
		expectedErr := errors.New("initial error")
		iter := newIterWithError[string](expectedErr)

		if iter.Next() {
			t.Error("expected Next() to return false")
		}
		if iter.Err() != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, iter.Err())
		}
		if iter.Current() != nil {
			t.Error("expected Current() to return nil")
		}
	})

	t.Run("Next always returns false after error", func(t *testing.T) {
		iter := newIterWithError[string](errors.New("error"))

		// Call Next multiple times
		for i := 0; i < 3; i++ {
			if iter.Next() {
				t.Errorf("Next() returned true on call %d", i)
			}
		}
	})
}

func ptr[T any](v T) *T {
	return &v
}
