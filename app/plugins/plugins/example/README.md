# NetScout Plugin: Example Plugin
Message
Repeat Count

This is a plugin for the NetScout-Go network diagnostics tool. It provides An example plugin that demonstrates how to create a modular plugin
A message to echo back
Number of times to repeat the message.

## Installation

To install this plugin, clone this repository into your NetScout-Go plugins directory:

```bash
git clone https://github.com/NetScout-Go/Plugin_example.git ~/.netscout/plugins/example
message
repeat
```

Or use the NetScout-Go plugin manager to install it:

```
// In your NetScout application
pluginLoader.InstallPlugin("https://github.com/NetScout-Go/Plugin_example")
```

## Features

- Network diagnostics for example
- Easy integration with NetScout-Go

## License

MIT License
