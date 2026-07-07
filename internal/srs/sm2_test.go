package srs

import (
	"math"
	"testing"
	"time"
)

var now = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestNewState(t *testing.T) {
	s := NewState(now)
	if s.Easiness != 2.5 || s.IntervalDays != 0 || s.Repetitions != 0 {
		t.Fatalf("unexpected new state: %+v", s)
	}
	if !s.DueAt.Equal(now) {
		t.Fatalf("new card should be due immediately, got %v", s.DueAt)
	}
}

func TestConsecutiveCorrectSequence(t *testing.T) {
	// Classic SM-2 with q=5 on every review:
	// intervals 1, 6, then previous*EF; EF grows by 0.1 per correct review.
	s := NewState(now)

	s = Review(s, true, now)
	if s.IntervalDays != 1 || !almostEqual(s.Easiness, 2.6) || s.Repetitions != 1 {
		t.Fatalf("after 1st correct: %+v", s)
	}
	if !s.DueAt.Equal(now.AddDate(0, 0, 1)) {
		t.Fatalf("due date after 1st correct: %v", s.DueAt)
	}

	s = Review(s, true, now)
	if s.IntervalDays != 6 || !almostEqual(s.Easiness, 2.7) || s.Repetitions != 2 {
		t.Fatalf("after 2nd correct: %+v", s)
	}

	// 3rd correct: EF becomes 2.8, interval = round(6 * 2.8) = 17.
	s = Review(s, true, now)
	if s.IntervalDays != 17 || !almostEqual(s.Easiness, 2.8) || s.Repetitions != 3 {
		t.Fatalf("after 3rd correct: %+v", s)
	}

	// 4th correct: EF 2.9, interval = round(17 * 2.9) = 49.
	s = Review(s, true, now)
	if s.IntervalDays != 49 || !almostEqual(s.Easiness, 2.9) || s.Repetitions != 4 {
		t.Fatalf("after 4th correct: %+v", s)
	}
	if !s.DueAt.Equal(now.AddDate(0, 0, 49)) {
		t.Fatalf("due date after 4th correct: %v", s.DueAt)
	}
}

func TestIncorrectResetsRepetitionsAndInterval(t *testing.T) {
	s := NewState(now)
	s = Review(s, true, now)
	s = Review(s, true, now)
	s = Review(s, true, now) // interval 17, EF 2.8, reps 3

	s = Review(s, false, now)
	if s.Repetitions != 0 {
		t.Fatalf("failure should reset repetitions, got %d", s.Repetitions)
	}
	if s.IntervalDays != 1 {
		t.Fatalf("failure should schedule for next day, got interval %d", s.IntervalDays)
	}
	// EF updated with q=2: 2.8 - 0.32 = 2.48.
	if !almostEqual(s.Easiness, 2.48) {
		t.Fatalf("EF after failure: %v", s.Easiness)
	}
	if !s.DueAt.Equal(now.AddDate(0, 0, 1)) {
		t.Fatalf("due date after failure: %v", s.DueAt)
	}
}

func TestRecoveryAfterFailureRestartsIntervals(t *testing.T) {
	s := NewState(now)
	s = Review(s, true, now)
	s = Review(s, true, now)
	s = Review(s, true, now)
	s = Review(s, false, now)

	// First correct after failure restarts at interval 1, then 6.
	s = Review(s, true, now)
	if s.IntervalDays != 1 || s.Repetitions != 1 {
		t.Fatalf("first correct after failure: %+v", s)
	}
	s = Review(s, true, now)
	if s.IntervalDays != 6 || s.Repetitions != 2 {
		t.Fatalf("second correct after failure: %+v", s)
	}
}

func TestEasinessFloor(t *testing.T) {
	s := NewState(now)
	for i := 0; i < 10; i++ {
		s = Review(s, false, now)
	}
	if !almostEqual(s.Easiness, MinEasiness) {
		t.Fatalf("EF should clamp at %v, got %v", MinEasiness, s.Easiness)
	}
	if s.IntervalDays != 1 || s.Repetitions != 0 {
		t.Fatalf("state after repeated failures: %+v", s)
	}
}

func TestIntervalAtEasinessFloor(t *testing.T) {
	// A card starting at the EF floor still grows, and its EF recovers by
	// 0.1 per correct review: intervals 1, 6, then round(6*1.6)=10.
	s := State{Easiness: MinEasiness, IntervalDays: 0, Repetitions: 0, DueAt: now}
	s = Review(s, true, now)
	if s.IntervalDays != 1 {
		t.Fatalf("interval 1 expected, got %d", s.IntervalDays)
	}
	s = Review(s, true, now)
	if s.IntervalDays != 6 {
		t.Fatalf("interval 6 expected, got %d", s.IntervalDays)
	}
	s = Review(s, true, now)
	if s.IntervalDays != 10 {
		t.Fatalf("interval 10 expected, got %d", s.IntervalDays)
	}
	if !almostEqual(s.Easiness, 1.3+0.3) {
		t.Fatalf("EF should have grown from floor, got %v", s.Easiness)
	}
}
