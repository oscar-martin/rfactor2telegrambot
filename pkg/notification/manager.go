package notification

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/oscar-martin/rfactor2telegrambot/pkg/model"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/pubsub"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/settings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/nikoksr/notify"
)

const (
	TypeTestDay  = "testday"
	TypePractice = "practice1"
	TypeQual     = "qual1"
	TypeWarnup   = "warmup"
	TypeRace     = "race1"
)

type Lister interface {
	ListUsersForSessionStarted(sessionType string) ([]settings.TelegramUser, error)
}

type Manager struct {
	ctx    context.Context
	lister Lister
	bot    *tgbotapi.BotAPI
	loc    *i18n.Localizer
}

func NewManager(ctx context.Context, bot *tgbotapi.BotAPI, lister Lister, loc *i18n.Localizer) *Manager {
	return &Manager{
		ctx:    ctx,
		bot:    bot,
		lister: lister,
		loc:    loc,
	}
}

func (m *Manager) Start(exitChan <-chan bool) {
	startedChan := pubsub.FirstDriverEnteredPubSub.Subscribe(pubsub.PubSubFirstDriverEnteredPreffix)
	for {
		select {
		case <-exitChan:
			return
		case newSession := <-startedChan:
			sessionType := strings.ToLower(newSession.SessionType)
			if isSessionToBeNotified(sessionType) {
				log.Printf("Session to be notified started: %s -> %s\n", newSession.ServerName, newSession.SessionType)
				switch {
				case isTestDay(sessionType):
					m.handleNotification(newSession, settings.TestDay)
				case isPractice(sessionType):
					m.handleNotification(newSession, settings.Practice)
				case isQual(sessionType):
					m.handleNotification(newSession, settings.Qual)
				case isWarmup(sessionType):
					m.handleNotification(newSession, settings.Warmup)
				case isRace(sessionType):
					m.handleNotification(newSession, settings.Race)
				}
			}
		}
	}
}

func (m *Manager) handleNotification(newSession model.ServerStarted, sessionType string) {
	receipients, err := m.lister.ListUsersForSessionStarted(sessionType)
	log.Printf("Sending notification for %s -> %s to %d telegram users\n", newSession.ServerName, sessionType, len(receipients))
	if err != nil {
		log.Printf("Error listing users for session started: %s", err.Error())
		return
	}
	err = m.sendNotification(receipients, newSession)
	if err != nil {
		log.Printf("Error notifying users: %s", err.Error())
	}
}

func (m *Manager) sendNotification(tusers []settings.TelegramUser, newSession model.ServerStarted) error {
	if len(tusers) == 0 {
		return nil
	}

	tg := Telegram{}
	tg.SetClient(m.bot)

	for _, tuser := range tusers {
		chatId, _ := strconv.ParseInt(tuser.ChatID, 0, 64)
		tg.AddReceivers(chatId)
	}

	n := notify.NewWithServices(tg)

	msg := m.loc.MustLocalize(&i18n.LocalizeConfig{
		// MessageID: "notification.sessionStarted",
		DefaultMessage: &i18n.Message{
			ID:    "notification.sessionStarted",
			Other: "New session started:",
		},
	})

	err := n.Send(m.ctx, msg, newSession.String())
	if err != nil {
		return err
	}
	return nil
}

func isTestDay(sessionType string) bool {
	return sessionType == TypeTestDay
}

func isPractice(sessionType string) bool {
	return sessionType == TypePractice
}

func isQual(sessionType string) bool {
	return sessionType == TypeQual
}

func isWarmup(sessionType string) bool {
	return sessionType == TypeWarnup
}

func isRace(sessionType string) bool {
	return sessionType == TypeRace
}

func isSessionToBeNotified(sessionType string) bool {
	return isTestDay(sessionType) || isPractice(sessionType) || isQual(sessionType) || isWarmup(sessionType) || isRace(sessionType)
}
