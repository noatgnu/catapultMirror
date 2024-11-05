# Catapult Mirror

Catapult Mirror is a tool designed to monitor directories and mirror files to a destination directory. It includes features such as Slack notifications and database tracking of copied files.

## Features

- **Directory Monitoring**: Monitors specified directories for new files.
- **File Mirroring**: Copies files from source directories to a destination directory.
- **Slack Notifications**: Sends notifications to a Slack channel for various events.
- **Database Tracking**: Tracks copied files in an SQLite database to avoid duplicate copying.
- **Configurable via JSON**: Uses a JSON configuration file for easy setup.

## Configuration

The configuration is done via a JSON file. Below is an example configuration:

```json
{
  "configs": [
    {
      "name": "MRC-Astral",
      "directories": [
        "D:/watch_folder/MRC-Astral"
      ],
      "destinations": [
        "D:/watch_folder/out1",
        "D:/watch_folder/out2"
      ],
      "check_interval": "5s",
      "min_free_space": 10485760000,
      "min_file_size": 1048576
    }
  ],
  "slack_token": "",
  "slack_channel_id": ""
}
```

### Configuration Fields

- **configs**: An array of configuration objects for each directory to monitor.
    - **name**: A name for the configuration.
    - **directories**: An array of directories to monitor.
    - **destinations**: List of destinations to mirror files to.
    - **check_interval**: The interval at which to check the directories for new files.
    - **min_free_space**: The minimum free space required in the destination directory (in bytes).
    - **min_file_size**: The minimum file size required to be copied (in bytes).
- **slack_token**: (Optional) The Slack token for sending notifications.
- **slack_channel_id**: (Optional) The Slack channel ID where notifications will be sent.

## Environment Variables

If the `slack_token` and `slack_channel_id` are not provided in the configuration file, the application will look for the following environment variables:

- `SLACK_TOKEN`
- `SLACK_CHANNEL_ID`

## Usage

### Command Line

```sh
catapultMirror -config=<config_file> -db=<db_file> -log=<log_file>
```

- `-config`: Path to the JSON configuration file.
- `-db`: Path to the SQLite database file (optional).
- `-log`: Path to the log file (optional).

### Example

```sh
catapultMirror -config=config.json -db=file_sizes.db -log=transfer.log
```

## Logging

The application logs important events to both the console and a log file.

## Building

### GitHub Actions

The project includes a GitHub Actions workflow to build and release binaries for multiple platforms.

#### Workflow File: `.github/workflows/release.yaml`

```yaml
on:
  release:
    types: [created]

permissions:
    contents: write
    packages: write

jobs:
  releases-matrix:
    name: Release Go Binary
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux, windows, darwin]
        goarch: ["386", amd64, arm64]
        exclude:
          - goarch: "386"
            goos: darwin
          - goarch: arm64
            goos: windows
    steps:
    - uses: actions/checkout@v4
    - uses: wangyoucao577/go-release-action@v1
      with:
        github_token: ${{ secrets.GITHUB_TOKEN }}
        goos: ${{ matrix.goos }}
        goarch: ${{ matrix.goarch }}
        goversion: "https://dl.google.com/go/go1.23.0.linux-amd64.tar.gz"
        project_path: "."
        binary_name: "catapultMirror"
        extra_files: LICENSE README.md
        build_flags: ${{ matrix.goos == 'windows' && '-tags windows' || '' }}
```

## Running Tests

To run the tests, use the following command:

```sh
go test ./...
```

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.