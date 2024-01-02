package live

import (
	"bytes"
	"context"
	"f1champshotlapsbot/pkg/helper"
	"f1champshotlapsbot/pkg/menus"
	"f1champshotlapsbot/pkg/model"
	"f1champshotlapsbot/pkg/pubsub"
	"f1champshotlapsbot/pkg/resources"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	subcommandShowDrivers = "show_drivers"
	subcommandShowCars    = "show_cars"
)

type StintApp struct {
	bot                               *tgbotapi.BotAPI
	appMenu                           menus.ApplicationMenu
	serverID                          string
	serverURL                         string
	liveStandingHistoryData           model.LiveStandingHistoryData
	liveStandingHistoryDataUpdateChan <-chan model.LiveStandingHistoryData
	appName                           string

	liveSessionInfoData           model.LiveSessionInfoData
	liveSessionInfoDataUpdateChan <-chan model.LiveSessionInfoData

	loc *i18n.Localizer

	mu sync.Mutex
}

func NewStintApp(bot *tgbotapi.BotAPI, appMenu menus.ApplicationMenu, serverID, serverURL string, appName string, loc *i18n.Localizer) *StintApp {
	sa := &StintApp{
		bot:                               bot,
		appMenu:                           appMenu,
		serverID:                          serverID,
		serverURL:                         serverURL,
		loc:                               loc,
		appName:                           appName,
		liveStandingHistoryDataUpdateChan: pubsub.LiveStandingHistoryPubSub.Subscribe(pubsub.PubSubStintDataPreffix + serverID),
		liveSessionInfoDataUpdateChan:     pubsub.LiveSessionInfoDataPubSub.Subscribe(pubsub.PubSubSessionInfoPreffix + serverID),
	}

	go sa.liveStandingHistoryDataUpdater()
	go sa.liveSessionInfoDataUpdater()

	return sa
}

func (sa *StintApp) liveStandingHistoryDataUpdater() {
	for lsd := range sa.liveStandingHistoryDataUpdateChan {
		sa.update(lsd, sa.liveSessionInfoData)
	}
}

func (sa *StintApp) liveSessionInfoDataUpdater() {
	for lsi := range sa.liveSessionInfoDataUpdateChan {
		sa.update(sa.liveStandingHistoryData, lsi)
	}
}

func (sa *StintApp) update(lsd model.LiveStandingHistoryData, lsi model.LiveSessionInfoData) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.liveStandingHistoryData = lsd
	sa.liveSessionInfoData = lsi
}

func (sa *StintApp) AcceptCommand(command string) (bool, func(ctx context.Context, chatId int64) error) {
	return false, nil
}

func (sa *StintApp) AcceptCallback(query *tgbotapi.CallbackQuery) (bool, func(ctx context.Context, query *tgbotapi.CallbackQuery) error) {
	data := strings.Split(query.Data, ":")
	if data[0] == subcommandShowDrivers && data[1] == sa.serverID {
		sa.mu.Lock()
		defer sa.mu.Unlock()
		return true, func(ctx context.Context, query *tgbotapi.CallbackQuery) error {
			return sa.handleStintDataCallbackQuery(query.Message.Chat.ID, &query.Message.MessageID, data[2:]...)
		}
	} else if data[0] == subcommandShowCars && data[1] == sa.serverID {
		sa.mu.Lock()
		defer sa.mu.Unlock()
		return true, func(ctx context.Context, query *tgbotapi.CallbackQuery) error {
			return sa.handleCarDataCallbackQuery(query.Message.Chat.ID, &query.Message.MessageID, data[2])
		}
	}
	return false, nil
}

func (sa *StintApp) AcceptButton(button string) (bool, func(ctx context.Context, chatId int64) error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// fmt.Printf("STINT: button: %s. appName: %s\n", button, buttonStint+" "+sa.stintData.ServerName)
	if button == sa.appName+" "+sa.liveStandingHistoryData.ServerName {
		return true, sa.renderDrivers()
	} else if button == sa.appMenu.ButtonBackTo() {
		return true, func(ctx context.Context, chatId int64) error {
			msg := tgbotapi.NewMessage(chatId, "OK")
			msg.ReplyMarkup = sa.appMenu.PrevMenu()
			_, err := sa.bot.Send(msg)
			return err
		}
	}
	// fmt.Print("STINT: FALSE\n")
	return false, nil
}

func (sa *StintApp) renderDrivers() func(ctx context.Context, chatId int64) error {
	return func(ctx context.Context, chatId int64) error {
		if len(sa.liveStandingHistoryData.DriverNames) > 0 {
			err := sa.sendDriversData(chatId, nil)
			if err != nil {
				return err
			}
		} else {
			message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "stint.noDriversInSession",
					Other: "There are no drivers in the session",
				},
			})

			msg := tgbotapi.NewMessage(chatId, message)
			_, err := sa.bot.Send(msg)
			return err
		}
		return nil
	}
}

func (sa *StintApp) handleStintDataCallbackQuery(chatId int64, messageId *int, data ...string) error {
	infoType := data[0]
	driver := data[1]
	driverData, found := sa.liveStandingHistoryData.DriversData[driver]
	if found {
		err := sa.sendStintData(chatId, messageId, driverData, driver, sa.liveStandingHistoryData.ServerName, sa.liveStandingHistoryData.ServerID, infoType)
		if err != nil {
			log.Printf("An error occured: %s", err.Error())
		}
	} else {
		text := sa.loc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "stint.noDataForDriver",
				Other: "No data for driver %s",
			},
		})

		message := fmt.Sprintf(text, driver)
		msg := tgbotapi.NewMessage(chatId, message)
		_, err := sa.bot.Send(msg)
		return err
	}
	return nil
}

func (sa *StintApp) handleCarDataCallbackQuery(chatId int64, messageId *int, driver string) error {
	driverData, found := sa.liveStandingHistoryData.DriversData[driver]
	if found && len(driverData) > 0 {
		if driverData[0].CarId != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			carThChan := make(chan resources.Resource)
			errChan := make(chan error)
			go func() {
				carTh, err := resources.BuildCarThumbnail(ctx, sa.serverURL, driverData[0].CarId)
				if err != nil {
					errChan <- err
					return
				}
				carThChan <- carTh
			}()
			select {
			case <-ctx.Done():
				text := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "stint.timeoutDownloadingCarImage",
						Other: "The waiting time for downloading the car image for %s has expired",
					},
				})

				message := fmt.Sprintf(text, driver)
				msg := tgbotapi.NewMessage(chatId, message)
				_, err := sa.bot.Send(msg)
				return err
			case err := <-errChan:
				text := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "stint.couldNotReadCarImage",
						Other: "Could not read the image of the car %s: %v",
					},
				})

				message := fmt.Sprintf(text, driver, err)
				msg := tgbotapi.NewMessage(chatId, message)
				_, err = sa.bot.Send(msg)
				return err
			case carTh := <-carThChan:
				filePath := carTh.FilePath()
				carText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "stint.car",
						Other: "Car",
					},
				})
				classText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "stint.class",
						Other: "Class",
					},
				})
				driverText := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "stint.driver",
						Other: "Driver",
					},
				})

				text := fmt.Sprintf(`‣ %s: %s
‣ %s: %s
‣ %s: %s`,
					carText,
					driverData[0].VehicleName,
					classText,
					driverData[0].CarClass,
					driverText,
					driverData[0].DriverName)
				msg := tgbotapi.NewPhoto(chatId, tgbotapi.FilePath(filePath))
				msg.Caption = text
				_, err := sa.bot.Send(msg)
				return err
			}
		} else {
			text := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "stint.noDataForDriver",
					Other: "No data for driver %s",
				},
			})
			message := fmt.Sprintf(text, driver)
			msg := tgbotapi.NewMessage(chatId, message)
			_, err := sa.bot.Send(msg)
			return err
		}
	} else {
		text := sa.loc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "stint.noDataForDriver",
				Other: "No data for driver %s",
			},
		})

		message := fmt.Sprintf(text, driver)
		msg := tgbotapi.NewMessage(chatId, message)
		_, err := sa.bot.Send(msg)
		return err
	}
}

func (sa *StintApp) sendStintData(chatId int64, messageId *int, driverData []model.StandingHistoryDriverData, driverName, serverName, serverId, infoType string) error {
	if len(driverData) > 0 {
		var b bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&b)
		style := table.StyleRounded
		style.Options.DrawBorder = false
		t.SetStyle(style)
		t.AppendSeparator()
		switch infoType {
		case getInlineKeyboardTimes(sa.loc):
			t.AppendHeader(table.Row{getLapHeader(sa.loc), infoType, getTopSpeedHeader(sa.loc)})
		case getInlineKeyboardSectors(sa.loc):
			t.AppendHeader(table.Row{getLapHeader(sa.loc), infoType})
		}
		for idx, lapData := range driverData {
			switch infoType {
			case getInlineKeyboardTimes(sa.loc):
				topSpeed := "-"
				if lapData.TopSpeed > 0 && lapData.LapTime > 0 {
					topSpeed = fmt.Sprintf("%.1f km/h", lapData.TopSpeed)
				}

				t.AppendRow([]interface{}{
					fmt.Sprintf("%d", idx+1),
					helper.SecondsToMinutes(lapData.LapTime),
					topSpeed,
				})
			case getInlineKeyboardSectors(sa.loc):
				ls1 := lapData.SectorTime1
				ls2 := -1.0
				if ls1 > 0.0 && lapData.SectorTime2 > 0.0 {
					ls2 = lapData.SectorTime2 - ls1
				}
				ls3 := -1.0
				if ls2 > 0.0 && lapData.LapTime > 0.0 {
					ls3 = lapData.LapTime - ls2 - ls1
				}

				t.AppendRow([]interface{}{
					fmt.Sprintf("%d", idx+1),
					fmt.Sprintf("%s %s %s", helper.ToSectorTime(ls1), helper.ToSectorTime(ls2), helper.ToSectorTime(ls3)),
				})
			}
		}
		t.Render()

		keyboard := getStintInlineKeyboard(driverName, serverId, sa.loc)
		var cfg tgbotapi.Chattable
		remainingTime := helper.SecondsToHoursAndMinutes(sa.liveSessionInfoData.SessionInfo.EndEventTime - sa.liveSessionInfoData.SessionInfo.CurrentEventTime)

		message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "stint.sessionData",
				Other: "```\nTime left: %s\nData for %s in %q\n\n%s```",
			},
		})

		text := fmt.Sprintf(message, remainingTime, driverName, serverName, b.String())
		if messageId == nil {
			msg := tgbotapi.NewMessage(chatId, text)
			msg.ParseMode = tgbotapi.ModeMarkdownV2
			msg.ReplyMarkup = keyboard
			cfg = msg
		} else {
			msg := tgbotapi.NewEditMessageText(chatId, *messageId, text)
			msg.ParseMode = tgbotapi.ModeMarkdownV2
			msg.ReplyMarkup = &keyboard
			cfg = msg
		}
		_, err := sa.bot.Send(cfg)
		return err
	} else {
		message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "stint.noLapsInSession",
				Other: "There are no laps in the session",
			},
		})

		msg := tgbotapi.NewMessage(chatId, message)
		_, err := sa.bot.Send(msg)
		return err
	}
}

func getStintInlineKeyboard(driver, serverID string, loc *i18n.Localizer) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardTimes(loc)+" "+symbolTimes, fmt.Sprintf("%s:%s:%s:%s", subcommandShowDrivers, serverID, getInlineKeyboardTimes(loc), driver)),
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardSectors(loc)+" "+symbolSectors, fmt.Sprintf("%s:%s:%s:%s", subcommandShowDrivers, serverID, getInlineKeyboardSectors(loc), driver)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardCar(loc)+" "+symbolPhoto, fmt.Sprintf("%s:%s:%s", subcommandShowCars, serverID, driver)),
		),
	)
}

func (sa *StintApp) sendDriversData(chatId int64, messageId *int) error {
	text, keyboard := sa.driversTextMarkup()

	var cfg tgbotapi.Chattable
	if messageId == nil {
		msg := tgbotapi.NewMessage(chatId, text)
		msg.ReplyMarkup = keyboard
		cfg = msg
	} else {
		msg := tgbotapi.NewEditMessageText(chatId, *messageId, text)
		msg.ReplyMarkup = &keyboard
		cfg = msg
	}

	_, err := sa.bot.Send(cfg)
	return err
}

func (sa *StintApp) driversTextMarkup() (text string, markup tgbotapi.InlineKeyboardMarkup) {
	buttons := [][]tgbotapi.InlineKeyboardButton{}

	for idx, driver := range sa.liveStandingHistoryData.DriverNames {
		if idx%2 == 0 {
			buttons = append(buttons, []tgbotapi.InlineKeyboardButton{})
		}
		buttons[len(buttons)-1] = append(buttons[len(buttons)-1], tgbotapi.NewInlineKeyboardButtonData(driver, fmt.Sprintf("%s:%s:%s:%s", subcommandShowDrivers, sa.liveStandingHistoryData.ServerID, getInlineKeyboardTimes(sa.loc), driver)))
	}

	msg := sa.loc.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    "stint.chooseDriverFromList",
			Other: "Choose the driver from the list:",
		},
	})

	text = fmt.Sprintf("%s\n\n", msg)

	markup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	return
}
