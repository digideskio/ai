package hex

import (
	"log"
	"math"
	"math/rand"
)

// In reinforcement learning, there are two common functions to learn.
//
// V(s) is the value of being in a state s.
// Q(s, a) is the value of taking action a from state s.
//
// In the case of playing Hex, we are defining "value" to be the
// probability of winning. The network outputs a real number that
// maps onto probabilities - positive for this player winning,
// negative for the opponent winning.
//
// The QNet is a neural network that operates on a Hex board, and
// incrementally updates with each move to maintain state without
// recalculating the state of each neuron every time.
//
// Each QNet corresponds to a particular color. That means it's used
// to decide where that color should play. A QNet tracks two
// things for a particular state:
// A real value baseV
// An array of offsets for each possible action, deltaV[a]
// Q(s, a) is defined as baseV + deltaV(a).
// This makes it easy to choose the best action just by picking the a
// with the highest deltaV.
// Q(s, a) maps to the odds of winning the game.
//
// Another interpretation is that the neural net is calculating a
// function V(s), where it's the value of a state if it's the *other*
// player's turn to move. deltaV is then just tracking how the neural
// network would change with a particular move by this player. This
// explains how the neurons work - they don't add their output values
// directly to baseV; instead when they get one feature away from
// triggering they add their output values to deltaV.
//
// Q(s, a) returns a logit. The probability for our color winning is
// p(Q) = e^Q / (e^Q + 1)
// So a Q of positive infinity corresponds to a 100% chance that our
// color wins.


// The main component of the QNet is the QNeuron, which represents a
// set of basic features that add a particular weight to V if all of
// them trigger.
type QNeuron struct {
	features []QFeature

	weight float64

	// A count of how many of the features are active.
	active uint8
}

func (neuron QNeuron) Debug() {
	featureString := ""
	for _, feature := range neuron.features {
		if len(featureString) > 0 {
			featureString += "+"
		}
		featureString += feature.String()
	}
	if neuron.weight != 0.0 {
		log.Printf("%0.2f <- {%s}", neuron.weight, featureString)
	}
}

// Data surrounding a particular action. Enough to be used for Q-learning.
type QAction struct {
	// Which player took the action
	color Color

	// What spot was moved in
	spot TopoSpot

	// Q(s, a) for the player taking the action
	Q float64

	// The weight difference of Q(s, a_optimal) - Q(s, a).
	// In most cases this is zero because the player took the optimal
	// action according to them.
	// This is useful because if the exploration cost is high, it
	// indicates this move was an "exploration" move, so if it screwed
	// us we shouldn't necessarily penalize earlier decisions.
	// Specifically, Q + explorationCost is the "target Q" that we use
	// to train previous moves.
	explorationCost float64
}

func (action QAction) Debug() {
	log.Printf("%v%v Q=%.3f eC=%.1f", action.color, action.spot, action.Q,
		action.explorationCost)
}

// Turns a q-value into a probability.
func Logistic(q float64) float64 {
	return 1.0 - 1.0 / (1.0 + math.Exp(q))
}

// Turns a probability into a q-value.
func Logit(prob float64) float64 {
	return math.Log(prob / (1.0 - prob))
}

func (action QAction) Feature() QFeature {
	return MakeQFeature(action.color, action.spot)
}

type QNet struct {
	startingPosition *TopoBoard
	color Color
	
	// The extra output that would come from activated neurons if each
	// particular action were taken by this color
	deltaV [NumTopoSpots]float64

	// The output solely from the activated neurons
	baseV float64

	// A neuron with no features
	bias QNeuron

	// Neurons with one feature
	mono [NumFeatures]QNeuron

	// Neurons with two features.
	// By convention, we only access the features in sorted order,
	// so this is half empty.
	duo [NumFeatures][NumFeatures]QNeuron

	// The empty spots in the starting position.
	// This is useful for iterating on the spots in random order, which
	// seeds more intelligently than lexicographical spot order.
	emptySpots []TopoSpot

	// The fraction of the time we intentionally go off-policy in order
	// to handicap this player.
	handicap float64
}

// Creates a new qnet that has no values on any features and thus just
// plays random playouts.
func NewQNet(board *TopoBoard, color Color) *QNet {
	qnet := &QNet{
		startingPosition: board,
		color: color,
		emptySpots: board.PossibleTopoSpotMoves(),
		bias: QNeuron{},
	}

	for feature := MinFeature; feature <= MaxFeature; feature++ {
		qnet.mono[feature].features = []QFeature{feature}
	}
	for f1 := MinFeature; f1 <= MaxFeature; f1++ {
		for f2 := f1 + 1; f2 <= MaxFeature; f2++ {
			qnet.duo[f1][f2].features = []QFeature{f1, f2}
		}
	}

	return qnet
}

func (qnet *QNet) StartingPosition() *TopoBoard {
	return qnet.startingPosition
}

func (qnet *QNet) Color() Color {
	return qnet.color
}

// Acts on the board to make a move.
// This does not update any neurons directly.
func (qnet *QNet) Act(board *TopoBoard) QAction {
	action := qnet.IdealAction(board, true)

	// Actually make the move
	board.MakeMove(action.spot)

	return action
}

// Returns what this net thinks is the ideal action.
// maybeExplore is whether to add some chance of exploration.
// Does not actually mutate anything.
func (qnet *QNet) IdealAction(board *TopoBoard, maybeExplore bool) QAction {

	if qnet.color != board.GetToMove() {
		panic("wrong color to move")
	}

	action := QAction{
		color: qnet.color,
	}

	// Figure out which move to make.
	// We loop to figure out the first possible move, and the best
	// move.
	firstPossibleMove := NotASpot
	firstPossibleDeltaV := math.Inf(-1)
	bestMove := NotASpot
	bestDeltaV := math.Inf(-1)
	for _, spot := range qnet.emptySpots {
		if board.Get(spot) != Empty {
			continue
		}

		if firstPossibleMove == NotASpot {
			firstPossibleMove = spot
			firstPossibleDeltaV = qnet.deltaV[spot]
		}

		if qnet.deltaV[spot] > bestDeltaV {
			bestMove = spot
			bestDeltaV = qnet.deltaV[spot]
		}
	}
	if firstPossibleMove == NotASpot {
		panic("no empty spot found in Act")
	}

	// We might do firstPossibleMove as an exploration.
	// We don't want to do stupid explorations, though.
	explorationQ := qnet.baseV + firstPossibleDeltaV
	explorationCost := bestDeltaV - firstPossibleDeltaV

	// Determine whether we should explore
	explore := false
	if maybeExplore {
		// A Q of 3 corresponds to a win chance of around 95%.
		// If we are still 95% likely to win then it seems okay to explore.
		if explorationQ > 3.0 && rand.Float64() > 0.5 {
			explore = true
		}
		// Sometimes we explore extra in order to handicap a player
		if rand.Float64() < qnet.handicap {
			explore = true
		}
	}

	if explore {
		// Explore
		action.spot = firstPossibleMove
		action.Q = explorationQ
		action.explorationCost = explorationCost
	} else {
		// Exploit
		action.spot = bestMove
		action.Q = qnet.baseV + bestDeltaV
		action.explorationCost = 0.0
	}

	return action
}

func (qnet *QNet) Reset() {
	ShuffleTopoSpots(qnet.emptySpots)

	qnet.baseV = qnet.bias.weight

	// Make mono-neurons contribute to deltaV, while also deactivating
	// them.
	for f, neuron := range qnet.mono {
		feature := QFeature(f)
		neuron.active = 0
		if feature.Color() == qnet.color {
			qnet.deltaV[feature.Spot()] = neuron.weight
		}
	}

	// Deactivate duo-neurons
	for i := range qnet.duo {
		for j := range qnet.duo[i] {
			qnet.duo[i][j].active = 0
		}
	}
}

// A helper function to get a neuron from duo
func (qnet *QNet) GetNeuron(f1 QFeature, f2 QFeature) *QNeuron {
	if f1 == f2 {
		panic("no duo neuron for symmetric (f, f) feature pairs")
	}
	if f1 < f2 {
		return &qnet.duo[f1][f2]
	}
	return &qnet.duo[f2][f1]
}

// Updates the qnet to observe a new feature.
func (qnet *QNet) AddFeature(feature QFeature) {
	qnet.deltaV[feature.Spot()] = 0.0
	
	qnet.baseV += qnet.mono[feature].weight

	// Handle duo neurons
	for feature2 := MinFeature; feature2 <= MaxFeature; feature2++ {
		if feature == feature2 {
			continue
		}
		neuron := qnet.GetNeuron(feature, feature2)
		neuron.active++

		switch neuron.active {
		case 1:
			if feature2.Color() == qnet.color {
				qnet.deltaV[feature2.Spot()] += neuron.weight
			}
		case 2:
			qnet.baseV += neuron.weight
		default:
			log.Fatalf("unexpected neuron activity count: %d", neuron.active)
		}
	}
}

// Updates the weights on the qnet according to a gradient.
func (qnet *QNet) ApplyGradient(gradient *[NumFeatureSets]float64) {
	qnet.bias.weight += (*gradient)[EmptyFeatureSet]

	for fs := MinSingleton; fs <= MaxSingleton; fs++ {
		qnet.mono[fs.SingletonFeature()].weight += (*gradient)[fs]
	}

	for fs := MinDoubleton; fs <= MaxDoubleton; fs++ {
		f1, f2 := fs.Features()
		qnet.duo[f1][f2].weight += (*gradient)[fs]
	}
}

// Learns from a set of playouts.
// scalar is a learning parameter to control how fast we try to learn.
// These must be playouts generated by this net.
func (qnet *QNet) LearnFromPlayouts(playouts []*QPlayout, scalar float64) {
	var gradient [NumFeatureSets]float64

	for _, playout := range playouts {
		playout.AddGradient(qnet.color, scalar, &gradient)
	}

	qnet.ApplyGradient(&gradient)
}

func (qnet *QNet) Debug() {
	qnet.bias.Debug()
}

func (qnet *QNet) DebugSpot(spot TopoSpot) {
	// Print pair-neuron info
	for other := TopLeftCorner; other <= BottomRightCorner; other++ {
		if other == spot {
			continue
		}

		for _, color := range Colors {
			for _, otherColor := range Colors {
				feature := MakeQFeature(color, spot)
				otherFeature := MakeQFeature(otherColor, other)
				qnet.GetNeuron(feature, otherFeature).Debug()
			}
		}
	}

	// Print single-neuron info
	for _, color := range Colors {
		feature := MakeQFeature(color, spot)
		qnet.mono[feature].Debug()
	}
}
