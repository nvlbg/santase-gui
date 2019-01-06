package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten"

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
	r, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	img, err := png.Decode(r)
	if err != nil {
		panic(err)
	}

	result, err := ebiten.NewImageFromImage(img, ebiten.FilterLinear)
	if err != nil {
		panic(err)
	}

	return result
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
	cards               map[santase.Card]*ebiten.Image
	backCard            *ebiten.Image
	movesChan           chan santase.Move
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
	trumpCard := &allCards[12]
	isOpponentMove := false
	ai := santase.CreateGame(opponentHand, *trumpCard, !isOpponentMove)

	return game{
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
		movesChan:      make(chan santase.Move),
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

func (g *game) playResponse(card *santase.Card) {
	g.response = card
	g.blockUI = true
	go func() {
		<-time.After(2 * time.Second)
		g.blockUI = false
		g.cardPlayed = nil
		g.response = nil
	}()
}

func (g *game) update(screen *ebiten.Image) error {
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

	if selected != nil && !g.blockUI && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !g.isOpponentMove && g.hand.HasCard(*selected.card) {
			g.isOpponentMove = true
			g.hand.RemoveCard(*selected.card)
			if g.cardPlayed == nil {
				g.opponentPlayedFirst = false
				g.cardPlayed = selected.card
			} else {
				g.playResponse(selected.card)
			}
			move := santase.NewMove(*selected.card)
			g.movesChan <- move
		}
	}

	if selected != nil && g.hand.HasCard(*selected.card) {
		selected.y -= 20
		selected.rect.Sub(image.Pt(0, -20))
	}

	for _, obj := range objects {
		obj.draw(screen)
	}

	return nil
}

func (g *game) startAI() {
	for move := range g.movesChan {
		g.ai.UpdateOpponentMove(move)
		if g.trumpCard != nil && len(g.hand) == 5 &&
			len(g.opponentHand) == 5 {
			// TODO
		}
		opponentMove := g.ai.GetMove()
		g.opponentHand.RemoveCard(opponentMove.Card)
		if g.cardPlayed == nil {
			g.cardPlayed = &opponentMove.Card
		} else {
			g.playResponse(&opponentMove.Card)
		}
	}
}

func (g *game) Start() {
	go g.startAI()

	if err := ebiten.Run(g.update, 960, 720, 1, "Santase"); err != nil {
		panic(err)
	}
}

func main() {
	game := NewGame()
	game.Start()
}
