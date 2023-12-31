package live

import (
	"context"
	"fmt"
	"sync"

	"github.com/oscar-martin/rfactor2telegrambot/pkg/apps"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/menus"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/model"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/pubsub"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/servers"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/settings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	liveAppName = "LiveTiming"
)

type LiveApp struct {
	bot                        *tgbotapi.BotAPI
	appMenu                    menus.ApplicationMenu
	menuKeyboard               tgbotapi.ReplyKeyboardMarkup
	accepters                  []apps.Accepter
	servers                    []servers.Server
	liveSessionInfoUpdateChans []<-chan model.LiveSessionInfoData
	loc                        *i18n.Localizer
	mu                         sync.Mutex
}

func NewLiveApp(ctx context.Context, bot *tgbotapi.BotAPI, ss []servers.Server, appMenu menus.ApplicationMenu, sm *settings.Manager, loc *i18n.Localizer) (*LiveApp, error) {
	liveSessionInfoUpdateChans := []<-chan model.LiveSessionInfoData{}
	for _, server := range ss {
		liveSessionInfoUpdateChans = append(liveSessionInfoUpdateChans, pubsub.LiveSessionInfoDataPubSub.Subscribe(pubsub.PubSubSessionInfoPreffix+server.ID))
	}
	la := &LiveApp{
		bot:                        bot,
		appMenu:                    appMenu,
		liveSessionInfoUpdateChans: liveSessionInfoUpdateChans,
		loc:                        loc,
		servers:                    ss,
	}

	la.accepters = []apps.Accepter{}
	for _, server := range ss {
		serverAppMenu := menus.NewApplicationMenu(server.StatusAndName(), liveAppName, la, loc)
		serverApp := NewServerApp(la.bot, serverAppMenu, server.ID, server.URL, loc)
		la.accepters = append(la.accepters, serverApp)
	}

	settingsApp := NewSettingsApp(la.bot, appMenu, sm, la.getButtonSettingsTitle(), loc)
	la.accepters = append(la.accepters, settingsApp)

	la.updateKeyboard()

	for _, liveSessionInfoUpdateChan := range la.liveSessionInfoUpdateChans {
		go la.updater(liveSessionInfoUpdateChan)
	}

	return la, nil
}

func (la *LiveApp) updateKeyboard() {
	buttons := [][]tgbotapi.KeyboardButton{}
	for idx := range la.servers {
		if idx%2 == 0 {
			buttons = append(buttons, []tgbotapi.KeyboardButton{})
		}
		buttons[len(buttons)-1] = append(buttons[len(buttons)-1], tgbotapi.NewKeyboardButton(la.servers[idx].StatusAndName()))
	}
	backButtonRow := tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(la.appMenu.ButtonBackTo()),
		tgbotapi.NewKeyboardButton(la.getButtonSettingsTitle()),
	)

	buttons = append(buttons, backButtonRow)

	menuKeyboard := tgbotapi.NewReplyKeyboard()
	menuKeyboard.Keyboard = buttons
	la.menuKeyboard = menuKeyboard
}

func (la *LiveApp) getButtonSettingsTitle() string {
	msg := la.loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "live.buttonSettings",
			Other: "Settings",
		},
	})
	return msg
}

func (la *LiveApp) update(lsid model.LiveSessionInfoData) {
	la.mu.Lock()
	defer la.mu.Unlock()
	for idx := range la.servers {
		if la.servers[idx].ID == lsid.ServerID {
			if lsid.SessionInfo.ServerName != "" {
				la.servers[idx].Name = lsid.SessionInfo.ServerName
			}
			la.servers[idx].WebSocketRunning = lsid.SessionInfo.WebSocketRunning
			la.servers[idx].ReceivingData = lsid.SessionInfo.ReceivingData
		}
	}
	la.updateKeyboard()
}

func (la *LiveApp) updater(c <-chan model.LiveSessionInfoData) {
	for lsid := range c {
		la.update(lsid)
	}
}

func (la *LiveApp) Menu() tgbotapi.ReplyKeyboardMarkup {
	la.mu.Lock()
	defer la.mu.Unlock()

	return la.menuKeyboard
}

func (la *LiveApp) AcceptCommand(command string) (bool, func(ctx context.Context, chatId int64) error) {
	for _, accepter := range la.accepters {
		accept, handler := accepter.AcceptCommand(command)
		if accept {
			return true, handler
		}
	}

	return false, nil
}

func (la *LiveApp) AcceptCallback(query *tgbotapi.CallbackQuery) (bool, func(ctx context.Context, query *tgbotapi.CallbackQuery) error) {
	for _, accepter := range la.accepters {
		accept, handler := accepter.AcceptCallback(query)
		if accept {
			return true, handler
		}
	}

	return false, nil
}

func (la *LiveApp) AcceptButton(button string) (bool, func(ctx context.Context, chatId int64) error) {
	la.mu.Lock()
	defer la.mu.Unlock()

	// fmt.Printf("LIVE: button: %s. appName: %s\n", button, la.appMenu.Name)
	if button == la.appMenu.Name {
		return true, func(ctx context.Context, chatId int64) error {
			message := fmt.Sprintf("%s\n", la.appMenu.Name)
			msg := tgbotapi.NewMessage(chatId, message)
			msg.ReplyMarkup = la.menuKeyboard
			_, err := la.bot.Send(msg)
			return err
		}
	} else if button == la.appMenu.ButtonBackTo() {
		return true, func(ctx context.Context, chatId int64) error {
			msg := tgbotapi.NewMessage(chatId, "OK")
			msg.ReplyMarkup = la.appMenu.PrevMenu()
			_, err := la.bot.Send(msg)
			return err
		}
	}
	for _, accepter := range la.accepters {
		accept, handler := accepter.AcceptButton(button)
		if accept {
			return true, handler
		}
	}
	// fmt.Print("LIVE: FALSE\n")
	return false, nil

}
