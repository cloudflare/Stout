package ini

import (
	"testing"
)

const (
	magic1 = 123
	magic2 = 987
)

func TestStack(t *testing.T) {
	s := NewStack()

	if s.Empty() == false {
		t.Fatal("New stack is not empty")
	}

	if s.Size() != 0 {
		t.Fatal("Stack size is not empty at init")
	}

	if s.Pop() != nil {
		t.Fatal("Empty stack pop did not return nil")
	}

	if s.Peek() != nil {
		t.Fatal("Empty stack peek did not return nil")
	}

	s.Push(magic1)
	if s.Size() != 1 {
		t.Fatal("Stack size is incorrect after one push - should be 1, not", s.Size())
	}

	if s.Peek() != 123 {
		t.Fatal("Stack peek did not return expected result")
	}

	if s.Empty() == true {
		t.Fatal("Stack should not be reported as empty")
	}

	s.Push(magic2)

	if s.Size() != 2 {
		t.Fatal("Stack size is incorrect after two pushes - should be 2, not", s.Size())
	}

	if s.Peek() != magic2 {
		t.Fatal("Stack peek did not return expected result")
	}

	if s.Peek() != magic2 {
		t.Fatal("Stack peek did not preserve result")
	}

	if s.Pop() != magic2 {
		t.Fatal("Stack first pop did not return correct result")
	}

	if s.Size() != 1 {
		t.Fatal("Stack size is incorrect after pop - should be 1, not", s.Size())
	}

	if s.Pop() != magic1 {
		t.Fatal("Stack second pop did not return correct result")
	}

	if s.Size() != 0 {
		t.Fatal("Stack size is incorrect after second pop - should be 0, not", s.Size())
	}

	if s.Pop() != nil {
		t.Fatal("Empty stack pop (after pops) did not return nil")
	}

	if s.Empty() == false {
		t.Fatal("Empty stack is not reported as empty")
	}

}
