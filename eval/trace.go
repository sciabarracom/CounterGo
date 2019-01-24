package eval

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ChizhovVadim/CounterGo/common"
)

func featureName(feature int) string {
	return Feature(feature).String()
}

func PrintWeights() {
	var w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintf(w, "Feature\tOpening\tEndgame\t\n")
	for featureIndex := 0; featureIndex < int(fSize); featureIndex++ {
		fmt.Fprintf(w, "%v\t%v\t%v\t\n",
			featureName(featureIndex),
			autoGeneratedWeights[2*featureIndex],
			autoGeneratedWeights[2*featureIndex+1])
	}
	w.Flush()
}

func (e *EvaluationService) Trace(p *common.Position) {
	var pawnEg = e.weights[2*fPawnMaterial+1]
	var entry = e.computeEntry(p)
	var w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprintf(w, "Feature\tScore\t\n")
	for _, f := range entry.features {
		var scoreOp = e.weights[2*f.index] * f.value
		var scoreEg = e.weights[2*f.index+1] * f.value
		var score = (entry.phase*scoreOp + (maxPhase-entry.phase)*scoreEg) / maxPhase
		score = score * PawnValue / pawnEg
		fmt.Fprintf(w, "%v\t%v\t\n",
			featureName(f.index), score)
	}
	w.Flush()
	var score = entry.Evaluate(e.weights[:]) * PawnValue / pawnEg
	fmt.Println("Score:", score)
}
