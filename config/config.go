package config

import (
	"io/ioutil"
	"log"

	"github.com/BurntSushi/toml"
)

var Conf config

//LoadDefaultConfig imports the config from the default file (./config.toml)
func LoadDefaultConfig() {
	importConfig("config.toml")
}

func importConfig(path string) {
	b, err := ioutil.ReadFile(path) // just pass the file name
	if err != nil {
		log.Panic(err)
	}
	configString := string(b)

	if _, err := toml.Decode(configString, &Conf); err != nil {
		log.Panic(err)
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
