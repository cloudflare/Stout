package ini

// NewStack returns a new stack.
func NewStack() *Stack {
	return &Stack{}
}

// Stack is a basic LIFO stack that resizes as needed.
type Stack struct {
	items []interface{}
	count int
}

// Push adds an iterm to the top of the stack
func (s *Stack) Push(item interface{}) {
	s.items = append(s.items[:s.count], item)
	s.count++
}

// Pop removes the top item (LIFO) from the stack
func (s *Stack) Pop() interface{} {
	if s.count == 0 {
		return nil
	}

	s.count--
	return s.items[s.count]
}

// Peek returns item at top of stack without removing it
func (s *Stack) Peek() interface{} {
	if s.count == 0 {
		return nil
	}

	return s.items[s.count-1]
}

// Empty returns true when stack is empty, false otherwise
func (s *Stack) Empty() bool {
	return s.count == 0
}

// Size returns the number of items in the stack
func (s *Stack) Size() int {
	return s.count
}
