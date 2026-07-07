// Package chessval validates FENs and SAN move lines server-side, so cards
// can never be saved with illegal positions or solutions.
package chessval

import (
	"fmt"
	"strings"

	"github.com/notnil/chess"
)

// ValidateFEN checks that fen is a parseable, legal position and returns
// the side to move ("w" or "b").
func ValidateFEN(fen string) (string, error) {
	fenOpt, err := chess.FEN(fen)
	if err != nil {
		return "", fmt.Errorf("invalid FEN: %w", err)
	}
	game := chess.NewGame(fenOpt)
	if game.Position().Turn() == chess.White {
		return "w", nil
	}
	return "b", nil
}

// ValidateLine checks that each SAN move in line is legal, applied in
// sequence from fen. Returns the FEN after the whole line.
func ValidateLine(fen string, line []string) (string, error) {
	fenOpt, err := chess.FEN(fen)
	if err != nil {
		return "", fmt.Errorf("invalid FEN: %w", err)
	}
	game := chess.NewGame(fenOpt)
	for i, san := range line {
		san = strings.TrimSpace(san)
		if san == "" {
			return "", fmt.Errorf("move %d is empty", i+1)
		}
		if err := game.MoveStr(san); err != nil {
			return "", fmt.Errorf("move %d (%s) is not legal here: %w", i+1, san, err)
		}
	}
	return game.Position().String(), nil
}
