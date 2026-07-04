# rsync-srv-gui

`rsync-srv-gui` ist ein kleines Bash-Tool mit Whiptail-Menüoberfläche zur einfachen und kontrollierten Nutzung von `rsync` auf Systemen mit `/srv`-basierten Laufwerken.

Das Tool richtet sich besonders an Raspberry-Pi-, OpenMediaVault-, USB-Storage- und Homelab-Setups, bei denen Daten zwischen eingebundenen Laufwerken, Backup-Zielen oder Service-Verzeichnissen kopiert, verschoben oder synchronisiert werden sollen.

## Funktionen

* Ordnerauswahl per Menü ab `/srv`
* Kopieren neuer und aktualisierter Dateien
* Verschieben von Dateien mit anschließendem Entfernen leerer Quellordner
* Mirror-Synchronisation von Quelle zu Ziel
* Trockenlauf ohne Änderungen
* Fortschrittsanzeige über `rsync --info=progress2`
* Sicherheitsprüfung gegen identische Quelle und Ziel
* Sicherheitsprüfung gegen Zielordner innerhalb der Quelle
* Übersicht und finale Bestätigung vor Ausführung

## Modi

| Modus       | Beschreibung                                                            |
| ----------- | ----------------------------------------------------------------------- |
| Kopieren    | Überträgt neue und aktualisierte Dateien                                |
| Verschieben | Überträgt Dateien und entfernt sie anschließend aus der Quelle          |
| Sync        | Spiegelt die Quelle auf das Ziel und löscht abweichende Dateien im Ziel |
| Test        | Führt einen Trockenlauf ohne Änderungen aus                             |

## Zielgruppe

Dieses Script ist für Linux-Systeme gedacht, auf denen mehrere Laufwerke oder Datenverzeichnisse unter `/srv` eingebunden sind, zum Beispiel:

* Raspberry Pi Server
* OpenMediaVault-Systeme
* USB-HDD-/SSD-Setups
* Backup- und Restore-Szenarien
* Docker- oder Homelab-Speicherstrukturen

## Voraussetzungen

* Linux
* Bash
* rsync
* whiptail

Installation der Abhängigkeiten unter Debian/Ubuntu/Raspberry Pi OS:

```bash
sudo apt update
sudo apt install rsync whiptail
```

## Hinweis

Der Sync-Modus verwendet `rsync --delete`. Dabei wird das Ziel exakt an die Quelle angepasst. Dateien, die nur im Ziel vorhanden sind, können gelöscht werden. Deshalb zeigt das Script vor der Ausführung eine finale Übersicht mit Bestätigung an.
