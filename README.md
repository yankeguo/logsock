# logsock

A tool to create an unix domain socket for logging, it reads log messages from the socket and writes them to the specified file or stdout.

## Usage

```shell
./logsock -listen /var/run/log.sock -out -
./logsock -listen /var/run/log.sock -out /var/log/log.sock
```

## Credits

GUO YANKE, MIT License
