package main

import (
	"log"
	"os"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func main() {
	token := os.Getenv("TOKEN")
	if token == "" {
		token = "111:3333kkkk"
	}
	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: -1,
	})
	updater := ext.NewUpdater(dispatcher, nil)
	if err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates:    false,
		EnableWebhookDeletion: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			AllowedUpdates: []string{"message"},
			Timeout:        5,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 5,
			},
		},
	}); err != nil {
		log.Fatalln(err.Error())
	}
	log.Println(b.User.FirstName, " has been started!")
	updater.Idle()
}
