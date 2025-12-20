package http

import "context"

// PageFetcher is a function that fetches a page of items.
// Returns the items, whether there are more pages, and any error.
type PageFetcher[T any] func(ctx context.Context, page int) (items []T, hasMore bool, err error)

// PageIterator provides iteration over paginated API results.
// It lazily fetches pages as needed.
type PageIterator[T any] struct {
	fetch   PageFetcher[T]
	page    int
	buffer  []T
	done    bool
	err     error
	total   int // Total items if known, -1 otherwise
	fetched int // Total items fetched so far
}

// NewPageIterator creates a new iterator with the given fetch function.
func NewPageIterator[T any](fetch PageFetcher[T]) *PageIterator[T] {
	return &PageIterator[T]{
		fetch: fetch,
		page:  0,
		total: -1,
	}
}

// Next returns the next item from the iterator.
// Returns the item, true if an item was returned, and any error.
// When iteration is complete, returns (zero, false, nil).
func (p *PageIterator[T]) Next(ctx context.Context) (T, bool, error) {
	var zero T

	// Return any previous error
	if p.err != nil {
		return zero, false, p.err
	}

	// Fetch next page if buffer is empty
	if len(p.buffer) == 0 && !p.done {
		items, hasMore, err := p.fetch(ctx, p.page)
		if err != nil {
			p.err = err
			return zero, false, err
		}
		p.buffer = items
		p.done = !hasMore
		p.page++
	}

	// Return next item from buffer
	if len(p.buffer) == 0 {
		return zero, false, nil
	}

	item := p.buffer[0]
	p.buffer = p.buffer[1:]
	p.fetched++

	return item, true, nil
}

// All collects all items from the iterator into a slice.
// This will fetch all pages and may be slow for large result sets.
func (p *PageIterator[T]) All(ctx context.Context) ([]T, error) {
	var all []T
	for {
		item, ok, err := p.Next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		all = append(all, item)
	}
	return all, nil
}

// Err returns any error that occurred during iteration.
func (p *PageIterator[T]) Err() error {
	return p.err
}

// Total returns the total number of items if known, -1 otherwise.
// This may only be accurate after at least one page has been fetched.
func (p *PageIterator[T]) Total() int {
	return p.total
}

// SetTotal sets the total count (called by fetch functions that know the total).
func (p *PageIterator[T]) SetTotal(total int) {
	p.total = total
}

// Fetched returns the number of items fetched so far.
func (p *PageIterator[T]) Fetched() int {
	return p.fetched
}

// Reset resets the iterator to the beginning.
// Any buffered items are discarded.
func (p *PageIterator[T]) Reset() {
	p.page = 0
	p.buffer = nil
	p.done = false
	p.err = nil
	p.fetched = 0
}

// Take returns up to n items from the iterator.
func (p *PageIterator[T]) Take(ctx context.Context, n int) ([]T, error) {
	var items []T
	for len(items) < n {
		item, ok, err := p.Next(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		items = append(items, item)
	}
	return items, nil
}

// Skip advances the iterator by n items.
func (p *PageIterator[T]) Skip(ctx context.Context, n int) error {
	for range n {
		_, ok, err := p.Next(ctx)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}
	return nil
}

// ForEach calls fn for each item in the iterator.
// If fn returns an error, iteration stops and that error is returned.
func (p *PageIterator[T]) ForEach(ctx context.Context, fn func(T) error) error {
	for {
		item, ok, err := p.Next(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if err := fn(item); err != nil {
			return err
		}
	}
}
