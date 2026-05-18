package shared

type CircularInt struct {
	value int
	min   int
	max   int
}

func NewCircularInt(min, max int) CircularInt {
	return CircularInt{value: min, min: min, max: max}
}

func (c *CircularInt) Increment() {
	if c.value >= c.max {
		c.value = c.min
	} else {
		c.value++
	}
}

func (c *CircularInt) Decrement() {
	if c.value <= c.min {
		c.value = c.max
	} else {
		c.value--
	}
}

func (c CircularInt) Value() int {
	return c.value
}
