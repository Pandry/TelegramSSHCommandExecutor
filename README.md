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
    defaultUsername = "pandry"
    defaultPassword = "v3rys3cr3tPassw0rD!"

[features]
    [features.listroot]
        commands = ["ls -la /"]

[knownservers]
    [knownservers.home]
        IP="10.0.1.1:2221"
        Username = "test"
        Password = "testpsw"

[allowedUsers]
    [allowedUsers.pandry]
        Username = "Pandry"
```
**Screenshot**
![](https://vgy.me/UifhNJ.png)



## TODOs
- [X] Read from a config file
- [X] Live-reload the configuration file
- [X] Username whitelist to caps insensitive
- [ ] "Features" sounds bad in the config, change
- [ ] Invert command and status order
- [ ] Ellipsize commands too long in the status messages
- [ ] Create config file if it's not found
- [X] Split report message if exceed the max allowed message size
- [ ] Allow private key authentication
- [ ] Per-host private key
- [ ] Settings to interrupt script execution on error
