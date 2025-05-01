# Censei


**Censei** is a powerful tool for identifying and analyzing open directories on the internet using the Censys API, focusing on efficiency, customization, and detailed output.

## Project Description

### What is Censei?

Censei (pronounced like "Census" + "Eye") is a command-line tool written in Go that uses the Censys API to search for suspicious open directories. It automates the process of identifying hosts, checking their accessibility, crawling directory indexes, and filtering files based on specific criteria.

### Key Features

- Efficient searching for open directories with Censys
- Automatic verification of host accessibility
- Intelligent crawling of directory indexes
- Flexible filtering by file extensions
- Parallel processing for increased speed
- Detailed outputs (raw and filtered)
- Configurable logging
- Interactive and automated mode

### Use Cases

- Security research
- Identification of exposed files
- Open-Source Intelligence (OSINT)
- Security audits
- Automated scanning processes

## Installation

### Prerequisites

- Go 1.20 or higher
- Censys CLI ([Install via pip](https://github.com/censys/censys-command-line))
- **Censys subscription** for API access (important: this tool will not function without valid Censys subscription and API credentials!)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/censei.git
   cd censei
   ```

2. Compile the program:
   ```bash
   go build
   ```

3. Install the Censys CLI:
   ```bash
   pip install censys-command-line
   ```

4. Verify the installation:
   ```bash
   ./censei --help
   ```

### Setting Up Configuration Files

Create two configuration files:

1. **config.json** - Basic settings:
   ```json
   {
     "api_key": "your-censys-api-key",
     "api_secret": "your-censys-api-secret",
     "output_dir": "./output",
     "http_timeout_seconds": 5,
     "max_concurrent_requests": 10,
     "log_level": "INFO",
     "log_file": "./censei.log"
   }
   ```

2. **queries.json** - Predefined queries:
   ```json
   [
     {
       "name": "Russia Suspicious OpenDir",
       "query": "labels:suspicious-open-dir and location.country_code:RU",
       "filters": [".pdf", ".lnk", ".exe", ".elf"]
     },
     {
       "name": "US OpenDir",
       "query": "labels:open-dir and location.country_code:US",
       "filters": [".docx", ".zip", ".xlsx"]
     },
     {
       "name": "Germany Suspicious OpenDir",
       "query": "labels:suspicious-open-dir and location.country_code:DE",
       "filters": [".doc", ".pdf", ".exe"]
     }
   ]
   ```

> **IMPORTANT**: You need a valid Censys subscription to obtain API key and secret. Without these credentials, Censei cannot perform queries. Visit [https://censys.io/plans](https://censys.io/plans) for information about subscriptions.

## Usage

### Basic Usage

Start Censei in interactive mode:

```bash
./censei
```

### Examples for Different Scenarios

**Direct query with specific filters:**
```bash
./censei --query="labels:suspicious-open-dir and location.country_code:RU" --filter=".pdf,.exe"
```

**Query with custom output path:**
```bash
./censei --output=/path/to/results
```

**Query with different log level:**
```bash
./censei --log-level=DEBUG
```

**Using alternative configuration files:**
```bash
./censei --config=/path/to/config.json --queries=/path/to/queries.json
```

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Path to configuration file | `./config.json` |
| `--queries` | Path to queries file | `./queries.json` |
| `--query` | Run a specific query directly | - |
| `--filter` | Specify file extensions to filter (comma-separated) | - |
| `--output` | Override output directory | From config |
| `--log-level` | Set log level (DEBUG, INFO, ERROR) | From config |

### Interactive Mode vs. Direct Queries

When run without a `--query` parameter, Censei starts in interactive mode, presenting a menu of predefined queries from your queries.json file. This mode is user-friendly and ideal for exploration.

For automation or scripts, use the `--query` parameter to run a specific query directly without user interaction.

## Configuration

### config.json Structure

| Parameter | Description | Default |
|-----------|-------------|---------|
| `api_key` | Your Censys API key | - |
| `api_secret` | Your Censys API secret | - |
| `output_dir` | Directory for output files | `./output` |
| `http_timeout_seconds` | Timeout for HTTP requests | `5` |
| `max_concurrent_requests` | Maximum parallel requests | `10` |
| `log_level` | Logging level (DEBUG, INFO, ERROR) | `INFO` |
| `log_file` | Path to log file | `./censei.log` |

### queries.json Structure

The queries.json file contains an array of query objects, each with:

| Field | Description | Example |
|-------|-------------|---------|
| `name` | Display name for the query | `"Russia Suspicious OpenDir"` |
| `query` | Censys search query | `"labels:suspicious-open-dir and location.country_code:RU"` |
| `filters` | Array of file extensions to filter | `[".pdf", ".exe", ".elf"]` |

## Output Files

Censei generates two main output files in the configured output directory:

### raw.txt

Contains all discovered URLs and files, formatted as:

```
http://example.com
Found file: http://example.com/index.html
Found file: http://example.com/data/
Found file: http://example.com/backup.zip
```

### filtered.txt

Contains only files matching the filter criteria:

```
http://example.com/backup.zip
http://example.com/documents/secret.pdf
```

At the end of each file, a summary of the scan is appended with statistics and configuration details.

## Advanced Features

### Customizing Filters

You can customize file filters in three ways:

1. In the queries.json file for predefined queries
2. With the `--filter` command-line option
3. During interactive mode when selecting "Custom query"

The filter format is a comma-separated list of file extensions (e.g., `.pdf,.exe,.zip`).

### Adjusting Log Levels

Available log levels:

- `DEBUG`: Detailed information for troubleshooting
- `INFO`: General operational information
- `ERROR`: Only error messages

Set the log level in config.json or with the `--log-level` parameter.

### Optimizing Parallelization

Adjust the `max_concurrent_requests` setting in config.json based on your system capabilities and network conditions. Higher values increase performance but may cause rate limiting or resource exhaustion.

## Troubleshooting

### Common Issues

**Censys CLI not found:**
```
ERROR: The censys-cli tool was not found. Please install it with:
pip install censys-command-line
```
Solution: Install the Censys CLI as directed.

**API credentials invalid:**
```
Failed to execute Censys query: censys CLI error: Invalid API ID or Secret
```
Solution: Check your API credentials in config.json.

**No results found:**
```
Extracted 0 hosts from Censys results
```
Solution: Verify your query string or try a broader search.

### Debugging Tips

1. Set log level to DEBUG for detailed information:
   ```bash
   ./censei --log-level=DEBUG
   ```

2. Check the generated JSON file to ensure Censys is returning results:
   ```bash
   cat output/censys_results.json
   ```

3. Manually test Censys CLI to verify API functionality:
   ```bash
   censys search "labels:open-dir" --output test.json
   ```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Commit your changes: `git commit -m 'Add some feature'`
4. Push to the branch: `git push origin feature-name`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
