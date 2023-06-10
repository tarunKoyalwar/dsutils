package dsutils

import "fmt"

// Stack is a generic stack implementation.
type Stack[T any] []T

// NewStack returns a new stack.
func (s *Stack[T]) Push(v T) {
	*s = append(*s, v)
}

// Pop removes and returns the last element of the stack.
func (s *Stack[T]) Pop() T {
	var lastelement T
	if s.IsEmpty() {
		return lastelement
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

// Len returns the length of the stack.
func (s *Stack[T]) Len() int {
	return len(*s)
}

// IsEmpty returns true if the stack is empty.
func (s *Stack[T]) IsEmpty() bool {
	return len(*s) == 0
}

// Clear clears the stack.
func (s *Stack[T]) Clear() {
	*s = (*s)[:0]
}

// String returns a string vector representation of all elements in stack.
func (s *Stack[T]) Vector() []string {
	var str []string
	for _, v := range *s {
		str = append(str, fmt.Sprint(v))
	}
	return str
}

// NewStack returns a new stack.
func NewStack[T any]() *Stack[T] {
	return new(Stack[T])
}
