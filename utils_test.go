package main

import "testing"

func TestNormalizeString(t *testing.T) {
	value1 := "école"
	want1 := "ecole"

	v := normalizeString(value1)
	if v != want1 {
		t.Errorf("normalizeString(école) = %v is not what we expected!, %v", v, want1)
	}
}

func TestContains(t *testing.T) {
	value1 := []string{"go", "rust", "java", "javascript", "python", "php"}

	if contains(value1, "csharp") {
		t.Errorf("contains(value1, csharp) should not be in the list.")
	}

	if !contains(value1, "go") {
		t.Errorf("contains(value1, go) must be in the list.")
	}
}

func TestGetRandomStringFromArray(t *testing.T) {
	value1 := []string{}
	value2 := []string{"pain", "salade", "tomate", "mayonaise", "poulet", "fromage"}

	_, err := getRandomStringFromArray(value1)
	if err == nil {
		t.Errorf("getRandomStringFromArray(list) must return an error because the list is empty.")
	}

	word, err := getRandomStringFromArray(value2)
	if err != nil && !contains(value2, word) {
		t.Errorf("getRandomStringFromArray(list) the word randomly retrieved from the list should be contained in the list.")
	}
}
