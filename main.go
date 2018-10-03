package main

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"

	"./config"
	"./queue"

	"golang.org/x/crypto/ssh"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

//TODO
// - split report message if too long
// - improve comments

func loadConfig(features *[]string) {
	config.LoadDefaultConfig()
	*features = config.GetFeatures()
}

func main() {

	var features []string

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panic("ERROR", err)
	}
	defer watcher.Close()

	//
	go func() {
		for {
			select {
			// watch for events
			//case event := <-watcher.Events:
			case <-watcher.Events:
				loadConfig(&features)
				//log.Println("Config changed! Reloading! %#v\n", event)
				log.Println("Config changed! Reloading!")
				break

				// watch for errors
			case err := <-watcher.Errors:
				log.Panic(err)
				break
			}
		}
	}()

	// out of the box fsnotify can watch a single file, or a single directory
	if err := watcher.Add("config.toml"); err != nil {
		log.Panic(err)
	}

	loadConfig(&features)

	if config.Conf.Settings.Debug {
		log.Println("Initializig Telegram bot API library")
	}
	bot, err := tgbotapi.NewBotAPI(config.Conf.Telegram.TelegramAPIToken)
	if err != nil {
		log.Panic(err)
	}

	if config.Conf.Settings.Debug {
		log.Println("Connection established - Bot authorized on account " + bot.Self.UserName)
	}

	bot.Debug = config.Conf.Settings.Debug

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Text == "" {
			if config.Conf.Settings.Debug {
				log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent a message to the bot, but the body message is empty. Ignoring the message...")
			}
			continue
		}

		isFromAllowedUser := false
		for _, user := range config.Conf.AllowedUsers {
			if strings.ToLower(update.Message.From.UserName) != strings.ToLower(user.UserName) && update.Message.From.ID != user.ID {
				continue
			}
			isFromAllowedUser = true
		}

		if !isFromAllowedUser {
			if config.Conf.Settings.Debug {
				log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent a message to the bot, but it's not whitelisted. Ignoring the message...")
			}
			continue
		}

		if update.Message.IsCommand() {
			if config.Conf.Settings.Debug {
				log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\", but the text doesn't seems to be a command. Ignoring the message...")
			}
			continue
		}

		sshIP, sshUsername, sshPassword := "", config.Conf.Settings.DefaultUsername, config.Conf.Settings.DefaultPassword
		args := strings.Split(update.Message.Text, " ")
		presentFeature := false
		for _, feature := range features {
			arg := args[0][1:]
			if arg == feature {
				presentFeature = true
				break
			}
		}
		if !presentFeature {
			if config.Conf.Settings.Debug {
				log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\", but the command is not recognized. Ignoring the message...")
			}
			continue
		}

		if len(args) == 1 {
			//If there's only the update, throws a usage message
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Usage*:"+"\n"+
				"`/"+args[0][1:]+" [<ip>] {<username> <password>}`")
			msg.ReplyToMessageID = update.Message.MessageID
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)

			if config.Conf.Settings.Debug {
				log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\", but the argument is only one.")
			}
			continue
		}
		if len(args) > 1 {
			//Check if 2nd parameter is a valid IP
			rgx := regexp.MustCompile("^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(:\\d+)?$")
			if !rgx.MatchString(args[1]) {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
					args[1]+" does not look like an IP")
				msg.ReplyToMessageID = update.Message.MessageID
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

				if config.Conf.Settings.Debug {
					log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\", but the 2nd argument doesn't looks like a valid IP. Ignoring the message...")
				}
				continue
			}
			//Auto-add poort 22 if missing
			if !strings.ContainsAny(args[1], ":") {
				if config.Conf.Settings.Debug {
					log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\" but the port was not specified, assuming 22.")
				}

				args[1] += ":22"
			}
			for _, s := range config.Conf.KnownServers {
				if s.IP == args[1] {
					sshUsername = s.Username
					sshPassword = s.Password
				}
			}
			sshIP = args[1]
			if len(args) == 3 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
					"Unspecifies field(s)!"+"\n"+
					"*Expected usage*: `/"+args[0][1:]+" [<ip>] {<username> <password>}`")
				msg.ReplyToMessageID = update.Message.MessageID
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

				if config.Conf.Settings.Debug {
					log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\" but the arguments are ambiguous. ")
				}
				continue
			}
			if len(args) == 4 {
				sshUsername = args[2]
				sshPassword = args[3]
				if config.Conf.Settings.Debug {
					log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\". Command recognized!")
				}
			}
			if len(args) > 4 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
					"Unspecifies field(s)!"+"\n"+
					"*Expected usage*: `/"+args[0][1:]+" [<ip>] {<username> <password>}`")
				msg.ReplyToMessageID = update.Message.MessageID
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

				if config.Conf.Settings.Debug {
					log.Println("User @" + update.Message.From.UserName + "[" + strconv.Itoa(update.Message.From.ID) + "] sent the message \"" + update.Message.Text + "\" but the arguments are more than four. Ignoring the message...")
				}

				continue
			}

			//Sets the SSH connection settings
			/*
				clientConfig := &ssh.ClientConfig{
					User: sshUsername,
					Auth: []ssh.AuthMethod{
						ssh.Password(sshPassword),
						//publicKey,
					},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				}*/

			if config.Conf.Settings.Debug {
				log.Println("\tSetting SSH client configuration...")
			}

			clientConfig := &ssh.ClientConfig{
				User: sshUsername,
				Auth: []ssh.AuthMethod{
					ssh.KeyboardInteractive(sshInteractive),
					ssh.Password(sshPassword),
					//publicKey,
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}

			//Try to establish the SSH connection
			if config.Conf.Settings.Debug {
				log.Println("\tTrying to establish the SSH connection...")
			}
			conn, err := ssh.Dial("tcp", sshIP, clientConfig)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
					"Failed to dial: "+err.Error())
				msg.ReplyToMessageID = update.Message.MessageID
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)

				if config.Conf.Settings.Debug {
					log.Println("\tFailed to connect to establish a connection with the given host!")
				}
				continue
			}
			if config.Conf.Settings.Debug {
				log.Println("\tSSH connection established!")
			}
			defer conn.Close()

			//script := []string{"echo $PATH", "touch iwashere", "echo $PATH"}
			if config.Conf.Settings.Debug {
				log.Println("\tAdding the commands to the execution queue...")
			}
			Queue := &queue.Queue{}
			Queue.AddBulkCommandsAndOutput(config.Conf.Commands[args[0][1:]].Commands, config.Conf.Commands[args[0][1:]].ExpectedOutputs)
			Queue.SetRetry(config.Conf.Commands[args[0]].RetryOnFaliure)

			messageID := sendJobStatus(Queue, bot, &update)

			//for _, scriptLine := range Queue.GetNextCommandToExecute {
			for i := 0; !Queue.IsOver() && 3+i < Queue.GetQueueLength(); i++ {

				if config.Conf.Settings.Debug {
					log.Println("\tAttempting to create a new SSH session...")
				}

				//Try to create a new SSH session
				session, err := conn.NewSession()
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
						"Failed to create session: "+err.Error())
					msg.ReplyToMessageID = update.Message.MessageID
					msg.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(msg)
					if config.Conf.Settings.Debug {
						log.Println("\tFailed to create a new SSH session! Retrying...")
					}
					continue
				}

				if config.Conf.Settings.Debug {
					log.Println("\tSSH session successfully created")
				}
				defer session.Close()

				//Set virual terminal mode
				modes := ssh.TerminalModes{
					ssh.ECHO:          0,     // disable echoing
					ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
					ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
					ssh.IGNCR:         1,     // Ignore CR on input.
				}

				//Try to request a virtual terminal
				//err = session.RequestPty("xterm", 80, 40, modes)

				if config.Conf.Settings.Debug {
					log.Println("\tTrying to create the virtual terminal size (45 characters x 40 lines)...")
				}

				err = session.RequestPty("xterm", 45, 40, modes)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
						"Failed create the virtual terminal: "+err.Error())
					msg.ReplyToMessageID = update.Message.MessageID
					msg.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(msg)

					if config.Conf.Settings.Debug {
						log.Println("\tFailed to create the virtual terminal size! Retrying...")
					}

					continue
				}

				if config.Conf.Settings.Debug {
					log.Println("\tVirtual terminal correctly created")
				}

				var cmd string
				if Queue.GetCommandStatus() != queue.Success {
					//If retry is allowed, retry, otherwise go on
					if Queue.IsRetryAllowed() {
						if config.Conf.Settings.Debug {
							log.Println("\tLast command didn't succedeed - reloading command...")
						}
						cmd, _ = Queue.GetActualCommandAndExecute(true)
					} else {
						if config.Conf.Settings.Debug {
							log.Println("\tLast command didn't succedeed - ignoring and loading next command...")
						}
						cmd, _ = Queue.PopCommand()
					}
				} else {
					if config.Conf.Settings.Debug {
						log.Println("\tLoading next command...")
					}
					cmd, _ = Queue.PopCommand()
				}

				/*
					i indicates the iteration - that may not correspond with the actual command number
					if config.Conf.Settings.Debug {
						var suffix string
						switch (i+1)%10{
							case 1
								suffix = "st"
							case 2
								suffix = "nd"
							case 3
								suffix = "rd"
							default
								suffix = "th"
						}

						log.Println("\tExecuting the " + strconv.Itoa(i+1) + suffix +" command (" + cmd + ")...")
					}
				*/

				regexStr := Queue.GetExpectedOutput()
				rgx := regexp.MustCompile(regexStr)
				//output, outerr := session.Output(cmd)
				if config.Conf.Settings.Debug {
					log.Println("\tExecuting command...")
				}
				output, outerr := session.Output(cmd)
				outputString := strings.Replace(string(output), "\r", "\n", -1)
				defer session.Close()

				if outerr != nil {
					Queue.SetCommandError(outerr)

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
						"Failed to execute command (`"+cmd+"`): "+outerr.Error())
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)

					if config.Conf.Settings.Debug {
						log.Println("\tError executing the command! Retrying...")
					}
					continue
				} else if regexStr != "" {
					if !rgx.MatchString(outputString) {
						Queue.SetCommandOutputMismatch(outputString)
						if config.Conf.Settings.Debug {
							log.Println("\tThe comman executed successfully but the expected output doesn't match")
						}
					} else {
						Queue.SetCommandOutput(outputString)
						if config.Conf.Settings.Debug {
							log.Println("\tThe comman executed successfully and the expected output match")
						}
					}
				} else {
					Queue.SetCommandOutput(outputString)
					if config.Conf.Settings.Debug {
						log.Println("\tThe comman executed successfully ")
					}
				}

				editJobStatus(Queue, bot, &update, messageID)

				if Queue.IsOver() {
					if config.Conf.Settings.Debug {
						log.Println("\tThe queue is over")
					}
					sendReport(Queue, bot, &update)
				}

			}
		}
	}
}

func sshInteractive(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	answers = make([]string, len(questions))
	// The second parameter is unused
	for n := range questions {
		answers[n] = config.Conf.Settings.DefaultPassword
	}

	return answers, nil
}

func sendJobStatus(q *queue.Queue, bot *tgbotapi.BotAPI, update *tgbotapi.Update) int {
	Jobstatuses := q.GetScriptsStatus()
	var text string
	for _, s := range Jobstatuses {
		text += s + "\n"
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Status* ‏‏‎  ‏‏‎  ‏‏‎  ‏‏‎  ‏‏‎ | *Command list*:\n"+text) //Special char [ ‏‏‎ ]
	msg.ReplyToMessageID = update.Message.MessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
	sendStat, _ := bot.Send(msg)
	return sendStat.MessageID
}

func editJobStatus(q *queue.Queue, bot *tgbotapi.BotAPI, update *tgbotapi.Update, messageID int) {
	Jobstatuses := q.GetScriptsStatus()
	var text string
	for _, s := range Jobstatuses {
		text += s + "\n"
	}
	msg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, messageID, "*Status* | *Command list*:\n"+text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

func sendReport(q *queue.Queue, bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	Jobstatuses := q.GetCommandQueue()
	text := "📝 <strong>REPORT</strong>"
	for i, cmd := range Jobstatuses {
		text += "\n\n" + strconv.Itoa(i+1) + ") (<code>" + escapeXMLTags(cmd.Command) + "</code>) "
		switch cmd.Status {
		//✅🕐⚙️❌
		case queue.Queued:
			text += "🕐  (Queued)"
			break
		case queue.Executing:
			text += "⚙️  (Executing)"
			break
		case queue.Success:
			text += "✅  (Success)"
			break
		case queue.Error:
			text += "❌  (Error)"
			break
		case queue.OutputMismatch:
			text += "❗️  (Output Mismatch)"
			break
		}
		text += "\n<strong>OUTPUT</strong>:\n"
		if escapeXMLTags(strings.TrimSpace(cmd.Output)) == "" {
			text += "<i>None</i>"
		} else {
			text += "<code>" + escapeXMLTags(strings.TrimSpace(cmd.Output)) + "</code>\n"
		}

		if cmd.ExpectedOutput != "" && cmd.Status == queue.OutputMismatch {
			text += "\n<strong>EXPECTED OUTPUT REGEX</strong>:\n"
			text += "<code>" + escapeXMLTags(cmd.ExpectedOutput) + "</code>\n\n\n"

		}
	}
	//4096 is the telegram limit
	const maxMessageLen = 4080
	var reportMessages []string

	if len(text) > maxMessageLen {
		text = "⚠️ <strong>OUTPUT IS LONGER THAN EXPECTED, THE OUTPUT WILL BE SPLITTED IN MULTIPLE MESSAGES!</strong> ⚠️\n\n" + text
		log.Println(text)
	}

	closingTag, openingTag := "", ""
	messageContinuation := false
	assignOpenTag := false
	sensibleTags := [...]string{"code", "strong", "i"}

	for i := 1; i < len(text); i += maxMessageLen {
		var submessage string
		if i-1+maxMessageLen > len(text) {
			submessage = text[i-1:]
		} else {
			submessage = text[i-1 : i-1+maxMessageLen]
		}

		//Used the first cycle
		var tempOpenTag string

		if messageContinuation {
			if strings.Count(submessage, closingTag) != strings.Count(submessage, openingTag) {
				//Closing in this message
				messageContinuation = false
				closingTag = ""
			}
		} else {
			//The message is not closing

			for _, tag := range sensibleTags {
				if strings.Count(submessage, "</"+tag+">") < strings.Count(submessage, "<"+tag+">") {
					closingTag = "</" + tag + ">"
					tempOpenTag = "<" + tag + ">" //The opening tag needs to be setted  after the message send
					messageContinuation = true
					assignOpenTag = true
					break
				}
			}

		}

		reportMessages = append(reportMessages, openingTag+submessage+closingTag)

		if !messageContinuation {
			openingTag = ""
		} else if assignOpenTag {
			openingTag = tempOpenTag
			assignOpenTag = false
		}
	}

	for _, textF := range reportMessages {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, textF)
		msg.ReplyToMessageID = update.Message.MessageID
		msg.ParseMode = tgbotapi.ModeHTML
		_, err := bot.Send(msg)
		if err != nil && config.Conf.Settings.Debug {
			log.Println("Error sending message: " + "\n\n\n\n" + textF + "\n\n\n" + err.Error())
		}
	}
}

func escapeXMLTags(s string) string {
	return strings.Replace(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					s, "\"", "&quot;", -1),
				"&", "&amp;", -1),
			">", "&gt;", -1),
		"<", "&lt;", -1)
}
