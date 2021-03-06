package hex

import (
	"math"
	"math/rand"
	"testing"
)

func TestSimpleChain(t *testing.T) {
	mcts := MakeMCTS(0)
	board := NewNaiveBoard()
	root := mcts.NewRoot(board)
	if root.UCT() != math.Inf(1) {
		t.Fatalf("root.UCT() was not Inf")
	}

	middle := NewChild(root, MakeNaiveSpot(1, 1))
	if middle.Board == nil {
		t.Fatalf("middle should have a non-nil board")
	}
	if middle.Board.Get(MakeNaiveSpot(1, 1)) != Black {
		t.Fatalf("middle should have a black stone at 1,1")
	}
	t.Log("made middle ok")

	leaf := NewChild(middle, MakeNaiveSpot(5, 5))
	t.Log("made leaf ok")

	if leaf.NumPossibleMoves != 119 {
		t.Fatalf("bad num possible moves")
	}

	if root.SelectLeafByUCT() != root {
		t.Fatalf("the root should also be a leaf according to SelectLeafByUCT")
	}
}

func TestExpansion(t *testing.T) {
	board := NewNaiveBoard()
	mcts := MakeMCTS(0)
	root := mcts.NewRoot(board)
	for i := 0; i < 121; i++ {
		if root.SelectLeafByUCT() != root {
			t.Fatalf("root.SelectLeafByUCT() should be root at iteration %d", i)
		}
		depth := root.Depth()
		if i > 0 && depth != 2 {
			t.Fatalf("on iteration %d got depth %d", i, depth)
		}
		if root.Expand() == nil {
			t.Fatalf("could not expand root on iteration %d", i)
		}
	}
	
	// Finally the root should be full
	leaf := root.SelectLeafByUCT()
	if leaf == root {
		t.Fatalf("leaf should not be root when root is full")
	}
	leaf.Expand()
	if root.Depth() != 3 {
		t.Fatalf("root depth should be three after expanding a child")
	}
}

func TestMCTS(t *testing.T) {
	board := NewNaiveBoard()
	mcts := MakeMCTS(0)
	root := mcts.NewRoot(board)
	for i := 0; i < 5; i++ {
		mcts.RunOneRound(root)
	}
	if root.BlackWins + root.WhiteWins != 5 {
		t.Fatalf("five mcts loops should lead to 5 win counts in the root")
	}
}

func TestTopoMCTS(t *testing.T) {
	board := NewNaiveBoard()
	mcts := MakeMCTS(0)
	mcts.UseTopoBoards = true
	root := mcts.NewRoot(board)
	for i := 0; i < 5; i++ {
		mcts.RunOneRound(root)
	}
	if root.BlackWins + root.WhiteWins != 5 {
		t.Fatalf("five mcts loops should lead to 5 win counts in the root")
	}
}

func BenchmarkMCTS(b *testing.B) {
	rand.Seed(1)
	mcts := MonteCarloTreeSearch{Seconds: 0, Quiet: false, V: 1000}
	board := NewNaiveBoard()
	root := mcts.NewRoot(board)

	for i := 0; i < b.N; i++ {
		mcts.RunOneRound(root)
	}
}

func BenchmarkTopoMCTS(b *testing.B) {
	rand.Seed(1)
	mcts := MonteCarloTreeSearch{
		Seconds: 0, Quiet: false, V: 1000, UseTopoBoards: true,
	}
	board := NewNaiveBoard()
	root := mcts.NewRoot(board)

	for i := 0; i < b.N; i++ {
		mcts.RunOneRound(root)
	}

}
