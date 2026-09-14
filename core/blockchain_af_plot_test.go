//go:build plot

// These are plotting and exploration tools, not tests: they assert nothing a regression
// would trip, and each writes a PNG into the package directory. They sit behind a build
// tag so they neither run in CI nor linger as skipped tests. Run one with, for example:
//
//	go test -tags plot -run TestPlot_ecbp1100PolynomialV ./core/

package core

import (
	"fmt"
	"image/color"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func TestPlot_ecbp1100PolynomialV(t *testing.T) {
	p := plot.New()
	p.Title.Text = "ECBP1100 Polynomial Curve Function"
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	poly := plotter.NewFunction(func(f float64) float64 {
		n := big.NewInt(int64(f))
		y := ecbp1100PolynomialV(n)
		ff, _ := new(big.Float).SetInt(y).Float64()
		return ff
	})
	p.Add(poly)

	p.X.Min = 0
	p.X.Max = 30000
	p.Y.Min = 0
	p.Y.Max = 5000

	p.Y.Label.Text = "Antigravity imposition"
	p.X.Label.Text = "Seconds difference between local head and proposed common ancestor"

	if err := p.Save(1000, 1000, "ecbp1100-polynomial.png"); err != nil {
		t.Fatal(err)
	}
}

func TestDifficultyDelta(t *testing.T) {
	parent := &types.Header{
		Number:     big.NewInt(1_000_000),
		Difficulty: params.DefaultMessNetGenesisBlock().Difficulty,
		Time:       uint64(time.Now().Unix()),
		UncleHash:  types.EmptyUncleHash,
	}

	data := plotter.XYs{}

	for i := uint64(1); i <= 60; i++ {
		nextTime := parent.Time + i
		d := ethash.CalcDifficulty(params.MessNetConfig, nextTime, parent)

		rat, _ := new(big.Float).Quo(
			new(big.Float).SetInt(d),
			new(big.Float).SetInt(parent.Difficulty),
		).Float64()

		t.Log(i, rat)
		data = append(data, plotter.XY{X: float64(i), Y: rat})
	}

	p := plot.New()
	p.Title.Text = "Block Difficulty Delta by Timestamp Offset"
	p.X.Label.Text = "Timestamp Offset"
	p.Y.Label.Text = "Relative Difficulty (child/parent)"

	dataScatter, _ := plotter.NewScatter(data)
	p.Add(dataScatter)

	if err := p.Save(800, 600, "difficulty-adjustments.png"); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateChainTargetingHashrate(t *testing.T) {
	engine := ethash.NewFaker()

	db := rawdb.NewMemoryDatabase()
	genesis := params.DefaultMessNetGenesisBlock()
	// genesis.Timestamp = 1
	genesisB := MustCommitGenesis(db, triedb.NewDatabase(db, nil), genesis)

	chain, err := NewBlockChain(db, nil, genesis, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	chain.EnableArtificialFinality(true)

	easy, _ := GenerateChain(genesis.Config, genesisB, engine, db, 1000, func(i int, gen *BlockGen) {
		gen.OffsetTime(0)
	})
	if _, err := chain.InsertChain(easy); err != nil {
		t.Fatal(err)
	}

	baseDifficulty := chain.CurrentHeader().Difficulty
	targetDifficultyRatio := big.NewInt(4)
	targetDifficulty := new(big.Int).Mul(baseDifficulty, targetDifficultyRatio)

	data := plotter.XYs{}

	for chain.CurrentHeader().Difficulty.Cmp(targetDifficulty) < 0 {
		bl := chain.GetBlock(chain.CurrentHeader().Hash(), chain.CurrentHeader().Number.Uint64())
		next, _ := GenerateChain(genesis.Config, bl, engine, db, 1, func(i int, gen *BlockGen) {
			gen.OffsetTime(-9) // 8: (=10+8=18>(13+4=17).. // minimum value over stable range
		})
		if _, err := chain.InsertChain(next); err != nil {
			t.Fatal(err)
		}

		// f, _ := new(big.Float).SetInt(next[0].Difficulty()).Float64()
		// data = append(data, plotter.XY{X: float64(next[0].NumberU64()), Y: f})

		rat1, _ := new(big.Float).Quo(
			new(big.Float).SetInt(next[0].Difficulty()),
			new(big.Float).SetInt(targetDifficulty),
		).Float64()

		// rat, _ := new(big.Float).Quo(
		// 	new(big.Float).SetInt(next[0].Difficulty()),
		// 	new(big.Float).SetInt(targetDifficultyRatio),
		// ).Float64()

		data = append(data, plotter.XY{X: float64(next[0].NumberU64()), Y: rat1})
	}
	t.Log(chain.CurrentBlock().Number)

	p := plot.New()
	p.Title.Text = fmt.Sprintf("Block Difficulty Toward Target: %dx", targetDifficultyRatio.Uint64())
	p.X.Label.Text = "Block Number"
	p.Y.Label.Text = "Difficulty"

	dataScatter, _ := plotter.NewScatter(data)
	p.Add(dataScatter)

	if err := p.Save(800, 600, "difficulty-toward-target.png"); err != nil {
		t.Fatal(err)
	}
}

func TestBlockChain_GenerateMESSPlot(t *testing.T) {
	easyLen := 500
	maxHardLen := 400

	generatePlot := func(title, fileName string) {
		p := plot.New()
		p.Title.Text = title
		p.X.Label.Text = "Block Depth"
		p.Y.Label.Text = "Mode Block Time Offset (10 seconds + y)"

		accepteds := plotter.XYs{}
		rejecteds := plotter.XYs{}
		sides := plotter.XYs{}

		for i := 1; i <= maxHardLen; i++ {
			for j := -9; j <= 8; j++ {
				fmt.Println("running", i, j)
				hardHead, _, hard, easy := runMESSTest2(t, true, easyLen, i, easyLen-i, 0, int64(j))
				point := plotter.XY{X: float64(i), Y: float64(j)}
				switch {
				case hardHead:
					accepteds = append(accepteds, point)
				case segmentDifficulty(hard).Cmp(segmentDifficulty(easy[easyLen-i:])) > 0:
					// Heavier, yet not head: fork choice refused it without an error.
					rejecteds = append(rejecteds, point)
				default:
					sides = append(sides, point)
				}
			}
		}

		t.Logf("accepted=%d refused=%d sidechained=%d", len(accepteds), len(rejecteds), len(sides))

		scatterAccept, _ := plotter.NewScatter(accepteds)
		scatterReject, _ := plotter.NewScatter(rejecteds)
		scatterSide, _ := plotter.NewScatter(sides)

		pixelWidth := vg.Length(1000)

		scatterAccept.Color = color.RGBA{R: 152, G: 236, B: 161, A: 255}
		scatterAccept.Shape = draw.BoxGlyph{}
		scatterAccept.Radius = vg.Length((float64(pixelWidth) / float64(maxHardLen)) * 2 / 3)
		scatterReject.Color = color.RGBA{R: 236, G: 106, B: 94, A: 255}
		scatterReject.Shape = draw.BoxGlyph{}
		scatterReject.Radius = vg.Length((float64(pixelWidth) / float64(maxHardLen)) * 2 / 3)
		scatterSide.Color = color.RGBA{R: 190, G: 197, B: 236, A: 255}
		scatterSide.Shape = draw.BoxGlyph{}
		scatterSide.Radius = vg.Length((float64(pixelWidth) / float64(maxHardLen)) * 2 / 3)

		p.Add(scatterAccept)
		p.Legend.Add("Accepted", scatterAccept)
		p.Add(scatterReject)
		p.Legend.Add("Rejected", scatterReject)
		p.Add(scatterSide)
		p.Legend.Add("Sidechained", scatterSide)

		p.Legend.YOffs = -30

		err := p.Save(pixelWidth, 300, fileName)
		if err != nil {
			log.Panic(err)
		}
	}
	baseTitle := fmt.Sprintf("Accept/Reject Reorgs: Relative Time (Difficulty) over Proposed Segment Length (%d-block original chain)", easyLen)
	generatePlot(baseTitle, "reorgs-MESS.png")
}

// segmentDifficulty sums the difficulty of a chain segment.
func segmentDifficulty(blocks []*types.Block) *big.Int {
	sum := new(big.Int)
	for _, b := range blocks {
		sum.Add(sum, b.Difficulty())
	}
	return sum
}
