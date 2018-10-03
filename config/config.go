package config

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

//Conf is the configuration that the bot uses
var Conf config

//LoadDefaultConfig imports the config from the default file (./config.toml)
func LoadDefaultConfig() {
	importConfig("config.toml")
}

func importConfig(path string) {
	b, err := ioutil.ReadFile(path) // just pass the file name
	if err.Error() == "open "+path+": The system cannot find the file specified." {
		defConfig := `[telegram]
TelegramAPIToken = "123456789:AbCdEf5705JwQr948bRjr78buyG8UYUg95f"

[settings]
debug = false
defaultUsername = "ubnt"
defaultPassword = "ubnt"

[features]
[features.commandname]
	commands = ["echo $PATH", "touch iwashere"]
	expectedOutputs = [".*", ""]
	onFaliure = "interrupt" # retry, ignore or interrupt

[knownservers]
[knownservers.serverAlias]
	IP="1.2.3.4:2109"
	Username = "customuser"
	Password = "ezpassword"

[allowedUsers]
[allowedUsers.userAlias1]
	username = "yourAllowedUsername"
[allowedUsers.userAlias2]
	ID = 12345678
	`
		ioutil.WriteFile(path, []byte(defConfig), os.ModeExclusive)
		log.Println("Config file not found. It has been created.")
		os.Exit(1)
	} else {
		log.Panic(err)
	}
	configString := string(b)

	if _, err := toml.Decode(configString, &Conf); err != nil {
		if err != nil {
			log.Panic(err)
		}
	}

}

//GetFeatures returns all the features in the config file
func GetFeatures() []string {
	var features []string
	for k := range Conf.Commands {
		features = append(features, k)
	}
	return features
}

/*
func GenerateConfig() {
	defConfig := `[telegram]
    TelegramAPIToken = "123456789:jQWERTY1234567890SDFGHJ908765"

[features]
    [features.update]
        commands= ["echo $PATH", "touch iwashere"]
    [features.uname]
        commands= ["uname"]
        expectedOutput = ["Linu."]`

}
*/
