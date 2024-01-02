package mainapp

import (
	"context"
	"f1champshotlapsbot/pkg/apps"
	"f1champshotlapsbot/pkg/apps/live"
	"f1champshotlapsbot/pkg/menus"
	"f1champshotlapsbot/pkg/servers"
	"f1champshotlapsbot/pkg/settings"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	menuStart  = "/start"
	menuMenu   = "/menu"
	appName    = "menu"
	buttonLive = "Live"
)

var (
	menuKeyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonLive),
		),
	)
)

type menuer struct{}

func (m menuer) Menu() tgbotapi.ReplyKeyboardMarkup {
	return menuKeyboard
}

type MainApp struct {
	bot       *tgbotapi.BotAPI
	accepters []apps.Accepter
	loc       *i18n.Localizer
}

func NewMainApp(ctx context.Context, bot *tgbotapi.BotAPI, ss []servers.Server, exitChan chan bool, sm *settings.Manager, loc *i18n.Localizer) (*MainApp, error) {
	liveAppMenu := menus.NewApplicationMenu(buttonLive, appName, menuer{}, loc)
	liveApp, err := live.NewLiveApp(ctx, bot, ss, liveAppMenu, sm, loc)
	if err != nil {
		return nil, err
	}

	accepters := []apps.Accepter{liveApp}

	return &MainApp{
		bot:       bot,
		loc:       loc,
		accepters: accepters,
	}, nil
}

func (m *MainApp) AcceptCommand(command string) (bool, func(ctx context.Context, chatId int64) error) {
	if command == menuStart {
		return true, m.renderStart()
	} else if command == menuMenu {
		return true, m.renderMenu()
	}
	for _, accepter := range m.accepters {
		accept, handler := accepter.AcceptCommand(command)
		if accept {
			return true, handler
		}
	}

	return false, nil
}

func (m *MainApp) AcceptCallback(query *tgbotapi.CallbackQuery) (bool, func(ctx context.Context, query *tgbotapi.CallbackQuery) error) {
	for _, accepter := range m.accepters {
		accept, handler := accepter.AcceptCallback(query)
		if accept {
			return true, handler
		}
	}

	return false, nil
}

func (m *MainApp) AcceptButton(button string) (bool, func(ctx context.Context, chatId int64) error) {
	for _, accepter := range m.accepters {
		accept, handler := accepter.AcceptButton(button)
		if accept {
			return true, handler
		}
	}
	return false, nil
}

func (m *MainApp) renderStart() func(ctx context.Context, chatId int64) error {
	return func(ctx context.Context, chatId int64) error {
		msg1 := m.loc.MustLocalize(&i18n.LocalizeConfig{
			// MessageID: "mainapp.helloBot1",
			DefaultMessage: &i18n.Message{
				ID:    "mainapp.helloBot1",
				Other: "Hello, I am a bot that allows you to get information about ongoing sessions.",
			},
		})

		msg2 := m.loc.MustLocalize(&i18n.LocalizeConfig{
			// MessageID: "mainapp.helloBot2",
			DefaultMessage: &i18n.Message{
				ID:    "mainapp.helloBot2",
				Other: "You can use the following command:",
			},
		})

		msgStartMenu := m.loc.MustLocalize(&i18n.LocalizeConfig{
			// MessageID: "mainapp.startMenu",
			DefaultMessage: &i18n.Message{
				ID:    "mainapp.startMenu",
				Other: "Show the bot menu",
			},
		})

		message := fmt.Sprintf("%s\n\n", msg1) + fmt.Sprintf("%s\n\n", msg2) + fmt.Sprintf("%s - %s\n", menuMenu, msgStartMenu)
		msg := tgbotapi.NewMessage(chatId, message)
		msg.ReplyMarkup = menuKeyboard
		_, err := m.bot.Send(msg)
		return err
	}
}

func (m *MainApp) renderMenu() func(ctx context.Context, chatId int64) error {
	return func(ctx context.Context, chatId int64) error {
		msgMenuMenu := m.loc.MustLocalize(&i18n.LocalizeConfig{
			// MessageID: "mainapp.menuMenu",
			DefaultMessage: &i18n.Message{
				ID:    "mainapp.menuMenu",
				Other: "Bot menu.",
			},
		})
		message := fmt.Sprintf("%s\n\n", msgMenuMenu)
		msg := tgbotapi.NewMessage(chatId, message)
		msg.ReplyMarkup = menuKeyboard
		_, err := m.bot.Send(msg)
		return err
	}
}
