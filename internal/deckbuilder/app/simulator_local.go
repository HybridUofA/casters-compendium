package deckbuilder

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/decklibrary"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/ai"
	"github.com/HybridUofA/casters-compendium/internal/simulator/ai/tsubasa"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
	simulatorui "github.com/HybridUofA/casters-compendium/internal/simulator/ui"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// runLocalHotseatMatch opens dual PlayerSession boards (existing Play a Game path).
func runLocalHotseatMatch(
	window fyne.Window,
	playerSessions [2]*session.PlayerSession,
	repository *cards.Repository,
	onBackToMenu func(),
) {
	runLocalSimulatorUI(window, playerSessions, repository, onBackToMenu, nil)
}

// runSoloVsTsubasaMatch seats the human as P1 and drives P2 with Tsubasa.
func runSoloVsTsubasaMatch(
	window fyne.Window,
	playerSessions [2]*session.PlayerSession,
	repository *cards.Repository,
	onBackToMenu func(),
) {
	driver := &ai.Driver{
		Session: playerSessions[1],
		Agent:   tsubasa.Agent{Catalog: repository},
	}
	runLocalSimulatorUI(window, playerSessions, repository, onBackToMenu, driver)
}

// buildSoloVsTsubasaDecks loads the human deck and the official Ignus list for Tsubasa.
func buildSoloVsTsubasaDecks(
	human decks.Deck,
	repository *cards.Repository,
) ([2]decks.Deck, error) {
	aiDeck, err := decklibrary.BuildOfficialTemplate("dd01-ignus", repository)
	if err != nil {
		return [2]decks.Deck{}, fmt.Errorf("tsubasa deck (DD01 Ignus): %w", err)
	}
	return [2]decks.Deck{human, *aiDeck}, nil
}

func runLocalSimulatorUI(
	window fyne.Window,
	playerSessions [2]*session.PlayerSession,
	repository *cards.Repository,
	onBackToMenu func(),
	soloDriver *ai.Driver,
) {
	solo := soloDriver != nil
	playerWindows := [2]fyne.Window{window, nil}
	if !solo {
		playerWindows[1] = fyne.CurrentApp().NewWindow(applicationName + " — Player Two")
		playerWindows[1].Resize(fyne.NewSize(1400, 850))
	}
	if solo {
		playerWindows[0].SetTitle(applicationName + " — Solo vs Tsubasa")
	} else {
		playerWindows[0].SetTitle(applicationName + " — Player One")
	}
	cardDefinitions := repository.All()
	var playerScreens [2]*simulatorui.BoardScreen

	var renderPlayersWithView func(int, simulatorview.MatchView)
	var kickSoloAI func()

	scheduleBoardRefresh := func(updatedIndex int, updatedView simulatorview.MatchView) {
		fyne.Do(func() {
			renderPlayersWithView(updatedIndex, updatedView)
			if kickSoloAI != nil {
				kickSoloAI()
			}
		})
	}

	kickSoloAI = func() {
		if soloDriver == nil {
			return
		}
		go func() {
			_, runErr := soloDriver.RunUntilIdle()
			fyne.Do(func() {
				if runErr != nil {
					dialog.ShowError(fmt.Errorf("Tsubasa AI: %w", runErr), window)
				}
				matchView, viewErr := playerSessions[0].View()
				if viewErr != nil {
					dialog.ShowError(viewErr, window)
					return
				}
				applySimulatorCosmetics(&matchView)
				if playerScreens[0] != nil {
					playerScreens[0].Update(matchView)
				}
			})
		}()
	}

	renderPlayersWithView = func(updatedIndex int, updatedView simulatorview.MatchView) {
		for index, playerSession := range playerSessions {
			if solo && index == 1 {
				continue
			}
			matchView := updatedView
			if index != updatedIndex {
				var viewErr error
				matchView, viewErr = playerSession.View()
				if viewErr != nil {
					dialog.ShowError(viewErr, playerWindows[index])
					continue
				}
			}
			applySimulatorCosmetics(&matchView)
			if playerScreens[index] == nil {
				playerIndex := index
				back := func() {
					if solo {
						onBackToMenu()
						return
					}
					if playerIndex == 0 {
						if playerWindows[1] != nil {
							playerWindows[1].Close()
						}
						onBackToMenu()
						return
					}
					if playerWindows[1] != nil {
						playerWindows[1].Close()
					}
				}
				backLabel := "Back to Main Menu"
				if index == 1 {
					backLabel = "Close Player Two Window"
				}
				actions := localBoardActions(playerSessions, playerWindows, playerIndex, cardDefinitions, scheduleBoardRefresh)
				actions.BackLabel = backLabel
				playerScreens[index] = simulatorui.NewBoardController(
					matchView,
					cardDefinitions,
					actions,
					back,
				)
				setWindowContent(playerWindows[index], playerScreens[index].Content())
				simulatorui.BindBoardKeyboard(playerWindows[index], playerScreens[index])
				continue
			}
			playerScreens[index].Update(matchView)
		}
	}

	renderPlayersWithView(-1, simulatorview.MatchView{})
	if solo {
		kickSoloAI()
	} else if playerWindows[1] != nil {
		playerWindows[1].Show()
	}
}

func localBoardActions(
	playerSessions [2]*session.PlayerSession,
	playerWindows [2]fyne.Window,
	playerIndex int,
	cardDefinitions []cards.Card,
	scheduleBoardRefresh func(int, simulatorview.MatchView),
) simulatorui.BoardActions {
	return simulatorui.BoardActions{
		ShouldAutoPassPriority: makeShouldAutoPassPriority(cardDefinitions),
		UseCasterToken: func(tokenID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].UseCasterToken(tokenID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		GenerateCasterAether: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].GenerateCasterAether(cardID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		GenerateNonElementalAether: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].GenerateNonElementalAether(cardID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CallFaceDownLevelOne: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CallFaceDownLevelOne(cardID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CallFaceUpLevelOne: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CallFaceUpLevelOne(cardID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		LevelUpCaster: func(upperCardID, targetCasterID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].LevelUpCaster(upperCardID, targetCasterID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CastServant: func(cardID model.MatchCardID, payment model.AetherPayment, orientation model.CardOrientation, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CastServant(cardID, payment, orientation, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CastConjure: func(cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CastConjure(cardID, payment, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CastBarrier: func(cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CastBarrier(cardID, payment, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CastServantWithPlan: func(cardID model.MatchCardID, plan model.CastPaymentPlan, orientation model.CardOrientation, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CastServantWithPlan(cardID, plan, orientation, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CastConjureWithPlan: func(cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CastConjureWithPlan(cardID, plan, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CastBarrierWithPlan: func(cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CastBarrierWithPlan(cardID, plan, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		MoveCard: func(command model.MoveCardCommand, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].MoveCard(command, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		AdjustAether: func(command model.AdjustAetherCommand, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].AdjustAether(command, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		DrawCards: func(count int, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].DrawCards(count, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		ShuffleDeck: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].ShuffleDeck(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		PeekDeckTops: func(ownerID model.PlayerID, count int, done func([]simulatorview.CardView, error)) {
			peeked, updatedView, err := playerSessions[playerIndex].PeekDeckTops(ownerID, count)
			if err != nil {
				if done != nil {
					done(nil, err)
				}
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
			if done != nil {
				done(peeked, nil)
			}
		},
		MoveDeckTopToBottom: func(ownerID model.PlayerID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].MoveDeckTopToBottom(ownerID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		ActivateHildeWillow: func(cardID model.MatchCardID, expectedRevision model.Revision, done func(simulatorview.CardView, model.PlayerID, error)) {
			peeked, enemyID, updatedView, err := playerSessions[playerIndex].ActivateHildeWillow(cardID, expectedRevision)
			if err != nil {
				if done != nil {
					done(simulatorview.CardView{}, "", err)
				}
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
			if done != nil {
				done(peeked, enemyID, nil)
			}
		},
		ResolveDeckDig: func(keep model.MatchCardID, bottomOrder []model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].ResolveDeckDig(keep, bottomOrder, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		PassPriority: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].PassPriority(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		DeclareAttack: func(attackerID model.MatchCardID, targetKind model.AttackTargetKind, targetCardID model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].DeclareAttack(attackerID, targetKind, targetCardID, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CorruptOrbs: func(orbIndexes []int, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CorruptOrbs(orbIndexes, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		SetGrantedDoubleCorrupt: func(cardID model.MatchCardID, enabled bool, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].SetGrantedDoubleCorrupt(cardID, enabled, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		PlayBreak: func(cardID model.MatchCardID, orientation model.CardOrientation, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].PlayBreak(cardID, orientation, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		DeclineBreak: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].DeclineBreak(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		ResolvePendingChoice: func(optionID string, selectedCardIDs []model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].ResolvePendingChoice(optionID, selectedCardIDs, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		DeclinePendingChoice: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].DeclinePendingChoice(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		AcceptSageAdvice: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].AcceptSageAdvice(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		DeclineDrawReplacement: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].DeclineDrawReplacement(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		PeekOrb: func(ownerID model.PlayerID, orbIndex int, expectedRevision model.Revision, done func(simulatorview.CardView, error)) {
			peeked, updatedView, err := playerSessions[playerIndex].PeekOrb(ownerID, orbIndex, expectedRevision)
			if err != nil {
				if done != nil {
					done(simulatorview.CardView{}, err)
				}
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
			if done != nil {
				done(peeked, nil)
			}
		},
		RevealOrb: func(orbIndex int, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].RevealOrb(orbIndex, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		CompleteCurrentPhase: func(expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].CompleteCurrentPhase(expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
		SubmitOpeningHand: func(replace []model.MatchCardID, expectedRevision model.Revision) {
			updatedView, err := playerSessions[playerIndex].SubmitOpeningHandDecision(replace, expectedRevision)
			if err != nil {
				dialog.ShowError(err, playerWindows[playerIndex])
				return
			}
			scheduleBoardRefresh(playerIndex, updatedView)
		},
	}
}
