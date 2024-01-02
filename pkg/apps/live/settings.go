package live

import (
	"context"
	"fmt"
	"log"
	"rfactor2telegrambot/pkg/menus"
	"rfactor2telegrambot/pkg/settings"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type ContextUser string
type ContextChatID string

const (
	UserContextKey         ContextUser   = "user"
	ChatContextKey         ContextChatID = "chat"
	inlineKeyboardTestday                = settings.TestDay
	inlineKeyboardPractice               = settings.Practice
	inlineKeyboardQual                   = settings.Qual
	inlineKeyboardWarmup                 = settings.Warmup
	inlineKeyboardRace                   = settings.Race

	symbolNotifications     = "🔔"
	subcommandNotifications = "notifications"
)

type SettingsApp struct {
	bot          *tgbotapi.BotAPI
	appMenu      menus.ApplicationMenu
	menuKeyboard tgbotapi.ReplyKeyboardMarkup
	sm           *settings.Manager
	loc          *i18n.Localizer
	title        string
	mu           sync.Mutex
}

func NewSettingsApp(bot *tgbotapi.BotAPI, appMenu menus.ApplicationMenu, sm *settings.Manager, appName string, loc *i18n.Localizer) *SettingsApp {
	sa := &SettingsApp{
		bot:     bot,
		sm:      sm,
		loc:     loc,
		title:   appName,
		appMenu: appMenu,
	}

	return sa
}

func (sa *SettingsApp) Menu() tgbotapi.ReplyKeyboardMarkup {
	return sa.menuKeyboard
}

func (sa *SettingsApp) AcceptCommand(command string) (bool, func(ctx context.Context, chatId int64) error) {
	return false, nil
}

func (sa *SettingsApp) AcceptCallback(query *tgbotapi.CallbackQuery) (bool, func(ctx context.Context, query *tgbotapi.CallbackQuery) error) {
	data := strings.Split(query.Data, ":")
	if data[0] == subcommandNotifications {
		sa.mu.Lock()
		defer sa.mu.Unlock()
		return true, func(ctx context.Context, query *tgbotapi.CallbackQuery) error {
			userID := data[1]
			sessionType := data[2]

			chatCtxValue := ctx.Value(ChatContextKey)
			if chatCtxValue == nil {
				message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "settings.chatNotFound",
						Other: "Could not read chat information",
					},
				})

				msg := tgbotapi.NewMessage(query.Message.Chat.ID, message)
				msg.ReplyMarkup = sa.appMenu.PrevMenu()
				_, err := sa.bot.Send(msg)
				return err
			}
			chat := chatCtxValue.(*tgbotapi.Chat)
			chatID := fmt.Sprintf("%d", chat.ID)

			err := sa.sm.ToggleNotificationForSessionStarted(userID, chatID, sessionType)
			if err != nil {
				message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID:    "settings.couldNotChangeNotificationStatus",
						Other: "Could not change notification status",
					},
				})

				msg := tgbotapi.NewMessage(query.Message.Chat.ID, message)
				msg.ReplyMarkup = sa.appMenu.PrevMenu()
				_, err := sa.bot.Send(msg)
				return err
			}
			return sa.renderNotifications(&query.Message.MessageID)(ctx, query.Message.Chat.ID)
		}
	}
	return false, nil
}

func (sa *SettingsApp) AcceptButton(button string) (bool, func(ctx context.Context, chatId int64) error) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// fmt.Printf("SETTINGS: button: %s. appName: %s\n", button, buttonSettings)
	if button == sa.title {
		return true, sa.renderNotifications(nil)
	} else if button == sa.appMenu.ButtonBackTo() {
		return true, func(ctx context.Context, chatId int64) error {
			msg := tgbotapi.NewMessage(chatId, "OK")
			msg.ReplyMarkup = sa.appMenu.PrevMenu()
			_, err := sa.bot.Send(msg)
			return err
		}
	}
	return false, nil
}

func (sa *SettingsApp) renderNotifications(messageID *int) func(ctx context.Context, chatId int64) error {
	return func(ctx context.Context, chatId int64) error {
		userCtxValue := ctx.Value(UserContextKey)
		if userCtxValue == nil {
			message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "settings.userNotFound",
					Other: "Could not read user",
				},
			})

			msg := tgbotapi.NewMessage(chatId, message)
			msg.ReplyMarkup = sa.appMenu.PrevMenu()
			_, err := sa.bot.Send(msg)
			return err
		}
		user := userCtxValue.(*tgbotapi.User)
		userID := fmt.Sprintf("%d", user.ID)
		notificationStatus, err := sa.sm.ListNotifications(userID)
		if err != nil {
			log.Println(err)
			message := sa.loc.MustLocalize(&i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "settings.couldNotReadNotifications",
					Other: "Could not read notifications for user",
				},
			})

			msg := tgbotapi.NewMessage(chatId, message)
			msg.ReplyMarkup = sa.appMenu.PrevMenu()
			_, err := sa.bot.Send(msg)
			return err
		}
		keyboard := getSettingsInlineKeyboard(userID, notificationStatus)
		text := sa.loc.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "settings.notifications",
				Other: "Notification status\n(Only notifies the first session)",
			},
		})

		var cfg tgbotapi.Chattable
		if messageID == nil {
			msg := tgbotapi.NewMessage(chatId, text)
			msg.ReplyMarkup = keyboard
			cfg = msg
		} else {
			msg := tgbotapi.NewEditMessageText(chatId, *messageID, text)
			msg.ReplyMarkup = &keyboard
			cfg = msg
		}
		_, err = sa.bot.Send(cfg)
		return err
	}
}

func getSettingsInlineKeyboard(userID string, n settings.Notifications) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(inlineKeyboardTestday+" "+n.TestDaySymbol(), fmt.Sprintf("%s:%s:%s", subcommandNotifications, userID, inlineKeyboardTestday)),
			tgbotapi.NewInlineKeyboardButtonData(inlineKeyboardPractice+" "+n.PracticeSymbol(), fmt.Sprintf("%s:%s:%s", subcommandNotifications, userID, inlineKeyboardPractice)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(inlineKeyboardQual+" "+n.QualSymbol(), fmt.Sprintf("%s:%s:%s", subcommandNotifications, userID, inlineKeyboardQual)),
			tgbotapi.NewInlineKeyboardButtonData(inlineKeyboardWarmup+" "+n.WarmupSymbol(), fmt.Sprintf("%s:%s:%s", subcommandNotifications, userID, inlineKeyboardWarmup)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(inlineKeyboardRace+" "+n.RaceSymbol(), fmt.Sprintf("%s:%s:%s", subcommandNotifications, userID, inlineKeyboardRace)),
		),
	)
}
