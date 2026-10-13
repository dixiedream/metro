package model

// Pattern describes subdivision of each beat.
type Pattern struct {
	ID          string
	Name        string
	Compound    bool
	Subdivs     []float64 // relative lengths within one beat that sum to 1.0
	SubAccents  []bool    // true = stronger subdiv click within the beat
}

var simplePatterns = []Pattern{
	{
		ID: "quarter", Name: "Quarter", Compound: false,
		Subdivs: []float64{1}, SubAccents: []bool{true},
	},
	{
		ID: "eighth", Name: "Eighths", Compound: false,
		Subdivs: []float64{0.5, 0.5}, SubAccents: []bool{true, false},
	},
	{
		ID: "triplet", Name: "Triplets", Compound: false,
		Subdivs: []float64{1.0 / 3, 1.0 / 3, 1.0 / 3}, SubAccents: []bool{true, false, false},
	},
	{
		ID: "sixteenth", Name: "Sixteenths", Compound: false,
		Subdivs: []float64{0.25, 0.25, 0.25, 0.25}, SubAccents: []bool{true, false, false, false},
	},
	{
		ID: "dotted_eighth", Name: "Dotted 8th + 16th", Compound: false,
		Subdivs: []float64{0.75, 0.25}, SubAccents: []bool{true, false},
	},
	{
		ID: "eighth_two_sixteenth", Name: "8th + two 16ths", Compound: false,
		Subdivs: []float64{0.5, 0.25, 0.25}, SubAccents: []bool{true, false, false},
	},
}

var compoundPatterns = []Pattern{
	{
		ID: "compound_dotted", Name: "Dotted Quarter", Compound: true,
		Subdivs: []float64{1}, SubAccents: []bool{true},
	},
	{
		ID: "compound_eighth", Name: "Eighths", Compound: true,
		Subdivs: []float64{1.0 / 3, 1.0 / 3, 1.0 / 3}, SubAccents: []bool{true, false, false},
	},
	{
		ID: "compound_mixed", Name: "Mixed Compound", Compound: true,
		Subdivs: []float64{2.0 / 3, 1.0 / 3}, SubAccents: []bool{true, false},
	},
}

func PatternsFor(compound bool) []Pattern {
	if compound {
		return compoundPatterns
	}
	return simplePatterns
}

func AllPatterns() []Pattern {
	out := make([]Pattern, 0, len(simplePatterns)+len(compoundPatterns))
	out = append(out, simplePatterns...)
	out = append(out, compoundPatterns...)
	return out
}

func PatternByID(id string) Pattern {
	for _, p := range AllPatterns() {
		if p.ID == id {
			return p
		}
	}
	return simplePatterns[0]
}

func DefaultPatternID(compound bool) string {
	if compound {
		return compoundPatterns[0].ID
	}
	return simplePatterns[0].ID
}

func NextPatternID(current string, compound bool) string {
	ps := PatternsFor(compound)
	for i, p := range ps {
		if p.ID == current {
			return ps[(i+1)%len(ps)].ID
		}
	}
	return ps[0].ID
}

func PrevPatternID(current string, compound bool) string {
	ps := PatternsFor(compound)
	for i, p := range ps {
		if p.ID == current {
			return ps[(i-1+len(ps))%len(ps)].ID
		}
	}
	return ps[0].ID
}
