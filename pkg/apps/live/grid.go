package live

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"rfactor2telegrambot/pkg/helper"
	"rfactor2telegrambot/pkg/menus"
	"rfactor2telegrambot/pkg/model"
	"rfactor2telegrambot/pkg/pubsub"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	subcommandShowLiveTiming = "show_live_timing"
)

type GridApp struct {
	bot                        *tgbotapi.BotAPI
	appMenu                    menus.ApplicationMenu
	serverID                   string
	liveStandingData           model.LiveStandingData
	liveStandingDataUpdateChan <-chan model.LiveStandingData
	appName                    string

	liveSessionInfoData           model.LiveSessionInfoData
	liveSessionInfoDataUpdateChan <-chan model.LiveSessionInfoData

	loc *i18n.Localizer

	mu sync.Mutex
}

func NewGridApp(bot *tgbotapi.BotAPI, appMenu menus.ApplicationMenu, serverID string, appName string, loc *i18n.Localizer) *GridApp {
	ga := &GridApp{
		bot:                           bot,
		appMenu:                       appMenu,
		serverID:                      serverID,
		loc:                           loc,
		appName:                       appName,
		liveStandingDataUpdateChan:    pubsub.LiveStandingDataPubSub.Subscribe(pubsub.PubSubDriversSessionPreffix + serverID),
		liveSessionInfoDataUpdateChan: pubsub.LiveSessionInfoDataPubSub.Subscribe(pubsub.PubSubSessionInfoPreffix + serverID),
	}

	go ga.liveStandingDataUpdater()
	go ga.liveSessionInfoDataUpdater()

	return ga
}

func (ga *GridApp) liveStandingDataUpdater() {
	for dss := range ga.liveStandingDataUpdateChan {
		ga.update(dss, ga.liveSessionInfoData)
	}
}

func (ga *GridApp) liveSessionInfoDataUpdater() {
	for lsi := range ga.liveSessionInfoDataUpdateChan {
		ga.update(ga.liveStandingData, lsi)
	}
}

func (ga *GridApp) update(lsd model.LiveStandingData, lsi model.LiveSessionInfoData) {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	ga.liveStandingData = lsd
	ga.liveSessionInfoData = lsi
}

func (ga *GridApp) AcceptCommand(command string) (bool, func(ctx context.Context, chatId int64) error) {
	return false, nil
}

func (ga *GridApp) AcceptCallback(query *tgbotapi.CallbackQuery) (bool, func(ctx context.Context, query *tgbotapi.CallbackQuery) error) {
	data := strings.Split(query.Data, ":")
	if data[0] == subcommandShowLiveTiming && data[1] == ga.serverID {
		ga.mu.Lock()
		defer ga.mu.Unlock()
		return true, func(ctx context.Context, query *tgbotapi.CallbackQuery) error {
			return ga.handleSessionDataCallbackQuery(query.Message.Chat.ID, &query.Message.MessageID, data[2:]...)
		}
	}
	return false, nil
}

func (ga *GridApp) AcceptButton(button string) (bool, func(ctx context.Context, chatId int64) error) {
	ga.mu.Lock()
	defer ga.mu.Unlock()

	// fmt.Printf("GRID: button: %s. appName: %s\n", button, buttonGrid+" "+ga.driversSession.ServerName)
	if button == ga.appName+" "+ga.liveStandingData.ServerName {
		return true, ga.renderGrid()
	} else if button == ga.appMenu.ButtonBackTo() {
		return true, func(ctx context.Context, chatId int64) error {
			msg := tgbotapi.NewMessage(chatId, "OK")
			msg.ReplyMarkup = ga.appMenu.PrevMenu()
			_, err := ga.bot.Send(msg)
			return err
		}
	}
	// fmt.Print("GRID: FALSE\n")
	return false, nil
}

func (ga *GridApp) renderGrid() func(ctx context.Context, chatId int64) error {
	return func(ctx context.Context, chatId int64) error {
		err := ga.sendSessionData(chatId, nil, ga.liveStandingData, getInlineKeyboardBestLap(ga.loc))
		if err != nil {
			log.Printf("An error occured: %s", err.Error())
		}
		return nil
	}
}

func (ga *GridApp) handleSessionDataCallbackQuery(chatId int64, messageId *int, data ...string) error {
	infoType := data[0]
	return ga.sendSessionData(chatId, messageId, ga.liveStandingData, infoType)
}

func (ga *GridApp) sendSessionData(chatId int64, messageId *int, driversSession model.LiveStandingData, infoType string) error {
	if len(driversSession.Drivers) > 0 {
		var b bytes.Buffer
		t := table.NewWriter()
		t.SetOutputMirror(&b)
		style := table.StyleRounded
		style.Options.DrawBorder = false
		t.SetStyle(style)
		t.AppendSeparator()

		switch infoType {
		case getInlineKeyboardStatus(ga.loc):
			t.AppendHeader(table.Row{getDriverHeader(ga.loc), getSectorsHeader(ga.loc), "S" /*, "FUEL"*/})
		case getInlineKeyboardInfo(ga.loc):
			t.AppendHeader(table.Row{getDriverHeader(ga.loc), getNameHeader(ga.loc) /*, "Núm"*/, getLapHeader(ga.loc)})
		case getInlineKeyboardLastLap(ga.loc):
			t.AppendHeader(table.Row{getDriverHeader(ga.loc), getLastHeader(ga.loc), getBestHeader(ga.loc)})
		case getInlineKeyboardOptimumLap(ga.loc):
			t.AppendHeader(table.Row{getDriverHeader(ga.loc), getOptimalHeader(ga.loc), getBestHeader(ga.loc)})
		case getInlineKeyboardBestLap(ga.loc):
			t.AppendHeader(table.Row{getDriverHeader(ga.loc), getBestHeader(ga.loc), getTopSpeedHeader(ga.loc)})
		default:
			t.AppendHeader(table.Row{getDriverHeader(ga.loc), infoType})
		}
		for idx, driverStat := range driversSession.Drivers {
			switch infoType {
			case getInlineKeyboardStatus(ga.loc):
				// state := "🟢"
				state := ""
				if driverStat.InGarageStall {
					state = "P"
					// state = "🔴"
				} else if driverStat.Pitting {
					state = "P"
					// state = "🟡"
				}
				var s1 float64
				s2 := -1.0
				s3 := -1.0
				if driverStat.CurrentSectorTime1 > 0.0 {
					// s1 is done in current lap
					s1 = driverStat.CurrentSectorTime1
					if s1 > 0.0 && driverStat.CurrentSectorTime2 > 0.0 {
						// s2 is done in current lap
						s2 = driverStat.CurrentSectorTime2 - s1
					}
				} else {
					s1 = driverStat.LastSectorTime1
					if s1 > 0.0 && driverStat.LastSectorTime2 > 0.0 {
						s2 = driverStat.LastSectorTime2 - s1
					}
					if s2 > 0.0 && driverStat.LastLapTime > 0.0 {
						s3 = driverStat.LastLapTime - s2 - s1
					}
				}
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					fmt.Sprintf("%s %s %s", helper.ToSectorTime(s1), helper.ToSectorTime(s2), helper.ToSectorTime(s3)),
					// fmt.Sprintf("%.0f%%", driverStat.FuelFraction*100),
					state,
				})
			case getInlineKeyboardInfo(ga.loc):
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					driverStat.DriverName,
					// driverStat.CarNumber,
					driverStat.LapsCompleted,
				})
			case getInlineKeyboardDiff(ga.loc):
				diff := ""
				if idx == 0 {
					diff = helper.SecondsToMinutes(driverStat.BestLapTime)
				} else {
					diff = helper.SecondsToDiff(driverStat.BestLapTime - driversSession.Drivers[0].BestLapTime)
				}
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					diff,
				})
			case getInlineKeyboardBestLap(ga.loc):
				topSpeed := "-"
				if driverStat.BestLap > 0 {
					kph, found := driverStat.TopSpeedPerLap[driverStat.BestLap]
					if found {
						topSpeed = fmt.Sprintf("%.1f km/h", kph)
					}
				}
				// fmt.Printf("Driver: %s\n   BestLap: %d\n   Data: %+v\n   Top Speed: %s\n", driverStat.DriverName, driverStat.BestLap, driverStat.TopSpeedPerLap, topSpeed)
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					helper.SecondsToMinutes(driverStat.BestLapTime),
					topSpeed,
				})
			case getInlineKeyboardLastLap(ga.loc):
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					helper.SecondsToMinutes(driverStat.LastLapTime),
					helper.SecondsToMinutes(driverStat.BestLapTime),
				})
			case getInlineKeyboardOptimumLap(ga.loc):
				optimumLap := -1.0
				if driverStat.BestSectorTime1 > 0.0 && driverStat.BestSectorTime2 > 0.0 && driverStat.BestSectorTime3 > 0.0 {
					optimumLap = driverStat.BestSectorTime1 + driverStat.BestSectorTime2 + driverStat.BestSectorTime3
				}
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					helper.SecondsToMinutes(optimumLap),
					helper.SecondsToMinutes(driverStat.BestLapTime),
				})
			case getInlineKeyboardOptimumLapSectors(ga.loc):
				ls1 := driverStat.BestSectorTime1
				ls2 := driverStat.BestSectorTime2
				ls3 := driverStat.BestSectorTime3
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					fmt.Sprintf("%s %s %s", helper.ToSectorTime(ls1), helper.ToSectorTime(ls2), helper.ToSectorTime(ls3)),
				})
			case getInlineKeyboardBestLapSectors(ga.loc):
				bs1 := driverStat.BestLapSectorTime1
				bs2 := -1.0
				if bs1 > 0.0 {
					bs2 = driverStat.BestLapSectorTime2 - bs1
				}
				bs3 := -1.0
				if bs2 > 0.0 && driverStat.BestLapTime > 0.0 {
					bs3 = driverStat.BestLapTime - bs2 - bs1
				}
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					fmt.Sprintf("%s %s %s", helper.ToSectorTime(bs1), helper.ToSectorTime(bs2), helper.ToSectorTime(bs3)),
				})
			case getInlineKeyboardLastLapSectors(ga.loc):
				ls1 := driverStat.LastSectorTime1
				ls2 := -1.0
				if ls1 > 0.0 && driverStat.LastSectorTime2 > 0.0 {
					ls2 = driverStat.LastSectorTime2 - ls1
				}
				ls3 := -1.0
				if ls2 > 0.0 && driverStat.LastLapTime > 0.0 {
					ls3 = driverStat.LastLapTime - ls2 - ls1
				}
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					fmt.Sprintf("%s %s %s", helper.ToSectorTime(ls1), helper.ToSectorTime(ls2), helper.ToSectorTime(ls3)),
				})
			case getInlineKeyboardLaps(ga.loc):
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					fmt.Sprintf("%d", driverStat.LapsCompleted),
				})
			case getInlineKeyboardTeam(ga.loc):
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					driverStat.CarClass,
				})
			case getInlineKeyboardDriver(ga.loc):
				t.AppendRow([]interface{}{
					helper.GetDriverCodeName(driverStat.DriverName),
					driverStat.DriverName,
				})
			}
		}
		t.Render()

		keyboard := getGridInlineKeyboard(driversSession.ServerID, fmt.Sprintf("%s%s/live", ga.liveSessionInfoData.SessionInfo.LiveMapDomain, ga.liveSessionInfoData.SessionInfo.LiveMapPath), ga.loc)
		var cfg tgbotapi.Chattable
		remainingTime := helper.SecondsToHoursAndMinutes(ga.liveSessionInfoData.SessionInfo.EndEventTime - ga.liveSessionInfoData.SessionInfo.CurrentEventTime)
		text := fmt.Sprintf("```\nTime left: %s\nServer: %q\n\n%s```", remainingTime, driversSession.ServerName, b.String())
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
		_, err := ga.bot.Send(cfg)
		return err
	} else {
		message := "There are no drivers in the session"
		msg := tgbotapi.NewMessage(chatId, message)
		_, err := ga.bot.Send(msg)
		return err
	}
}

func getGridInlineKeyboard(serverID, liveMapUrl string, loc *i18n.Localizer) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardBestLap(loc)+" "+symbolTimes, fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardBestLap(loc))),
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardBestLapSectors(loc), fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardBestLapSectors(loc))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardLastLap(loc)+" "+symbolTimes, fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardLastLap(loc))),
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardLastLapSectors(loc), fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardLastLapSectors(loc))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardOptimumLap(loc)+" "+symbolTimes, fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardOptimumLap(loc))),
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardOptimumLapSectors(loc), fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardOptimumLapSectors(loc))),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardStatus(loc), fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardStatus(loc))),
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardInfo(loc), fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardInfo(loc))),
			tgbotapi.NewInlineKeyboardButtonData(getInlineKeyboardDiff(loc), fmt.Sprintf("%s:%s:%s", subcommandShowLiveTiming, serverID, getInlineKeyboardDiff(loc))),
			tgbotapi.NewInlineKeyboardButtonURL(getInlineKeyboardLiveMap(loc), liveMapUrl),
		),
	)
}
