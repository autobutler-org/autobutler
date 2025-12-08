package maputil_test

import (
	"testing"

	"autobutler/pkg/util/maputil"
)

func TestNewOrderedMap(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	if om == nil {
		t.Fatal("NewOrderedMap returned nil")
	}

	keys := om.Keys()
	if len(keys) != 0 {
		t.Errorf("Expected empty map, got %d keys", len(keys))
	}
}

func TestNewOrderedMapFromValues(t *testing.T) {
	keys := []string{"a", "b", "c"}
	values := []int{1, 2, 3}

	om := maputil.NewOrderedMapFromValues(keys, values)

	if len(om.Keys()) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(om.Keys()))
	}

	for i, key := range keys {
		val, exists := om.Get(key)
		if !exists {
			t.Errorf("Key %s not found", key)
		}
		if val != values[i] {
			t.Errorf("Expected value %d for key %s, got %d", values[i], key, val)
		}
	}
}

func TestSet(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()

	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	keys := om.Keys()
	if len(keys) != 3 {
		t.Fatalf("Expected 3 keys, got %d", len(keys))
	}

	if keys[0] != "first" || keys[1] != "second" || keys[2] != "third" {
		t.Errorf("Keys not in insertion order: %v", keys)
	}
}

func TestSetOverwrite(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()

	om.Set("key", 1)
	om.Set("key", 2)

	val, exists := om.Get("key")
	if !exists {
		t.Fatal("Key not found")
	}
	if val != 2 {
		t.Errorf("Expected value 2, got %d", val)
	}

	keys := om.Keys()
	if len(keys) != 1 {
		t.Errorf("Expected 1 key after overwrite, got %d", len(keys))
	}
}

func TestGet(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("exists", 42)

	val, exists := om.Get("exists")
	if !exists {
		t.Error("Expected key to exist")
	}
	if val != 42 {
		t.Errorf("Expected value 42, got %d", val)
	}

	_, exists = om.Get("not-exists")
	if exists {
		t.Error("Expected key to not exist")
	}
}

func TestKeys(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("alpha", 1)
	om.Set("beta", 2)
	om.Set("gamma", 3)

	keys := om.Keys()
	expected := []string{"alpha", "beta", "gamma"}

	if len(keys) != len(expected) {
		t.Fatalf("Expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected key %s at position %d, got %s", expected[i], i, key)
		}
	}
}

func TestValues(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("a", 10)
	om.Set("b", 20)
	om.Set("c", 30)

	values := om.Values()
	expected := []int{10, 20, 30}

	if len(values) != len(expected) {
		t.Fatalf("Expected %d values, got %d", len(expected), len(values))
	}

	for i, val := range values {
		if val != expected[i] {
			t.Errorf("Expected value %d at position %d, got %d", expected[i], i, val)
		}
	}
}

func TestRange_BasicIteration(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("one", 1)
	om.Set("two", 2)
	om.Set("three", 3)

	expectedKeys := []string{"one", "two", "three"}
	expectedValues := []int{1, 2, 3}

	i := 0
	for k, v := range om.Range() {
		if i >= len(expectedKeys) {
			t.Errorf("Iterator yielded more items than expected")
			break
		}
		if k != expectedKeys[i] {
			t.Errorf("Expected key %s at position %d, got %s", expectedKeys[i], i, k)
		}
		if v != expectedValues[i] {
			t.Errorf("Expected value %d at position %d, got %d", expectedValues[i], i, v)
		}
		i++
	}

	if i != len(expectedKeys) {
		t.Errorf("Expected to iterate %d times, iterated %d times", len(expectedKeys), i)
	}
}

func TestRange_EmptyMap(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()

	count := 0
	for range om.Range() {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 iterations on empty map, got %d", count)
	}
}

func TestRange_EarlyBreak(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)
	om.Set("d", 4)

	count := 0
	for k, v := range om.Range() {
		count++
		// Break after 2 iterations
		if count == 2 {
			if k != "b" || v != 2 {
				t.Errorf("Expected to break at key 'b' with value 2, got key '%s' with value %d", k, v)
			}
			break
		}
	}

	if count != 2 {
		t.Errorf("Expected to iterate 2 times before break, iterated %d times", count)
	}
}

func TestRange_ConditionalBreak(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("stop", 3)
	om.Set("d", 4)

	var visitedKeys []string
	for k, _ := range om.Range() {
		visitedKeys = append(visitedKeys, k)
		if k == "stop" {
			break
		}
	}

	expected := []string{"a", "b", "stop"}
	if len(visitedKeys) != len(expected) {
		t.Fatalf("Expected to visit %d keys, visited %d", len(expected), len(visitedKeys))
	}

	for i, key := range visitedKeys {
		if key != expected[i] {
			t.Errorf("Expected key %s at position %d, got %s", expected[i], i, key)
		}
	}
}

func TestRange_MultipleIterations(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("x", 10)
	om.Set("y", 20)

	// First iteration
	count1 := 0
	for range om.Range() {
		count1++
	}

	// Second iteration
	count2 := 0
	for range om.Range() {
		count2++
	}

	if count1 != 2 {
		t.Errorf("First iteration: expected 2 items, got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("Second iteration: expected 2 items, got %d", count2)
	}
}

func TestRange_DifferentTypes(t *testing.T) {
	// Test with int keys
	omInt := maputil.NewOrderedMap[int, string]()
	omInt.Set(1, "one")
	omInt.Set(2, "two")
	omInt.Set(3, "three")

	countInt := 0
	for k, v := range omInt.Range() {
		if k == 1 && v != "one" {
			t.Errorf("Expected value 'one' for key 1, got '%s'", v)
		}
		countInt++
	}
	if countInt != 3 {
		t.Errorf("Expected 3 iterations, got %d", countInt)
	}

	// Test with struct values
	type Person struct {
		Name string
		Age  int
	}

	omStruct := maputil.NewOrderedMap[string, Person]()
	omStruct.Set("alice", Person{"Alice", 30})
	omStruct.Set("bob", Person{"Bob", 25})

	countStruct := 0
	for k, v := range omStruct.Range() {
		if k == "alice" && v.Name != "Alice" {
			t.Errorf("Expected Alice, got %s", v.Name)
		}
		countStruct++
	}
	if countStruct != 2 {
		t.Errorf("Expected 2 iterations, got %d", countStruct)
	}
}

func TestRange_OrderPreservation(t *testing.T) {
	om := maputil.NewOrderedMap[int, string]()

	// Insert in specific order
	insertOrder := []int{5, 1, 9, 3, 7}
	for _, key := range insertOrder {
		om.Set(key, "value")
	}

	var iteratedOrder []int
	for k, _ := range om.Range() {
		iteratedOrder = append(iteratedOrder, k)
	}

	if len(iteratedOrder) != len(insertOrder) {
		t.Fatalf("Expected %d items, got %d", len(insertOrder), len(iteratedOrder))
	}

	for i, key := range iteratedOrder {
		if key != insertOrder[i] {
			t.Errorf("Order not preserved: expected key %d at position %d, got %d", insertOrder[i], i, key)
		}
	}
}

func TestRange_ModificationDuringIteration(t *testing.T) {
	om := maputil.NewOrderedMap[string, int]()
	om.Set("a", 1)
	om.Set("b", 2)
	om.Set("c", 3)

	// Collect keys during iteration (not modifying the map being iterated)
	var keys []string
	for k, _ := range om.Range() {
		keys = append(keys, k)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
}

func TestRange_LargeMap(t *testing.T) {
	om := maputil.NewOrderedMap[int, int]()

	// Insert 1000 items
	n := 1000
	for i := 0; i < n; i++ {
		om.Set(i, i*2)
	}

	count := 0
	for k, v := range om.Range() {
		if v != k*2 {
			t.Errorf("Expected value %d for key %d, got %d", k*2, k, v)
		}
		count++
	}

	if count != n {
		t.Errorf("Expected %d iterations, got %d", n, count)
	}
}
