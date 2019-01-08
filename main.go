package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/text"
	"golang.org/x/image/font/inconsolata"

	"santase"
)

var ranks = map[santase.Rank]string{
	santase.Nine:  "9",
	santase.Jack:  "J",
	santase.Queen: "Q",
	santase.King:  "K",
	santase.Ten:   "10",
	santase.Ace:   "A",
}

var suits = map[santase.Suit]string{
	santase.Clubs:    "C",
	santase.Diamonds: "D",
	santase.Hearts:   "H",
	santase.Spades:   "S",
}

func createImageFromPath(path string) *ebiten.Image {
	image, _, err := ebitenutil.NewImageFromFile(path, ebiten.FilterLinear)
	if err != nil {
		panic(err)
	}
	return image
}

func createImage(card santase.Card) *ebiten.Image {
	rank := ranks[card.Rank]
	suit := suits[card.Suit]
	path := "assets/" + rank + suit + ".png"

	return createImageFromPath(path)
}

type card struct {
	card    *santase.Card
	rect    image.Rectangle
	image   *ebiten.Image
	x       int
	y       int
	zIndex  int
	flipped bool
}

func (c *card) draw(screen *ebiten.Image) {
	width, height := c.image.Size()
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(-float64(width)/2, -float64(height)/2)
	opts.GeoM.Scale(0.2, 0.2)
	if c.flipped {
		opts.GeoM.Rotate(1.570796)
	}
	opts.GeoM.Translate(float64(c.x), float64(c.y))
	screen.DrawImage(c.image, &opts)
}

func (c *card) intersects(x, y int) bool {
	return x >= c.rect.Min.X && x <= c.rect.Max.X && y >= c.rect.Min.Y && y <= c.rect.Max.Y
}

type game struct {
	score               int
	opponentScore       int
	trump               santase.Suit
	hand                santase.Hand
	opponentHand        santase.Hand
	trumpCard           *santase.Card
	stack               []santase.Card
	cardPlayed          *santase.Card
	response            *santase.Card
	opponentPlayedFirst bool
	isOpponentMove      bool
	blockUI             bool
	switchTrumpCard     bool
	cards               map[santase.Card]*ebiten.Image
	backCard            *ebiten.Image
	userMoves           chan santase.Move
	ai                  santase.Game
}

func NewGame() game {
	cards := make(map[santase.Card]*ebiten.Image)
	for _, card := range santase.AllCards {
		cards[card] = createImage(card)
	}
	backCard := createImageFromPath("assets/red_back.png")

	allCards := make([]santase.Card, 0, 24)
	for _, card := range santase.AllCards {
		allCards = append(allCards, card)
	}
	rand.Shuffle(len(allCards), func(i, j int) {
		allCards[i], allCards[j] = allCards[j], allCards[i]
	})

	hand := santase.NewHand(allCards[:6]...)
	opponentHand := santase.NewHand(allCards[6:12]...)
	aiHand := santase.NewHand(allCards[6:12]...)
	trumpCard := &allCards[12]
	isOpponentMove := false
	ai := santase.CreateGame(aiHand, *trumpCard, !isOpponentMove)

	return game{
		score:          0,
		opponentScore:  0,
		trump:          allCards[12].Suit,
		hand:           hand,
		opponentHand:   opponentHand,
		trumpCard:      trumpCard,
		stack:          allCards[13:],
		cardPlayed:     nil,
		response:       nil,
		isOpponentMove: isOpponentMove,
		blockUI:        false,
		cards:          cards,
		backCard:       backCard,
		userMoves:      make(chan santase.Move),
		ai:             ai,
	}
}

func (g *game) getHand() []santase.Card {
	var cards []santase.Card
	for card := range g.hand {
		cards = append(cards, card)
	}
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Suit < cards[j].Suit || (cards[i].Suit == cards[j].Suit && cards[i].Rank < cards[j].Rank)
	})

	return cards
}

func (g *game) getOpponentHand() []santase.Card {
	var cards []santase.Card
	for card := range g.opponentHand {
		cards = append(cards, card)
	}
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Suit < cards[j].Suit || (cards[i].Suit == cards[j].Suit && cards[i].Rank < cards[j].Rank)
	})

	return cards
}

func (g *game) newCard(c *santase.Card, x, y, z int, flipped bool) *card {
	var img *ebiten.Image
	if !g.hand.HasCard(*c) && c != g.trumpCard && c != g.cardPlayed && c != g.response {
		img = g.backCard
	} else {
		img = g.cards[*c]
	}

	width, height := img.Size()

	if flipped {
		width, height = height, width
	}

	width /= 5
	height /= 5

	return &card{
		card:    c,
		rect:    image.Rect(x-width/2, y-width/2, x+width/2, y+height/2),
		image:   img,
		x:       x,
		y:       y,
		zIndex:  z,
		flipped: flipped,
	}
}

func (g *game) drawCard() santase.Card {
	if len(g.stack) > 0 {
		result := g.stack[len(g.stack)-1]
		g.stack = g.stack[:len(g.stack)-1]
		return result
	}

	if g.trumpCard != nil {
		result := *g.trumpCard
		g.trumpCard = nil
		return result
	}

	panic("all cards have been drawn")
}

func (g *game) drawCards(aiWon bool) {
	if g.trumpCard != nil && len(g.hand) == 5 && len(g.opponentHand) == 5 {
		var opponentDrawnCard, playerDrawnCard santase.Card
		if aiWon {
			opponentDrawnCard, playerDrawnCard = g.drawCard(), g.drawCard()
		} else {
			playerDrawnCard, opponentDrawnCard = g.drawCard(), g.drawCard()
		}
		g.ai.UpdateDrawnCard(opponentDrawnCard)
		g.hand.AddCard(playerDrawnCard)
		g.opponentHand.AddCard(opponentDrawnCard)
	}
}

func (g *game) playResponse(card *santase.Card) {
	g.blockUI = true
	g.response = card

	go func() {
		<-time.After(2 * time.Second)

		stronger := g.ai.StrongerCard(g.cardPlayed, g.response)
		aiWon := (g.opponentPlayedFirst && stronger == g.cardPlayed) ||
			(!g.opponentPlayedFirst && stronger == g.response)

		g.drawCards(aiWon)
		handPoints := santase.Points(g.cardPlayed) + santase.Points(g.response)
		g.cardPlayed = nil
		g.response = nil

		if aiWon {
			g.isOpponentMove = true
			g.opponentScore += handPoints
			g.playAIMove()
		} else {
			g.isOpponentMove = false
			g.score += handPoints
		}

		g.blockUI = false
	}()
}

func (g *game) update(screen *ebiten.Image) error {
	fmt.Println(g.score, g.opponentScore)
	screen.Fill(color.NRGBA{0x00, 0xaa, 0x00, 0xff})

	var objects []*card
	cardX := 270
	z := 0
	for _, card := range g.getHand() {
		func(card santase.Card) {
			objects = append(objects, g.newCard(&card, cardX, 600, z, false))
			cardX += 80
			z++
		}(card)
	}

	cardX = 270
	z = 0
	for _, card := range g.getOpponentHand() {
		func(card santase.Card) {
			objects = append(objects, g.newCard(&card, cardX, 120, z, false))
			cardX += 80
			z++
		}(card)
	}

	if g.trumpCard != nil {
		objects = append(objects, g.newCard(g.trumpCard, 120, 360, 0, true))
		objects = append(objects, g.newCard(&g.stack[0], 84, 360, 1, false))
	}

	if g.cardPlayed != nil {
		var x, y int
		if g.opponentPlayedFirst {
			x, y = 500, 340
		} else {
			x, y = 540, 360
		}
		objects = append(objects, g.newCard(g.cardPlayed, x, y, 0, false))
	}

	if g.response != nil {
		var x, y int
		if g.opponentPlayedFirst {
			x, y = 540, 360
		} else {
			x, y = 500, 340
		}
		objects = append(objects, g.newCard(g.response, x, y, 1, false))
	}

	x, y := ebiten.CursorPosition()

	var selected *card
	for _, obj := range objects {
		if obj.intersects(x, y) && (selected == nil || selected.zIndex < obj.zIndex) {
			selected = obj
		}
	}

	if selected != nil && !g.blockUI && !g.isOpponentMove &&
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		// TODO: check that selected card is allowed to be played
		if g.hand.HasCard(*selected.card) {
			var move santase.Move
			var isAnnouncement bool
			if (selected.card.Rank == santase.Queen || selected.card.Rank == santase.King) &&
				g.cardPlayed == nil && len(g.stack) < 11 {
				var other santase.Card
				if selected.card.Rank == santase.Queen {
					other = santase.NewCard(santase.King, selected.card.Suit)
				} else {
					other = santase.NewCard(santase.Queen, selected.card.Suit)
				}
				if g.hand.HasCard(other) {
					isAnnouncement = true
					if selected.card.Suit == g.trump {
						g.score += 40
					} else {
						g.score += 20
					}
				}
			}

			if isAnnouncement && g.switchTrumpCard {
				move = santase.NewMoveWithAnnouncementAndTrumpCardSwitch(*selected.card)
				g.switchTrumpCard = false
			} else if g.switchTrumpCard {
				move = santase.NewMoveWithTrumpCardSwitch(*selected.card)
				g.switchTrumpCard = false
			} else if isAnnouncement {
				move = santase.NewMoveWithAnnouncement(*selected.card)
			} else {
				move = santase.NewMove(*selected.card)
			}

			g.userMoves <- move
		} else if selected.card == g.trumpCard && len(g.stack) > 1 && len(g.stack) < 11 {
			nineTrump := santase.NewCard(santase.Nine, g.trump)
			if g.hand.HasCard(nineTrump) {
				g.hand.RemoveCard(nineTrump)
				g.hand.AddCard(*g.trumpCard)
				g.trumpCard = &nineTrump
				g.switchTrumpCard = true
			}
		}
	}

	if selected != nil && g.hand.HasCard(*selected.card) {
		selected.y -= 20
		selected.rect.Sub(image.Pt(0, -20))
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	for _, obj := range objects {
		obj.draw(screen)
	}

	// text.Draw(screen, string(g.score), inconsolata.Bold8x16, 850, 620, color.White)
	text.Draw(screen, "asdf", inconsolata.Bold8x16, 0, 0, color.White)

	return nil
}

func (g *game) playAIMove() {
	opponentMove := g.ai.GetMove()
	if opponentMove.SwitchTrumpCard {
		nineTrump := santase.NewCard(santase.Nine, g.trump)
		g.opponentHand.RemoveCard(nineTrump)
		g.opponentHand.AddCard(*g.trumpCard)
		g.trumpCard = &nineTrump
	}
	if opponentMove.IsAnnouncement {
		if opponentMove.Card.Suit == g.trump {
			g.opponentScore += 40
		} else {
			g.opponentScore += 20
		}
	}
	g.opponentHand.RemoveCard(opponentMove.Card)

	if g.cardPlayed == nil {
		g.opponentPlayedFirst = true
		g.isOpponentMove = false
		g.cardPlayed = &opponentMove.Card
	} else {
		g.playResponse(&opponentMove.Card)
	}
}

func (g *game) handleUserMoves() {
	for move := range g.userMoves {
		g.hand.RemoveCard(move.Card)
		g.ai.UpdateOpponentMove(move)

		if g.cardPlayed == nil {
			g.opponentPlayedFirst = false
			g.isOpponentMove = true
			g.cardPlayed = &move.Card
			g.playAIMove()
		} else {
			g.playResponse(&move.Card)
		}
	}
}

func (g *game) Start() {
	go g.handleUserMoves()

	if err := ebiten.Run(g.update, 960, 720, 1, "Santase"); err != nil {
		panic(err)
	}
}

func main() {
	game := NewGame()
	game.Start()
}
