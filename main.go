package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	Token     = flag.String("t", os.Getenv("TOKEN"), "Provide Bot Token")
	GuildID   = flag.String("g", os.Getenv("GUILD_ID_TEST"), "Guild ID for test command slash.")
	RemoveCmd = flag.Bool("rmcmd", true, "Remove command during shutting down bot.")

	Test                 = flag.Bool("test", envToBool(os.Getenv("TEST")), "Allows to launch the robot in test.")
	UsersAllowInTestGrap = flag.String("userstest", os.Getenv("USERS_TEST"), "List of users allow use bot in test mode.")
	UsersAllowInTest     = envToArrString(*UsersAllowInTestGrap)
)

var dgs *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	dgs, err = discordgo.New("Bot " + *Token)
	if err != nil {
		log.Println("Error during creating Discord Client Session: ", err)
		return
	}

	dgs.Identify.Intents = discordgo.IntentsGuildMessages
}

const TUSMO_AMOUNT_TRY int = 6

type TusmoPartyGamePlayer struct {
	ID, Username, Discriminator string
}

func (player *TusmoPartyGamePlayer) Tag() string {
	return fmt.Sprintf("%v#%v", player.Username, player.Discriminator)
}

type TusmoPartyGame struct {
	RetryRemaining, RaxRound, Round, RoundWin int
	ReferenceWord                             string
	CurrentWord                               []string
	Player                                    *TusmoPartyGamePlayer
}

type TusmoPartyGameLaunchParams struct {
	ChannelID string
	Number    int64
	Player    *TusmoPartyGamePlayer
}

var (
	tusmoMinAmount      = 0.0
	tusmoGameInProgress = make(map[string]*TusmoPartyGame)

	wordAlreadySeen = []string{}

	messageTipQuite = "**Tips:** _If you don't want to continue playing **Tusmo Game** just write `>quit`._\n"
	messageBotTest  = "This bot is currently under test, you are not an authorized user to use the commands of this bot in the tests."

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

				TusmoPartyGameLaunch(TusmoPartyGameLaunchParams{Player: &TusmoPartyGamePlayer{
					ID:            i.Member.User.ID,
					Username:      i.Member.User.Username,
					Discriminator: i.Member.User.Discriminator,
				}, ChannelID: i.Interaction.ChannelID, Number: number})

				break
			}

		},
	}
)

var (
	dictionnary = []string{}
)

func init() {
	file, err := os.Open("dictionary.csv")
	if err != nil {
		log.Printf("An error is occured during loading dico file: %v", err)
		return
	}
	log.Printf("Dictionary file loaded successfully.")
	defer file.Close()

	csvReader := csv.NewReader(file)
	// Allows not to send an error if the rule of the number of column of a CSV is not respected
	csvReader.FieldsPerRecord = -1
	fileLines, err := csvReader.ReadAll()
	if err != nil {
		log.Printf("An error is occured during read dico file csv: %v", err)
		return
	}

	replacer := strings.NewReplacer(".", "", "_", "", "-", "", "'", "")

	for _, line := range fileLines {
		if len(line) >= 2 {
			// Do not load words with spaces!
			if !strings.Contains(line[1], " ") {
				dictionnary = append(dictionnary, replacer.Replace(normalizeString(line[1])))
			}
		}
	}
}

func getNewWordFromDico() string {
	newWord := false
	for !newWord {
		word, err := getRandomStringFromArray(dictionnary)

		if err != nil {
			break
		}

		if !contains(wordAlreadySeen, word) {
			return word
		}
	}
	return ""
}

func TusmoPartyGameLaunch(params TusmoPartyGameLaunchParams) *TusmoPartyGame {
	var referenceWord = strings.ToUpper(getNewWordFromDico())
	var a = strings.Split(referenceWord, "")

	var currentWord = []string{a[0]}
	for i := 1; i < len(a); i++ {
		currentWord = append(currentWord, "\\_")
	}

	okReStart := true

	if _, ok := tusmoGameInProgress[params.Player.ID]; !ok {
		tusmoGameInProgress[params.Player.ID] = &TusmoPartyGame{
			ReferenceWord:  referenceWord,
			CurrentWord:    currentWord,
			RetryRemaining: TUSMO_AMOUNT_TRY,
			RaxRound:       int(params.Number),
			Round:          1,
			RoundWin:       0,
			Player:         params.Player,
		}
		okReStart = false
	}
	var tusmoParty = tusmoGameInProgress[params.Player.ID]

	numberString := fmt.Sprintf("%v", tusmoParty.RaxRound)
	if tusmoParty.RaxRound == 0 {
		numberString = "infinite"
	}

	if okReStart {
		tusmoParty.CurrentWord = currentWord
		tusmoParty.ReferenceWord = referenceWord
		tusmoParty.RetryRemaining = TUSMO_AMOUNT_TRY
	}

	log.Printf("[DEBUG] User: %v | Word wanted: %v", params.Player.Tag(), tusmoParty.ReferenceWord)

	dgs.ChannelMessageSend(
		params.ChannelID,
		fmt.Sprintf(
			messageTipQuite+
				"Round: **%v**/**%v** | Retry remaining: **%v** | Lenght current word : **%v** \n"+
				"--- --- --- --- --- --- --- --- ---\n"+
				"You have to guess the word: %v",
			tusmoParty.Round,
			numberString,
			tusmoParty.RetryRemaining,
			len(tusmoParty.CurrentWord),
			strings.Join(tusmoParty.CurrentWord, " **|** "),
		),
	)

	TusmoPartyGameSendImage(*tusmoParty, params.ChannelID)

	return tusmoParty
}

func TusmoPartyGameFinish(userID string) {
	delete(tusmoGameInProgress, userID)
}

func TusmoPartyGameSendImage(tusmoParty TusmoPartyGame, ChannelID string) {
	reader := generateTusmoImage(TusmoPartyGameImageParams{
		WaterMark: TusmoPartyGameImageWaterMark{
			Name: fmt.Sprintf(".%v", dgs.State.User.Username),
			URL:  dgs.State.User.AvatarURL("512"),
		},
		Username: tusmoParty.Player.Tag(),
		Word:     strings.ReplaceAll(strings.Join(tusmoParty.CurrentWord, " "), "\\", ""),
	})

	dgs.ChannelFileSend(ChannelID, fmt.Sprintf("tusmo_game_%v.png", tusmoParty.Player.ID), reader)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Forbid the management of messages not concerned
	if m.Author.ID == s.State.User.ID || m.Author.Bot == true {
		return
	}

	authorID := m.Author.ID

	// Check if player
	if tusmoParty, ok := tusmoGameInProgress[authorID]; ok {
		var message = "A party of **Tusmo** is in progress for you."

		args := strings.Split(m.Content, " ")

		max := fmt.Sprintf("%v", tusmoParty.RaxRound)
		if tusmoParty.RaxRound == 0 {
			max = "infinite"
		}

		if len(args) == 1 {
			// Command tusmo party game
			term := strings.Split(strings.ToUpper(normalizeString(args[0])), "")

			switch strings.ToLower(args[0]) {
			case ">quit":
				s.ChannelMessageSend(
					m.ChannelID,
					fmt.Sprintf(
						"🚪 You have just left the game and your score is **%v**/**%v** games won.",
						tusmoParty.Round-1,
						max,
					),
				)
				TusmoPartyGameFinish(authorID)
				return
			case ">relaunch":
				numberRound := tusmoParty.RaxRound

				TusmoPartyGameFinish(authorID)
				s.ChannelMessageSend(m.ChannelID, "🔄 You have just restarted the whole game with the same parameters.")
				TusmoPartyGameLaunch(TusmoPartyGameLaunchParams{ChannelID: m.ChannelID, Number: int64(numberRound), Player: tusmoParty.Player})
				return
			}
			// End command tusmo party game

			// Normalize string
			referenceWord := tusmoParty.ReferenceWord
			referenceWordArr := strings.Split(referenceWord, "")

			if len(term) == len(referenceWordArr) {
				currentWordBefore := strings.Split(strings.Join(tusmoParty.CurrentWord, "|"), "|")

				word := []string{}
				for k, v := range referenceWordArr {
					if term[k] == v {
						word = append(word, "("+term[k]+")")
						tusmoParty.CurrentWord[k] = term[k]
					} else if strings.Contains(referenceWord, term[k]) {
						word = append(word, "<"+term[k]+">")
					} else {
						word = append(word, "~~"+term[k]+"~~")
					}
				}

				// Don't missing, i use backslash because discord md format :3
				if !contains(tusmoParty.CurrentWord, "\\_") {
					tusmoParty.Round++

					message = fmt.Sprintf(
						"👏 You have found the word you were looking for which was: **%v**",
						tusmoParty.ReferenceWord,
					)
					s.ChannelMessageSend(m.ChannelID, message)

					if tusmoParty.RaxRound != 0 && (tusmoParty.RaxRound+1) == tusmoParty.Round {
						message = fmt.Sprintf(
							"🎉 Congratulations %v! You succeeded in finding all the words.\n",
							m.Author,
						)

						TusmoPartyGameFinish(authorID)
					} else {
						TusmoPartyGameLaunch(TusmoPartyGameLaunchParams{ChannelID: m.ChannelID})
						return
					}

				} else {
					tusmoParty.RetryRemaining--

					// Parti perdu
					if tusmoParty.RetryRemaining <= 0 {
						message = fmt.Sprintf(
							"**Game Over** | Search word was: **%v**\n"+
								"Your score is **%v**/**%v** games won.",
							tusmoParty.ReferenceWord,
							tusmoParty.Round-1,
							max,
						)
						TusmoPartyGameFinish(authorID)
					} else {
						message = fmt.Sprintf(
							messageTipQuite+
								"Round: **%v**/**%v** | Retry remaining: **%v** | Lenght current word : **%v** \n"+
								"--- --- --- --- --- --- --- --- ---\n"+
								"**Letter Legend:** \n- `(A)` Good position and present\n- `<B>` Bad position and present\n- ~~`C`~~ Bad position and not present\n"+
								"--- --- --- --- --- --- --- --- ---\n"+
								"**Letter found before:** %v\n"+
								"**Tips copy:** %v\n"+
								"**Letter status:** %v",
							tusmoParty.Round,
							max,
							tusmoParty.RetryRemaining,
							len(tusmoParty.CurrentWord),
							strings.Join(currentWordBefore, " **|** "),
							strings.Join(tusmoParty.CurrentWord, ""),
							strings.Join(word, " **|** "),
						)
						s.ChannelMessageSend(m.ChannelID, message)

						if strings.Join(currentWordBefore, "") != strings.Join(tusmoParty.CurrentWord, "") {
							TusmoPartyGameSendImage(*tusmoParty, m.ChannelID)
						}

						return
					}
				}
			} else {
				message = "The length of the first word you wrote does not match the length of the word you are looking for."
			}

			s.ChannelMessageSend(m.ChannelID, message)
		}
	}
}

func init() {
	dgs.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandsHandler[i.ApplicationCommandData().Name]; ok {

			if *Test && !contains(UsersAllowInTest, i.Member.User.ID) {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:   1 << 6,
						Content: messageBotTest,
					},
				})

				return
			}

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
	if *Test {
		log.Println("This Application is running in test mode.")
	}
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
