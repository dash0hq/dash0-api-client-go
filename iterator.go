package dash0

// Iter provides iteration over paginated API results.
// Use Next() to advance, Current() to get the item, and Err() to check for errors.
//
// Example:
//
//	iter := client.GetSpansIter(ctx, request)
//	for iter.Next() {
//	    span := iter.Current()
//	    // process span
//	}
//	if err := iter.Err(); err != nil {
//	    // handle error
//	}
//
// Iterators are not thread-safe. Do not share an iterator across goroutines.
type Iter[T any] struct {
	cur     *T
	err     error
	items   []*T
	idx     int
	hasMore bool
	fetch   func(cursor *string) ([]*T, *string, error)
	cursor  *string
}

// Next advances the iterator to the next item.
// Returns true if there is a next item, false if iteration is complete or an error occurred.
func (it *Iter[T]) Next() bool {
	if it.err != nil {
		return false
	}

	it.idx++
	if it.idx < len(it.items) {
		it.cur = it.items[it.idx]
		return true
	}

	// Need to fetch more?
	if !it.hasMore || it.fetch == nil {
		return false
	}

	items, nextCursor, err := it.fetch(it.cursor)
	if err != nil {
		it.err = err
		return false
	}

	it.items = items
	it.idx = 0
	it.cursor = nextCursor
	it.hasMore = nextCursor != nil

	if len(it.items) > 0 {
		it.cur = it.items[0]
		return true
	}
	return false
}

// Current returns the current item in the iteration.
// Returns nil if Next() has not been called or returned false.
func (it *Iter[T]) Current() *T {
	return it.cur
}

// Err returns any error that occurred during iteration.
// Should be checked after Next() returns false.
func (it *Iter[T]) Err() error {
	return it.err
}

// newIter creates a new iterator with the given initial items and fetch function.
func newIter[T any](items []*T, hasMore bool, cursor *string, fetch func(cursor *string) ([]*T, *string, error)) *Iter[T] {
	return &Iter[T]{
		items:   items,
		idx:     -1,
		hasMore: hasMore,
		cursor:  cursor,
		fetch:   fetch,
	}
}

// newIterWithError creates a new iterator that immediately returns the given error.
func newIterWithError[T any](err error) *Iter[T] {
	return &Iter[T]{
		err: err,
		idx: -1,
	}
}
