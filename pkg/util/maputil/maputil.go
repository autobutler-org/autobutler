package maputil

type OrderedMap[K comparable, V any] interface {
	Set(key K, value V)
	Get(key K) (V, bool)
	Keys() []K
	Values() []V
	Range() func(yield func(K, V) bool)
}

type orderedMap[K comparable, V any] struct {
	keys   []K
	values map[K]V
}

func NewOrderedMap[K comparable, V any]() OrderedMap[K, V] {
	return &orderedMap[K, V]{
		keys:   make([]K, 0),
		values: make(map[K]V),
	}
}

func NewOrderedMapFromValues[K comparable, V any](
	keys []K,
	values []V,
) OrderedMap[K, V] {
	om := NewOrderedMap[K, V]()
	for i, key := range keys {
		om.Set(key, values[i])
	}
	return om
}

func (om *orderedMap[K, V]) Set(key K, value V) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

func (om *orderedMap[K, V]) Get(key K) (V, bool) {
	value, exists := om.values[key]
	return value, exists
}

func (om *orderedMap[K, V]) Keys() []K {
	return om.keys
}

func (om *orderedMap[K, V]) Values() []V {
	values := make([]V, 0, len(om.keys))
	for _, key := range om.keys {
		values = append(values, om.values[key])
	}
	return values
}

func (om *orderedMap[K, V]) Range() func(yield func(K, V) bool) {
	return func(yield func(K, V) bool) {
		for _, key := range om.Keys() {
			value := om.values[key]
			if !yield(key, value) {
				break
			}
		}
	}
}
