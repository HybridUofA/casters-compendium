package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const sidePanelWidth float32 = 220

type sidePanel struct {
	root *fyne.Container

	chaseWrap    *fyne.Container
	chaseCards   *fyne.Container
	chaseCaption *widget.Label

	eventList *widget.List
	eventData []simulatorview.MatchMessage

	chatList *widget.List
	chatData []simulatorview.MatchMessage

	chatEntry  *widget.Entry
	chatSend   *widget.Button
	onSendChat func(string)

	definitions cardLookup
	match       simulatorview.MatchView
}

func newSidePanel(definitions cardLookup, onSendChat func(string)) *sidePanel {
	panel := &sidePanel{
		definitions: definitions,
		onSendChat:  onSendChat,
	}
	panel.chaseCaption = widget.NewLabel("Chase (top resolves first)")
	panel.chaseCaption.TextStyle = fyne.TextStyle{Bold: true}
	panel.chaseCards = container.NewHBox()
	panel.chaseWrap = container.NewVBox(
		panel.chaseCaption,
		container.NewHScroll(panel.chaseCards),
	)
	panel.chaseWrap.Hide()

	panel.eventList = newMessageList(panel, func() []simulatorview.MatchMessage { return panel.eventData })
	panel.chatList = newMessageList(panel, func() []simulatorview.MatchMessage { return panel.chatData })

	panel.chatEntry = widget.NewEntry()
	panel.chatEntry.SetPlaceHolder("Message opponent…")
	panel.chatEntry.OnSubmitted = func(text string) {
		panel.submitChat(text)
	}
	panel.chatSend = widget.NewButton("Send", func() {
		panel.submitChat(panel.chatEntry.Text)
	})
	if onSendChat == nil {
		panel.chatEntry.Disable()
		panel.chatSend.Disable()
	}

	chatRow := container.NewBorder(nil, nil, nil, panel.chatSend, panel.chatEntry)
	eventsTab := container.NewBorder(nil, nil, nil, nil, panel.eventList)
	chatTab := container.NewBorder(nil, chatRow, nil, nil, panel.chatList)
	tabs := container.NewAppTabs(
		container.NewTabItem("Events", eventsTab),
		container.NewTabItem("Chat", chatTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	bg := canvas.NewRectangle(color.NRGBA{R: 24, G: 40, B: 44, A: 255})
	bg.StrokeColor = zoneBorder
	bg.StrokeWidth = 1
	inner := container.NewBorder(
		panel.chaseWrap,
		nil,
		nil,
		nil,
		tabs,
	)
	sizer := canvas.NewRectangle(color.Transparent)
	sizer.SetMinSize(fyne.NewSize(sidePanelWidth, 0))
	panel.root = container.NewStack(sizer, bg, container.NewPadded(inner))
	return panel
}

func newMessageList(panel *sidePanel, data func() []simulatorview.MatchMessage) *widget.List {
	return widget.NewList(
		func() int { return len(data()) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			messages := data()
			if id < 0 || id >= len(messages) {
				label.SetText("")
				return
			}
			label.SetText(formatMatchMessage(panel.match, messages[id]))
		},
	)
}

func (panel *sidePanel) submitChat(text string) {
	if panel == nil || panel.onSendChat == nil {
		return
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	panel.onSendChat(trimmed)
	if panel.chatEntry != nil {
		panel.chatEntry.SetText("")
	}
}

func (panel *sidePanel) update(match simulatorview.MatchView) {
	if panel == nil {
		return
	}
	panel.match = match
	panel.refreshChase(match.ChaseLinks)

	events := make([]simulatorview.MatchMessage, 0, len(match.Messages))
	chats := make([]simulatorview.MatchMessage, 0, len(match.Messages))
	for _, msg := range match.Messages {
		if msg.Kind == simulatorview.MessageKindChat {
			chats = append(chats, msg)
			continue
		}
		events = append(events, msg)
	}
	panel.eventData = events
	panel.chatData = chats
	panel.eventList.Refresh()
	panel.chatList.Refresh()
	if len(panel.eventData) > 0 {
		panel.eventList.ScrollToBottom()
	}
	if len(panel.chatData) > 0 {
		panel.chatList.ScrollToBottom()
	}
}

func (panel *sidePanel) refreshChase(links []simulatorview.ChaseLinkView) {
	if panel == nil || panel.chaseWrap == nil || panel.chaseCards == nil {
		return
	}
	objects := make([]fyne.CanvasObject, 0, len(links))
	// Show top of stack on the right (resolves first visually last→first left-to-right).
	for i := len(links) - 1; i >= 0; i-- {
		link := links[i]
		def := cards.Card{}
		if panel.definitions != nil {
			def = panel.definitions[link.Card.CardID]
		}
		tile := NewCardTile(link.Card, def, fyne.NewSize(fieldCardWidth, fieldCardHeight), nil, nil)
		label := widget.NewLabel(chaseLinkKindLabel(link))
		label.TextStyle = fyne.TextStyle{Italic: true}
		objects = append(objects, container.NewVBox(tile, label))
	}
	panel.chaseCards.Objects = objects
	refreshContainerStructure(panel.chaseCards)
	if len(links) == 0 {
		panel.chaseWrap.Hide()
	} else {
		panel.chaseCaption.SetText(fmt.Sprintf("Chase (%d) — top resolves first", len(links)))
		panel.chaseWrap.Show()
	}
	refreshContainerStructure(panel.chaseWrap)
}

func chaseLinkKindLabel(link simulatorview.ChaseLinkView) string {
	switch link.Kind {
	case model.ChaseLinkCardPlay:
		return "play"
	case model.ChaseLinkActivatedAbility:
		return "activate"
	case model.ChaseLinkTriggeredAbility:
		return "trigger"
	default:
		return string(link.Kind)
	}
}

func formatMatchMessage(match simulatorview.MatchView, msg simulatorview.MatchMessage) string {
	who := "System"
	if msg.PlayerID != "" {
		who = displayNameFor(match, msg.PlayerID)
	}
	switch msg.Kind {
	case simulatorview.MessageKindChat:
		return fmt.Sprintf("%s: %s", who, msg.Text)
	default:
		if msg.PlayerID == "" {
			return msg.Text
		}
		return fmt.Sprintf("[%s] %s", who, msg.Text)
	}
}
