# Catapult Mirror

Catapult Mirror is a Go application that monitors specified directories and mirrors completed files to a destination directory. It ensures that files are fully copied and verified before being moved to their final destination.

## Features

- Monitors multiple directories for new files.
- Copies completed files to a destination directory.
- Verifies file integrity using SHA-256 hash.
- Logs important events to both console and a log file.
- Gracefully shuts down on receiving OS signals or when disk space is low.
- Deletes partially copied files on shutdown.

## Configuration

The application is configured using a JSON file. Below is an example configuration:

```json
{
    "directories": ["D:/watch_folder/MRC-Astral"],
    "destination": "D:/watch_folder",
    "check_interval": "5s",
    "min_free_space": 10485760000
}
```

- `directories`: List of directories to monitor.
- `destination`: Directory where completed files will be copied.
- `check_interval`: Interval at which directories are checked for new files.
- `min_free_space`: Minimum free space (in bytes) required on the destination drive.

## Usage

### Command Line

```sh
catapultMirror -config=<config_file> -db=<db_file> -log=<log_file>
```

- `-config`: Path to the JSON configuration file.
- `-db`: Path to the SQLite database file.
- `-log`: Path to the log file.

### Example

```sh
catapultMirror -config=config.json -db=file_sizes.db -log=transfer.log
```

## Logging

The application logs important events to both the console and a log file. The log file does not record transfer progress but only what has finished transferring.

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

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.