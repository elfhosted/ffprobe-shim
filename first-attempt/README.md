# ffprobe-shim

This project provides a simple shim for `ffprobe`, allowing users to adjust the `-probesize` and `-analyzeduration` values when invoking the real `ffprobe` command.

## Usage

To use the `ffprobe-shim`, you can run it from the command line with the desired arguments. The shim will parse the command line options, modify the `-probesize` and `-analyzeduration` values if specified, and then execute the real `ffprobe` with the adjusted arguments.

### Command Line Arguments

- `-probesize <size>`: Set the probe size for the `ffprobe` command. If not specified, the default value will be used.
- `-analyzeduration <duration>`: Set the analyze duration for the `ffprobe` command. If not specified, the default value will be used.

### Example

```bash
./ffprobe-shim -probesize 10000000 -analyzeduration 10000000 <other ffprobe arguments>
```

This command will invoke `ffprobe` with the specified probe size and analyze duration, along with any other arguments you provide.

## Installation

To install the shim, clone the repository and build the project:

```bash
git clone <repository-url>
cd ffprobe-shim
go build -o ffprobe-shim ./cmd/ffprobe-shim
```

## License

This project is licensed under the MIT License. See the LICENSE file for more details.# ffprobe-shim
