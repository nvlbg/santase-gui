package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"sort"

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

type game struct {
	trump          santase.Suit
	hand           santase.Hand
	opponentHand   santase.Hand
	trumpCard      *santase.Card
	stack          []santase.Card
	cardPlayed     *santase.Card
	isOpponentMove bool
}

func newGame() game {
	allCards := make([]santase.Card, 0, 24)
	for _, card := range santase.AllCards {
		allCards = append(allCards, card)
	}
	rand.Shuffle(len(allCards), func(i, j int) {
		allCards[i], allCards[j] = allCards[j], allCards[i]
	})

	return game{
		trump:          allCards[12].Suit,
		hand:           santase.NewHand(allCards[:6]...),
		opponentHand:   santase.NewHand(allCards[6:12]...),
		trumpCard:      &allCards[12],
		stack:          allCards[13:],
		cardPlayed:     nil,
		isOpponentMove: false,
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

type Card struct {
	card    *santase.Card
	rect    image.Rectangle
	image   *ebiten.Image
	x       int
	y       int
	zIndex  int
	flipped bool
}

func NewCard(card *santase.Card, x, y, z int, flipped bool) *Card {
	var img *ebiten.Image
	if !currentGame.hand.HasCard(*card) && card != currentGame.trumpCard &&
		card != currentGame.cardPlayed {
		img = backCard
	} else {
		img = cards[*card]
	}

	width, height := img.Size()

	if flipped {
		width, height = height, width
	}

	width /= 5
	height /= 5

	return &Card{
		card:    card,
		rect:    image.Rect(x-width/2, y-width/2, x+width/2, y+height/2),
		image:   img,
		x:       x,
		y:       y,
		zIndex:  z,
		flipped: flipped,
	}
}

func (c *Card) Draw(screen *ebiten.Image) {
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

func (c *Card) Intersects(x, y int) bool {
	return x >= c.rect.Min.X && x <= c.rect.Max.X && y >= c.rect.Min.Y && y <= c.rect.Max.Y
}

var cards = make(map[santase.Card]*ebiten.Image)
var backCard *ebiten.Image
var currentGame game

func update(screen *ebiten.Image) error {
	screen.Fill(color.NRGBA{0x00, 0xaa, 0x00, 0xff})

	var objects []*Card
	cardX := 270
	z := 0
	for _, card := range currentGame.getHand() {
		func(card santase.Card) {
			objects = append(objects, NewCard(&card, cardX, 600, z, false))
			cardX += 80
			z++
		}(card)
	}

	cardX = 270
	z = 0
	for _, card := range currentGame.getOpponentHand() {
		func(card santase.Card) {
			objects = append(objects, NewCard(&card, cardX, 120, z, false))
			cardX += 80
			z++
		}(card)
	}

	if currentGame.trumpCard != nil {
		objects = append(objects, NewCard(currentGame.trumpCard, 120, 360, 0, true))
		objects = append(objects, NewCard(&currentGame.stack[0], 84, 360, 1, false))
	}

	if currentGame.cardPlayed != nil {
		objects = append(objects, NewCard(currentGame.cardPlayed, 540, 360, 0, false))
	}

	x, y := ebiten.CursorPosition()

	var selected *Card
	for _, obj := range objects {
		if obj.Intersects(x, y) && (selected == nil || selected.zIndex < obj.zIndex) {
			selected = obj
		}
	}

	if selected != nil && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !currentGame.isOpponentMove && currentGame.hand.HasCard(*selected.card) {
			if currentGame.cardPlayed == nil {
				currentGame.isOpponentMove = true
				currentGame.cardPlayed = selected.card
				currentGame.hand.RemoveCard(*selected.card)
			} else {

			}
		}
	}

	if selected != nil && currentGame.hand.HasCard(*selected.card) {
		selected.y -= 20
		selected.rect.Sub(image.Pt(0, -20))
	}

	for _, obj := range objects {
		obj.Draw(screen)
	}

	return nil
}

func main() {
	for _, card := range santase.AllCards {
		cards[card] = createImage(card)
	}
	backCard = createImageFromPath("assets/red_back.png")

	currentGame = newGame()

	if err := ebiten.Run(update, 960, 720, 1, "Hello world!"); err != nil {
		panic(err)
	}
}
