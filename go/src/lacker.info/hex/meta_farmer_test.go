package hex

import (
	"testing"
)

func TestMetaFarmerInit(t *testing.T) {
	board := NewTopoBoard()
	mf := &MetaFarmer{Seconds:-1, Quiet:true}
	mf.init(board)
}