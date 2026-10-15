package session

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const (
	maxMatchMessages = 200
	maxChatRunes     = 280
)

// appendMessageLocked records a log/chat line. Caller must hold match.mu.
func (match *LocalMatch) appendMessageLocked(msg view.MatchMessage) {
	if match == nil {
		return
	}
	match.nextMsgSeq++
	msg.Seq = match.nextMsgSeq
	msg.Text = strings.TrimSpace(msg.Text)
	if msg.Text == "" {
		return
	}
	match.messages = append(match.messages, msg)
	if len(match.messages) > maxMatchMessages {
		match.messages = append([]view.MatchMessage(nil), match.messages[len(match.messages)-maxMatchMessages:]...)
	}
}

func (match *LocalMatch) copyMessagesLocked() []view.MatchMessage {
	if match == nil || len(match.messages) == 0 {
		return nil
	}
	return append([]view.MatchMessage(nil), match.messages...)
}

// projectLocked builds the viewer MatchView including Chase + shared messages.
// Caller must hold match.mu (read or write).
func (session *PlayerSession) projectLocked() (view.MatchView, error) {
	matchView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, err
	}
	matchView.Messages = session.match.copyMessagesLocked()
	return matchView, nil
}

func (session *PlayerSession) finishLocked(event string) (view.MatchView, error) {
	if event != "" {
		session.match.appendMessageLocked(view.MatchMessage{
			Kind:     view.MessageKindEvent,
			PlayerID: session.playerID,
			Text:     event,
		})
	}
	return session.projectLocked()
}

// SendChat posts a player chat line visible to all seats. Does not bump game revision.
func (session *PlayerSession) SendChat(text string) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return view.MatchView{}, fmt.Errorf("chat message cannot be empty")
	}
	if utf8.RuneCountInString(trimmed) > maxChatRunes {
		return view.MatchView{}, fmt.Errorf("chat message too long (max %d characters)", maxChatRunes)
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	session.match.appendMessageLocked(view.MatchMessage{
		Kind:     view.MessageKindChat,
		PlayerID: session.playerID,
		Text:     trimmed,
	})
	return session.projectLocked()
}

func (session *PlayerSession) cardLabelLocked(cardID model.MatchCardID) string {
	if session == nil || session.match == nil {
		return string(cardID)
	}
	instance, ok := session.match.state.CardInstances[cardID]
	if !ok {
		return string(cardID)
	}
	if session.match.catalog != nil {
		if def, found := session.match.catalog.FindByID(string(instance.CardID)); found && def.Name != "" {
			return def.Name
		}
	}
	return string(instance.CardID)
}
