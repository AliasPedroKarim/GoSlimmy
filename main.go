package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	Token     = flag.String("t", os.Getenv("TOKEN"), "Provide Bot Token")
	GuildID   = flag.String("g", os.Getenv("GUILD_ID_TEST"), "Guild ID for test command slash.")
	RemoveCmd = flag.Bool("rmcmd", true, "Remove command during shutting down bot.")
)

var dgs *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	dgs, err = discordgo.New("Bot " + *Token)
	if err != nil {
		log.Println("Error durring creating Discord Client Session: ", err)
		return
	}

	dgs.Identify.Intents = discordgo.IntentsGuildMessages
}

const TUSMO_AMOUNT_TRY int = 6

type TusmoPartyGame struct {
	referenceWord  string
	currentWord    []string
	retryRemaining int
	maxRound       int
	Round          int
}

var (
	tusmoMinAmount = 0.0

	tusmoGameInProgress = make(map[string]*TusmoPartyGame)

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Check if the bot isn't die !!!",
		},
		{
			Name:        "tusmo",
			Description: "The poor man's tusmo (it's a little bit stupid since tusmo is not even paid :joy:)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "new-game",
					Description: "Launched a new tusmo",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "number",
							Description: "Maximum number of rounds you want to play",
							MinValue:    &tusmoMinAmount,
							MaxValue:    10,
							Required:    false,
							Type:        discordgo.ApplicationCommandOptionInteger,
						},
					},
					Type: discordgo.ApplicationCommandOptionSubCommand,
				},
			},
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

		"tusmo": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			switch i.ApplicationCommandData().Options[0].Name {
			case "new-game":
				// Error with "i.User.ID" because in here is "nil"

				if _, ok := tusmoGameInProgress[i.Member.User.ID]; ok {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "You've already started a game of Tusmo that's already in progress.",
						},
					})

					return
				}

				var number int64 = 0
				var content string

				if len(i.ApplicationCommandData().Options[0].Options) > 0 && i.ApplicationCommandData().Options[0].Options[0].Name == "number" {
					number = i.ApplicationCommandData().Options[0].Options[0].IntValue()
				}

				if number == 0 {
					content = fmt.Sprintf("You have just launched an **infinite** game!!! (well almost)")
				} else {
					content = fmt.Sprintf("You have just started a game of **%v** rounds.", number)
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: content,
					},
				})

				var referenceWord = strings.ToUpper("Discord")
				var a = strings.Split(referenceWord, "")

				var currentWord = []string{a[0]}
				for i := 1; i < len(a); i++ {
					currentWord = append(currentWord, "\\_")
				}

				tusmoGameInProgress[i.Member.User.ID] = &TusmoPartyGame{referenceWord, currentWord, TUSMO_AMOUNT_TRY, int(number), 1}

				_, err := dgs.FollowupMessageCreate(dgs.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
					Content: fmt.Sprintf(
						"Round: **%v** | Retry remaining: **%v** \n"+
							"You have to guess the word: %v",
						tusmoGameInProgress[i.Member.User.ID].Round,
						tusmoGameInProgress[i.Member.User.ID].retryRemaining,
						strings.Join(tusmoGameInProgress[i.Member.User.ID].currentWord, " | "),
					),
				})

				if err != nil {
					dgs.FollowupMessageCreate(dgs.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
						Content: "Something went wrong",
					})
					return
				}

				break
			default:
				log.Println("[Tusmo] >> BAH rien !!")
			}

		},
	}
)

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Forbid the management of messages not concerned
	if m.Author.ID == s.State.User.ID || m.Author.Bot == true {
		return
	}

	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)

	authorID := m.Author.ID

	// Check if player
	if tusmoParty, ok := tusmoGameInProgress[authorID]; ok {
		var message = "A party of **Tusmo** is in progress for you."

		args := strings.Split(m.Content, " ")
		if len(args) > 0 {
			// Normalize string
			stringNorm, _, _ := transform.String(t, args[0])

			term := strings.Split(strings.ToUpper(stringNorm), "")
			referenceWord := tusmoParty.referenceWord
			referenceWordArr := strings.Split(referenceWord, "")

			if len(term) == len(referenceWordArr) {
				currentWordBefore := strings.Split(strings.Join(tusmoParty.currentWord, "|"), "|")

				word := []string{}
				for k, v := range referenceWordArr {
					if term[k] == v {
						word = append(word, "("+term[k]+")")
						tusmoParty.currentWord[k] = term[k]
					} else if strings.Contains(referenceWord, term[k]) {
						word = append(word, "<"+term[k]+">")
					} else {
						word = append(word, "~~"+term[k]+"~~")
					}
				}

				// Don't missing, i use backslash because discord md format :3
				if !contains(tusmoParty.currentWord, "\\_") {
					message =
						fmt.Sprintf(
							"🎉 Congratulations! You have found the word you were looking for which was: **%v**",
							tusmoParty.referenceWord,
						)
					delete(tusmoGameInProgress, authorID)
				} else {
					tusmoParty.retryRemaining = tusmoParty.retryRemaining - 1

					// Parti perdu
					if tusmoParty.retryRemaining <= 0 {
						message = fmt.Sprintf(
							"**Game Over** | Search word was: **%v**",
							tusmoParty.referenceWord,
						)
						delete(tusmoGameInProgress, authorID)
					} else {
						message = fmt.Sprintf(
							"Round: **%v** | Retry remaining: **%v** \n"+
								"--- --- --- --- --- --- --- --- ---\n"+
								"**Letter Legend:** \n- `(A)` Good position and present\n- `<B>` Bad position and present\n- ~~`C`~~ Bad position and not present\n"+
								"--- --- --- --- --- --- --- --- ---\n"+
								"**Letter found before:** %v\n"+
								"**Letter status:** %v",
							tusmoParty.Round,
							tusmoParty.retryRemaining,
							strings.Join(currentWordBefore, " **|** "),
							strings.Join(word, " **|** "),
						)
					}
				}
			} else {
				message = "The length of the first word you wrote does not match the length of the word you are looking for."
			}

		}

		s.ChannelMessageSend(m.ChannelID, message)
	}
}

func init() {
	dgs.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandsHandler[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	dgs.AddHandler(messageCreate)
}

func main() {
	dgs.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logging as %v#%v in running now.", dgs.State.User.Username, dgs.State.User.Discriminator)
	})

	err := dgs.Open()
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
