package ast

import (
	"fmt"
	"math"
	"strconv"
	"testing"
)

func TestScalarTypes(t *testing.T) {
	t.Run("StringNode", func(t *testing.T) {
		cases := []struct {
			value    string
			line     int
			column   int
			expected string
		}{
			{"hello", 1, 2, "\"hello\" @ 1:2"},
			{"world", 3, 4, "\"world\" @ 3:4"},
			{"hello world", 5, 6, "\"hello world\" @ 5:6"},
			{"\"escaped\"", 7, 8, "\"\"escaped\"\" @ 7:8"},
		}

		for _, c := range cases {
			t.Run(c.value, func(t *testing.T) {
				node := &StringNode{
					Value: c.value,
					Pos: Position{
						Line:   c.line,
						Column: c.column,
					},
				}

				if got := node.String(); got != c.expected {
					t.Errorf("String() = %v; want %v", got, c.expected)
				}

				if got := node.TokenLiteral(); got != strconv.Quote(c.value) {
					t.Errorf("TokenLiteral() = %v; want %v", got, strconv.Quote(c.value))
				}

				if got := node.NodeType(); got != "String" {
					t.Errorf("NodeType() = %v; want %v", got, "String")
				}

				if got := node.GetPos(); got != node.Pos {
					t.Errorf("GetPos() = %v; want %v", got, node.Pos)
				}

				newPos := Position{Line: 10, Column: 10}
				node.SetPos(newPos)
				if got := node.GetPos(); got != newPos {
					t.Errorf("SetPos() failed: got %v; want %v", got, newPos)
				}
			})
		}
	})

	t.Run("IntegerNode", func(t *testing.T) {
		cases := []struct {
			value    int64
			line     int
			column   int
			expected string
		}{
			{42, 1, 2, "42 @ 1:2"},
			{-100, 3, 4, "-100 @ 3:4"},
			{0, 5, 6, "0 @ 5:6"},
			{math.MaxInt64, 7, 8, fmt.Sprintf("%d @ 7:8", math.MaxInt64)},
			{math.MinInt64, 9, 10, fmt.Sprintf("%d @ 9:10", math.MinInt64)},
		}

		for _, c := range cases {
			t.Run(fmt.Sprintf("%d", c.value), func(t *testing.T) {
				node := &IntegerNode{
					Value: c.value,
					Pos: Position{
						Line:   c.line,
						Column: c.column,
					},
				}

				if got := node.String(); got != c.expected {
					t.Errorf("String() = %v; want %v", got, c.expected)
				}

				if got := node.TokenLiteral(); got != strconv.FormatInt(c.value, 10) {
					t.Errorf("TokenLiteral() = %v; want %v", got, strconv.FormatInt(c.value, 10))
				}

				if got := node.NodeType(); got != "Integer" {
					t.Errorf("NodeType() = %v; want %v", got, "Integer")
				}

				if got := node.GetPos(); got != node.Pos {
					t.Errorf("GetPos() = %v; want %v", got, node.Pos)
				}

				newPos := Position{Line: 10, Column: 10}
				node.SetPos(newPos)
				if got := node.GetPos(); got != newPos {
					t.Errorf("SetPos() failed: got %v; want %v", got, newPos)
				}
			})
		}
	})

	t.Run("FloatNode", func(t *testing.T) {
		cases := []struct {
			value    float64
			line     int
			column   int
			expected string
		}{
			{3.14, 1, 2, "3.140000 @ 1:2"},
			{-0.001, 3, 4, "-0.001000 @ 3:4"},
			{0.0, 5, 6, "0.000000 @ 5:6"},
			{math.MaxFloat64, 7, 8, fmt.Sprintf("%f @ 7:8", math.MaxFloat64)},
			{math.SmallestNonzeroFloat64, 9, 10, fmt.Sprintf("%f @ 9:10", math.SmallestNonzeroFloat64)},
		}

		for _, c := range cases {
			t.Run(fmt.Sprintf("%f", c.value), func(t *testing.T) {
				node := &FloatNode{
					Value: c.value,
					Pos: Position{
						Line:   c.line,
						Column: c.column,
					},
				}

				if got := node.String(); got != c.expected {
					t.Errorf("String() = %v; want %v", got, c.expected)
				}

				if got := node.TokenLiteral(); got != strconv.FormatFloat(c.value, 'f', -1, 64) {
					t.Errorf("TokenLiteral() = %v; want %v", got, strconv.FormatFloat(c.value, 'f', -1, 64))
				}

				if got := node.NodeType(); got != "Float" {
					t.Errorf("NodeType() = %v; want %v", got, "Float")
				}

				if got := node.GetPos(); got != node.Pos {
					t.Errorf("GetPos() = %v; want %v", got, node.Pos)
				}

				newPos := Position{Line: 10, Column: 10}
				node.SetPos(newPos)
				if got := node.GetPos(); got != newPos {
					t.Errorf("SetPos() failed: got %v; want %v", got, newPos)
				}
			})
		}
	})

	t.Run("IsScalarType", func(t *testing.T) {
		cases := []struct {
			input    string
			expected bool
		}{
			{"int", true},
			{"float", true},
			{"string", true},
			{"bool", true},
			{"array", false},
			{"object", false},
			{"custom", false},
		}
		for _, c := range cases {
			if got := IsScalarType(c.input); got != c.expected {
				t.Errorf("IsScalarType(%q) = %v; want %v", c.input, got, c.expected)
			}
		}
	})
}
