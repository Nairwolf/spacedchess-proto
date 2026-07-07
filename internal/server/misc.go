package server

import (
	"net/http"
)

// --- Tags ---

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.store.ListTags(r.Context(), userFrom(r).ID)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, tags)
}

type nameBody struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var in nameBody
	if !s.decode(w, r, &in) {
		return
	}
	tag, err := s.store.CreateTag(r.Context(), userFrom(r).ID, in.Name)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, tag)
}

func (s *Server) handleRenameTag(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	var in nameBody
	if !s.decode(w, r, &in) {
		return
	}
	if err := s.store.RenameTag(r.Context(), userFrom(r).ID, id, in.Name); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid tag id")
		return
	}
	if err := s.store.DeleteTag(r.Context(), userFrom(r).ID, id); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- Sets ---

func (s *Server) handleListSets(w http.ResponseWriter, r *http.Request) {
	sets, err := s.store.ListSets(r.Context(), userFrom(r).ID)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, sets)
}

func (s *Server) handleCreateSet(w http.ResponseWriter, r *http.Request) {
	var in nameBody
	if !s.decode(w, r, &in) {
		return
	}
	set, err := s.store.CreateSet(r.Context(), userFrom(r).ID, in.Name)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, set)
}

func (s *Server) handleRenameSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	var in nameBody
	if !s.decode(w, r, &in) {
		return
	}
	if err := s.store.RenameSet(r.Context(), userFrom(r).ID, id, in.Name); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid set id")
		return
	}
	if err := s.store.DeleteSet(r.Context(), userFrom(r).ID, id); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAddCardToSet(w http.ResponseWriter, r *http.Request) {
	setID, err1 := pathID(r, "id")
	cardID, err2 := pathID(r, "cardId")
	if err1 != nil || err2 != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.AddCardToSet(r.Context(), userFrom(r).ID, setID, cardID); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleRemoveCardFromSet(w http.ResponseWriter, r *http.Request) {
	setID, err1 := pathID(r, "id")
	cardID, err2 := pathID(r, "cardId")
	if err1 != nil || err2 != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.RemoveCardFromSet(r.Context(), userFrom(r).ID, setID, cardID); err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- Review ---

func (s *Server) handleDueCards(w http.ResponseWriter, r *http.Request) {
	f := cardFilterFromQuery(r)
	f.DueOnly = true
	cards, err := s.store.ListCards(r.Context(), userFrom(r).ID, f)
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, cards)
}

type reviewBody struct {
	Correct *bool `json:"correct"`
}

func (s *Server) handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid card id")
		return
	}
	var in reviewBody
	if !s.decode(w, r, &in) {
		return
	}
	if in.Correct == nil {
		s.jsonError(w, http.StatusBadRequest, `"correct" is required`)
		return
	}
	state, err := s.store.SubmitReview(r.Context(), userFrom(r).ID, id, *in.Correct, timeNow())
	if err != nil {
		s.storeError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, state)
}
