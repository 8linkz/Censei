# Censei

**Censei** ist ein leistungsstarkes Tool zur Identifizierung und Analyse von offenen Verzeichnissen im Internet mithilfe der Censys API. Der Fokus liegt dabei auf Effizienz, Anpassungsfähigkeit und detaillierter Ausgabe.

## Projektbeschreibung

### Was ist Censei?

Censei (ausgesprochen wie "Sensei") ist ein Kommandozeilen-Tool, geschrieben in Go, das die Censys API verwendet, um nach verdächtigen offenen Verzeichnissen zu suchen. Es automatisiert den Prozess der Identifizierung von Hosts, der Überprüfung ihrer Erreichbarkeit, dem Crawlen von Verzeichnisindizes und dem Filtern von Dateien nach bestimmten Kriterien.

### Hauptfunktionen

- Effiziente Suche nach offenen Verzeichnissen mit Censys
- Automatische Überprüfung der Host-Erreichbarkeit
- Intelligentes Crawlen von Verzeichnisindizes
- Flexibles Filtern nach Dateierweiterungen
- Parallele Verarbeitung für höhere Geschwindigkeit
- Detaillierte Ausgaben (raw und gefiltert)
- Konfigurierbares Logging
- Interaktiver und automatisierter Modus
- Spezieller Überprüfungsmodus für binäre Dateien ohne HTML-Verarbeitung

### Anwendungsfälle

- Sicherheitsforschung
- Identifizierung von exponierten Dateien
- Open-Source Intelligence (OSINT)
- Sicherheitsaudits
- Automatisierte Scan-Prozesse

## Installation

### Voraussetzungen

- Go 1.20 oder höher
- Censys CLI ([Installation über pip](https://github.com/censys/censys-command-line))
- **Censys-Abonnement** für API-Zugriff (wichtig: Dieses Tool funktioniert nicht ohne gültiges Censys-Abonnement und API-Zugangsdaten!)

### Installation

1. Repository klonen:
   ```bash
   git clone https://github.com/8linkz/censei.git
   cd censei
   ```

2. Programm kompilieren:
   ```bash
   go build
   ```

3. Censys CLI installieren:
   ```bash
   pip install censys-command-line
   ```

4. Installation überprüfen:
   ```bash
   ./censei --help
   ```

### Konfigurationsdateien einrichten

Erstellen Sie zwei Konfigurationsdateien:

1. **config.json** - Grundeinstellungen:
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

2. **queries.json** - Vordefinierte Abfragen:
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

> **WICHTIG**: Sie benötigen ein gültiges Censys-Abonnement, um API-Schlüssel und -Secret zu erhalten. Ohne diese Zugangsdaten kann Censei keine Abfragen durchführen. Besuchen Sie [https://censys.io/plans](https://censys.io/plans) für Informationen zu Abonnements.

## Verwendung

### Grundlegende Verwendung

Starten Sie Censei im interaktiven Modus:

```bash
./censei
```

### Beispiele für verschiedene Szenarien

**Direkte Abfrage mit spezifischen Filtern:**
```bash
./censei --query="labels:suspicious-open-dir and location.country_code:RU" --filter=".pdf,.exe"
```

**Abfrage mit benutzerdefiniertem Ausgabepfad:**
```bash
./censei --output=/path/to/results
```

**Abfrage mit anderem Log-Level:**
```bash
./censei --log-level=DEBUG
```

**Verwendung alternativer Konfigurationsdateien:**
```bash
./censei --config=/path/to/config.json --queries=/path/to/queries.json
```

**Aktivierung des Filechecker-Modus:**
```bash
./censei --check --target-file="suspicious.exe"
```

### Kommandozeilen-Optionen

| Option | Beschreibung | Standard |
|--------|-------------|---------|
| `--config` | Pfad zur Konfigurationsdatei | `./config.json` |
| `--queries` | Pfad zur Abfragedatei | `./queries.json` |
| `--query` | Direkte Ausführung einer spezifischen Abfrage | - |
| `--filter` | Angabe von zu filternden Dateierweiterungen (kommagetrennt) | - |
| `--output` | Überschreiben des Ausgabeverzeichnisses | Aus der Konfiguration |
| `--log-level` | Log-Level setzen (DEBUG, INFO, ERROR) | Aus der Konfiguration |
| `--check` | Aktiviert den Filechecker-Modus - überspringt HTML-Verarbeitung und Link-Extraktion, prüft stattdessen Hosts direkt auf bestimmte Binärdateien | `false` |
| `--target-file` | Gibt die spezifische Datei an, nach der im Filechecker-Modus gesucht werden soll | - |

### Interaktiver Modus vs. Direkte Abfragen

Wenn das Tool ohne den `--query`-Parameter gestartet wird, startet Censei im interaktiven Modus und präsentiert ein Menü mit vordefinierten Abfragen aus Ihrer queries.json-Datei. Dieser Modus ist benutzerfreundlich und ideal für die Exploration.

Für Automatisierung oder Skripte verwenden Sie den `--query`-Parameter, um eine bestimmte Abfrage direkt ohne Benutzerinteraktion auszuführen.

## Konfiguration

### config.json-Struktur

| Parameter | Beschreibung | Standard |
|-----------|-------------|---------|
| `api_key` | Ihr Censys API-Schlüssel | - |
| `api_secret` | Ihr Censys API-Secret | - |
| `output_dir` | Verzeichnis für Ausgabedateien | `./output` |
| `http_timeout_seconds` | Timeout für HTTP-Anfragen | `5` |
| `max_concurrent_requests` | Maximale parallele Anfragen | `10` |
| `log_level` | Logging-Level (DEBUG, INFO, ERROR) | `INFO` |
| `log_file` | Pfad zur Log-Datei | `./censei.log` |

### queries.json-Struktur

Die queries.json-Datei enthält ein Array von Abfrageobjekten, jedes mit:

| Feld | Beschreibung | Beispiel |
|-------|-------------|---------|
| `name` | Anzeigename für die Abfrage | `"Russia Suspicious OpenDir"` |
| `query` | Censys-Suchabfrage | `"labels:suspicious-open-dir and location.country_code:RU"` |
| `filters` | Array von zu filternden Dateierweiterungen | `[".pdf", ".exe", ".elf"]` |
| `check` | Aktiviert den Filechecker-Modus für diese Abfrage | `true` |
| `target_filename` | Spezifische Datei, nach der im Filechecker-Modus gesucht werden soll | `"02.08.2022.exe"` |

## Ausgabedateien

Censei generiert zwei Hauptausgabedateien im konfigurierten Ausgabeverzeichnis:

### raw.txt

Enthält alle entdeckten URLs und Dateien, formatiert als:

```
http://example.com
Found file: http://example.com/index.html
Found file: http://example.com/data/
Found file: http://example.com/backup.zip
```

Bei aktiviertem Filechecker-Modus werden gefundene binäre Dateien wie folgt angezeigt:

```
http://example.com
Found binary file: http://example.com/02.08.2022.exe with Content-Type: application/x-msdownload
```

### filtered.txt

Enthält nur Dateien, die den Filterkriterien entsprechen:

```
http://example.com/backup.zip
http://example.com/documents/secret.pdf
```

Am Ende jeder Datei wird eine Zusammenfassung des Scans mit Statistiken und Konfigurationsdetails angehängt.

## Erweiterte Funktionen

### Filechecker-Modus

Der Filechecker-Modus ist eine spezielle Betriebsart, die für die gezielte Suche nach binären Dateien optimiert ist:

- Aktiviert durch `check: true` in queries.json oder `--check` auf der Kommandozeile
- Überspringt HTML-Verarbeitung und Link-Extraktion vollständig
- Prüft jeden Host nur auf das Vorhandensein einer bestimmten Datei (angegeben durch `target_filename`)
- Überprüft nur kleine Teile der Datei (Header), um den Dateityp zu bestimmen
- Speichert keine Dateien auf der Festplatte
- Optimiert für die schnelle Identifizierung von potenziell schädlichen Binärdateien

Dieser Modus ist besonders nützlich für Sicherheitsanalysten, die nach bestimmten binären Dateien suchen, ohne den gesamten Inhalt von Verzeichnissen zu durchsuchen.

### Anpassen von Filtern

Sie können Filter auf drei Arten anpassen:

1. In der queries.json-Datei für vordefinierte Abfragen
2. Mit der Option `--filter` auf der Kommandozeile
3. Im interaktiven Modus bei der Auswahl von "Custom query"

Das Filterformat ist eine kommagetrennte Liste von Dateierweiterungen (z.B. `.pdf,.exe,.zip`).

### Anpassen der Log-Level

Verfügbare Log-Level:

- `DEBUG`: Detaillierte Informationen zur Fehlerbehebung
- `INFO`: Allgemeine Betriebsinformationen
- `ERROR`: Nur Fehlermeldungen

Setzen Sie das Log-Level in config.json oder mit dem Parameter `--log-level`.

### Optimierung der Parallelisierung

Passen Sie die Einstellung `max_concurrent_requests` in config.json basierend auf Ihren Systemfähigkeiten und Netzwerkbedingungen an. Höhere Werte erhöhen die Leistung, können aber zu Rate-Limiting oder Ressourcenerschöpfung führen.

## Fehlerbehebung

### Häufige Probleme

**Censys CLI nicht gefunden:**
```
ERROR: The censys-cli tool was not found. Please install it with:
pip install censys-command-line
```
Lösung: Installieren Sie die Censys CLI wie angegeben.

**API-Zugangsdaten ungültig:**
```
Failed to execute Censys query: censys CLI error: Invalid API ID or Secret
```
Lösung: Überprüfen Sie Ihre API-Zugangsdaten in config.json.

**Keine Ergebnisse gefunden:**
```
Extracted 0 hosts from Censys results
```
Lösung: Überprüfen Sie Ihren Abfragestring oder versuchen Sie eine breitere Suche.

### Debugging-Tipps

1. Setzen Sie das Log-Level auf DEBUG für detaillierte Informationen:
   ```bash
   ./censei --log-level=DEBUG
   ```

2. Überprüfen Sie die generierte JSON-Datei, um sicherzustellen, dass Censys Ergebnisse zurückgibt:
   ```bash
   cat output/censys_results.json
   ```

3. Testen Sie die Censys CLI manuell, um die API-Funktionalität zu überprüfen:
   ```bash
   censys search "labels:open-dir" --output test.json
   ```

## Beitragen

Beiträge sind willkommen! Bitte folgen Sie diesen Schritten:

1. Forken Sie das Repository
2. Erstellen Sie einen Feature-Branch: `git checkout -b feature-name`
3. Committen Sie Ihre Änderungen: `git commit -m 'Add some feature'`
4. Pushen Sie zum Branch: `git push origin feature-name`
5. Reichen Sie einen Pull Request ein

## Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert - siehe die LICENSE-Datei für Details.