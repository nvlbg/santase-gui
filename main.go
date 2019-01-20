package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/text"
	santase "github.com/nvlbg/santase-ai"
	"golang.org/x/image/font"

	cardAssets "github.com/nvlbg/santase-gui/assets/cards"
	"github.com/nvlbg/santase-gui/assets/fonts"
)

func createImageFromBytes(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	result, err := ebiten.NewImageFromImage(img, ebiten.FilterLinear)
	if err != nil {
		panic(err)
	}

	return result
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
	isOver              bool
	isClosed            bool
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
	closeGame           bool
	cards               map[santase.Card]*ebiten.Image
	backCard            *ebiten.Image
	userMoves           chan santase.Move
	ai                  santase.Game
	fontFace            font.Face
	fontFaceSmall       font.Face
	fontFaceBig         font.Face
	debugMode           bool
	debugBtnPressedFlag bool
	announcement        int
}

func NewGame() game {
	cards := make(map[santase.Card]*ebiten.Image)

	cards[santase.NewCard(santase.Nine, santase.Clubs)] = createImageFromBytes(cardAssets.Card9C)
	cards[santase.NewCard(santase.Jack, santase.Clubs)] = createImageFromBytes(cardAssets.CardJC)
	cards[santase.NewCard(santase.Queen, santase.Clubs)] = createImageFromBytes(cardAssets.CardQC)
	cards[santase.NewCard(santase.King, santase.Clubs)] = createImageFromBytes(cardAssets.CardKC)
	cards[santase.NewCard(santase.Ten, santase.Clubs)] = createImageFromBytes(cardAssets.Card10C)
	cards[santase.NewCard(santase.Ace, santase.Clubs)] = createImageFromBytes(cardAssets.CardAC)

	cards[santase.NewCard(santase.Nine, santase.Diamonds)] = createImageFromBytes(cardAssets.Card9D)
	cards[santase.NewCard(santase.Jack, santase.Diamonds)] = createImageFromBytes(cardAssets.CardJD)
	cards[santase.NewCard(santase.Queen, santase.Diamonds)] = createImageFromBytes(cardAssets.CardQD)
	cards[santase.NewCard(santase.King, santase.Diamonds)] = createImageFromBytes(cardAssets.CardKD)
	cards[santase.NewCard(santase.Ten, santase.Diamonds)] = createImageFromBytes(cardAssets.Card10D)
	cards[santase.NewCard(santase.Ace, santase.Diamonds)] = createImageFromBytes(cardAssets.CardAD)

	cards[santase.NewCard(santase.Nine, santase.Hearts)] = createImageFromBytes(cardAssets.Card9H)
	cards[santase.NewCard(santase.Jack, santase.Hearts)] = createImageFromBytes(cardAssets.CardJH)
	cards[santase.NewCard(santase.Queen, santase.Hearts)] = createImageFromBytes(cardAssets.CardQH)
	cards[santase.NewCard(santase.King, santase.Hearts)] = createImageFromBytes(cardAssets.CardKH)
	cards[santase.NewCard(santase.Ten, santase.Hearts)] = createImageFromBytes(cardAssets.Card10H)
	cards[santase.NewCard(santase.Ace, santase.Hearts)] = createImageFromBytes(cardAssets.CardAH)

	cards[santase.NewCard(santase.Nine, santase.Spades)] = createImageFromBytes(cardAssets.Card9S)
	cards[santase.NewCard(santase.Jack, santase.Spades)] = createImageFromBytes(cardAssets.CardJS)
	cards[santase.NewCard(santase.Queen, santase.Spades)] = createImageFromBytes(cardAssets.CardQS)
	cards[santase.NewCard(santase.King, santase.Spades)] = createImageFromBytes(cardAssets.CardKS)
	cards[santase.NewCard(santase.Ten, santase.Spades)] = createImageFromBytes(cardAssets.Card10S)
	cards[santase.NewCard(santase.Ace, santase.Spades)] = createImageFromBytes(cardAssets.CardAS)

	backCard := createImageFromBytes(cardAssets.CardBack)

	allCards := make([]santase.Card, 0, 24)
	for _, card := range santase.AllCards {
		allCards = append(allCards, card)
	}

	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(allCards), func(i, j int) {
		allCards[i], allCards[j] = allCards[j], allCards[i]
	})

	hand := santase.NewHand(allCards[:6]...)
	opponentHand := santase.NewHand(allCards[6:12]...)
	aiHand := santase.NewHand(allCards[6:12]...)
	trumpCard := &allCards[12]
	isOpponentMove := false
	ai := santase.CreateGame(aiHand, *trumpCard, !isOpponentMove, 0.7, 2*time.Second)

	font, err := truetype.Parse(fonts.ArcadeTTF)
	if err != nil {
		panic(err)
	}
	face := truetype.NewFace(font, &truetype.Options{Size: 22})
	smallFace := truetype.NewFace(font, &truetype.Options{Size: 16})
	bigFace := truetype.NewFace(font, &truetype.Options{Size: 50})

	return game{
		score:               0,
		opponentScore:       0,
		isOver:              false,
		isClosed:            false,
		trump:               allCards[12].Suit,
		hand:                hand,
		opponentHand:        opponentHand,
		trumpCard:           trumpCard,
		stack:               allCards[13:],
		cardPlayed:          nil,
		response:            nil,
		isOpponentMove:      isOpponentMove,
		blockUI:             false,
		switchTrumpCard:     false,
		closeGame:           false,
		cards:               cards,
		backCard:            backCard,
		userMoves:           make(chan santase.Move),
		ai:                  ai,
		fontFace:            face,
		fontFaceSmall:       smallFace,
		fontFaceBig:         bigFace,
		debugMode:           false,
		debugBtnPressedFlag: false,
		announcement:        0,
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
	if !g.hand.HasCard(*c) && c != g.trumpCard && c != g.cardPlayed && c != g.response && !g.debugMode {
		img = g.backCard
	} else if g.isClosed && c == g.trumpCard && !g.debugMode {
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
	if g.trumpCard != nil && !g.isClosed && len(g.hand) == 5 && len(g.opponentHand) == 5 {
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

		if g.score >= 66 || g.opponentScore >= 66 {
			g.isOver = true
		}

		g.blockUI = false
	}()
}

func (g *game) isCardLegal(card santase.Card) bool {
	// you're first to play or the game is not closed
	if g.cardPlayed == nil || (g.trumpCard != nil && !g.isClosed) {
		return true
	}

	// playing stronger card of the requested suit
	if card.Suit == g.cardPlayed.Suit && card.Rank > g.cardPlayed.Rank {
		return true
	}

	if g.cardPlayed.Suit == card.Suit {
		for c := range g.hand {
			if c.Suit == g.cardPlayed.Suit && c.Rank > g.cardPlayed.Rank {
				// you're holding stronger card of the same suit that you must play
				return false
			}
		}
		// you don't have stronger card of the same suit
		return true
	}

	for c := range g.hand {
		if c.Suit == g.cardPlayed.Suit {
			// you're holding card of the requested suit that you must play
			return false
		}
	}

	if g.cardPlayed.Suit != g.trump && card.Suit == g.trump {
		// you are forced to play trump card in this case
		return true
	}

	if g.cardPlayed.Suit != g.trump {
		for c := range g.hand {
			if c.Suit == g.trump {
				// you're holding a trump card that you should play
				return false
			}
		}
	}

	// your move is valid
	return true
}

func (g *game) update(screen *ebiten.Image) error {
	screen.Fill(color.NRGBA{0x00, 0xaa, 0x00, 0xff})

	if g.isOver {
		if ebiten.IsDrawingSkipped() {
			return nil
		}

		var message string
		if g.score > g.opponentScore && g.score >= 66 {
			message = "You win!"
		} else if g.opponentScore > g.score && g.opponentScore >= 66 {
			message = "You lose!"
		} else if g.isOpponentMove {
			message = "You lose!"
		} else {
			message = "You win!"
		}

		text.Draw(screen, message, g.fontFaceBig, 300, 300, color.NRGBA{0xff, 0xff, 0xff, 0xff})

		scores := fmt.Sprintf("%3s %3s", strconv.Itoa(g.score), strconv.Itoa(g.opponentScore))
		text.Draw(screen, scores, g.fontFaceBig, 300, 360, color.NRGBA{0xff, 0xff, 0xff, 0xff})
		return nil
	}

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
		if g.isClosed {
			objects = append(objects, g.newCard(&g.stack[len(g.stack)-1], 84, 360, 0, false))
			objects = append(objects, g.newCard(g.trumpCard, 120, 360, 1, true))
		} else {
			objects = append(objects, g.newCard(g.trumpCard, 120, 360, 0, true))
			objects = append(objects, g.newCard(&g.stack[len(g.stack)-1], 84, 360, 1, false))
		}
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

	if !g.debugBtnPressedFlag && ebiten.IsKeyPressed(ebiten.KeyF12) {
		g.debugBtnPressedFlag = ebiten.IsKeyPressed(ebiten.KeyF12)
		g.debugMode = !g.debugMode
	}
	g.debugBtnPressedFlag = ebiten.IsKeyPressed(ebiten.KeyF12)

	if selected != nil && !g.blockUI && !g.isOpponentMove &&
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if g.hand.HasCard(*selected.card) && g.isCardLegal(*selected.card) {
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

			move = santase.Move{
				Card: *selected.card,
			}

			if g.switchTrumpCard {
				move.SwitchTrumpCard = true
				g.switchTrumpCard = false
			}

			if isAnnouncement {
				move.IsAnnouncement = true
			}

			if g.closeGame {
				move.CloseGame = true
				g.closeGame = false
			}

			g.userMoves <- move
		} else if !g.isClosed && selected.card == g.trumpCard && len(g.stack) > 1 && len(g.stack) < 11 {
			nineTrump := santase.NewCard(santase.Nine, g.trump)
			if g.hand.HasCard(nineTrump) {
				g.hand.RemoveCard(nineTrump)
				g.hand.AddCard(*g.trumpCard)
				g.trumpCard = &nineTrump
				g.switchTrumpCard = true
			}
		} else if !g.isClosed && len(g.stack) > 1 && len(g.stack) < 11 && selected.card == &g.stack[len(g.stack)-1] {
			g.isClosed = true
			g.closeGame = true
		}
	}

	if selected != nil && !g.isOpponentMove && !g.blockUI &&
		g.hand.HasCard(*selected.card) && g.isCardLegal(*selected.card) {
		selected.y -= 20
		selected.rect.Sub(image.Pt(0, -20))
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	for _, obj := range objects {
		obj.draw(screen)
	}

	text.Draw(screen, "Score:"+strconv.Itoa(g.score), g.fontFace, 760, 680, color.White)

	if g.trumpCard != nil {
		text.Draw(screen, strconv.Itoa(1+len(g.stack))+" cards", g.fontFaceSmall, 20, 490, color.White)
	}

	if g.debugMode {
		text.Draw(screen, "Score:"+strconv.Itoa(g.opponentScore), g.fontFace, 760, 40, color.White)
	}

	if g.announcement != 0 {
		var x, y int
		if g.isOpponentMove {
			x, y = 650, 450
		} else {
			x, y = 275, 300
		}
		text.Draw(screen, strconv.Itoa(g.announcement), g.fontFaceBig, x, y, color.NRGBA{0xff, 0x00, 0x00, 0xff})
	}

	return nil
}

func (g *game) playAIMove() {
	opponentMove := g.ai.GetMove()
	if opponentMove.SwitchTrumpCard {
		g.blockUI = true
		nineTrump := santase.NewCard(santase.Nine, g.trump)
		g.opponentHand.RemoveCard(nineTrump)
		g.opponentHand.AddCard(*g.trumpCard)
		g.trumpCard = &nineTrump
		<-time.After(2 * time.Second)
		g.blockUI = false
	}
	if opponentMove.CloseGame {
		g.blockUI = true
		g.isClosed = true
		<-time.After(2 * time.Second)
		g.blockUI = false
	}
	if opponentMove.IsAnnouncement {
		if opponentMove.Card.Suit == g.trump {
			g.opponentScore += 40
			g.announcement = 40
		} else {
			g.opponentScore += 20
			g.announcement = 20
		}

		if g.opponentScore >= 66 {
			g.isOver = true
		}
	} else {
		g.announcement = 0
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
		if move.IsAnnouncement {
			if move.Card.Suit == g.trump {
				g.announcement = 40
			} else {
				g.announcement = 20
			}

			if g.score >= 66 {
				g.isOver = true
				return
			}
		} else {
			g.announcement = 0
		}
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
