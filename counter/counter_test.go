package counter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CounterSuite struct {
	suite.Suite
	sharedCounter Counter
}

func (suite *CounterSuite) TestAdd() {
	tests := []struct {
		name string
		give int32
		want int32
	}{
		{"1", 1, 1},
		{"2", 2, 3},
		{"3", 3, 6},
		{"4", 4, 10},
		{"5", 5, 15},
		{"6", -15, 0},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.sharedCounter.Add(tt.give)
			got := int32(suite.sharedCounter)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *CounterSuite) TestSet() {
	tests := []struct {
		name string
		give int32
		want int32
	}{
		{"1", 10, 10},
		{"2", -100, -100},
		{"3", 61, 61},
		{"4", 71, 71},
		{"5", -27, -27},
		{"6", 0, 0},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.sharedCounter.Set(tt.give)
			got := int32(suite.sharedCounter)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *CounterSuite) TestGet() {
	suite.NotEqual(suite.sharedCounter.Get(), int32(61))
	suite.Equal(suite.sharedCounter.Get(), int32(0))
}

func (suite *CounterSuite) TestInc() {
	suite.sharedCounter = 0

	for i := 1; i <= 10; i++ {
		suite.Run(fmt.Sprint(i), func() {
			suite.sharedCounter.Inc()
			got := int32(suite.sharedCounter)
			suite.Equal(int32(i), got)
		})
	}
}

func (suite *CounterSuite) TestDec() {
	suite.sharedCounter = 10

	for i := 9; i >= 0; i-- {
		suite.Run(fmt.Sprint(i), func() {
			suite.sharedCounter.Dec()
			fmt.Println("Asdadaddad")
			got := int32(suite.sharedCounter)
			suite.Equal(int32(i), got)
		})
	}
}

func (suite *CounterSuite) TestStop() {
	suite.sharedCounter = 10
	suite.sharedCounter.Stop()
	suite.Equal(0, int(suite.sharedCounter))
}

func TestCounterSuite(t *testing.T) {
	suite.Run(t, new(CounterSuite))
}
