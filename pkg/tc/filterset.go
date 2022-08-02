package tc

import "github.com/Mellanox/multi-networkpolicy-tc/pkg/tc/types"

// FilterSet interface defines an API for Filter set, which allows
// to perform set operations on a collection of Filters
type FilterSet interface {
	// Add adds filter element to set
	Add(filter types.Filter)
	// Remove removes filter element from set. if filter element does not exist, the call is a no-op
	Remove(filter types.Filter)
	// Has returns true if filter element is in the set, else returns false
	Has(filter types.Filter) bool
	// Len returns the number of elements in the set
	Len() int
	// In returns true if every element in other is an alement of this set. else it returns false
	In(other FilterSet) bool
	// Intersect returns a new FilterSet with elements from both this FilterSet and other FilterSet
	Intersect(other FilterSet) FilterSet
	// Difference returns the difference between this and other FilterSet, that is, elements in this FilterSet
	// and not the other FilterSet
	Difference(other FilterSet) FilterSet
	// Equals returns true if this and other FilterSet are equal (have the same elements)
	Equals(other FilterSet) bool
	// List returns the Filter elements in FilterSet
	List() []types.Filter
}

// NewFilterSetImpl returns a new *FilterSetImpl
func NewFilterSetImpl() *FilterSetImpl {
	return &FilterSetImpl{
		items: make([]types.Filter, 0),
	}
}

// FilterSetImpl implements FilterSet
type FilterSetImpl struct {
	items []types.Filter
}

// Add implements FilterSet
func (f *FilterSetImpl) Add(filter types.Filter) {
	if !f.Has(filter) {
		f.items = append(f.items, filter)
	}
}

// Remove implements FilterSet
func (f *FilterSetImpl) Remove(filter types.Filter) {
	foundIdx := -1
	for idx, fl := range f.items {
		if filter.Equals(fl) {
			foundIdx = idx
			break
		}
	}

	if foundIdx != -1 {
		f.items[foundIdx] = f.items[len(f.items)-1]
		f.items = f.items[:len(f.items)-1]
	}
}

// Has implements FilterSet
func (f *FilterSetImpl) Has(filter types.Filter) bool {
	for _, fl := range f.items {
		if filter.Equals(fl) {
			return true
		}
	}
	return false
}

// Len implements FilterSet
func (f *FilterSetImpl) Len() int {
	return len(f.items)
}

// In implements FilterSet
func (f *FilterSetImpl) In(other FilterSet) bool {
	if f.Len() > other.Len() {
		return false
	}

	for _, fl := range f.items {
		if !other.Has(fl) {
			return false
		}
	}
	return true
}

// Intersect implements FilterSet
func (f *FilterSetImpl) Intersect(other FilterSet) FilterSet {
	fs := NewFilterSetImpl()
	for _, fl := range f.items {
		if other.Has(fl) {
			fs.Add(fl)
		}
	}
	return fs
}

// Difference implements FilterSet
func (f *FilterSetImpl) Difference(other FilterSet) FilterSet {
	fs := NewFilterSetImpl()
	for _, fl := range f.items {
		if !other.Has(fl) {
			fs.Add(fl)
		}
	}
	return fs
}

// Equals implements FilterSet
func (f *FilterSetImpl) Equals(other FilterSet) bool {
	return f.Len() == other.Len() && f.In(other)
}

// List implements FilterSet
func (f *FilterSetImpl) List() []types.Filter {
	return f.items
}
