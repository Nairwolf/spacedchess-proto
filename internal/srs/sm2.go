// Package srs implements the SM-2 spaced-repetition scheduling algorithm
// (SPEC.md §10). Scheduling state is kept isolated from card content so the
// algorithm can be swapped later (e.g. for FSRS) without touching cards.
//
// The product's grading signal is binary (SPEC.md §5). SM-2's quality scale
// is 0–5, so grades are mapped: correct/remembered → q=5, incorrect/didn't
// remember → q=2. A failing quality (<3) resets repetitions and schedules
// the card for the next day; easiness is updated on every review as in the
// original algorithm, clamped to the standard 1.3 floor.
package srs

import (
	"math"
	"time"
)

const (
	MinEasiness     = 1.3
	InitialEasiness = 2.5

	qualityCorrect   = 5.0
	qualityIncorrect = 2.0
)

// State is a card's scheduling state, mirroring the review_state table.
type State struct {
	Easiness     float64
	IntervalDays int
	Repetitions  int
	DueAt        time.Time
}

// NewState returns the scheduling state for a freshly created card:
// due immediately, never reviewed.
func NewState(now time.Time) State {
	return State{
		Easiness:     InitialEasiness,
		IntervalDays: 0,
		Repetitions:  0,
		DueAt:        now,
	}
}

// Review applies one SM-2 review to the state and returns the new state.
func Review(s State, correct bool, now time.Time) State {
	q := qualityIncorrect
	if correct {
		q = qualityCorrect
	}

	ef := s.Easiness + (0.1 - (5-q)*(0.08+(5-q)*0.02))
	if ef < MinEasiness {
		ef = MinEasiness
	}

	var interval, reps int
	if !correct {
		reps = 0
		interval = 1
	} else {
		reps = s.Repetitions + 1
		switch reps {
		case 1:
			interval = 1
		case 2:
			interval = 6
		default:
			interval = int(math.Round(float64(s.IntervalDays) * ef))
			if interval < 1 {
				interval = 1
			}
		}
	}

	return State{
		Easiness:     ef,
		IntervalDays: interval,
		Repetitions:  reps,
		DueAt:        now.AddDate(0, 0, interval),
	}
}
