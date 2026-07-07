package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/nairwolf/spacedchess/internal/chessval"
	"github.com/nairwolf/spacedchess/internal/store"
)

// cardPayload is the request body for creating/updating a card.
type cardPayload struct {
	CardType   string          `json:"card_type"`
	FEN        string          `json:"fen"`
	Details    json.RawMessage `json:"details"`
	SourceNote string          `json:"source_note"`
	Tags       []string        `json:"tags"`
	SetIDs     []int64         `json:"set_ids"`
}

type tacticalDetails struct {
	Solution []string `json:"solution"`
}

type blunderDetails struct {
	IntendedMove       string   `json:"intended_move"`
	Refutation         []string `json:"refutation"`
	CorrectAlternative []string `json:"correct_alternative"`
}

type strategicDetails struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// validateCard checks the payload (FEN legality, solution-line legality per
// card type) and returns a normalized store input. Details are re-marshaled
// from typed structs so only known fields are persisted.
func validateCard(p cardPayload) (store.CardInput, error) {
	var in store.CardInput

	side, err := chessval.ValidateFEN(strings.TrimSpace(p.FEN))
	if err != nil {
		return in, err
	}

	trimLine := func(line []string) []string {
		out := make([]string, 0, len(line))
		for _, m := range line {
			if m = strings.TrimSpace(m); m != "" {
				out = append(out, m)
			}
		}
		return out
	}

	fen := strings.TrimSpace(p.FEN)
	var details any

	switch p.CardType {
	case "tactical_opportunity":
		var d tacticalDetails
		if err := json.Unmarshal(p.Details, &d); err != nil {
			return in, fmt.Errorf("invalid details")
		}
		d.Solution = trimLine(d.Solution)
		if len(d.Solution) == 0 {
			return in, fmt.Errorf("solution is required")
		}
		if _, err := chessval.ValidateLine(fen, d.Solution); err != nil {
			return in, fmt.Errorf("solution: %w", err)
		}
		details = d

	case "blunder":
		var d blunderDetails
		if err := json.Unmarshal(p.Details, &d); err != nil {
			return in, fmt.Errorf("invalid details")
		}
		d.IntendedMove = strings.TrimSpace(d.IntendedMove)
		d.Refutation = trimLine(d.Refutation)
		d.CorrectAlternative = trimLine(d.CorrectAlternative)
		if d.IntendedMove == "" {
			return in, fmt.Errorf("intended move is required")
		}
		if len(d.Refutation) == 0 {
			return in, fmt.Errorf("refutation is required")
		}
		if len(d.CorrectAlternative) == 0 {
			return in, fmt.Errorf("correct alternative is required")
		}
		afterBlunder, err := chessval.ValidateLine(fen, []string{d.IntendedMove})
		if err != nil {
			return in, fmt.Errorf("intended move: %w", err)
		}
		if _, err := chessval.ValidateLine(afterBlunder, d.Refutation); err != nil {
			return in, fmt.Errorf("refutation: %w", err)
		}
		if _, err := chessval.ValidateLine(fen, d.CorrectAlternative); err != nil {
			return in, fmt.Errorf("correct alternative: %w", err)
		}
		if d.CorrectAlternative[0] == d.IntendedMove {
			return in, fmt.Errorf("correct alternative must differ from the intended move")
		}
		details = d

	case "strategic_mistake":
		var d strategicDetails
		if err := json.Unmarshal(p.Details, &d); err != nil {
			return in, fmt.Errorf("invalid details")
		}
		d.Question = strings.TrimSpace(d.Question)
		d.Answer = strings.TrimSpace(d.Answer)
		if d.Question == "" {
			return in, fmt.Errorf("question is required")
		}
		if d.Answer == "" {
			return in, fmt.Errorf("answer is required")
		}
		details = d

	default:
		return in, fmt.Errorf("unknown card type")
	}

	raw, err := json.Marshal(details)
	if err != nil {
		return in, err
	}
	return store.CardInput{
		CardType:   p.CardType,
		FEN:        fen,
		SideToMove: side,
		Details:    raw,
		SourceNote: strings.TrimSpace(p.SourceNote),
		Tags:       p.Tags,
		SetIDs:     p.SetIDs,
	}, nil
}

func (s *Server) handleCreateCard(w http.ResponseWriter, r *http.Request) {
	var p cardPayload
	if !s.decode(w, r, &p) {
		return
	}
	in, err := validateCard(p)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	card, err := s.store.CreateCard(r.Context(), userFrom(r).ID, in)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, card)
}

func (s *Server) handleUpdateCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid card id")
		return
	}
	var p cardPayload
	if !s.decode(w, r, &p) {
		return
	}
	in, err := validateCard(p)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	card, err := s.store.UpdateCard(r.Context(), userFrom(r).ID, id, in)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, card)
}

func (s *Server) handleGetCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid card id")
		return
	}
	card, err := s.store.GetCard(r.Context(), userFrom(r).ID, id)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, card)
}

func (s *Server) handleDeleteCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid card id")
		return
	}
	if err := s.store.DeleteCard(r.Context(), userFrom(r).ID, id); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func cardFilterFromQuery(r *http.Request) store.CardFilter {
	q := r.URL.Query()
	f := store.CardFilter{
		CardType: q.Get("type"),
		Tag:      q.Get("tag"),
		Search:   q.Get("q"),
	}
	if setID, err := strconv.ParseInt(q.Get("set_id"), 10, 64); err == nil {
		f.SetID = setID
	}
	return f
}

func (s *Server) handleListCards(w http.ResponseWriter, r *http.Request) {
	cards, err := s.store.ListCards(r.Context(), userFrom(r).ID, cardFilterFromQuery(r))
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, cards)
}
