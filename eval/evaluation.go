package eval

import (
	"math"

	. "github.com/ChizhovVadim/CounterGo/common"
)

const (
	PawnValue = 100
	maxPhase  = 24
	evalScale = 100
)

type EvaluationService struct {
	opening       int
	endgame       int
	phase         int
	whiteFactor   int
	blackFactor   int
	weights       [2 * fSize]int
	features      [fSize]int
	mobilityBonus [1 + 27]int
	pst           pst
	pstKingShield [64]int
}

type pst struct {
	wn, bn, wb, bb, wq, bq, wkOp, bkOp, wkEg, bkEg [64]int
}

func NewEvaluationService() *EvaluationService {
	var srv = &EvaluationService{}
	srv.initPst()
	srv.initMobility()
	srv.initWeights(autoGeneratedWeights)
	return srv
}

func (e *EvaluationService) initPst() {
	var (
		knightLine = [8]int{0, 2, 3, 4, 4, 3, 2, 0}
		bishopLine = [8]int{0, 1, 2, 3, 3, 2, 1, 0}
		kingLine   = [8]int{0, 2, 3, 4, 4, 3, 2, 0}
	)
	for sq := 0; sq < 64; sq++ {
		var f = File(sq)
		var r = Rank(sq)
		e.pst.wn[sq] = knightLine[f] + knightLine[r]
		//e.pst.wb[sq] = Min(bishopLine[f], bishopLine[r])
		e.pst.wq[sq] = Min(bishopLine[f], bishopLine[r])
		e.pst.wkOp[sq] = Min(dist[sq][SquareG1], dist[sq][SquareB1])
		e.pst.wkEg[sq] = kingLine[f] + kingLine[r]
		e.pstKingShield[sq] = computePstKingShield(sq)
	}
	e.pst.initBlack()
}

func computePstKingShield(sq int) int {
	switch sq {
	case SquareG2, SquareB2:
		return 4
	case SquareH2, SquareH3, SquareG3, SquareF2,
		SquareA2, SquareA3, SquareB3, SquareC2:
		return 3
	default:
		return 2
	}
}

func (e *EvaluationService) initMobility() {
	for i := range e.mobilityBonus {
		e.mobilityBonus[i] = int(math.Round(20 * math.Sqrt(float64(i))))
	}
}

func (e *EvaluationService) initWeights(weights []int) {
	if len(weights) != len(e.weights) {
		return
	}
	copy(e.weights[:], weights)
}

var pawnPassedBonus = [8]int{0, 0, 0, 2, 6, 12, 21, 0}

const (
	darkSquares  = uint64(0xAA55AA55AA55AA55)
	whiteOutpost = (FileCMask | FileDMask | FileEMask | FileFMask) & (Rank4Mask | Rank5Mask | Rank6Mask)
	blackOutpost = (FileCMask | FileDMask | FileEMask | FileFMask) & (Rank5Mask | Rank4Mask | Rank3Mask)
)

var (
	dist            [64][64]int
	whitePawnSquare [64]uint64
	blackPawnSquare [64]uint64
	kingZone        [64]uint64
)

func (e *EvaluationService) add(feature Feature, value int) {
	e.opening += value * e.weights[2*feature]
	e.endgame += value * e.weights[2*feature+1]
	e.features[feature] += value
}

func (e *EvaluationService) Evaluate(p *Position) int {
	var score = e.evaluateCore(p)
	if !p.WhiteMove {
		score = -score
	}
	const Tempo = 8
	return score*PawnValue/e.weights[2*fPawnMaterial+1] + Tempo
}

type evalEntry struct {
	phase       int
	whiteFactor int
	blackFactor int
	features    []featureEntry
}

type featureEntry struct{ index, value int }

func (entry *evalEntry) Evaluate(weights []int) int {
	var scoreOp, scoreEg int
	for _, f := range entry.features {
		scoreOp += weights[2*f.index] * f.value
		scoreEg += weights[2*f.index+1] * f.value
	}
	var score = (scoreOp*entry.phase + scoreEg*(maxPhase-entry.phase)) / maxPhase
	if score > 0 {
		score /= entry.whiteFactor
	} else {
		score /= entry.blackFactor
	}
	return score
}

func (e *EvaluationService) computeEntry(p *Position) evalEntry {
	for i := range e.features {
		e.features[i] = 0
	}
	e.evaluateCore(p)
	var result = evalEntry{
		phase:       e.phase,
		whiteFactor: e.whiteFactor,
		blackFactor: e.blackFactor,
	}
	for i, v := range e.features {
		if v != 0 {
			result.features = append(result.features, featureEntry{i, v})
		}
	}
	return result
}

const (
	WHITE = iota
	BLACK
)

type evalInfo struct {
	pawns           uint64
	pawnCount       int
	knightCount     int
	bishopCount     int
	rookCount       int
	queenCount      int
	pawnAttacks     uint64
	knightAttacks   uint64
	bishopAttacks   uint64
	rookAttacks     uint64
	queenAttacks    uint64
	kingAttacks     uint64
	mobilityArea    uint64
	kingZone        uint64
	king            int
	force           int
	kingAttackNb    int
	kingAttackCount int
}

func (e *EvaluationService) evaluateCore(p *Position) int {
	var (
		x, b         uint64
		sq           int
		keySq, bonus int
		white, black evalInfo
	)

	// init

	var allPieces = p.White | p.Black
	e.opening = 0
	e.endgame = 0

	white.pawns = p.Pawns & p.White
	black.pawns = p.Pawns & p.Black

	white.pawnCount = PopCount(white.pawns)
	black.pawnCount = PopCount(black.pawns)

	white.pawnAttacks = AllWhitePawnAttacks(white.pawns)
	black.pawnAttacks = AllBlackPawnAttacks(black.pawns)

	white.king = FirstOne(p.Kings & p.White)
	black.king = FirstOne(p.Kings & p.Black)

	white.kingAttacks = KingAttacks[white.king]
	black.kingAttacks = KingAttacks[black.king]

	white.kingZone = kingZone[white.king]
	black.kingZone = kingZone[black.king]

	white.mobilityArea = ^(white.pawns | black.pawnAttacks)
	black.mobilityArea = ^(black.pawns | white.pawnAttacks)

	// eval pieces

	for x = p.Knights & p.White; x != 0; x &= x - 1 {
		white.knightCount++
		sq = FirstOne(x)
		e.add(fKnightPst, e.pst.wn[sq])
		b = KnightAttacks[sq]
		e.add(fKnightMobility, e.mobilityBonus[PopCount(b&white.mobilityArea)])
		white.knightAttacks |= b
		if (b & black.kingZone & white.mobilityArea) != 0 {
			white.kingAttackNb++
			white.kingAttackCount += PopCount(b & black.kingZone & white.mobilityArea)
		}
	}

	for x = p.Knights & p.Black; x != 0; x &= x - 1 {
		black.knightCount++
		sq = FirstOne(x)
		e.add(fKnightPst, e.pst.bn[sq])
		b = KnightAttacks[sq]
		e.add(fKnightMobility, -e.mobilityBonus[PopCount(b&black.mobilityArea)])
		black.knightAttacks |= b
		if (b & white.kingZone & black.mobilityArea) != 0 {
			black.kingAttackNb++
			black.kingAttackCount += PopCount(b & white.kingZone & black.mobilityArea)
		}
	}

	for x = p.Bishops & p.White; x != 0; x &= x - 1 {
		white.bishopCount++
		sq = FirstOne(x)
		//e.add(fBishopPst, e.pst.wb[sq])
		b = BishopAttacks(sq, allPieces)
		e.add(fBishopMobility, e.mobilityBonus[PopCount(b&white.mobilityArea)])
		white.bishopAttacks |= b
		if (b & black.kingZone & white.mobilityArea) != 0 {
			white.kingAttackNb++
			white.kingAttackCount += PopCount(b & black.kingZone & white.mobilityArea)
		}
		e.add(fBishopRammedPawns, PopCount(sameColorSquares(sq)&white.pawns&Down(black.pawns)))
	}

	for x = p.Bishops & p.Black; x != 0; x &= x - 1 {
		black.bishopCount++
		sq = FirstOne(x)
		//e.add(fBishopPst, e.pst.bb[sq])
		b = BishopAttacks(sq, allPieces)
		e.add(fBishopMobility, -e.mobilityBonus[PopCount(b&black.mobilityArea)])
		black.bishopAttacks |= b
		if (b & white.kingZone & black.mobilityArea) != 0 {
			black.kingAttackNb++
			black.kingAttackCount += PopCount(b & white.kingZone & black.mobilityArea)
		}
		e.add(fBishopRammedPawns, -PopCount(sameColorSquares(sq)&black.pawns&Up(white.pawns)))
	}

	for x = p.Rooks & p.White; x != 0; x &= x - 1 {
		white.rookCount++
		sq = FirstOne(x)
		if Rank(sq) == Rank7 &&
			((p.Pawns&p.Black&Rank7Mask) != 0 || Rank(black.king) == Rank8) {
			e.add(fRook7th, 1)
		}
		b = RookAttacks(sq, allPieces^(p.Rooks&p.White))
		//b = RookAttacks(sq, allPieces)
		e.add(fRookMobility, e.mobilityBonus[PopCount(b&white.mobilityArea)])
		white.rookAttacks |= b
		if (b & black.kingZone & white.mobilityArea) != 0 {
			white.kingAttackNb++
			white.kingAttackCount += PopCount(b & black.kingZone & white.mobilityArea)
		}
		b = FileMask[File(sq)]
		if (b & white.pawns) == 0 {
			if (b & p.Pawns) == 0 {
				e.add(fRookOpen, 1)
			} else {
				e.add(fRookSemiopen, 1)
			}
		}
	}

	for x = p.Rooks & p.Black; x != 0; x &= x - 1 {
		black.rookCount++
		sq = FirstOne(x)
		if Rank(sq) == Rank2 &&
			((p.Pawns&p.White&Rank2Mask) != 0 || Rank(white.king) == Rank1) {
			e.add(fRook7th, -1)
		}
		b = RookAttacks(sq, allPieces^(p.Rooks&p.Black))
		//b = RookAttacks(sq, allPieces)
		e.add(fRookMobility, -e.mobilityBonus[PopCount(b&black.mobilityArea)])
		black.rookAttacks |= b
		if (b & white.kingZone & black.mobilityArea) != 0 {
			black.kingAttackNb++
			black.kingAttackCount += PopCount(b & white.kingZone & black.mobilityArea)
		}
		b = FileMask[File(sq)]
		if (b & black.pawns) == 0 {
			if (b & p.Pawns) == 0 {
				e.add(fRookOpen, -1)
			} else {
				e.add(fRookSemiopen, -1)
			}
		}
	}

	for x = p.Queens & p.White; x != 0; x &= x - 1 {
		white.queenCount++
		sq = FirstOne(x)
		e.add(fQueenPst, e.pst.wq[sq])
		b = QueenAttacks(sq, allPieces)
		e.add(fQueenMobility, e.mobilityBonus[PopCount(b&white.mobilityArea)])
		white.queenAttacks |= b
		if (b & black.kingZone & white.mobilityArea) != 0 {
			white.kingAttackNb++
			white.kingAttackCount += PopCount(b & black.kingZone & white.mobilityArea)
		}
		e.add(fKingQueenTropism, dist[sq][black.king])
	}

	for x = p.Queens & p.Black; x != 0; x &= x - 1 {
		black.queenCount++
		sq = FirstOne(x)
		e.add(fQueenPst, e.pst.bq[sq])
		b = QueenAttacks(sq, allPieces)
		e.add(fQueenMobility, -e.mobilityBonus[PopCount(b&black.mobilityArea)])
		black.queenAttacks |= b
		if (b & white.kingZone & black.mobilityArea) != 0 {
			black.kingAttackNb++
			black.kingAttackCount += PopCount(b & white.kingZone & black.mobilityArea)
		}
		e.add(fKingQueenTropism, -dist[sq][white.king])
	}

	white.force = white.knightCount + white.bishopCount +
		2*white.rookCount + 4*white.queenCount
	black.force = black.knightCount + black.bishopCount +
		2*black.rookCount + 4*black.queenCount

	e.add(fKingCastlingPst, e.pst.wkOp[white.king]+e.pst.bkOp[black.king])
	e.add(fKingCenterPst, e.pst.wkEg[white.king]+e.pst.bkEg[black.king])

	var kingShield = 0
	for x = white.pawns & KingShieldMask[File(white.king)] &^ LowerRanks[Rank(white.king)]; x != 0; x &= x - 1 {
		sq = FirstOne(x)
		kingShield += e.pstKingShield[sq]
	}
	for x = black.pawns & KingShieldMask[File(black.king)] &^ UpperRanks[Rank(black.king)]; x != 0; x &= x - 1 {
		sq = FirstOne(x)
		kingShield -= e.pstKingShield[FlipSquare(sq)]
	}
	e.add(fKingShelter, kingShield)

	if white.kingAttackNb >= 2 {
		e.add(fKingAttack, white.kingAttackCount-1)
	}
	if black.kingAttackNb >= 2 {
		e.add(fKingAttack, -(black.kingAttackCount - 1))
	}

	// eval threats

	/*var wattacks = white.pawnAttacks |
		white.knightAttacks |
		white.bishopAttacks |
		white.rookAttacks |
		white.queenAttacks |
		white.kingAttacks

	var battacks = black.pawnAttacks |
		black.knightAttacks |
		black.bishopAttacks |
		black.rookAttacks |
		black.queenAttacks |
		black.kingAttacks*/

	e.add(fThreatPawn,
		PopCount(white.pawnAttacks&p.Black&^(p.Pawns|p.Queens))-
			PopCount(black.pawnAttacks&p.White&^(p.Pawns|p.Queens)))

	e.add(fThreatForPawn,
		PopCount((white.rookAttacks|white.kingAttacks)&black.pawns&^black.pawnAttacks)-
			PopCount((black.rookAttacks|black.kingAttacks)&p.White&p.Pawns&^white.pawnAttacks))

	e.add(fThreatPiece,
		PopCount((white.knightAttacks|white.bishopAttacks|white.rookAttacks)&p.Black&(p.Knights|p.Bishops|p.Rooks))-
			PopCount((black.knightAttacks|black.bishopAttacks|black.rookAttacks)&p.White&(p.Knights|p.Bishops|p.Rooks)))

	e.add(fThreatPieceForQueen,
		PopCount((white.pawnAttacks|white.knightAttacks|white.bishopAttacks|white.rookAttacks)&p.Black&p.Queens)-
			PopCount((black.pawnAttacks|black.knightAttacks|black.bishopAttacks|black.rookAttacks)&p.White&p.Queens))

	// eval pawns

	e.add(fPawnWeak,
		PopCount(getWhiteWeakPawns(p))-
			PopCount(getBlackWeakPawns(p)))

	e.add(fPawnDoubled,
		PopCount(getIsolatedPawns(p.Pawns&p.White)&getDoubledPawns(p.Pawns&p.White))-
			PopCount(getIsolatedPawns(p.Pawns&p.Black)&getDoubledPawns(p.Pawns&p.Black)))

	e.add(fPawnDuo,
		PopCount(p.Pawns&p.White&(Left(p.Pawns&p.White)|Right(p.Pawns&p.White)))-
			PopCount(p.Pawns&p.Black&(Left(p.Pawns&p.Black)|Right(p.Pawns&p.Black))))

	e.add(fPawnProtected,
		PopCount(white.pawns&white.pawnAttacks)-
			PopCount(black.pawns&black.pawnAttacks))

	e.add(fMinorProtected,
		PopCount((p.Knights|p.Bishops)&p.White&white.pawnAttacks)-
			PopCount((p.Knights|p.Bishops)&p.Black&black.pawnAttacks))

	var wstrongFields = whiteOutpost &^ DownFill(black.pawnAttacks)
	var bstrongFields = blackOutpost &^ UpFill(white.pawnAttacks)

	e.add(fKnightOutpost,
		PopCount(p.Knights&p.White&wstrongFields)-
			PopCount(p.Knights&p.Black&bstrongFields))

	e.add(fPawnBlockedByOwnPiece,
		PopCount(p.Pawns&p.White&^white.kingZone&(Rank2Mask|Rank3Mask)&Down(p.White))-
			PopCount(p.Pawns&p.Black&^black.kingZone&(Rank7Mask|Rank6Mask)&Up(p.Black)))

	e.add(fPawnRammed,
		PopCount(p.Pawns&p.White&(Rank2Mask|Rank3Mask)&Down(p.Pawns&p.Black))-
			PopCount(p.Pawns&p.Black&(Rank7Mask|Rank6Mask)&Up(p.Pawns&p.White)))

	for x = getWhitePassedPawns(p); x != 0; x &= x - 1 {
		sq = FirstOne(x)
		bonus = pawnPassedBonus[Rank(sq)]
		e.add(fPawnPassed, bonus)
		keySq = sq + 8
		e.add(fPawnPassedOppKing, bonus*dist[keySq][black.king])
		e.add(fPawnPassedOwnKing, bonus*dist[keySq][white.king])
		if (SquareMask[keySq] & p.Black) == 0 {
			e.add(fPawnPassedFree, bonus)
		}
		/*if (UpFill(SquareMask[keySq]) & (p.Black | (battacks &^ wattacks))) == 0 {
			e.add(fPawnPassedSafeAdvance, bonus)
		}*/

		if black.force == 0 {
			var f1 = sq
			if !p.WhiteMove {
				f1 -= 8
			}
			if (whitePawnSquare[f1] & p.Kings & p.Black) == 0 {
				e.add(fPawnPassedSquare, Rank(f1)-Rank1)
				//pawnScore.endgame += 200 * Rank(f1) / Rank7
			}
		}
	}

	for x = getBlackPassedPawns(p); x != 0; x &= x - 1 {
		sq = FirstOne(x)
		bonus = -pawnPassedBonus[Rank(FlipSquare(sq))]
		e.add(fPawnPassed, bonus)
		keySq = sq - 8
		e.add(fPawnPassedOppKing, bonus*dist[keySq][white.king])
		e.add(fPawnPassedOwnKing, bonus*dist[keySq][black.king])
		if (SquareMask[keySq] & p.White) == 0 {
			e.add(fPawnPassedFree, bonus)
		}
		/*if (DownFill(SquareMask[keySq]) & (p.White | (wattacks &^ battacks))) == 0 {
			e.add(fPawnPassedSafeAdvance, bonus)
		}*/

		if white.force == 0 {
			var f1 = sq
			if p.WhiteMove {
				f1 += 8
			}
			if (blackPawnSquare[f1] & p.Kings & p.White) == 0 {
				e.add(fPawnPassedSquare, Rank(f1)-Rank8)
				//pawnScore.endgame -= 200 * (Rank8 - Rank(f1)) / Rank7
			}
		}
	}

	// eval material

	e.add(fPawnMaterial, white.pawnCount-black.pawnCount)
	e.add(fKnightMaterial, white.knightCount-black.knightCount)
	e.add(fBishopMaterial, white.bishopCount-black.bishopCount)
	e.add(fRookMaterial, white.rookCount-black.rookCount)
	e.add(fQueenMaterial, white.queenCount-black.queenCount)
	if white.bishopCount >= 2 {
		e.add(fBishopPairMaterial, 1)
	}
	if black.bishopCount >= 2 {
		e.add(fBishopPairMaterial, -1)
	}

	// mix score

	var phase = white.force + black.force
	if phase > maxPhase {
		phase = maxPhase
	}
	var result = (e.opening*phase + e.endgame*(maxPhase-phase)) / maxPhase

	e.phase = phase
	var ocb = white.force == 1 && black.force == 1 &&
		(p.Bishops&darkSquares) != 0 && (p.Bishops & ^darkSquares) != 0
	e.whiteFactor = computeFactor(&white, &black, ocb)
	e.blackFactor = computeFactor(&black, &white, ocb)

	if result > 0 {
		result /= e.whiteFactor
	} else {
		result /= e.blackFactor
	}

	return result
}

func computeFactor(own, their *evalInfo, ocb bool) int {
	if own.force >= 6 {
		return 1
	}
	if own.pawnCount == 0 {
		if own.force <= 1 {
			return 16
		}
		if own.force == 2 && own.knightCount == 2 && their.pawnCount == 0 {
			return 16
		}
		if own.force-their.force <= 1 {
			return 4
		}
	} else if own.pawnCount == 1 {
		if own.force <= 1 && their.knightCount+their.bishopCount != 0 {
			return 8
		}
		if own.force == their.force && their.knightCount+their.bishopCount != 0 {
			return 2
		}
	} else if ocb && own.pawnCount-their.pawnCount <= 2 {
		return 2
	}
	return 1
}

func getDoubledPawns(pawns uint64) uint64 {
	return DownFill(Down(pawns)) & pawns
}

func getIsolatedPawns(pawns uint64) uint64 {
	return ^FileFill(Left(pawns)|Right(pawns)) & pawns
}

func getWhitePassedPawns(p *Position) uint64 {
	return p.Pawns & p.White &^
		DownFill(Down(Left(p.Pawns&p.Black)|p.Pawns|Right(p.Pawns&p.Black)))
}

func getBlackPassedPawns(p *Position) uint64 {
	return p.Pawns & p.Black &^
		UpFill(Up(Left(p.Pawns&p.White)|p.Pawns|Right(p.Pawns&p.White)))
}

func getWhiteWeakPawns(p *Position) uint64 {
	var pawns = p.Pawns & p.White
	var supported = UpFill(Left(pawns) | Right(pawns))
	var weak = uint64(0)
	weak |= getIsolatedPawns(pawns)
	weak |= (Rank2Mask | Rank3Mask | Rank4Mask) & Down(AllBlackPawnAttacks(p.Pawns&p.Black)) &^ supported
	return pawns & weak

}

func getBlackWeakPawns(p *Position) uint64 {
	var pawns = p.Pawns & p.Black
	var supported = DownFill(Left(pawns) | Right(pawns))
	var weak = uint64(0)
	weak |= getIsolatedPawns(pawns)
	weak |= (Rank7Mask | Rank6Mask | Rank5Mask) & Up(AllWhitePawnAttacks(p.Pawns&p.White)) &^ supported
	return pawns & weak
}

func (pst *pst) initBlack() {
	for sq := 0; sq < 64; sq++ {
		var flipSq = FlipSquare(sq)
		pst.bn[sq] = -pst.wn[flipSq]
		pst.bb[sq] = -pst.wb[flipSq]
		pst.bq[sq] = -pst.wq[flipSq]
		pst.bkOp[sq] = -pst.wkOp[flipSq]
		pst.bkEg[sq] = -pst.wkEg[flipSq]
	}
}

func sameColorSquares(sq int) uint64 {
	if IsDarkSquare(sq) {
		return darkSquares
	}
	return ^darkSquares
}

func limit(v, min, max int) int {
	if v <= min {
		return min
	}
	if v >= max {
		return max
	}
	return v
}

var (
	UpperRanks [8]uint64
	LowerRanks [8]uint64
)

var (
	Ranks          = [8]uint64{Rank1Mask, Rank2Mask, Rank3Mask, Rank4Mask, Rank5Mask, Rank6Mask, Rank7Mask, Rank8Mask}
	KingShieldMask = [8]uint64{
		FileAMask | FileBMask | FileCMask,
		FileAMask | FileBMask | FileCMask,
		FileAMask | FileBMask | FileCMask,
		FileCMask | FileDMask | FileEMask,
		FileDMask | FileEMask | FileFMask,
		FileFMask | FileGMask | FileHMask,
		FileFMask | FileGMask | FileHMask,
		FileFMask | FileGMask | FileHMask}
)

func init() {
	for i := 0; i < 64; i++ {
		for j := 0; j < 64; j++ {
			dist[i][j] = SquareDistance(i, j)
		}
	}
	for sq := 0; sq < 64; sq++ {
		var x = UpFill(SquareMask[sq])
		for j := 0; j < Rank(FlipSquare(sq)); j++ {
			x |= Left(x) | Right(x)
		}
		whitePawnSquare[sq] = x
	}
	for sq := 0; sq < 64; sq++ {
		var x = DownFill(SquareMask[sq])
		for j := 0; j < Rank(sq); j++ {
			x |= Left(x) | Right(x)
		}
		blackPawnSquare[sq] = x
	}
	for i := Rank7; i >= Rank1; i-- {
		UpperRanks[i] = UpperRanks[i+1] | Ranks[i+1]
	}
	for i := Rank2; i <= Rank8; i++ {
		LowerRanks[i] = LowerRanks[i-1] | Ranks[i-1]
	}
	for sq := range kingZone {
		//kingZone[sq] = SquareMask[sq] | KingAttacks[sq]
		var x = MakeSquare(limit(File(sq), FileB, FileG), limit(Rank(sq), Rank2, Rank7))
		kingZone[sq] = SquareMask[x] | KingAttacks[x]
	}
}
