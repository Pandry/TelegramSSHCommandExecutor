# TelegramSSHCommandExecutor
## Introduction
This program was created to remotely execute pre-made scripts via a telegram bot.

### Demo
**Config file:**
```
[telegram]
    TelegramAPIToken = "*Censored*"

[settings]
    debug = false
    maxMessageColumns = 35
    defaultUsername = "pandry"
    defaultPassword = "v3rys3cr3tPassw0rD!"
    
[features]
    [features.list]
        commands = ["echo $PATH", "notexistingcommand", "cd /var && pwd ", "wget -nv -O /dev/null https://speed.hetzner.de/100MB.bin", "whoami"]
        expectedOutputs = [".*/usr/bin.*", "", "/etc"]

[knownservers]
    [knownservers.home]
        IP="10.0.1.1:2221"
        Username = "test"
        Password = "testpsw"

[allowedUsers]
    [allowedUsers.pandry]
        Username = "Pandry"
```

**Screenshots**   

![](https://vgy.me/8zZ6Wm.png)
![](https://vgy.me/DejKOR.png)



## TODOs
- [X] Read from a config file
- [X] Live-reload the configuration file
- [X] Username whitelist to caps insensitive
- [ ] "Features" sounds bad in the config, change
- [X] Invert command and status order
- [X] Ellipsize commands too long in the status messages
- [X] Create config file if it's not found
- [X] Split report message if exceed the max allowed message size
- [ ] Allow private key authentication
- [ ] Per-host private key
