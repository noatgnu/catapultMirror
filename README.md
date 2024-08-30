# CatapultMirror

CatapultMirror is a Go-based tool for monitoring directories and mirroring completed files to a destination directory. It uses an SQLite database to keep track of whether a file has been copied or not, ensuring efficient file management.

## Features

- Monitors specified directories for completed files.
- Copies files to a destination directory with a `.cat.part` suffix.
- Verifies the integrity of copied files using SHA-256 hash comparison.
- Renames files to remove the `.cat.part` suffix upon successful verification.
- Deletes the `.cat.part` file if the hash comparison fails.
- Displays copy progress in the terminal.

## Prerequisites

- Go 1.16 or later
- SQLite3

## Installation

1. Clone the repository:

   ```sh
   git clone https://github.com/yourusername/catapultmirror.git
   cd catapultmirror
   ```

2. Build the project:

   ```sh
   go build -o catapultMirror main.go
   ```

## Usage

1. Create a configuration file:

   If the configuration file does not exist, the program will create a template configuration file for you. Run the following command to generate the template:

   ```sh
   ./catapultMirror -config=config.json
   ```

   Fill in the `config.json` file with the directories you want to monitor, the destination directory, and the check interval.

2. Run the program:

   ```sh
   ./catapultMirror -config=config.json -db=file_sizes.db
   ```

    - `-config`: Path to the JSON configuration file.
    - `-db`: Path to the SQLite database file.

## Configuration

The configuration file (`config.json`) should be in the following format:

```json
{
  "directories": ["exampleDir1", "exampleDir2"],
  "destination": "exampleDestinationDir",
  "check_interval": "1m"
}
```

- `directories`: List of directories to monitor.
- `destination`: Directory where completed files will be copied.
- `check_interval`: Interval for checking file completion (e.g., `1m` for 1 minute).

## Example

1. Create a configuration file `config.json`:

   ```json
   {
     "directories": ["/path/to/sourceDir1", "/path/to/sourceDir2"],
     "destination": "/path/to/destinationDir",
     "check_interval": "1m"
   }
   ```

2. Run the program:

   ```sh
   ./catapultMirror -config=config.json -db=file_sizes.db
   ```

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
