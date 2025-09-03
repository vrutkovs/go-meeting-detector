# Go Meeting Detector

A Go application that monitors PipeWire audio devices to automatically detect when you're in a meeting and updates your status accordingly via MQTT and GNOME Shell Do Not Disturb mode.

## Features

- **PipeWire Integration**: Monitors audio devices using native Go implementation with `pw-cli`
- **MQTT Publishing**: Publishes meeting status to MQTT broker for home automation integration
- **GNOME Shell Integration**: Automatically enables/disables Do Not Disturb mode
- **Structured Logging**: Uses `log/slog` for comprehensive observability
- **Graceful Shutdown**: Handles SIGTERM and SIGINT signals properly
- **Real-time Monitoring**: Checks audio device status every second

## How It Works

The application monitors a specified PipeWire audio node (typically your microphone or audio input device). When the device state changes to "running" (indicating active audio input/output), it assumes you're in a meeting and:

1. Publishes the meeting status to an MQTT topic
2. Enables GNOME Shell's Do Not Disturb mode
3. Logs all activities with structured logging for monitoring

When the audio device becomes inactive, it reverses these actions.

## Requirements

- Go 1.21 or later
- PipeWire audio system
- GNOME Shell (for DND functionality)
- Access to an MQTT broker
- `pw-cli` command-line tool

## Installation

### From Source

```bash
git clone <repository-url>
cd go-meeting-detector
go mod tidy
go build -o go-meeting-detector .
```

### Dependencies

The project uses the following Go modules:
- `log/slog` (standard library) - structured logging
- MQTT client library (imported in the code)

## Configuration

The application is configured entirely through environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `MQTT_HOST` | MQTT broker hostname or IP | `192.168.1.100` |
| `MQTT_PORT` | MQTT broker port | `1883` |
| `MQTT_USER` | MQTT username | `homeassistant` |
| `MQTT_PASSWORD` | MQTT password | `your-password` |
| `MQTT_TOPIC` | MQTT topic to publish to | `home/office/meeting` |
| `PW_NODE_NAME` | PipeWire node name to monitor | `alsa_input.usb-Blue_Microphones_Yeti_Stereo_Microphone` |

### Finding Your PipeWire Node Name

To find the correct node name for your audio device:

```bash
pw-cli ls | grep -A 10 -B 10 "your-device-name"
```

Look for the `node.name` property in the output.

## Usage

### Basic Usage

```bash
export MQTT_HOST="your-mqtt-broker.local"
export MQTT_PORT="1883"
export MQTT_USER="your-username"
export MQTT_PASSWORD="your-password"
export MQTT_TOPIC="home/office/meeting"
export PW_NODE_NAME="your-audio-device-node-name"

./go-meeting-detector
```

### With Docker

Create a `.env` file:

```env
MQTT_HOST=your-mqtt-broker.local
MQTT_PORT=1883
MQTT_USER=your-username
MQTT_PASSWORD=your-password
MQTT_TOPIC=home/office/meeting
PW_NODE_NAME=your-audio-device-node-name
```

### As a Systemd Service

Create `/etc/systemd/system/go-meeting-detector.service`:

```ini
[Unit]
Description=Go Meeting Detector
After=network.target pipewire.service

[Service]
Type=simple
User=your-username
ExecStart=/path/to/go-meeting-detector
Environment=MQTT_HOST=your-mqtt-broker.local
Environment=MQTT_PORT=1883
Environment=MQTT_USER=your-username
Environment=MQTT_PASSWORD=your-password
Environment=MQTT_TOPIC=home/office/meeting
Environment=PW_NODE_NAME=your-audio-device-node-name
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable go-meeting-detector
sudo systemctl start go-meeting-detector
```

## Logging

The application uses structured logging with the following log levels:

- **Debug**: Detailed operational information (device checks, state comparisons)
- **Info**: General application flow (connections, state changes, meeting detection)
- **Warn**: Non-critical issues (device not found temporarily)
- **Error**: Errors that need attention (connection failures, command errors)

### Log Output Example

```json
{
  "time": "2024-01-15T10:30:45.123Z",
  "level": "INFO",
  "msg": "Successfully found PipeWire device ID",
  "nodeName": "alsa_input.usb-Blue_Microphones_Yeti_Stereo_Microphone",
  "deviceID": 42
}
```

## Architecture

### Core Components

- **PipeWireClient**: Handles all PipeWire interactions with structured logging
- **MQTT Client**: Manages MQTT connections and publishing
- **GNOME Shell Integration**: Controls Do Not Disturb mode
- **Main Loop**: Coordinates all components with graceful shutdown

### Error Handling

The application implements comprehensive error handling:
- Retries on temporary failures
- Graceful degradation when services are unavailable
- Detailed error logging for debugging
- Clean shutdown on termination signals

## Troubleshooting

### Common Issues

1. **PipeWire node not found**
   - Verify the node name with `pw-cli ls`
   - Ensure PipeWire is running
   - Check device permissions

2. **MQTT connection failed**
   - Verify broker connectivity: `mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t test -m "hello"`
   - Check credentials and network access

3. **GNOME Shell DND not working**
   - Ensure you're running GNOME Shell (not other DEs)
   - Check that the user has access to D-Bus

### Debug Mode

Set the log level to debug for verbose output:

```bash
export SLOG_LEVEL=DEBUG
./go-meeting-detector
```

### Monitoring

Monitor the application with:

```bash
# View logs
journalctl -u go-meeting-detector -f

# Check MQTT messages
mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t $MQTT_TOPIC

# Monitor PipeWire
pw-cli monitor
```

## Development

### Building

```bash
go build -o go-meeting-detector .
```

### Testing

```bash
go test ./...
```

### Code Structure

```
.
├── main.go              # Application entry point and main loop
├── pipewire.go          # PipeWire integration with structured logging
├── mqtt.go              # MQTT client wrapper
├── gnome.go             # GNOME Shell D-Bus integration
└── go.mod               # Go module dependencies
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with appropriate logging
4. Add tests if applicable
5. Submit a pull request

## License

Apache License 2.0

## Changelog

### v1.0.0
- Initial release with PipeWire monitoring
- MQTT integration
- GNOME Shell Do Not Disturb support
- Structured logging with slog
- Graceful shutdown handling
