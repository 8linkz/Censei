# Censei

**Censei** is a powerful tool for identifying and analyzing open directories on the internet using the Censys API. The focus is on efficiency, adaptability, and detailed output.

## Project Description

### What is Censei?

Censei (pronounced like "Sensei") is a command-line tool written in Go that uses the Censys API to search for suspicious open directories. It automates the process of identifying hosts, checking their availability, crawling directory indexes, and filtering files according to specific criteria.

### Key Features

- Efficient search for open directories with Censys
- Automatic host availability checking
- Intelligent crawling of directory indexes
- Recursive directory scanning with configurable depth
- Flexible filtering by file extensions
- Parallel processing for higher speed
- Smart host blocking with persistent blocklist
- Performance limits to prevent resource exhaustion
- Detailed outputs (raw and filtered)
- Configurable logging
- Interactive and automated modes
- Special verification mode for binary files without HTML processing
- File content verification for binary detection
- Target-specific file search functionality

### Use Cases

- Security research
- Identification of exposed files
- Open-Source Intelligence (OSINT)
- Security audits
- Automated scanning processes

## Installation

### Prerequisites

- Go 1.20 or higher
- Censys CLI ([Installation via pip](https://github.com/censys/censys-command-line))
- **Censys subscription** for API access (important: This tool does not work without a valid Censys subscription and API credentials!)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/8linkz/censei.git
   cd censei
   ```

2. Compile the program:
   ```bash
   go build
   ```

3. Install Censys CLI:
   ```bash
   pip install censys-command-line
   ```

4. Verify installation:
   ```bash
   ./censei --help
   ```

### Setting up configuration files

Create two configuration files:

1. **config.json** - Basic settings:
   ```json
   {
     "api_key": "your-censys-api-key",
     "api_secret": "your-censys-api-secret",
     "output_dir": "./output",
     "binary_output_file": "./output/binary_found.txt",
     "http_timeout_seconds": 5,
     "max_concurrent_requests": 10,
     "log_level": "INFO",
     "log_file": "./censei.log",
     "max_links_per_directory": 500,
     "max_total_links": 10000,
     "max_skips_before_block": 5,
     "enable_blocklist": true,
     "blocklist_file": "./blocked_hosts.txt"
   }
   ```

2. **queries.json** - Predefined queries:
   ```json
   [
     {
       "name": "Russia Suspicious OpenDir",
       "query": "labels:suspicious-open-dir and location.country_code:RU",
       "recursive": "yes",
       "max-depth": 3,
       "filters": [".pdf", ".lnk", ".exe", ".elf"]
     },
     {
       "name": "US OpenDir",
       "query": "labels:open-dir and location.country_code:US",
       "recursive": "yes",
       "max-depth": 3,
       "filters": [".docx", ".zip", ".xlsx"]
     },
     {
       "name": "Cobalt Strike Scanner",
       "query": "services.labels:`open-dir` and services.labels:`c2`",
       "filters": [],
       "check": true,
       "target_filename": "02.08.2022.exe"
     }
   ]
   ```

> **IMPORTANT**: You need a valid Censys subscription to obtain API key and secret. Without these credentials, Censei cannot perform queries. Visit [https://censys.io/plans](https://censys.io/plans) for subscription information.

## Usage

### Basic Usage

Start Censei in interactive mode:

```bash
./censei
```

### Examples for different scenarios

**Direct query with specific filters:**
```bash
./censei --query="labels:suspicious-open-dir and location.country_code:RU" --filter=".pdf,.exe"
```

**Recursive scanning with depth limit:**
```bash
./censei --recursive --max-depth=3 --query="labels:open-dir and location.country_code:US"
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

**Enabling File Checker mode:**
```bash
./censei --check --target-file="suspicious.exe"
```

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Path to configuration file | `./config.json` |
| `--queries` | Path to queries file | `./queries.json` |
| `--query` | Direct execution of a specific query | - |
| `--filter` | Specification of file extensions to filter (comma-separated) | - |
| `--output` | Override output directory | From configuration |
| `--log-level` | Set log level (DEBUG, INFO, ERROR) | From configuration |
| `--check` | Enables the File Checker mode - skips HTML processing and link extraction, instead checks hosts directly for specific binary files | `false` |
| `--target-file` | Specifies the specific file to search for in File Checker mode | - |
| `--recursive` | Enable recursive directory scanning | `false` |
| `--max-depth` | Maximum depth for recursive scanning (requires --recursive) | `1` |

### Interactive Mode vs. Direct Queries

When the tool is started without the `--query` parameter, Censei starts in interactive mode and presents a menu with predefined queries from your queries.json file. This mode is user-friendly and ideal for exploration.

For automation or scripts, use the `--query` parameter to execute a specific query directly without user interaction.

## Configuration

### config.json Structure

| Parameter | Description | Default |
|-----------|-------------|---------|
| `api_key` | Your Censys API key | - |
| `api_secret` | Your Censys API secret | - |
| `output_dir` | Directory for output files | `./output` |
| `binary_output_file` | Path for binary file outputs | `./output/binary_found.txt` |
| `http_timeout_seconds` | Timeout for HTTP requests | `5` |
| `max_concurrent_requests` | Maximum parallel requests | `10` |
| `log_level` | Logging level (DEBUG, INFO, ERROR) | `INFO` |
| `log_file` | Path to log file | `./censei.log` |
| `max_links_per_directory` | Maximum links to process per directory | `500` |
| `max_total_links` | Total link limit per host before skipping | `10000` |
| `max_skips_before_block` | Number of skips before blocking entire host | `5` |
| `enable_blocklist` | Enable persistent host blocking functionality | `true` |
| `blocklist_file` | Path to file storing permanently blocked hosts | `./blocked_hosts.txt` |

### queries.json Structure

The queries.json file contains an array of query objects, each with:

| Field | Description | Example |
|-------|-------------|---------|
| `name` | Display name for the query | `"Russia Suspicious OpenDir"` |
| `query` | Censys search query | `"labels:suspicious-open-dir and location.country_code:RU"` |
| `filters` | Array of file extensions to filter | `[".pdf", ".exe", ".elf"]` |
| `check` | Enables File Checker mode for this query | `true` |
| `target_filename` | Specific file to search for in File Checker mode | `"02.08.2022.exe"` |
| `recursive` | Enable recursive scanning ("yes"/"no") | `"yes"` |
| `max-depth` | Maximum scanning depth for recursive mode | `3` |

## Output Files

Censei generates three main output files in the configured output directory:

### raw.txt

Contains all discovered URLs and files, formatted as:

```
http://example.com
Found file: http://example.com/index.html
Found file: http://example.com/data/
Found file: http://example.com/backup.zip
```

With File Checker mode enabled, binary files found will be displayed as:

```
http://example.com
Found binary file: http://example.com/02.08.2022.exe with Content-Type: application/x-msdownload
```

### filtered.txt

Contains only files that match the filter criteria:

```
http://example.com/backup.zip
http://example.com/documents/secret.pdf
```

### binary_found.txt

Contains paths to binary files identified during scanning:

```
http://example.com/02.08.2022.exe with Content-Type: application/x-msdownload
http://example.com/tools/app.exe with Content-Type: application/octet-stream
```

At the end of each file, a summary of the scan with statistics and configuration details is appended.

## Advanced Features

### Recursive Directory Scanning

Censei supports recursive directory scanning to automatically explore subdirectories:

- **Enable recursion**: Set `"recursive": "yes"` in queries.json or use `--recursive` flag
- **Control depth**: Configure `"max-depth": 3` or use `--max-depth=3` to limit scanning depth
- **Performance protection**: Built-in limits prevent infinite recursion and resource exhaustion

### Smart Host Blocking

The tool includes intelligent host management to improve scanning efficiency:

- **Automatic blocking**: Hosts exceeding limits are automatically blocked
- **Persistent blocklist**: Blocked hosts are saved to file and persist between sessions
- **Configurable thresholds**: Adjust `max_skips_before_block` to control blocking sensitivity
- **Performance optimization**: Prevents wasting time on problematic hosts

### Performance Limits

Built-in safeguards prevent resource exhaustion:

- **Per-directory limits**: `max_links_per_directory` controls links processed per directory
- **Total link limits**: `max_total_links` sets overall limit per host
- **Memory protection**: Prevents excessive memory usage during large directory scans

### File Checker Mode

The File Checker mode is a special operating mode optimized for targeted searching for binary files:

- Activated by `check: true` in queries.json or `--check` on the command line
- Completely skips HTML processing and link extraction
- Checks each host only for the presence of a specific file (specified by `target_filename`)
- Checks only small parts of the file (headers) to determine the file type
- Does not save files to disk
- Optimized for quick identification of potentially harmful binary files

This mode is especially useful for security analysts looking for specific binary files without having to search through entire directory contents.

### Binary File Detection

Censei can detect binary files based on their Content-Type headers, including:
- application/octet-stream
- application/x-executable
- application/x-msdos-program
- application/x-msdownload
- application/exe
- application/binary

The tool checks HTTP headers first (using HEAD requests) to efficiently determine file types without downloading large files.

### Customizing Filters

You can customize filters in three ways:

1. In the queries.json file for predefined queries
2. With the `--filter` option on the command line
3. In interactive mode when selecting "Custom query"

The filter format is a comma-separated list of file extensions (e.g. `.pdf,.exe,.zip`).

### Customizing Log Levels

Available log levels:

- `DEBUG`: Detailed information for troubleshooting
- `INFO`: General operational information
- `ERROR`: Error messages only

Set the log level in config.json or with the `--log-level` parameter.

### Optimizing Parallelization

Adjust the `max_concurrent_requests` setting in config.json based on your system capabilities and network conditions. Higher values increase performance but may lead to rate limiting or resource exhaustion.

## Troubleshooting

### Common Problems

**Censys CLI not found:**
```
ERROR: The censys-cli tool was not found. Please install it with:
pip install censys-command-line
```
Solution: Install the Censys CLI as indicated.

**API credentials invalid:**
```
Failed to execute Censys query: censys CLI error: Invalid API ID or Secret
```
Solution: Check your API credentials in config.json.

**No results found:**
```
Extracted 0 hosts from Censys results
```
Solution: Check your query string or try a broader search.

### Debugging Tips

1. Set the log level to DEBUG for detailed information:
   ```bash
   ./censei --log-level=DEBUG
   ```

2. Check the generated JSON file to ensure Censys is returning results:
   ```bash
   cat output/censys_results.json
   ```

3. Test the Censys CLI manually to verify API functionality:
   ```bash
   censys search "labels:open-dir" --output test.json
   ```

## License

This project is licensed under the MIT License - see the LICENSE file for details.