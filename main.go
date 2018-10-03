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

	bot, err := tgbotapi.NewBotAPI(config.Conf.Telegram.TelegramAPIToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = config.Conf.Settings.Debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Text == "" {
			continue
		}

		isFromAlowedUser := false
		for _, user := range config.Conf.AllowedUsers {
			if update.Message.From.UserName != user.UserName && update.Message.From.ID != user.ID {
				continue
			}
			isFromAlowedUser = true
		}

		if !isFromAlowedUser {
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
			continue
		}

		if len(args) == 1 {
			//If there's only the update, throws a usage message
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Usage*:"+"\n"+
				"`/"+args[0][1:]+" [<ip>] {<username> <password>}`")
			msg.ReplyToMessageID = update.Message.MessageID
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
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
				continue
			}
			//Auto-add poort 22 if missing
			if !strings.ContainsAny(args[1], ":") {
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
				continue
			}
			if len(args) == 4 {
				sshUsername = args[2]
				sshPassword = args[3]
			}
			if len(args) > 4 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
					"Unspecifies field(s)!"+"\n"+
					"*Expected usage*: `/"+args[0][1:]+" [<ip>] {<username> <password>}`")
				msg.ReplyToMessageID = update.Message.MessageID
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)
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
			conn, err := ssh.Dial("tcp", sshIP, clientConfig)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
					"Failed to dial: "+err.Error())
				msg.ReplyToMessageID = update.Message.MessageID
				msg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(msg)
				continue
			}
			defer conn.Close()

			//script := []string{"echo $PATH", "touch iwashere", "echo $PATH"}
			Queue := &queue.Queue{}
			Queue.AddBulkCommandsAndOutput(config.Conf.Commands[args[0][1:]].Commands, config.Conf.Commands[args[0][1:]].ExpectedOutputs)

			messageID := sendJobStatus(Queue, bot, &update)

			//for _, scriptLine := range Queue.GetNextCommandToExecute {
			for i := 0; i < Queue.GetQueueLength(); i++ {

				//Try to create a new SSH session
				session, err := conn.NewSession()
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
						"Failed to create session: "+err.Error())
					msg.ReplyToMessageID = update.Message.MessageID
					msg.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(msg)
					continue
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
				err = session.RequestPty("xterm", 45, 40, modes)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
						"Failed create the virtual terminal: "+err.Error())
					msg.ReplyToMessageID = update.Message.MessageID
					msg.ParseMode = tgbotapi.ModeMarkdown
					bot.Send(msg)
					continue
				}

				cmd, _ := Queue.GetNextCommandToExecute()
				regexStr := Queue.GetExpectedOutput()
				rgx := regexp.MustCompile(regexStr)
				//output, outerr := session.Output(cmd)
				output, _ := session.Output(cmd)
				outputString := strings.Replace(string(output), "\r", "\n", -1)
				defer session.Close()

				if err != nil {
					Queue.SetCommandError(err)

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "*Error*:"+"\n"+
						"Failed to execute command (`"+cmd+"`): "+err.Error())
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)
					continue
				} else if regexStr != "" {
					if !rgx.MatchString(outputString) {
						Queue.SetCommandOutputMismatch(outputString)
					} else {
						Queue.SetCommandOutput(outputString)
					}
				} else {
					Queue.SetCommandOutput(outputString)
				}

				editJobStatus(Queue, bot, &update, messageID)

				if i+1 == Queue.GetQueueLength() {
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
			log.Println("Error sending message: " + "\n\n\n\n" + textF + "\n\n\n\n\n\n")
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
