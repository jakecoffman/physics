package physics

type HashValue uintptr
type HashSetEqual func(ptr, elt interface{}) bool
type HashSetTrans func(ptr, data interface{}) interface{}
type HashSetIterator func(elt interface{})
type HashSetFilter func(elt, data interface{}) bool

type HashSetBin struct {
	elt  interface{}
	hash HashValue
	next *HashSetBin
}

type HashSet struct {
	// number of bins in the table, not just table size
	entries      uint
	eql          HashSetEqual
	defaultValue interface{}

	table      map[HashValue]*HashSetBin
	pooledBins *HashSetBin
}

func NewHashSet(eql HashSetEqual) *HashSet {
	return &HashSet{
		eql:   eql,
		table: map[HashValue]*HashSetBin{},
	}
}

func (set *HashSet) Free() {
	if set != nil {
		set.table = map[HashValue]*HashSetBin{}
	}
}

func (set *HashSet) Count() uint {
	return set.entries
}

func (set *HashSet) Insert(hash HashValue, ptr interface{}, trans HashSetTrans, data interface{}) interface{} {
	// Find the bin with the matching element.
	bin := set.table[hash]
	for bin != nil && !set.eql(ptr, bin.elt) {
		bin = bin.next
	}

	// Create it if necessary.
	if bin == nil {
		bin = set.GetUnusedBin()
		bin.hash = hash
		if trans != nil {
			bin.elt = trans(ptr, data)
		} else {
			bin.elt = data
		}

		bin.next = set.table[hash]
		set.table[hash] = bin

		set.entries++
	}

	return bin.elt
}

func (set *HashSet) Remove(hash HashValue, ptr interface{}) interface{} {
	bin := set.table[hash]
	// In Go we can't take the address of a map entry, so this differs a bit.
	var prevPtr **HashSetBin

	// Find the bin
	for bin != nil && !set.eql(ptr, bin.elt) {
		prevPtr = &bin.next
		bin = bin.next
	}

	// Remove the bin if it exists
	if bin != nil {
		// Update the previous linked list pointer
		if prevPtr != nil {
			*prevPtr = bin.next
		} else {
			delete(set.table, hash)
		}

		set.entries--

		elt := bin.elt
		set.Recycle(bin)

		return elt
	}

	return nil
}

func (set *HashSet) Find(hash HashValue, ptr interface{}) interface{} {
	bin := set.table[hash]
	for bin != nil && !set.eql(ptr, bin.elt) {
		bin = bin.next
	}

	if bin != nil {
		return bin.elt
	} else {
		return set.defaultValue
	}
}

func (set *HashSet) Each(f HashSetIterator) {
	for _, bin := range set.table {
		for bin != nil {
			next := bin.next
			f(bin.elt)
			bin = next
		}
	}
}

func (set *HashSet) Filter(f HashSetFilter, data interface{}) {
	for i, first := range set.table {
		prevPtr := &first
		bin := first
		for bin != nil {
			next := bin.next

			if f(bin.elt, data) {
				prevPtr = &bin.next
			} else {
				if first == *prevPtr {
					set.table[i] = nil
				} else {
					*prevPtr = bin.next
				}

				set.entries--
				set.Recycle(bin)
			}

			bin = next
		}
	}
}

func (set *HashSet) Recycle(bin *HashSetBin) {
	bin.next = set.pooledBins
	set.pooledBins = bin
	bin.elt = nil
}

func (set *HashSet) GetUnusedBin() *HashSetBin {
	bin := set.pooledBins

	if bin != nil {
		set.pooledBins = bin.next
		return bin
	}

	for i := 0; i < POOLED_BUFFER_SIZE; i++ {
		set.Recycle(&HashSetBin{})
	}

	return &HashSetBin{}
}
