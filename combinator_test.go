package goparsify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeq(t *testing.T) {
	parser := Seq("hello", "world")

	t.Run("matches sequence", func(t *testing.T) {
		node, p2 := runParser("hello world", parser)
		assertSequence(t, node, "hello", "world")
		require.Equal(t, "", p2.Get())
	})

	t.Run("returns errors", func(t *testing.T) {
		_, p2 := runParser("hello there", parser)
		require.Equal(t, "world", p2.Error.expected)
		require.Equal(t, 6, p2.Error.pos)
		require.Equal(t, 0, p2.Pos)
	})
}

func TestMaybe(t *testing.T) {
	t.Run("matches sequence", func(t *testing.T) {
		node, p2 := runParser("hello world", Maybe("hello"))
		require.Equal(t, "hello", node.Token)
		require.Equal(t, " world", p2.Get())
	})

	t.Run("returns no errors", func(t *testing.T) {
		node, p3 := runParser("hello world", Maybe("world"))
		require.Equal(t, Result{}, node)
		require.False(t, p3.Errored())
		require.Equal(t, 0, p3.Pos)
	})
}

func TestAny(t *testing.T) {
	t.Run("Matches any", func(t *testing.T) {
		node, p2 := runParser("hello world!", Any("hello", "world"))
		require.Equal(t, "hello", node.Token)
		require.Equal(t, 5, p2.Pos)
	})

	t.Run("Returns longest error", func(t *testing.T) {
		_, p2 := runParser("hello world!", Any(
			"nope",
			Seq("hello", "world", "."),
			Seq("hello", "brother"),
		))
		require.Equal(t, "offset 11: expected .", p2.Error.Error())
		require.Equal(t, 11, p2.Error.Pos())
		require.Equal(t, 0, p2.Pos)
	})

	t.Run("Accepts nil matches", func(t *testing.T) {
		node, p2 := runParser("hello world!", Any(Exact("ffffff")))
		require.Equal(t, Result{}, node)
		require.Equal(t, 0, p2.Pos)
	})

	t.Run("branch prediction", func(t *testing.T) {
		p := Any("hello", Seq("{", Cut(), "world", "}"), Seq("[", Cut(), "a", "]"))
		// warm up the predictor
		_, _ = Run(p, "hello")
		_, _ = Run(p, "{world}")

		t.Run("matches", func(t *testing.T) {
			node, ps := runParser("hello world!", p)
			require.Equal(t, "hello", node.Token)
			require.Equal(t, 5, ps.Pos)
		})

		t.Run("errors", func(t *testing.T) {
			_, ps := runParser("help world!", p)
			require.Equal(t, "offset 0: expected [", ps.Error.Error())
			require.Equal(t, 0, ps.Error.Pos())
			require.Equal(t, 0, ps.Pos)
		})

		t.Run("errors with cuts", func(t *testing.T) {
			_, ps := runParser("{world", p)
			require.Equal(t, "offset 6: expected }", ps.Error.Error())
			require.Equal(t, 6, ps.Error.Pos())
			require.Equal(t, 0, ps.Pos)
		})

		t.Run("misprededicted cut", func(t *testing.T) {
			// This should probably only happen when the predictor is cold
			_, ps := runParser("[a", p)
			require.Equal(t, "offset 2: expected ]", ps.Error.Error())
			require.Equal(t, 2, ps.Error.Pos())
			require.Equal(t, 0, ps.Pos)
		})
	})
}

func TestSome(t *testing.T) {
	t.Run("Matches sequence with sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", Some(Chars("a-g"), ","))
		require.False(t, p2.Errored())
		assertSequence(t, node, "a", "b", "c", "d", "e")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("Matches sequence without trailing sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e1111", Some(Chars("a-g"), ","))
		require.False(t, p2.Errored())
		assertSequence(t, node, "a", "b", "c", "d", "e")
		require.Equal(t, "1111", p2.Get())
	})

	t.Run("Matches sequence without sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", Some(Any(Chars("a-g"), ",")))
		assertSequence(t, node, "a", ",", "b", ",", "c", ",", "d", ",", "e", ",")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("splits words automatically on space", func(t *testing.T) {
		node, p2 := runParser("hello world", Some(Chars("a-z")))
		assertSequence(t, node, "hello", "world")
		require.Equal(t, "", p2.Get())
	})

	t.Run("Stops on error", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", Some(Chars("a-c"), ","))
		assertSequence(t, node, "a", "b", "c")
		require.Equal(t, 6, p2.Pos)
		require.Equal(t, "d,e,", p2.Get())
	})
}

func TestMany(t *testing.T) {
	t.Run("Matches sequence with sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", Many(Chars("a-g"), Exact(",")))
		assertSequence(t, node, "a", "b", "c", "d", "e")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("Matches sequence without sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", Many(Any(Chars("abcdefg"), Exact(","))))
		assertSequence(t, node, "a", ",", "b", ",", "c", ",", "d", ",", "e", ",")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("Stops on error", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", Many(Chars("abc"), Exact(",")))
		assertSequence(t, node, "a", "b", "c")
		require.Equal(t, 6, p2.Pos)
		require.Equal(t, "d,e,", p2.Get())
	})

	t.Run("Returns error if nothing matches", func(t *testing.T) {
		_, p2 := runParser("a,b,c,d,e,", Many(Chars("def"), Exact(",")))
		require.Equal(t, "offset 0: expected def", p2.Error.Error())
		require.Equal(t, "a,b,c,d,e,", p2.Get())
	})
}

type htmlTag struct {
	Name string
}

func TestMap(t *testing.T) {
	parser := Seq("<", Chars("a-zA-Z0-9"), ">").Map(func(n *Result) {
		n.Result = htmlTag{n.Child[1].Token}
	})

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("<html>", parser)
		require.Equal(t, htmlTag{"html"}, result.Result)
	})

	t.Run("error", func(t *testing.T) {
		_, ps := runParser("<html", parser)
		require.Equal(t, "offset 5: expected >", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestBind(t *testing.T) {
	parser := Bind("true", true)

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("true", parser)
		require.Equal(t, true, result.Result)
	})

	t.Run("error", func(t *testing.T) {
		result, ps := runParser("nil", parser)
		require.Nil(t, result.Result)
		require.Equal(t, "offset 0: expected true", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestCut(t *testing.T) {
	t.Run("test any", func(t *testing.T) {
		_, ps := runParser("var world", Any(Seq("var", Cut(), "hello"), "var world"))
		require.Equal(t, "offset 4: expected hello", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})

	t.Run("test many", func(t *testing.T) {
		_, ps := runParser("hello <world", Many(Any(Seq("<", Cut(), Chars("a-z"), ">"), Chars("a-z"))))
		require.Equal(t, "offset 12: expected >", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})

	t.Run("test maybe", func(t *testing.T) {
		_, ps := runParser("var", Maybe(Seq("var", Cut(), "hello")))
		require.Equal(t, "offset 3: expected hello", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestMerge(t *testing.T) {
	var bracer Parser
	bracer = Seq("(", Maybe(&bracer), ")")
	parser := Merge(bracer)

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("((()))", parser)
		require.Equal(t, "((()))", result.Token)
	})

	t.Run("error", func(t *testing.T) {
		_, ps := runParser("((())", parser)
		require.Equal(t, "offset 5: expected )", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestMapShorthand(t *testing.T) {
	Chars("a-z").Map(func(n *Result) {
		n.Result = n.Token
	})
}

func assertSequence(t *testing.T, node Result, expected ...string) {
	require.NotNil(t, node)
	actual := []string{}

	for _, child := range node.Child {
		actual = append(actual, child.Token)
	}

	require.Equal(t, expected, actual)
}
