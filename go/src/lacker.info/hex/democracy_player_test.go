package hex

import (
	"log"
	"testing"
)

func TestDemocracyPlayerPlayout(t *testing.T) {
	board := NewTopoBoard()

	black1 := NewLinearPlayer(board, Black)
	black2 := NewLinearPlayer(board, Black)
	black3 := NewLinearPlayer(board, Black)

	white1 := NewLinearPlayer(board, White)
	white2 := NewLinearPlayer(board, White)
	white3 := NewLinearPlayer(board, White)

	black := NewDemocracyPlayer(board, Black)
	black.Add(black1)
	black.Add(black2)
	black.Add(black3)
	
	white := NewDemocracyPlayer(board, White)
	white.Add(white1)
	white.Add(white2)
	white.Add(white3)
	
	ending := Playout(black, white, false)
	if ending.Winner != Black {
		log.Fatal("expected Black to win because Black wins individuals")
	}
}

func TestDemocracyPlayerEmptiness(t *testing.T) {
	board := NewTopoBoard()
	black := NewDemocracyPlayer(board, Black)
	white := NewDemocracyPlayer(board, White)
	ending := Playout(black, white, false)
	if ending.Winner != Black {
		log.Fatal("expected Black to win along column 0 via fallbacks")
	}
	if ending.GetByRowCol(10, 1) != Empty {
		log.Fatal("expected fallback to never get to 10, 1")
	}
}

func TestDemocracyPlayerWeights(t *testing.T) {
	board := NewTopoBoard()
	black := NewDemocracyPlayer(board, Black)
	sub := NewLinearPlayer(board, Black)
	black.AddWithWeight(sub, 1000000.0)
	black.NormalizeWeights()
	if black.weights[0] != 10000.0 {
		log.Fatal("expected weights to be normalized")
	}
}
