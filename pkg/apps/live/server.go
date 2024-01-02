package live

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/oscar-martin/rfactor2telegrambot/pkg/apps"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/helper"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/menus"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/model"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/pubsub"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/resources"
	"github.com/oscar-martin/rfactor2telegrambot/pkg/servers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	subcommandShowLiveMap = "show_live_map"
)

type ServerApp struct {
	bot                           *tgbotapi.BotAPI
	appMenu                       menus.ApplicationMenu
	menuKeyboard                  tgbotapi.ReplyKeyboardMarkup
	gridApp                       *GridApp
	stintApp                      *StintApp
	accepters                     []apps.Accepter
	serverID                      string
	liveSessionInfoData           model.LiveSessionInfoData
	liveSessionInfoDataUpdateChan <-chan model.LiveSessionInfoData

	trackThumbnailData           resources.Resource
	trackThumbnailDataUpdateChan <-chan resources.Resource

	loc *i18n.Localizer

	mu sync.Mutex
}

func sanitizeServerName(name string) string {
	fixed := strings.TrimPrefix(name, servers.ServerStatusOnline)
	fixed = strings.TrimPrefix(fixed, servers.ServerStatusOffline)
	fixed = strings.TrimPrefix(fixed, servers.ServerStatusOnlineButNotData)
	return strings.TrimSpace(fixed)
}

func NewServerApp(bot *tgbotapi.BotAPI, appMenu menus.ApplicationMenu, serverID, serverURL string, loc *i18n.Localizer) *ServerApp {
	sa := &ServerApp{
		bot:                           bot,
		appMenu:                       appMenu,
		serverID:                      serverID,
		loc:                           loc,
		liveSessionInfoDataUpdateChan: pubsub.LiveSessionInfoDataPubSub.Subscribe(pubsub.PubSubSessionInfoPreffix + serverID),
		trackThumbnailDataUpdateChan:  pubsub.TrackThumbnailPubSub.Subscribe(pubsub.PubSubThumbnailPreffix + serverID),
	}

	go sa.liveSessionInfoUpdater()
	go sa.trackThumbnailUpdater()

	gridAppMenu := menus.NewApplicationMenu("", serverID, sa, loc)
	gridApp := NewGridApp(bot, gridAppMenu, serverID, sa.getButtonGridTitle(), loc)

	stintAppMenu := menus.NewApplicationMenu("", serverID, sa, loc)
	stintApp := NewStintApp(bot, stintAppMenu, serverID, serverURL, sa.getButtonStintTitle(), loc)

	accepters := []apps.Accepter{gridApp, stintApp}

	sa.accepters = accepters
	sa.gridApp = gridApp
	sa.stintApp = stintApp
	return sa
}

func (sa *ServerApp) update(lsid model.LiveSessionInfoData, t resources.Resource) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	stint := sa.getButtonStintTitle() + " " + lsid.ServerName
	grid := sa.getButtonGridTitle() + " " + lsid.ServerName
	info := sa.getButtonInfoTitle() + " " + lsid.ServerName

	menuKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(stint),
			tgbotapi.NewKeyboardButton(grid),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(sa.appMenu.ButtonBackTo()),
			tgbotapi.NewKeyboardButton(info),
		),
	)

	sa.menuKeyboard = menuKeyboard
	sa.liveSessionInfoData = lsid
	sa.trackThumbnailData = t
}

func (sa *ServerApp) getButtonStintTitle() string {
	msg := sa.loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "serverapp.buttonStint",
			Other: "Stint",
		},
	})
	return msg
}

func (sa *ServerApp) getButtonGridTitle() string {
	msg := sa.loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "serverapp.buttonGrid",
			Other: "Grid",
		},
	})
	return msg
}

func (sa *ServerApp) getButtonInfoTitle() string {
	msg := sa.loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "serverapp.buttonInfo",
			Other: "Info",
		},
	})
	return msg
}

func (sa *ServerApp) liveSessionInfoUpdater() {
	for si := range sa.liveSessionInfoDataUpdateChan {
		sa.update(si, sa.trackThumbnailData)
	}
}

func (sa *ServerApp) trackThumbnailUpdater() {
	for t := range sa.trackThumbnailDataUpdateChan {
		sa.update(sa.liveSessionInfoData, t)
	}
}

func (sa *ServerApp) Menu() tgbotapi.ReplyKeyboardMarkup {
	return sa.menuKeyboard
}

func (sa *ServerApp) AcceptCommand(command string) (bool, func(ctx context.Context, chatId int64) error) {
	for _, accepter := range sa.accepters {
		accept, handler := accepter.AcceptCommand(command)
		if accept {
			return true, handler
		}
	}

	return false, nil
}

func (sa *ServerApp) AcceptCallback(query *tgbotapi.CallbackQuery) (bool, func(ctx context.Context, query *tgbotapi.CallbackQuery) error) {
	for _, accepter := range sa.accepters {
		accept, handler := accepter.AcceptCallback(query)
		if accept {
			return true, handler
		}
	}

	return false, nil
}

func (sa *ServerApp) AcceptButton(button string) (bool, func(ctx context.Context, chatId int64) error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// fmt.Printf("SERVER: button: %s. appName: %s\n", button, sa.sessionInfo.ServerName)
	if sanitizeServerName(button) == sa.liveSessionInfoData.ServerName ||
		sanitizeServerName(button) == sa.getButtonInfoTitle()+" "+sa.liveSessionInfoData.ServerName {
		return true, func(ctx context.Context, chatId int64) error {
			if !sa.liveSessionInfoData.SessionInfo.WebSocketRunning {
				message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "server.serverIsOffline",
						Other: "Server %s is offline",
					},
				})

				msg := tgbotapi.NewMessage(chatId, fmt.Sprintf(message, sa.liveSessionInfoData.ServerName))
				msg.ReplyMarkup = sa.appMenu.PrevMenu()
				_, err := sa.bot.Send(msg)
				return err
			} else if !sa.liveSessionInfoData.SessionInfo.ReceivingData {
				message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "server.noDataReceived",
						Other: "No data received from server %s",
					},
				})

				msg := tgbotapi.NewMessage(chatId, fmt.Sprintf(message, sa.liveSessionInfoData.ServerName))
				msg.ReplyMarkup = sa.appMenu.PrevMenu()
				_, err := sa.bot.Send(msg)
				return err
			}
			si := sa.liveSessionInfoData.SessionInfo
			laps := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.notLimited",
					Other: "Not Limited",
				},
			})
			if si.MaximumLaps < 100 {
				laps = fmt.Sprintf("%d", si.MaximumLaps)
			}
			trackText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.track",
					Other: "Track",
				},
			})
			timeLeftText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.timeLeft",
					Other: "Time left",
				},
			})
			sessionText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.session",
					Other: "Session",
				},
			})
			lapsText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.laps",
					Other: "Laps",
				},
			})
			carsInSessionText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.carsInSession",
					Other: "Cars in session",
				},
			})
			rainText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.rain",
					Other: "Rain",
				},
			})

			tempText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "server.temp",
					Other: "Temperature (Track/Ambient)",
				},
			})

			text := fmt.Sprintf(`%s:
			‣ %s: %s (%0.fm)
			‣ %s: %s
			‣ %s: %s (%s: %s)
			‣ %s: %d
			‣ %s: %.1f%% (min: %.1f%%. max: %.1f%%)
			‣ %s: %0.fºC/%0.fºC
			`,
				sa.liveSessionInfoData.ServerName,
				trackText,
				si.TrackName,
				si.LapDistance,
				timeLeftText,
				helper.SecondsToHoursAndMinutes(si.EndEventTime-si.CurrentEventTime),
				sessionText,
				si.Session,
				lapsText,
				laps,
				carsInSessionText,
				si.NumberOfVehicles,
				rainText,
				si.Raining,
				si.MinPathWetness,
				si.MaxPathWetness,
				tempText,
				si.TrackTemp,
				si.AmbientTemp)
			err := fmt.Errorf("No track thumbnail available")
			var filePath string
			if !sa.trackThumbnailData.IsZero() {
				filePath = sa.trackThumbnailData.FilePath()
				err = nil
			}
			var cfg tgbotapi.Chattable
			if err != nil {
				log.Printf("Error getting thumbnail data: %s\n", err.Error())
				msg := tgbotapi.NewMessage(chatId, text)
				msg.ReplyMarkup = sa.menuKeyboard
				cfg = msg
			} else {
				msg := tgbotapi.NewPhoto(chatId, tgbotapi.FilePath(filePath))
				msg.Caption = text
				msg.ReplyMarkup = sa.menuKeyboard
				cfg = msg
			}
			_, err = sa.bot.Send(cfg)
			return err
		}
	} else if button == sa.appMenu.ButtonBackTo() {
		return true, func(ctx context.Context, chatId int64) error {
			msg := tgbotapi.NewMessage(chatId, "OK")
			msg.ReplyMarkup = sa.appMenu.PrevMenu()
			_, err := sa.bot.Send(msg)
			return err
		}
	} else {
		for _, accepter := range sa.accepters {
			accept, handler := accepter.AcceptButton(button)
			if accept {
				return true, handler
			}
		}
		// fmt.Print("SERVER: FALSE\n")
		return false, nil
	}
}
