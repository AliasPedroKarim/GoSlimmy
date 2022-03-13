package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	Token     = flag.String("token", "", "Provide Bot Token")
	GuildID   = flag.String("guildtest", "", "Guild ID for test command slash.")
	RemoveCmd = flag.Bool("rmcmd", true, "Remove command during shutting down bot.")

	commands = []*discordgo.ApplicationCommand{}
)

func init() { flag.Parse() }

func main() {
	dg, err := discordgo.New("Bot " + *Token)
	if err != nil {
		fmt.Println("Error durring creating Discord Client Session: ", err)
		return
	}

	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Error occured during open websocket client: ", err)
		return
	}

	fmt.Println("Bot " + dg.State.User.Username + " in running now. Press Ctrl + C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot == true {
		return
	}

	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong, i think i'm alive !!")
	}
}
