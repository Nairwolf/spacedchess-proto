package chessval

import "testing"

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// Scholar's mate threat position: 1.e4 e5 2.Bc4 Nc6 3.Qh5 Nf6?? — White to move.
const scholarFEN = "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4"

func TestValidateFEN(t *testing.T) {
	side, err := ValidateFEN(startFEN)
	if err != nil || side != "w" {
		t.Fatalf("start position: side=%q err=%v", side, err)
	}
	side, err = ValidateFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1")
	if err != nil || side != "b" {
		t.Fatalf("after 1.e4: side=%q err=%v", side, err)
	}
	if _, err := ValidateFEN("not a fen"); err == nil {
		t.Fatal("garbage FEN should fail")
	}
	if _, err := ValidateFEN(""); err == nil {
		t.Fatal("empty FEN should fail")
	}
}

func TestValidateLine(t *testing.T) {
	if _, err := ValidateLine(scholarFEN, []string{"Qxf7#"}); err != nil {
		t.Fatalf("Qxf7# should be legal: %v", err)
	}
	// A multi-move line with replies.
	if _, err := ValidateLine(startFEN, []string{"e4", "e5", "Nf3"}); err != nil {
		t.Fatalf("legal line rejected: %v", err)
	}
	// Illegal move mid-line.
	if _, err := ValidateLine(startFEN, []string{"e4", "e4"}); err == nil {
		t.Fatal("illegal move should fail")
	}
	// Nonsense SAN.
	if _, err := ValidateLine(startFEN, []string{"Zz9"}); err == nil {
		t.Fatal("unparseable SAN should fail")
	}
	// Empty move.
	if _, err := ValidateLine(startFEN, []string{""}); err == nil {
		t.Fatal("empty move should fail")
	}
}

func TestValidateLineReturnsResultingFEN(t *testing.T) {
	fen, err := ValidateLine(startFEN, []string{"e4"})
	if err != nil {
		t.Fatal(err)
	}
	side, err := ValidateFEN(fen)
	if err != nil || side != "b" {
		t.Fatalf("resulting FEN should be Black to move: side=%q err=%v (fen=%s)", side, err, fen)
	}
}
