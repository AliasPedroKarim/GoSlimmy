package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	Token     = flag.String("token", "", "Provide Bot Token")
	GuildID   = flag.String("guildtest", "", "Guild ID for test command slash.")
	RemoveCmd = flag.Bool("rmcmd", true, "Remove command during shutting down bot.")
)

var dgs *discordgo.Session

func init() { flag.Parse() }

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Check if bot is not die !!!",
		},
	}
	commandsHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			dgs.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Great! I'm still alive, it seems! (%d ms)", dgs.HeartbeatLatency().Milliseconds()),
				},
			})
		},
	}
)

func init() {
	dgs.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandsHandler[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	var err error
	dgs, err = discordgo.New("Bot " + *Token)
	if err != nil {
		log.Println("Error durring creating Discord Client Session: ", err)
		return
	}

	dgs.Identify.Intents = discordgo.IntentsGuildMessages

	dgs.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logging as %v#%v in running now.", dgs.State.User.Username, dgs.State.User.Discriminator)
	})

	// dg.AddHandler(messageCreate)

	err = dgs.Open()
	if err != nil {
		log.Fatalln("Error occured during open websocket client: ", err)
		return
	}

	log.Println("Start to registering all commands...")
	registeredCommand := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dgs.ApplicationCommandCreate(dgs.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Connot create '%v' command: %v", v.Name, err)
			return
		}

		registeredCommand[i] = cmd
	}

	defer dgs.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	log.Println("Press Ctrl + C to exit.")
	<-sc

	if *RemoveCmd {
		log.Print("Start to suppress all commands...")

		for _, v := range registeredCommand {
			err := dgs.ApplicationCommandDelete(dgs.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	log.Println("MakeFranceGreateAgain x3")
}
